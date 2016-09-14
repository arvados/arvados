import apiclient
import arvados
import arvados_fuse
import logging
import mock
import multiprocessing
import os
import re
import sys
import time
import unittest

from .integration_test import IntegrationTest

logger = logging.getLogger('arvados.arv-mount')

class TokenExpiryTest(IntegrationTest):
    def __init__(self, *args, **kwargs):
        super(TokenExpiryTest, self).__init__(*args, **kwargs)
        self.test_start_time = time.time()
        self.time_now = int(time.time())+1

    def fake_time(self):
        self.time_now += 1
        return self.time_now

    orig_open = arvados_fuse.Operations.open
    def fake_open(self, operations, *args, **kwargs):
        self.time_now += 86400*13
        logger.debug('opening file at time=%f', self.time_now)
        return self.orig_open(operations, *args, **kwargs)

    @mock.patch.object(arvados_fuse.Operations, 'open', autospec=True)
    @mock.patch('time.time')
    @mock.patch('arvados.keep.KeepClient.get')
    @IntegrationTest.mount(argv=['--mount-by-id', 'zzz'])
    def test_refresh_old_manifest(self, mocked_get, mocked_time, mocked_open):
        mocked_get.return_value = 'fake data'
        mocked_time.side_effect = self.fake_time
        mocked_open.side_effect = self.fake_open

        with mock.patch.object(self.mount.api, 'collections', wraps=self.mount.api.collections) as mocked_collections:
            mocked_collections.return_value = mocked_collections()
            with mock.patch.object(self.mount.api.collections(), 'get', wraps=self.mount.api.collections().get) as mocked_get:
                self.pool_test(os.path.join(self.mnt, 'zzz'))

        self.assertEqual(3, mocked_open.call_count)
        self.assertEqual(
            4, mocked_get.call_count,
            'Not enough calls to collections().get(): expected 4, got {!r}'.format(
                mocked_get.mock_calls))

    @staticmethod
    def _test_refresh_old_manifest(self, zzz):
        uuid = 'zzzzz-4zz18-op4e2lbej01tcvu'
        fnm = 'zzzzz-8i9sb-0vsrcqi7whchuil.log.txt'
        os.listdir(os.path.join(zzz, uuid))
        for _ in range(3):
            with open(os.path.join(zzz, uuid, fnm)) as f:
                f.read()
