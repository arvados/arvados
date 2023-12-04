# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import
from __future__ import print_function
from six import assertRegex
import arvados
import arvados_fuse
import arvados_fuse.command
import contextlib
import functools
import io
import json
import llfuse
import logging
import mock
import os
from . import run_test_server
import sys
import tempfile
import unittest
import resource

def noexit(func):
    """If argparse or arvados_fuse tries to exit, fail the test instead"""
    class SystemExitCaught(Exception):
        pass
    @functools.wraps(func)
    def wrapper(*args, **kwargs):
        try:
            return func(*args, **kwargs)
        except SystemExit:
            raise SystemExitCaught
    return wrapper

@contextlib.contextmanager
def nostderr():
    with open(os.devnull, 'w') as dn:
        orig, sys.stderr = sys.stderr, dn
        try:
            yield
        finally:
            sys.stderr = orig


class MountArgsTest(unittest.TestCase):
    def setUp(self):
        self.mntdir = tempfile.mkdtemp()
        run_test_server.authorize_with('active')

    def tearDown(self):
        os.rmdir(self.mntdir)

    def lookup(self, mnt, *path):
        ent = mnt.operations.inodes[llfuse.ROOT_INODE]
        for p in path:
            ent = ent[p]
        return ent

    @contextlib.contextmanager
    def stderrMatches(self, stderr):
        orig, sys.stderr = sys.stderr, stderr
        try:
            yield
        finally:
            sys.stderr = orig

    def check_ent_type(self, cls, *path):
        ent = self.lookup(self.mnt, *path)
        self.assertEqual(ent.__class__, cls)
        return ent

    @noexit
    def test_default_all(self):
        args = arvados_fuse.command.ArgumentParser().parse_args([
            '--foreground', self.mntdir])
        self.assertEqual(args.mode, None)
        self.mnt = arvados_fuse.command.Mount(args)
        e = self.check_ent_type(arvados_fuse.ProjectDirectory, 'home')
        self.assertEqual(e.project_object['uuid'],
                         run_test_server.fixture('users')['active']['uuid'])
        e = self.check_ent_type(arvados_fuse.MagicDirectory, 'by_id')

        e = self.check_ent_type(arvados_fuse.StringFile, 'README')
        readme = e.readfrom(0, -1).decode()
        assertRegex(self, readme, r'active-user@arvados\.local')
        assertRegex(self, readme, r'\n$')

        e = self.check_ent_type(arvados_fuse.StringFile, 'by_id', 'README')
        txt = e.readfrom(0, -1).decode()
        assertRegex(self, txt, r'portable data hash')
        assertRegex(self, txt, r'\n$')

    @noexit
    def test_by_id(self):
        args = arvados_fuse.command.ArgumentParser().parse_args([
            '--by-id',
            '--foreground', self.mntdir])
        self.assertEqual(args.mode, 'by_id')
        self.mnt = arvados_fuse.command.Mount(args)
        e = self.check_ent_type(arvados_fuse.MagicDirectory)
        self.assertEqual(e.pdh_only, False)
        self.assertEqual(True, self.mnt.listen_for_events)

    @noexit
    def test_by_pdh(self):
        args = arvados_fuse.command.ArgumentParser().parse_args([
            '--by-pdh',
            '--foreground', self.mntdir])
        self.assertEqual(args.mode, 'by_pdh')
        self.mnt = arvados_fuse.command.Mount(args)
        e = self.check_ent_type(arvados_fuse.MagicDirectory)
        self.assertEqual(e.pdh_only, True)
        self.assertEqual(False, self.mnt.listen_for_events)

    @noexit
    def test_by_tag(self):
        args = arvados_fuse.command.ArgumentParser().parse_args([
            '--by-tag',
            '--foreground', self.mntdir])
        self.assertEqual(args.mode, 'by_tag')
        self.mnt = arvados_fuse.command.Mount(args)
        e = self.check_ent_type(arvados_fuse.TagsDirectory)
        self.assertEqual(True, self.mnt.listen_for_events)

    @noexit
    def test_collection(self, id_type='uuid'):
        c = run_test_server.fixture('collections')['public_text_file']
        cid = c[id_type]
        args = arvados_fuse.command.ArgumentParser().parse_args([
            '--collection', cid,
            '--foreground', self.mntdir])
        self.mnt = arvados_fuse.command.Mount(args)
        e = self.check_ent_type(arvados_fuse.CollectionDirectory)
        self.assertEqual(e.collection_locator, cid)
        self.assertEqual(id_type == 'uuid', self.mnt.listen_for_events)

    def test_collection_pdh(self):
        self.test_collection('portable_data_hash')

    @noexit
    def test_home(self):
        args = arvados_fuse.command.ArgumentParser().parse_args([
            '--home',
            '--foreground', self.mntdir])
        self.assertEqual(args.mode, 'home')
        self.mnt = arvados_fuse.command.Mount(args)
        e = self.check_ent_type(arvados_fuse.ProjectDirectory)
        self.assertEqual(e.project_object['uuid'],
                         run_test_server.fixture('users')['active']['uuid'])
        self.assertEqual(True, self.mnt.listen_for_events)

    def test_mutually_exclusive_args(self):
        cid = run_test_server.fixture('collections')['public_text_file']['uuid']
        gid = run_test_server.fixture('groups')['aproject']['uuid']
        for badargs in [
                ['--mount-tmp', 'foo', '--collection', cid],
                ['--mount-tmp', 'foo', '--project', gid],
                ['--collection', cid, '--project', gid],
                ['--by-id', '--project', gid],
                ['--mount-tmp', 'foo', '--by-id'],
        ]:
            with nostderr():
                with self.assertRaises(SystemExit):
                    args = arvados_fuse.command.ArgumentParser().parse_args(
                        badargs + ['--foreground', self.mntdir])
                    arvados_fuse.command.Mount(args)
    @noexit
    def test_project(self):
        uuid = run_test_server.fixture('groups')['aproject']['uuid']
        args = arvados_fuse.command.ArgumentParser().parse_args([
            '--project', uuid,
            '--foreground', self.mntdir])
        self.mnt = arvados_fuse.command.Mount(args)
        e = self.check_ent_type(arvados_fuse.ProjectDirectory)
        self.assertEqual(e.project_object['uuid'], uuid)

    @noexit
    def test_shared(self):
        args = arvados_fuse.command.ArgumentParser().parse_args([
            '--shared',
            '--foreground', self.mntdir])
        self.assertEqual(args.mode, 'shared')
        self.mnt = arvados_fuse.command.Mount(args)
        e = self.check_ent_type(arvados_fuse.SharedDirectory)
        self.assertEqual(e.current_user['uuid'],
                         run_test_server.fixture('users')['active']['uuid'])
        self.assertEqual(True, self.mnt.listen_for_events)

    def test_version_argument(self):
        # The argparse version action prints to stderr in Python 2 and stdout
        # in Python 3.4 and up. Write both to the same stream so the test can pass
        # in both cases.
        origerr = sys.stderr
        origout = sys.stdout
        sys.stderr = io.StringIO()
        sys.stdout = sys.stderr

        with self.assertRaises(SystemExit):
            args = arvados_fuse.command.ArgumentParser().parse_args(['--version'])
        assertRegex(self, sys.stdout.getvalue(), "[0-9]+\.[0-9]+\.[0-9]+")
        sys.stderr.close()
        sys.stderr = origerr
        sys.stdout = origout

    @noexit
    @mock.patch('arvados.events.subscribe')
    def test_disable_event_listening(self, mock_subscribe):
        args = arvados_fuse.command.ArgumentParser().parse_args([
            '--disable-event-listening',
            '--by-id',
            '--foreground', self.mntdir])
        self.mnt = arvados_fuse.command.Mount(args)
        self.assertEqual(True, self.mnt.listen_for_events)
        self.assertEqual(True, self.mnt.args.disable_event_listening)
        with self.mnt:
            pass
        self.assertEqual(0, mock_subscribe.call_count)

    @noexit
    @mock.patch('arvados.events.subscribe')
    def test_custom(self, mock_subscribe):
        args = arvados_fuse.command.ArgumentParser().parse_args([
            '--mount-tmp', 'foo',
            '--mount-tmp', 'bar',
            '--mount-home', 'my_home',
            '--foreground', self.mntdir])
        self.assertEqual(args.mode, None)
        self.mnt = arvados_fuse.command.Mount(args)
        self.check_ent_type(arvados_fuse.Directory)
        self.check_ent_type(arvados_fuse.TmpCollectionDirectory, 'foo')
        self.check_ent_type(arvados_fuse.TmpCollectionDirectory, 'bar')
        e = self.check_ent_type(arvados_fuse.ProjectDirectory, 'my_home')
        self.assertEqual(e.project_object['uuid'],
                         run_test_server.fixture('users')['active']['uuid'])
        self.assertEqual(True, self.mnt.listen_for_events)
        with self.mnt:
            pass
        self.assertEqual(1, mock_subscribe.call_count)

    @noexit
    @mock.patch('arvados.events.subscribe')
    def test_custom_no_listen(self, mock_subscribe):
        args = arvados_fuse.command.ArgumentParser().parse_args([
            '--mount-by-pdh', 'pdh',
            '--mount-tmp', 'foo',
            '--mount-tmp', 'bar',
            '--foreground', self.mntdir])
        self.mnt = arvados_fuse.command.Mount(args)
        self.assertEqual(False, self.mnt.listen_for_events)
        with self.mnt:
            pass
        self.assertEqual(0, mock_subscribe.call_count)

    def test_custom_unsupported_layouts(self):
        for name in ['.', '..', '', 'foo/bar', '/foo']:
            with nostderr():
                with self.assertRaises(SystemExit):
                    args = arvados_fuse.command.ArgumentParser().parse_args([
                        '--mount-tmp', name,
                        '--foreground', self.mntdir])
                    arvados_fuse.command.Mount(args)

    @noexit
    @mock.patch('resource.setrlimit')
    @mock.patch('resource.getrlimit')
    def test_file_cache(self, getrlimit, setrlimit):
        args = arvados_fuse.command.ArgumentParser().parse_args([
            '--foreground', '--file-cache=256000000000', self.mntdir])
        self.assertEqual(args.mode, None)

        getrlimit.return_value = (1024, 1048576)

        self.mnt = arvados_fuse.command.Mount(args)

        setrlimit.assert_called_with(resource.RLIMIT_NOFILE, (30517, 1048576))


    @noexit
    @mock.patch('resource.setrlimit')
    @mock.patch('resource.getrlimit')
    def test_file_cache_hard_limit(self, getrlimit, setrlimit):
        args = arvados_fuse.command.ArgumentParser().parse_args([
            '--foreground', '--file-cache=256000000000', self.mntdir])
        self.assertEqual(args.mode, None)

        getrlimit.return_value = (1024, 2048)

        self.mnt = arvados_fuse.command.Mount(args)

        setrlimit.assert_called_with(resource.RLIMIT_NOFILE, (2048, 2048))

