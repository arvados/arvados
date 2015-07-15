import arvados
import arvados.safeapi
import arvados_fuse as fuse
import llfuse
import os
import shutil
import subprocess
import sys
import tempfile
import threading
import time
import unittest
import logging
import multiprocessing
import run_test_server

logger = logging.getLogger('arvados.arv-mount')

class MountTestBase(unittest.TestCase):
    def setUp(self):
        # The underlying C implementation of open() makes a fstat() syscall
        # with the GIL still held.  When the GETATTR message comes back to
        # llfuse (which in these tests is in the same interpreter process) it
        # can't acquire the GIL, so it can't service the fstat() call, so it
        # deadlocks.  The workaround is to run some of our test code in a
        # separate process.  Forturnately the multiprocessing module makes this
        # relatively easy.
        self.pool = multiprocessing.Pool(1)

        self.keeptmp = tempfile.mkdtemp()
        os.environ['KEEP_LOCAL_STORE'] = self.keeptmp
        self.mounttmp = tempfile.mkdtemp()
        run_test_server.run()
        run_test_server.authorize_with("admin")
        self.api = arvados.safeapi.ThreadSafeApiCache(arvados.config.settings())

    def make_mount(self, root_class, **root_kwargs):
        self.operations = fuse.Operations(os.getuid(), os.getgid(), enable_write=True)
        self.operations.inodes.add_entry(root_class(
            llfuse.ROOT_INODE, self.operations.inodes, self.api, 0, **root_kwargs))
        llfuse.init(self.operations, self.mounttmp, [])
        threading.Thread(None, llfuse.main).start()
        # wait until the driver is finished initializing
        self.operations.initlock.wait()
        return self.operations.inodes[llfuse.ROOT_INODE]

    def tearDown(self):
        self.pool.terminate()
        self.pool.join()
        del self.pool

        # llfuse.close is buggy, so use fusermount instead.
        #llfuse.close(unmount=True)

        count = 0
        success = 1
        while (count < 9 and success != 0):
          success = subprocess.call(["fusermount", "-u", self.mounttmp])
          time.sleep(0.1)
          count += 1

        self.operations.destroy()

        os.rmdir(self.mounttmp)
        shutil.rmtree(self.keeptmp)
        run_test_server.reset()

    def assertDirContents(self, subdir, expect_content):
        path = self.mounttmp
        if subdir:
            path = os.path.join(path, subdir)
        self.assertEqual(sorted(expect_content), sorted(llfuse.listdir(path)))
