# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import arvados
import arvados_fuse
import arvados_fuse.command
import atexit
import functools
import inspect
import logging
import multiprocessing
import os
import signal
import sys
import tempfile
import unittest

import pytest

from . import run_test_server

@atexit.register
def _pool_cleanup():
    if _pool is None:
        return
    _pool.close()
    _pool.join()


def wrap_static_test_method(modName, clsName, funcName, args, kwargs):
    class Test(unittest.TestCase):
        def runTest(self, *args, **kwargs):
            getattr(getattr(sys.modules[modName], clsName), funcName)(self, *args, **kwargs)
    Test().runTest(*args, **kwargs)


# To avoid Python's threading+multiprocessing=deadlock problems, we
# use a single global pool with maxtasksperchild=None for the entire
# test suite.
_pool = None
def workerPool():
    global _pool
    if _pool is None:
        _pool = multiprocessing.Pool(processes=1, maxtasksperchild=None)
    return _pool


class IntegrationTest(unittest.TestCase):
    def pool_test(self, *args, **kwargs):
        """Run a static method as a unit test, in a different process.

        If called by method 'foobar', the static method '_foobar' of
        the same class will be called in the other process.
        """
        modName = inspect.getmodule(self).__name__
        clsName = self.__class__.__name__
        funcName = inspect.currentframe().f_back.f_code.co_name
        workerPool().apply(
            wrap_static_test_method,
            (modName, clsName, '_'+funcName, args, kwargs))

    @classmethod
    def setUpClass(cls):
        run_test_server.run()
        run_test_server.run_keep(blob_signing=True, num_servers=2)

    @classmethod
    def tearDownClass(cls):
        run_test_server.stop_keep(num_servers=2)

    def setUp(self):
        self.mnt = tempfile.mkdtemp()
        run_test_server.authorize_with('active')

    def tearDown(self):
        os.rmdir(self.mnt)
        run_test_server.reset()

    @staticmethod
    def mount(argv):
        """Decorator. Sets up a FUSE mount at self.mnt with the given args."""
        def decorator(func):
            @functools.wraps(func)
            def wrapper(self, *args, **kwargs):
                self.mount = None
                try:
                    with arvados_fuse.command.Mount(
                            arvados_fuse.command.ArgumentParser().parse_args(
                                argv + ['--foreground',
                                        '--unmount-timeout=60',
                                        self.mnt])) as self.mount:
                        return func(self, *args, **kwargs)
                finally:
                    if self.mount and self.mount.llfuse_thread.is_alive():
                        logging.warning("IntegrationTest.mount:"
                                            " llfuse thread still alive after umount"
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
            return wrapper
        return decorator
