# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import hashlib
import json
import os
import unittest

from apiclient import http as apiclient_http
from unittest import mock

import arvados
from . import run_test_server
from . import arvados_testutil as tutil

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
        self.api_client = arvados.api('v1', cache=False, num_retries=0)
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
