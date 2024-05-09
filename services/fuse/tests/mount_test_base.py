# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import arvados
import arvados.keep
import arvados_fuse as fuse
import arvados.safeapi
import llfuse
import logging
import multiprocessing
import os
import shutil
import signal
import subprocess
import sys
import tempfile
import threading
import time
import unittest

import pytest

from . import run_test_server
from .integration_test import workerPool

logger = logging.getLogger('arvados.arv-mount')

def make_block_cache(disk_cache):
    if disk_cache:
        disk_cache_dir = os.path.join(os.path.expanduser("~"), ".cache", "arvados", "keep")
        shutil.rmtree(disk_cache_dir, ignore_errors=True)
    block_cache = arvados.keep.KeepBlockCache(disk_cache=disk_cache)
    return block_cache

class MountTestBase(unittest.TestCase):
    disk_cache = False

    def setUp(self, api=None, local_store=True):
        # The underlying C implementation of open() makes a fstat() syscall
        # with the GIL still held.  When the GETATTR message comes back to
        # llfuse (which in these tests is in the same interpreter process) it
        # can't acquire the GIL, so it can't service the fstat() call, so it
        # deadlocks.  The workaround is to run some of our test code in a
        # separate process.  Forturnately the multiprocessing module makes this
        # relatively easy.

        self.pool = workerPool()
        if local_store:
            self.keeptmp = tempfile.mkdtemp()
            os.environ['KEEP_LOCAL_STORE'] = self.keeptmp
        else:
            self.keeptmp = None
        self.mounttmp = tempfile.mkdtemp()
        run_test_server.run()
        run_test_server.authorize_with("admin")

        self.api = api if api else arvados.safeapi.ThreadSafeApiCache(
            arvados.config.settings(),
            keep_params={"block_cache": make_block_cache(self.disk_cache)},
            version='v1',
        )
        self.llfuse_thread = None

    # This is a copy of Mount's method.  TODO: Refactor MountTestBase
    # to use a Mount instead of copying its code.
    def _llfuse_main(self):
        try:
            llfuse.main()
        except:
            llfuse.close(unmount=False)
            raise
        llfuse.close()

    def make_mount(self, root_class, **root_kwargs):
        enable_write = root_kwargs.pop('enable_write', True)
        self.operations = fuse.Operations(
            os.getuid(),
            os.getgid(),
            api_client=self.api,
            enable_write=enable_write,
        )
        self.operations.inodes.add_entry(root_class(
            llfuse.ROOT_INODE,
            self.operations.inodes,
            self.api,
            0,
            enable_write,
            root_kwargs.pop('filters', None),
            **root_kwargs,
        ))
        llfuse.init(self.operations, self.mounttmp, [])
        self.llfuse_thread = threading.Thread(None, lambda: self._llfuse_main())
        self.llfuse_thread.daemon = True
        self.llfuse_thread.start()
        # wait until the driver is finished initializing
        self.operations.initlock.wait()
        return self.operations.inodes[llfuse.ROOT_INODE]

    def tearDown(self):
        if self.llfuse_thread:
            if self.operations.events:
                self.operations.events.close(timeout=10)
            subprocess.call(["fusermount", "-u", "-z", self.mounttmp])
            t0 = time.time()
            self.llfuse_thread.join(timeout=60)
            if self.llfuse_thread.is_alive():
                logger.warning("MountTestBase.tearDown():"
                               " llfuse thread still alive 60s after umount"
                               " -- ending test suite to avoid deadlock")
                # pytest uses exit status 2 when test collection failed.
                # A UnitTest failing in setup/teardown counts as a
                # collection failure, so pytest will exit with status 2
                # no matter what status you specify here. run-tests.sh
                # looks for this status, so specify 2 just to keep
                # everything as consistent as possible.
                # TODO: If we refactor these tests so they're not built
                # on unittest, consider using a dedicated, non-pytest
                # exit code like TEMPFAIL.
                pytest.exit("llfuse thread outlived test", 2)
            waited = time.time() - t0
            if waited > 0.1:
                logger.warning("MountTestBase.tearDown(): waited %f s for llfuse thread to end", waited)

        os.rmdir(self.mounttmp)
        if self.keeptmp:
            shutil.rmtree(self.keeptmp)
            os.environ.pop('KEEP_LOCAL_STORE')
        run_test_server.reset()

    def assertDirContents(self, subdir, expect_content):
        path = self.mounttmp
        if subdir:
            path = os.path.join(path, subdir)
        self.assertEqual(sorted(expect_content), sorted(llfuse.listdir(str(path))))
