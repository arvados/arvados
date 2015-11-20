import arvados
import arvados_fuse
import arvados_fuse.command
import functools
import inspect
import multiprocessing
import os
import sys
import tempfile
import unittest
import run_test_server

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
        modName = inspect.getmodule(self).__name__
        clsName = self.__class__.__name__
        funcName = inspect.currentframe().f_back.f_code.co_name
        pool = multiprocessing.Pool(1)
        try:
            pool.apply(
                wrap_static_test_method,
                (modName, clsName, '_'+funcName, args, kwargs))
        finally:
            pool.terminate()
            pool.join()

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
                            argv + ['--foreground', self.mnt])):
                    return func(self, *args, **kwargs)
            return wrapper
        return decorator
