# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import os
import subprocess
import time

from integration_test import IntegrationTest

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
             '--unmount-timeout', '10',
             self.mnt])
        subprocess.check_call(
            ['./bin/arv-mount', '--subtype', 'test', '--replace',
             '--unmount-timeout', '10',
             self.mnt,
             '--exec', 'true'])
        for m in subprocess.check_output(['mount']).splitlines():
            self.assertNotIn(' '+self.mnt+' ', m)

    def _mounted(self, mounts):
        all_mounts = subprocess.check_output(['mount'])
        return [m for m in mounts
                if ' '+m+' ' in all_mounts]

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
