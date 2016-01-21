import arvados
import arvados_fuse.command
import mock
import os
import run_test_server
import tempfile
import unittest

class KeepClientRetry(unittest.TestCase):
    origKeepClient = arvados.keep.KeepClient

    def setUp(self):
        self.mnt = tempfile.mkdtemp()
        run_test_server.authorize_with('active')

    def tearDown(self):
        os.rmdir(self.mnt)

    @mock.patch('arvados_fuse.arvados.keep.KeepClient')
    def _test_retry(self, num_retries, argv, kc):
        kc.side_effect = lambda *args, **kw: self.origKeepClient(*args, **kw)
        with arvados_fuse.command.Mount(
                arvados_fuse.command.ArgumentParser().parse_args(
                    argv+[self.mnt])):
            pass
        self.assertEqual(num_retries, kc.call_args[1].get('num_retries'))

    def test_default_retry_3(self):
        self._test_retry(3, [])

    def test_retry_2(self):
        self._test_retry(2, ['--retries=2'])

    def test_no_retry(self):
        self._test_retry(0, ['--retries=0'])
