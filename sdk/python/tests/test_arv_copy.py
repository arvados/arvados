# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import itertools
import os
import sys
import tempfile
import unittest
import shutil
import arvados.api
import arvados.util
from arvados.collection import Collection, CollectionReader

import pytest

import arvados.commands.arv_copy as arv_copy
from arvados._internal import basedirs
from . import arvados_testutil as tutil
from . import run_test_server

class ArvCopyVersionTestCase(run_test_server.TestCaseWithServers, tutil.VersionChecker):
    MAIN_SERVER = {}
    KEEP_SERVER = {}

    def run_copy(self, args):
        sys.argv = ['arv-copy'] + args
        return arv_copy.main()

    def test_unsupported_arg(self):
        with self.assertRaises(SystemExit):
            self.run_copy(['-x=unknown'])

    def test_version_argument(self):
        with tutil.redirected_streams(
                stdout=tutil.StringIO, stderr=tutil.StringIO) as (out, err):
            with self.assertRaises(SystemExit):
                self.run_copy(['--version'])
        self.assertVersionOutput(out, err)

    def test_copy_project(self):
        api = arvados.api()
        src_proj = api.groups().create(body={"group": {"name": "arv-copy project", "group_class": "project"}}).execute()["uuid"]

        c = Collection()
        with c.open('foo', 'wt') as f:
            f.write('foo')
        c.save_new("arv-copy foo collection", owner_uuid=src_proj)
        coll_record = api.collections().get(uuid=c.manifest_locator()).execute()
        assert coll_record['storage_classes_desired'] == ['default']

        dest_proj = api.groups().create(body={"group": {"name": "arv-copy dest project", "group_class": "project"}}).execute()["uuid"]

        tmphome = tempfile.mkdtemp()
        home_was = os.environ['HOME']
        os.environ['HOME'] = tmphome
        try:
            cfgdir = os.path.join(tmphome, ".config", "arvados")
            os.makedirs(cfgdir)
            with open(os.path.join(cfgdir, "zzzzz.conf"), "wt") as f:
                f.write("ARVADOS_API_HOST=%s\n" % os.environ["ARVADOS_API_HOST"])
                f.write("ARVADOS_API_TOKEN=%s\n" % os.environ["ARVADOS_API_TOKEN"])
                f.write("ARVADOS_API_HOST_INSECURE=1\n")

            contents = api.groups().list(filters=[["owner_uuid", "=", dest_proj]]).execute()
            assert len(contents["items"]) == 0

            with tutil.redirected_streams(
                    stdout=tutil.StringIO, stderr=tutil.StringIO) as (out, err):
                try:
                    self.run_copy(["--project-uuid", dest_proj, "--storage-classes", "foo", src_proj])
                except SystemExit as e:
                    assert e.code == 0
                copy_uuid_from_stdout = out.getvalue().strip()

            contents = api.groups().list(filters=[["owner_uuid", "=", dest_proj]]).execute()
            assert len(contents["items"]) == 1

            assert contents["items"][0]["name"] == "arv-copy project"
            copied_project = contents["items"][0]["uuid"]

            assert copied_project == copy_uuid_from_stdout

            contents = api.collections().list(filters=[["owner_uuid", "=", copied_project]]).execute()
            assert len(contents["items"]) == 1

            assert contents["items"][0]["uuid"] != c.manifest_locator()
            assert contents["items"][0]["name"] == "arv-copy foo collection"
            assert contents["items"][0]["portable_data_hash"] == c.portable_data_hash()
            assert contents["items"][0]["storage_classes_desired"] == ["foo"]

        finally:
            os.environ['HOME'] = home_was
            shutil.rmtree(tmphome)


class TestApiForInstance:
    _token_counter = itertools.count(1)

    @staticmethod
    def api_config(version, **kwargs):
        assert version == 'v1'
        return kwargs

    @pytest.fixture
    def patch_api(self, monkeypatch):
        monkeypatch.setattr(arvados, 'api', self.api_config)

    @pytest.fixture
    def config_file(self, tmp_path):
        count = next(self._token_counter)
        path = tmp_path / f'config{count}.conf'
        with path.open('w') as config_file:
            print(
                "ARVADOS_API_HOST=localhost",
                f"ARVADOS_API_TOKEN={self.expected_token(path)}",
                sep="\n", file=config_file,
            )
        return path

    @pytest.fixture
    def patch_search(self, tmp_path, monkeypatch):
        def search(self, name):
            path = tmp_path / name
            if path.exists():
                yield path
        monkeypatch.setattr(basedirs.BaseDirectories, 'search', search)

    def expected_token(self, path):
        return f"v2/zzzzz-gj3su-{path.stem:>015s}/{path.stem:>050s}"

    def test_from_environ(self, patch_api):
        actual = arv_copy.api_for_instance('', 0)
        assert actual == {}

    def test_relative_path(self, patch_api, config_file, monkeypatch):
        monkeypatch.chdir(config_file.parent)
        actual = arv_copy.api_for_instance(f'./{config_file.name}', 0)
        assert actual['host'] == 'localhost'
        assert actual['token'] == self.expected_token(config_file)

    def test_absolute_path(self, patch_api, config_file):
        actual = arv_copy.api_for_instance(str(config_file), 0)
        assert actual['host'] == 'localhost'
        assert actual['token'] == self.expected_token(config_file)

    def test_search_path(self, patch_api, patch_search, config_file):
        actual = arv_copy.api_for_instance(config_file.stem, 0)
        assert actual['host'] == 'localhost'
        assert actual['token'] == self.expected_token(config_file)

    def test_search_failed(self, patch_api, patch_search):
        with pytest.raises(SystemExit) as exc_info:
            arv_copy.api_for_instance('NotFound', 0)
        assert exc_info.value.code > 0

    def test_path_unreadable(self, patch_api, tmp_path):
        with pytest.raises(SystemExit) as exc_info:
            arv_copy.api_for_instance(str(tmp_path / 'nonexistent.conf'), 0)
        assert exc_info.value.code > 0
