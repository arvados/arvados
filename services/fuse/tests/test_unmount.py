import os
import subprocess

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
            ['arv-mount', '--subtype', 'test', '--replace',
             self.mnt])
        subprocess.check_call(
            ['arv-mount', '--subtype', 'test', '--replace',
             '--unmount-timeout', '10',
             self.mnt])
        subprocess.check_call(
            ['arv-mount', '--subtype', 'test', '--replace',
             '--unmount-timeout', '10',
             self.mnt,
             '--exec', 'true'])
        for m in subprocess.check_output(['mount']).splitlines():
            self.assertNotIn(' '+self.mnt+' ', m)

    def test_unmount_children(self):
        for d in ['foo', 'foo/bar', 'bar']:
            mnt = self.tmp+'/'+d
            os.mkdir(mnt)
            self.to_delete.insert(0, mnt)
        for d in ['bar', 'foo/bar']:
            mnt = self.tmp+'/'+d
            subprocess.check_call(
                ['arv-mount', '--subtype', 'test', mnt])
        subprocess.check_call(['arv-mount', '--unmount', self.tmp+'/...'])
        for m in subprocess.check_output(['mount']).splitlines():
            self.assertNotIn(' '+self.tmp+'/', m)