class MountErrorTest(unittest.TestCase):
    def setUp(self):
        self.mntdir = tempfile.mkdtemp()
        run_test_server.run()
        run_test_server.authorize_with("active")
        self.logger = logging.getLogger("null")
        self.logger.setLevel(logging.CRITICAL+1)

    def tearDown(self):
        if os.path.exists(self.mntdir):
            # If the directory was not unmounted, this will raise an exception.
            os.rmdir(self.mntdir)
        run_test_server.reset()

    def test_no_token(self):
        del arvados.config._settings["ARVADOS_API_TOKEN"]
        arvados.config._settings = {}
        with self.assertRaises(SystemExit) as ex:
            args = arvados_fuse.command.ArgumentParser().parse_args([self.mntdir])
            arvados_fuse.command.Mount(args, logger=self.logger).run()
        self.assertEqual(1, ex.exception.code)

    def test_no_host(self):
        del arvados.config._settings["ARVADOS_API_HOST"]
        with self.assertRaises(SystemExit) as ex:
            args = arvados_fuse.command.ArgumentParser().parse_args([self.mntdir])
            arvados_fuse.command.Mount(args, logger=self.logger).run()
        self.assertEqual(1, ex.exception.code)

    def test_bogus_host(self):
        arvados.config._settings["ARVADOS_API_HOST"] = "100::"
        with self.assertRaises(SystemExit) as ex, mock.patch('time.sleep'):
            args = arvados_fuse.command.ArgumentParser().parse_args([self.mntdir])
            arvados_fuse.command.Mount(args, logger=self.logger).run()
        self.assertEqual(1, ex.exception.code)

    def test_bogus_token(self):
        arvados.config._settings["ARVADOS_API_TOKEN"] = "zzzzzzzzzzzzz"
        with self.assertRaises(SystemExit) as ex:
            args = arvados_fuse.command.ArgumentParser().parse_args([self.mntdir])
            arvados_fuse.command.Mount(args, logger=self.logger).run()
        self.assertEqual(1, ex.exception.code)

    def test_bogus_mount_dir(self):
        # All FUSE errors in llfuse.init() are raised as RuntimeError
        # An easy error to trigger is to supply a nonexistent mount point,
        # so test that one.
        #
        # Other possible errors that also raise RuntimeError (but are much
        # harder to test automatically because they depend on operating
        # system configuration):
        #
        # The user doesn't have permission to use FUSE
        # The user specified --allow-other but user_allow_other is not set
        # in /etc/fuse.conf
        os.rmdir(self.mntdir)
        with self.assertRaises(SystemExit) as ex:
            args = arvados_fuse.command.ArgumentParser().parse_args([self.mntdir])
            arvados_fuse.command.Mount(args, logger=self.logger).run()
        self.assertEqual(1, ex.exception.code)

    def test_unreadable_collection(self):
        with self.assertRaises(SystemExit) as ex:
            args = arvados_fuse.command.ArgumentParser().parse_args([
                "--collection", "zzzzz-4zz18-zzzzzzzzzzzzzzz", self.mntdir])
            arvados_fuse.command.Mount(args, logger=self.logger).run()
        self.assertEqual(1, ex.exception.code)

    def test_unreadable_project(self):
        with self.assertRaises(SystemExit) as ex:
            args = arvados_fuse.command.ArgumentParser().parse_args([
                "--project", "zzzzz-j7d0g-zzzzzzzzzzzzzzz", self.mntdir])
            arvados_fuse.command.Mount(args, logger=self.logger).run()
        self.assertEqual(1, ex.exception.code)
