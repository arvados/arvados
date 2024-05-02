# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import apiclient
import arvados
import arvados_fuse
import logging
import multiprocessing
import os
import re
import sys
import time
import unittest

from unittest import mock

from .integration_test import IntegrationTest

logger = logging.getLogger('arvados.arv-mount')

class TokenExpiryTest(IntegrationTest):
    def setUp(self):
        super(TokenExpiryTest, self).setUp()
        self.test_start_time = time.time()
        self.time_now = int(time.time())+1

    def fake_time(self):
        self.time_now += 1
        return self.time_now

    orig_open = arvados_fuse.Operations.open
    def fake_open(self, operations, *args, **kwargs):
        self.time_now += 86400*13
        logger.debug('opening file at time=%f', self.time_now)
        return TokenExpiryTest.orig_open(operations, *args, **kwargs)

    @mock.patch.object(arvados_fuse.Operations, 'open', autospec=True)
    @mock.patch.object(time, 'time', return_value=0)
    @mock.patch('arvados.keep.KeepClient.get')
    @IntegrationTest.mount(argv=['--mount-by-id', 'zzz'])
    def test_refresh_old_manifest(self, mocked_get, mocked_time, mocked_open):
        # This test (and associated behavior) is still not strong
        # enough. We should ensure old tokens are never used even if
        # blobSignatureTtl seconds elapse between open() and
        # read(). See https://dev.arvados.org/issues/10008

        mocked_get.return_value = b'fake data'
        mocked_time.side_effect = self.fake_time
        mocked_open.side_effect = self.fake_open

        with mock.patch.object(self.mount.api, 'collections', wraps=self.mount.api.collections) as mocked_collections:
            mocked_collections.return_value = mocked_collections()
            with mock.patch.object(self.mount.api.collections(), 'get', wraps=self.mount.api.collections().get) as mocked_get:
                self.pool_test(os.path.join(self.mnt, 'zzz'))

        # open() several times here to make sure we don't reach our
        # quota of mocked_get.call_count dishonestly (e.g., the first
        # open causes 5 mocked_get, and the rest cause none).
        self.assertEqual(8, mocked_open.call_count)
        self.assertGreaterEqual(
            mocked_get.call_count, 8,
            'Not enough calls to collections().get(): expected 8, got {!r}'.format(
                mocked_get.mock_calls))

    @staticmethod
    def _test_refresh_old_manifest(self, zzz):
        uuid = 'zzzzz-4zz18-op4e2lbej01tcvu'
        fnm = 'zzzzz-8i9sb-0vsrcqi7whchuil.log.txt'
        os.listdir(os.path.join(zzz, uuid))
        for _ in range(8):
            with open(os.path.join(zzz, uuid, fnm)) as f:
                f.read()
