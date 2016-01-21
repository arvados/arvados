import arvados
import arvados_fuse.command
import json
import mock
import os
import pycurl
import Queue
import run_test_server
import tempfile
import unittest

from .integration_test import IntegrationTest


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

class RetryPUT(IntegrationTest):
    @mock.patch('time.sleep')
    @IntegrationTest.mount(argv=['--read-write', '--mount-tmp=zzz'])
    def test_retry_write(self, sleep):
        mockedCurl = mock.Mock(spec=pycurl.Curl(), wraps=pycurl.Curl())
        mockedCurl.perform.side_effect = Exception('mock error (ok)')
        q = Queue.Queue()
        q.put(mockedCurl)
        q.put(pycurl.Curl())
        q.put(pycurl.Curl())
        with mock.patch('arvados.keep.KeepClient.KeepService._get_user_agent', side_effect=lambda: q.get(block=None)):
            self.pool_test(os.path.join(self.mnt, 'zzz'))
            self.assertTrue(mockedCurl.perform.called)
    @staticmethod
    def _test_retry_write(self, tmp):
        with open(os.path.join(tmp, 'foo'), 'w') as f:
            f.write('foo')
        json.load(open(os.path.join(tmp, '.arvados#collection')))
