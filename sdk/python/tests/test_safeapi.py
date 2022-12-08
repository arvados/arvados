# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import os
import unittest

import googleapiclient

from arvados import safeapi

from . import run_test_server

class SafeApiTest(run_test_server.TestCaseWithServers):
    MAIN_SERVER = {}

    def test_constructor(self):
        env_mapping = {
            key: value
            for key, value in os.environ.items()
            if key.startswith('ARVADOS_API_')
        }
        extra_params = {
            'timeout': 299,
        }
        base_params = {
            key[12:].lower(): value
            for key, value in env_mapping.items()
        }
        try:
            base_params['insecure'] = base_params.pop('host_insecure')
        except KeyError:
            pass
        expected_keep_params = {}
        for config, params, subtest in [
                (None, {}, "default arguments"),
                (None, extra_params, "extra params"),
                (env_mapping, {}, "explicit config"),
                (env_mapping, extra_params, "explicit config and params"),
                ({}, base_params, "params only"),
        ]:
            with self.subTest(f"test constructor with {subtest}"):
                expected_timeout = params.get('timeout', 300)
                expected_params = dict(params)
                keep_params = dict(expected_keep_params)
                client = safeapi.ThreadSafeApiCache(config, keep_params, params, 'v1')
                self.assertTrue(hasattr(client, 'localapi'), "client missing localapi method")
                self.assertEqual(client.api_token, os.environ['ARVADOS_API_TOKEN'])
                self.assertEqual(client._http.timeout, expected_timeout)
                self.assertEqual(params, expected_params,
                                 "api_params was modified in-place")
                self.assertEqual(keep_params, expected_keep_params,
                                 "keep_params was modified in-place")

    def test_constructor_no_args(self):
        client = safeapi.ThreadSafeApiCache()
        self.assertTrue(hasattr(client, 'localapi'), "client missing localapi method")
        self.assertEqual(client.api_token, os.environ['ARVADOS_API_TOKEN'])
        self.assertTrue(client.insecure)

    def test_constructor_bad_version(self):
        with self.assertRaises(googleapiclient.errors.UnknownApiNameOrVersion):
            safeapi.ThreadSafeApiCache(version='BadTestVersion')
