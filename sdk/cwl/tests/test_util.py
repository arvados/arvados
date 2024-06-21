# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import datetime
import httplib2
import unittest

from unittest import mock

from arvados_cwl.util import *
from arvados.errors import ApiError
from arvados_cwl.util import common_prefix

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
        self.assertEqual(info["properties"], {"type" : "intermediate", "container_uuid" : "zzzzz-8i9sb-zzzzzzzzzzzzzzz"})

    def test_get_current_container_success(self):
        api = mock.MagicMock()
        api.containers().current().execute.return_value = {"uuid" : "zzzzz-8i9sb-zzzzzzzzzzzzzzz"}

        current_container = get_current_container(api)

        self.assertEqual(current_container, {"uuid" : "zzzzz-8i9sb-zzzzzzzzzzzzzzz"})

    def test_get_current_container_error(self):
        api = mock.MagicMock()
        api.containers().current().execute.side_effect = ApiError(httplib2.Response({"status": 300}), bytes(b""))
        logger = mock.MagicMock()

        with self.assertRaises(ApiError):
            get_current_container(api, num_retries=0, logger=logger)

    def test_get_current_container_404_error(self):
        api = mock.MagicMock()
        api.containers().current().execute.side_effect = ApiError(httplib2.Response({"status": 404}), bytes(b""))
        logger = mock.MagicMock()

        current_container = get_current_container(api, num_retries=0, logger=logger)
        self.assertEqual(current_container, None)

    def test_common_prefix(self):
        self.assertEqual(common_prefix("file:///foo/bar", ["file:///foo/bar/baz"]), "file:///foo/")
        self.assertEqual(common_prefix("file:///foo", ["file:///foo", "file:///foo/bar", "file:///foo/bar/"]), "file:///")
        self.assertEqual(common_prefix("file:///foo/", ["file:///foo/", "file:///foo/bar", "file:///foo/bar/"]), "file:///foo/")
        self.assertEqual(common_prefix("file:///foo/bar", ["file:///foo/bar", "file:///foo/baz", "file:///foo/quux/q2"]), "file:///foo/")
        self.assertEqual(common_prefix("file:///foo/bar/", ["file:///foo/bar/", "file:///foo/baz", "file:///foo/quux/q2"]), "file:///foo/")
        self.assertEqual(common_prefix("file:///foo/bar/splat", ["file:///foo/bar/splat", "file:///foo/baz", "file:///foo/quux/q2"]), "file:///foo/")
        self.assertEqual(common_prefix("file:///foo/bar/splat", ["file:///foo/bar/splat", "file:///nope", "file:///foo/quux/q2"]), "file:///")
        self.assertEqual(common_prefix("file:///blub/foo", ["file:///blub/foo", "file:///blub/foo/bar", "file:///blub/foo/bar/"]), "file:///blub/")

        # sanity check, the subsequent code strips off the prefix so
        # just confirm the logic doesn't have a fencepost error
        prefix = "file:///"
        self.assertEqual("file:///foo/bar"[len(prefix):], "foo/bar")
