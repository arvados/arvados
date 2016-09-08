import arvados
import arvados_fuse
import arvados_fuse.command
import atexit
import functools
import inspect
import multiprocessing
import os
import run_test_server
import signal
import sys
import tempfile
import unittest

_pool = None


@atexit.register
def _pool_cleanup():
    global _pool
    if _pool is None:
        return
    _pool.close()
    _pool.join()


def wrap_static_test_method(modName, clsName, funcName, args, kwargs):
    class Test(unittest.TestCase):
        def runTest(self, *args, **kwargs):
            getattr(getattr(sys.modules[modName], clsName), funcName)(self, *args, **kwargs)
    Test().runTest(*args, **kwargs)


class IntegrationTest(unittest.TestCase):
    def pool_test(self, *args, **kwargs):
        """Run a static method as a unit test, in a different process.

        If called by method 'foobar', the static method '_foobar' of
        the same class will be called in the other process.
        """
        global _pool
        if _pool is None:
            _pool = multiprocessing.Pool(1, maxtasksperchild=1)
        modName = inspect.getmodule(self).__name__
        clsName = self.__class__.__name__
        funcName = inspect.currentframe().f_back.f_code.co_name
        _pool.apply(
            wrap_static_test_method,
            (modName, clsName, '_'+funcName, args, kwargs))

    @classmethod
    def setUpClass(cls):
        run_test_server.run()
        run_test_server.run_keep(enforce_permissions=True, num_servers=2)

    @classmethod
    def tearDownClass(cls):
        run_test_server.stop_keep(num_servers=2)

    def setUp(self):
        self.mnt = tempfile.mkdtemp()
        run_test_server.authorize_with('active')
        self.api = arvados.safeapi.ThreadSafeApiCache(arvados.config.settings())

    def tearDown(self):
        os.rmdir(self.mnt)
        run_test_server.reset()

    @staticmethod
    def mount(argv):
        """Decorator. Sets up a FUSE mount at self.mnt with the given args."""
        def decorator(func):
            @functools.wraps(func)
            def wrapper(self, *args, **kwargs):
                with arvados_fuse.command.Mount(
                        arvados_fuse.command.ArgumentParser().parse_args(
                            argv + ['--foreground',
                                    '--unmount-timeout=0.1',
                                    self.mnt])) as m:
                    return func(self, *args, **kwargs)
                if m.llfuse_thread.is_alive():
                    self.logger.warning("IntegrationTest.mount:"
                                        " llfuse thread still alive after umount"
                                        " -- killing test suite to avoid deadlock")
                    os.kill(os.getpid(), signal.SIGKILL)
            return wrapper
        return decorator
