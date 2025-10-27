# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import abc
import contextlib
import dataclasses
import enum
import functools
import hashlib
import itertools
import json
import os
import shlex
import shutil
import subprocess
import sys
import tempfile
import threading
import time

import typing as t

from collections import abc as cabc
from pathlib import Path, PurePath, PurePosixPath

import arvados
import arvados_fuse as fuse
import googleapiclient
import pytest

from . import run_test_server
from .mount_test_base import MountTestBase

@pytest.fixture(scope='module', autouse=True)
def keepstore_servers():
    cmd = [
        sys.executable,
        run_test_server.__file__,
        'start_keep',
         '--keep-blob-signing',
         '--num-keep-servers=2',
    ]
    yield subprocess.run(cmd, check=True, stdin=subprocess.DEVNULL)
    cmd[2] = 'stop_keep'
    subprocess.run(cmd, check=True, stdin=subprocess.DEVNULL)


def _cmd2str(cmd):
    return ' '.join(shlex.quote(s) for s in cmd)


@dataclasses.dataclass
class MountProc:
    env: cabc.Mapping[str, str]
    cmd: t.Sequence[str]
    mount_subdir: PurePath
    _mount_root: Path | None = dataclasses.field(init=False, default=None)
    _mount_proc: subprocess.Popen | None = dataclasses.field(init=False, default=None)

    BIN_PATH: t.ClassVar[Path] = Path(__file__).parent.parent / 'bin/arv-mount'

    @classmethod
    def for_collection(cls, env, coll_id, *, opts=()):
        cmd = [
            sys.executable, str(cls.BIN_PATH),
            '--foreground',
            '--ram-cache',
            '--read-write',
            '--refresh-time=0',
            '--collection', coll_id,
            *opts,
        ]
        return cls(env, cmd, PurePath('.'))

    @classmethod
    def for_tmp(cls, env, subdir=PurePath('tmp'), *, opts=()):
        cmd = [
            sys.executable, str(cls.BIN_PATH),
            '--foreground',
            '--ram-cache',
            '--read-write',
            '--refresh-time=0',
            '--mount-tmp', str(subdir),
            *opts,
        ]
        return cls(env, cmd, subdir)

    def __enter__(self):
        env = {
            key: val
            for key, val in os.environ.items()
            if not key.startswith('ARVADOS_API_')
        }
        env.update(self.env)
        self._mount_root = Path(tempfile.mkdtemp(prefix='arv-mount-'))
        cmd = list(self.cmd)
        cmd.append(str(self._mount_root))
        self._mount_proc = subprocess.Popen(cmd, env=env, stdin=subprocess.DEVNULL)

        retry_time = .2
        for retry_count in range(round(10 / retry_time)):
            if self._mount_root.is_mount():
                break
            time.sleep(retry_time)
        else:
            # The mount didn't come up in time. Do our best to clean up, but
            # don't assume anything is going to work.
            subprocess.Popen(
                ['fusermount', '-u', '-z', str(self._mount_root)],
                stdin=subprocess.DEVNULL,
            )
            assert self._mount_proc.wait(timeout=10) == os.EX_OK
            raise Exception("mount did not come up in time (but exited OK?)")

        return self

    def __exit__(self, exc_type, exc_value, exc_tb):
        try:
            subprocess.run(
                ['fusermount', '-u', '-z', str(self._mount_root)],
                stdin=subprocess.DEVNULL,
                check=True,
                timeout=60,
            )
            self._mount_proc.wait(timeout=60)
        except subprocess.CalledProcessError as err:
            pytest.exit(2, f"command `{_cmd2str(err.cmd)}` exited {err.returncode}")
        except subprocess.TimeoutExpired as err:
            pytest.exit(2, f"command `{_cmd2str(err.cmd)}` did not finish within {err.timeout} seconds")
        self._mount_root.rmdir()
        self._mount_root = None
        try:
            assert self._mount_proc.returncode == os.EX_OK
        finally:
            self._mount_proc = None

    @property
    def mount_path(self):
        return self._mount_root / self.mount_subdir


