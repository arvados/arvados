# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import unittest
import mock
import datetime
import httplib2

from arvados_cwl.util import *
from arvados.errors import ApiError

class MockDateTime(datetime.datetime):
    @classmethod
    def utcnow(cls):
        return datetime.datetime(2018, 1, 1, 0, 0, 0, 0)

datetime.datetime = MockDateTime

class TestUtil(unittest.TestCase):
    def test_get_intermediate_collection_info(self):
        name = "one"
        current_container = {"uuid": "zzzzz-8i9sb-zzzzzzzzzzzzzzz"}
        intermediate_output_ttl = 120

        info = get_intermediate_collection_info(name, current_container, intermediate_output_ttl)

        self.assertEqual(info["name"], "Intermediate collection for step one")
        self.assertEqual(info["trash_at"], datetime.datetime(2018, 1, 1, 0, 2, 0, 0))
        self.assertEqual(info["properties"], {"type" : "intermediate", "container" : "zzzzz-8i9sb-zzzzzzzzzzzzzzz"})

    def test_get_current_container_success(self):
        api = mock.MagicMock()
        api.containers().current().execute.return_value = {"uuid" : "zzzzz-8i9sb-zzzzzzzzzzzzzzz"}

        current_container = get_current_container(api)

        self.assertEqual(current_container, {"uuid" : "zzzzz-8i9sb-zzzzzzzzzzzzzzz"})

    def test_get_current_container_error(self):
        api = mock.MagicMock()
        api.containers().current().execute.side_effect = ApiError(httplib2.Response({"status": 300}), "")
        logger = mock.MagicMock()

        self.assertRaises(ApiError, get_current_container(api, num_retries=0, logger=logger))
