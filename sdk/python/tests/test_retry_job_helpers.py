# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import absolute_import
from builtins import object
import mock
import os
import unittest
import hashlib
from . import run_test_server
import json
import arvados
from . import arvados_testutil as tutil
from apiclient import http as apiclient_http


@tutil.skip_sleep
class ApiClientRetryTestMixin(object):

    TEST_UUID = 'zzzzz-zzzzz-zzzzzzzzzzzzzzz'
    TEST_LOCATOR = 'd41d8cd98f00b204e9800998ecf8427e+0'

    @classmethod
    def setUpClass(cls):
        run_test_server.run()

    def setUp(self):
        # Patch arvados.api() to return our mock API, so we can mock
        # its http requests.
        self.api_client = arvados.api('v1', cache=False)
        self.api_patch = mock.patch('arvados.api', return_value=self.api_client)
        self.api_patch.start()

    def tearDown(self):
        self.api_patch.stop()

    def run_method(self):
        raise NotImplementedError("test subclasses must define run_method")

    def test_immediate_success(self):
        with tutil.mock_api_responses(self.api_client, '{}', [200]):
            self.run_method()

    def test_immediate_failure(self):
        with tutil.mock_api_responses(self.api_client, '{}', [400]), self.assertRaises(self.DEFAULT_EXCEPTION):
            self.run_method()

    def test_retry_then_success(self):
        with tutil.mock_api_responses(self.api_client, '{}', [500, 200]):
            self.run_method()

    def test_error_after_default_retries_exhausted(self):
        with tutil.mock_api_responses(self.api_client, '{}', [500, 500, 500, 500, 500, 500, 200]), self.assertRaises(self.DEFAULT_EXCEPTION):
            self.run_method()

    def test_no_retry_after_immediate_success(self):
        with tutil.mock_api_responses(self.api_client, '{}', [200, 400]):
            self.run_method()


class CurrentJobTestCase(ApiClientRetryTestMixin, unittest.TestCase):

    DEFAULT_EXCEPTION = arvados.errors.ApiError

    def setUp(self):
        super(CurrentJobTestCase, self).setUp()
        os.environ['JOB_UUID'] = 'zzzzz-zzzzz-zzzzzzzzzzzzzzz'
        os.environ['JOB_WORK'] = '.'

    def tearDown(self):
        del os.environ['JOB_UUID']
        del os.environ['JOB_WORK']
        arvados._current_job = None
        super(CurrentJobTestCase, self).tearDown()

    def run_method(self):
        arvados.current_job()
