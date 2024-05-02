# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import arvados_fuse.unmount
import os
import subprocess
import shutil
import tempfile
import time
import unittest

from .integration_test import IntegrationTest

class UnmountTest(IntegrationTest):
    def setUp(self):
        super(UnmountTest, self).setUp()
        self.tmp = self.mnt
        self.to_delete = []

    def tearDown(self):
        for d in self.to_delete:
            os.rmdir(d)
        super(UnmountTest, self).tearDown()

    def test_replace(self):
        subprocess.check_call(
            ['./bin/arv-mount', '--subtype', 'test', '--replace',
             self.mnt])
        subprocess.check_call(
            ['./bin/arv-mount', '--subtype', 'test', '--replace',
             '--unmount-timeout', '60',
             self.mnt])
        subprocess.check_call(
            ['./bin/arv-mount', '--subtype', 'test', '--replace',
             '--unmount-timeout', '60',
             self.mnt,
             '--exec', 'true'])
        for m in subprocess.check_output(['mount']).splitlines():
            expected = bytes(' ' + self.mnt + ' ', encoding='utf-8')
            self.assertNotIn(expected, m)

    def _mounted(self, mounts):
        all_mounts = subprocess.check_output(['mount'])
        return [m for m in mounts
                if bytes(' ' + m + ' ', encoding='utf-8') in all_mounts]

    def _wait_for_mounts(self, mounts):
        deadline = time.time() + 10
        while self._mounted(mounts) != mounts:
            time.sleep(0.1)
            self.assertLess(time.time(), deadline)

    def test_unmount_subtype(self):
        mounts = []
        for d in ['foo', 'bar']:
            mnt = self.tmp+'/'+d
            os.mkdir(mnt)
            self.to_delete.insert(0, mnt)
            mounts.append(mnt)
            subprocess.check_call(
                ['./bin/arv-mount', '--subtype', d, mnt])

        self._wait_for_mounts(mounts)
        self.assertEqual(mounts, self._mounted(mounts))
        subprocess.call(['./bin/arv-mount', '--subtype', 'baz', '--unmount-all', self.tmp])
        self.assertEqual(mounts, self._mounted(mounts))
        subprocess.call(['./bin/arv-mount', '--subtype', 'bar', '--unmount', mounts[0]])
        self.assertEqual(mounts, self._mounted(mounts))
        subprocess.call(['./bin/arv-mount', '--subtype', '', '--unmount', self.tmp])
        self.assertEqual(mounts, self._mounted(mounts))
        subprocess.check_call(['./bin/arv-mount', '--subtype', 'foo', '--unmount', mounts[0]])
        self.assertEqual(mounts[1:], self._mounted(mounts))
        subprocess.check_call(['./bin/arv-mount', '--subtype', '', '--unmount-all', mounts[0]])
        self.assertEqual(mounts[1:], self._mounted(mounts))
        subprocess.check_call(['./bin/arv-mount', '--subtype', 'bar', '--unmount-all', self.tmp])
        self.assertEqual([], self._mounted(mounts))

    def test_unmount_children(self):
        for d in ['foo', 'foo/bar', 'bar']:
            mnt = self.tmp+'/'+d
            os.mkdir(mnt)
            self.to_delete.insert(0, mnt)
        mounts = []
        for d in ['bar', 'foo/bar']:
            mnt = self.tmp+'/'+d
            mounts.append(mnt)
            subprocess.check_call(
                ['./bin/arv-mount', '--subtype', 'test', mnt])

        self._wait_for_mounts(mounts)
        self.assertEqual(mounts, self._mounted(mounts))
        subprocess.check_call(['./bin/arv-mount', '--unmount', self.tmp])
        self.assertEqual(mounts, self._mounted(mounts))
        subprocess.check_call(['./bin/arv-mount', '--unmount-all', self.tmp])
        self.assertEqual([], self._mounted(mounts))



class SaferRealpath(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.mkdtemp()

    def tearDown(self):
        shutil.rmtree(self.tmp)

    def test_safer_realpath(self):
        os.mkdir(self.tmp+"/dir")
        os.mkdir(self.tmp+"/dir/dir2")
        os.symlink("missing", self.tmp+"/relative-missing")
        os.symlink("dir", self.tmp+"/./relative-dir")
        os.symlink("relative-dir", self.tmp+"/relative-indirect")
        os.symlink(self.tmp+"/dir", self.tmp+"/absolute-dir")
        os.symlink("./dir/../loop", self.tmp+"/loop")
        os.symlink(".", self.tmp+"/dir/self")
        os.symlink("..", self.tmp+"/dir/dir2/parent")
        os.symlink("../dir3", self.tmp+"/dir/dir2/sibling")
        os.symlink("../missing/../danger", self.tmp+"/dir/tricky")
        os.symlink("/proc/1/fd/12345", self.tmp+"/eperm")
        for (inpath, outpath, ok) in [
                ("dir/self", "dir", True),
                ("dir/dir2/parent", "dir", True),
                ("dir/dir2/sibling", "dir/dir3", False),
                ("dir", "dir", True),
                ("relative-dir", "dir", True),
                ("relative-missing", "missing", False),
                ("relative-indirect", "dir", True),
                ("absolute-dir", "dir", True),
                ("loop", "loop", False),
                # "missing" doesn't exist, so "missing/.." isn't our
                # tmpdir; it's important not to contract this to just
                # "danger".
                ("dir/tricky", "missing/../danger", False),
                ("eperm", "/proc/1/fd/12345", False),
        ]:
            if not outpath.startswith('/'):
                outpath = self.tmp + '/' + outpath
            self.assertEqual((outpath, ok), arvados_fuse.unmount.safer_realpath(self.tmp+"/"+inpath))