@dataclasses.dataclass
class AbstractChange(metaclass=abc.ABCMeta):
    mount_path: Path
    arv_client: googleapiclient.discovery.Resource | None = None
    coll_uuid: str | None = None
    filename: PurePath = PurePath('bar')

    @abc.abstractmethod
    def change_thread(self): ...
    @abc.abstractmethod
    def check_mount(self): ...
    @abc.abstractmethod
    def check_record(self):
        assert self.arv_client is not None
        assert self.coll_uuid is not None

    def check_all(self):
        self.check_mount()
        self.check_record()

    def _replace_files_thread(self, dst, src, body=None):
        request = self.arv_client.collections().update(
            uuid=self.coll_uuid,
            body=body or {},
            replace_files={
                str(PurePosixPath('/', dst)): src,
            },
        )
        return threading.Thread(target=request.execute)


@dataclasses.dataclass
class AddInMount(AbstractChange):
    filename: PurePath = PurePath('FUSEbar')
    _CONTENT: t.ClassVar[str] = b'bar'

    def change_thread(self):
        path = self.mount_path / self.filename
        return threading.Thread(target=path.write_bytes, args=(self._CONTENT,))

    def check_mount(self):
        path = self.mount_path / self.filename
        assert path.stat().st_size == 3
        assert path.read_bytes() == self._CONTENT

    def check_record(self):
        super().check_record()
        coll = arvados.collection.CollectionReader(self.coll_uuid, self.arv_client)
        with coll.open(str(self.filename), 'rb') as coll_file:
            assert coll_file.read() == self._CONTENT


@dataclasses.dataclass
class AddInRecord(AddInMount):
    filename: PurePath = PurePath('APIbar')

    def change_thread(self):
        return self._replace_files_thread(self.filename, 'current/bar')


@dataclasses.dataclass
class DelInMount(AbstractChange):
    def change_thread(self):
        path = self.mount_path / self.filename
        return threading.Thread(target=path.unlink, kwargs={'missing_ok': True})

    def check_mount(self):
        assert not (self.mount_path / self.filename).exists()

    def check_record(self):
        super().check_record()
        coll = arvados.collection.CollectionReader(self.coll_uuid, self.arv_client)
        assert self.filename not in coll


@dataclasses.dataclass
class DelInRecord(DelInMount):
    def change_thread(self):
        return self._replace_files_thread(self.filename, '')


@dataclasses.dataclass
class ModInMount(AbstractChange):
    _ORIG_CONTENT: t.ClassVar[bytes] = b'bar'
    _NEW_CONTENT: t.ClassVar[bytes] = b'FUSE'

    def change_thread(self):
        def append_bytes(path, content):
            with path.open('ab') as f:
                f.write(content)
        path = self.mount_path / self.filename
        return threading.Thread(target=append_bytes, args=(path, self._NEW_CONTENT))

    def check_mount(self):
        expected = self._ORIG_CONTENT + self._NEW_CONTENT
        path = self.mount_path / self.filename
        assert path.stat().st_size == len(expected)
        assert path.read_bytes() == expected

    def check_record(self):
        super().check_record()
        expected = self._ORIG_CONTENT + self._NEW_CONTENT
        coll = arvados.collection.CollectionReader(self.coll_uuid, self.arv_client)
        with coll.open(str(self.filename), 'rb') as coll_file:
            assert coll_file.read() == expected


@dataclasses.dataclass
class ModInRecord(ModInMount):
    _NEW_CONTENT: t.ClassVar[bytes] = b'API'
    _MANIFEST_TEXT: t.ClassVar[str | None] = None

    def change_thread(self):
        if self._MANIFEST_TEXT is None:
            coll = arvados.collection.Collection(self.coll_uuid, self.arv_client)
            coll.clone()
            with coll.open(str(self.filename), 'ab') as coll_file:
                coll_file.write(self._NEW_CONTENT)
            type(self)._MANIFEST_TEXT = coll.manifest_text()
        src = str(PurePosixPath('manifest_text', self.filename))
        return self._replace_files_thread(self.filename, src, body={
            'manifest_text': self._MANIFEST_TEXT,
        })


def _config_with_token(token_name):
    env = {
        'ARVADOS_API_HOST': os.environ['ARVADOS_API_HOST'],
        'ARVADOS_API_TOKEN': run_test_server.auth_token(token_name),
    }
    try:
        env['ARVADOS_API_HOST_INSECURE'] = os.environ['ARVADOS_API_HOST_INSECURE']
    except KeyError:
        pass
    return env


@pytest.fixture
def active_env():
    return _config_with_token('active')


