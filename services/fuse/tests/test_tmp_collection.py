import arvados
import arvados_fuse
import arvados_fuse.command
import json
import logging
import os
import tempfile
import unittest

from .integration_test import IntegrationTest
from .mount_test_base import MountTestBase

logger = logging.getLogger('arvados.arv-mount')


class TmpCollectionArgsTest(unittest.TestCase):
    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()

    def tearDown(self):
        os.rmdir(self.tmpdir)

    def test_tmp_only(self):
        args = arvados_fuse.command.ArgumentParser().parse_args([
            '--mount-tmp', 'tmp1',
            '--mount-tmp', 'tmp2',
            self.tmpdir,
        ])
        self.assertIn(args.mode, [None, 'custom'])
        self.assertEqual(['tmp1', 'tmp2'], args.mount_tmp)
        for mtype in ['home', 'shared', 'by_id', 'by_pdh', 'by_tag']:
            self.assertEqual([], getattr(args, 'mount_'+mtype))

    def test_tmp_and_home(self):
        args = arvados_fuse.command.ArgumentParser().parse_args([
            '--mount-tmp', 'test_tmp',
            '--mount-home', 'test_home',
            self.tmpdir,
        ])
        self.assertIn(args.mode, [None, 'custom'])
        self.assertEqual(['test_tmp'], args.mount_tmp)
        self.assertEqual(['test_home'], args.mount_home)

    def test_no_tmp(self):
        args = arvados_fuse.command.ArgumentParser().parse_args([
            self.tmpdir,
        ])
        self.assertEqual([], args.mount_tmp)


def current_manifest(tmpdir):
    return json.load(open(
        os.path.join(tmpdir, '.arvados#collection')
    ))['manifest_text']


class TmpCollectionTest(IntegrationTest):
    mnt_args = [
        '--read-write',
        '--mount-tmp', 'zzz',
    ]

    @IntegrationTest.mount(argv=mnt_args+['--mount-tmp', 'yyy'])
    def test_two_tmp(self):
        self.pool_test(os.path.join(self.mnt, 'zzz'),
                       os.path.join(self.mnt, 'yyy'))
    @staticmethod
    def _test_two_tmp(self, zzz, yyy):
        self.assertEqual(current_manifest(zzz), "")
        self.assertEqual(current_manifest(yyy), "")
        with open(os.path.join(zzz, 'foo'), 'w') as f:
            f.write('foo')
        self.assertNotEqual(current_manifest(zzz), "")
        self.assertEqual(current_manifest(yyy), "")
        os.unlink(os.path.join(zzz, 'foo'))
        with open(os.path.join(yyy, 'bar'), 'w') as f:
            f.write('bar')
        self.assertEqual(current_manifest(zzz), "")
        self.assertNotEqual(current_manifest(yyy), "")

    @IntegrationTest.mount(argv=mnt_args)
    def test_tmp_empty(self):
        self.pool_test(os.path.join(self.mnt, 'zzz'))
    @staticmethod
    def _test_tmp_empty(self, tmpdir):
        self.assertEqual(current_manifest(tmpdir), "")

    @IntegrationTest.mount(argv=mnt_args)
    def test_tmp_onefile(self):
        self.pool_test(os.path.join(self.mnt, 'zzz'))
    @staticmethod
    def _test_tmp_onefile(self, tmpdir):
        with open(os.path.join(tmpdir, 'foo'), 'w') as f:
            f.write('foo')
        self.assertRegexpMatches(
            current_manifest(tmpdir),
            r'^\. acbd18db4cc2f85cedef654fccc4a4d8\+3(\+\S+)? 0:3:foo\n$')

    @IntegrationTest.mount(argv=mnt_args)
    def test_tmp_snapshots(self):
        self.pool_test(os.path.join(self.mnt, 'zzz'))
    @staticmethod
    def _test_tmp_snapshots(self, tmpdir):
        ops = [
            ('foo', 'bar',
             r'^\. 37b51d194a7513e45b56f6524f2d51f2\+3(\+\S+)? 0:3:foo\n$'),
            ('foo', 'foo',
             r'^\. acbd18db4cc2f85cedef654fccc4a4d8\+3(\+\S+)? 0:3:foo\n$'),
            ('bar', 'bar',
             r'^\. 37b51d194a7513e45b56f6524f2d51f2\+3(\+\S+)? acbd18db4cc2f85cedef654fccc4a4d8\+3(\+\S+)? 0:3:bar 3:3:foo\n$'),
            ('foo', None,
             r'^\. 37b51d194a7513e45b56f6524f2d51f2\+3(\+\S+)? 0:3:bar\n$'),
            ('bar', None,
             r'^$'),
        ]
        for _ in range(10):
            for fn, content, expect in ops:
                path = os.path.join(tmpdir, fn)
                if content is None:
                    os.unlink(path)
                else:
                    with open(path, 'w') as f:
                        f.write(content)
                self.assertRegexpMatches(current_manifest(tmpdir), expect)
