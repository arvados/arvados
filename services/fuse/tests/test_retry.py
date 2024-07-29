# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import arvados
import arvados_fuse.command
import json
import os
import pycurl
import queue
import tempfile
import unittest

from unittest import mock

from . import run_test_server
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

    def test_default_retry_10(self):
        self._test_retry(10, [])

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
        q = queue.Queue()
        q.put(mockedCurl)
        q.put(pycurl.Curl())
        q.put(pycurl.Curl())
        with mock.patch('arvados.keep.KeepClient._KeepService._get_user_agent', side_effect=q.get_nowait):
            self.pool_test(os.path.join(self.mnt, 'zzz'))
            self.assertTrue(mockedCurl.perform.called)
    @staticmethod
    def _test_retry_write(self, tmp):
        with open(os.path.join(tmp, 'foo'), 'w') as f:
            f.write('foo')
        json.load(open(os.path.join(tmp, '.arvados#collection')))