@pytest.fixture(scope='session')
def git_src():
    try:
        workspace = Path(os.environ['WORKSPACE'])
    except (KeyError, ValueError):
        workspace_ok = False
    else:
        workspace_ok = workspace.is_dir()
    if not workspace_ok:
        raise ValueError("$WORKSPACE does not refer to a directory")
    git_proc = subprocess.run(
        ['git', 'rev-parse', '--git-dir'],
        capture_output=True,
        check=True,
        cwd=workspace,
        text=True,
    )
    git_path = Path(git_proc.stdout.removesuffix('\n'))
    if git_path.is_absolute():
        return git_path
    else:
        return workspace / git_path
    

def new_coll(arv_client, fixture_name='collection_owned_by_active'):
    coll_record = run_test_server.fixture('collections')[fixture_name]
    coll = arvados.collection.Collection(coll_record['uuid'], arv_client)
    coll.save_new()
    return coll


def run_changes(*changes):
    threads = [c.change_thread() for c in changes]
    for t in threads:
        t.start()
    errors = []
    for t in threads:
        try:
            t.join(timeout=10)
        except Exception as err:
            errors.append(err)
    assert not errors


@pytest.mark.parametrize('mount_ct,record_ct', itertools.product(
    [AddInMount, DelInMount, ModInMount],
    [AddInRecord, DelInRecord, ModInRecord],
))
def test_simultaneous_api_mount_updates(active_env, mount_ct, record_ct):
    if issubclass(mount_ct, ModInMount):
        pytest.skip(
            "TODO: mount writes usually fail with inconsistent state - "
            "this should probably pass",
        )
    arv_client = arvados.api.api_from_config('v1', active_env)
    coll_uuid = new_coll(arv_client).manifest_locator()
    with MountProc.for_collection(active_env, coll_uuid) as mount:
        mount_change = mount_ct(mount.mount_path, arv_client, coll_uuid)
        record_change = record_ct(mount.mount_path, arv_client, coll_uuid)
        run_changes(mount_change, record_change)
        # In this case, given that FUSE does not do idempotent writes, there is
        # always a possibility that it simply overwrites the API record change.
        # Therefore we only check that the mount change took.
        mount_change.check_all()


@pytest.mark.parametrize('change_type', [AddInMount, DelInMount, ModInMount])
def test_simultaneous_coll_mount_updates(active_env, change_type):
    arv_client = arvados.api.api_from_config('v1', active_env)
    coll_uuid = new_coll(arv_client).manifest_locator()
    with MountProc.for_collection(active_env, coll_uuid) as mount:
        add = AddInMount(mount.mount_path, arv_client, coll_uuid, change_type.__name__)
        change = change_type(mount.mount_path, arv_client, coll_uuid)
        run_changes(add, change)
        add.check_all()
        change.check_all()


@pytest.mark.parametrize('change_type', [AddInMount, DelInMount, ModInMount])
def test_simultaneous_tmp_mount_updates(active_env, change_type):
    with MountProc.for_tmp(active_env) as mount:
        (mount.mount_path / 'bar').write_bytes(b'bar')
        add = AddInMount(mount.mount_path, filename=change_type.__name__)
        change = change_type(mount.mount_path)
        run_changes(add, change)
        add.check_mount()
        change.check_mount()


@pytest.mark.skip("TODO: this test should probably pass but never has")
def test_git_clone_to_coll(active_env, git_src):
    arv_client = arvados.api.api_from_config('v1', active_env)
    coll = new_coll(arv_client, 'empty_collection_name_in_active_user_home_project')
    with MountProc.for_collection(active_env, coll.manifest_locator()) as mount:
        git_proc = subprocess.run([
            'git', 'clone',
            '--jobs=3',
            '--no-hardlinks',
            '--quiet',
            str(git_src),
            str(mount.mount_path),
        ], stdin=subprocess.DEVNULL)
    # assert outside the `with` block because if arv-mount exits nonzero,
    # that's a more interesting failure to report.
    assert git_proc.returncode == os.EX_OK


def test_git_clone_to_tmp(active_env, git_src):
    with MountProc.for_tmp(active_env) as mount:
        git_proc = subprocess.run([
            'git', 'clone',
            '--jobs=3',
            '--no-hardlinks',
            '--quiet',
            str(git_src),
            str(mount.mount_path),
        ], stdin=subprocess.DEVNULL)
    # assert outside the `with` block because if arv-mount exits nonzero,
    # that's a more interesting failure to report.
    assert git_proc.returncode == os.EX_OK
