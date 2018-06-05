# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import functools
import json
import logging
import mock
import os
import StringIO
import unittest

import arvados
import arvados_cwl
from .mock_discovery import get_rootDesc

class TestMakeOutput(unittest.TestCase):
    def setUp(self):
        self.api = mock.MagicMock()
        self.api._rootDesc = get_rootDesc()

    @mock.patch("arvados.collection.Collection")
    @mock.patch("arvados.collection.CollectionReader")
    def test_make_output_collection(self, reader, col):
        keep_client = mock.MagicMock()
        runner = arvados_cwl.ArvCwlRunner(self.api, keep_client=keep_client)
        runner.project_uuid = 'zzzzz-j7d0g-zzzzzzzzzzzzzzz'

        final = mock.MagicMock()
        col.return_value = final
        readermock = mock.MagicMock()
        reader.return_value = readermock

        final_uuid = final.manifest_locator()
        num_retries = runner.num_retries

        cwlout = StringIO.StringIO()
        openmock = mock.MagicMock()
        final.open.return_value = openmock
        openmock.__enter__.return_value = cwlout

        _, runner.final_output_collection = runner.make_output_collection("Test output", ["foo"], "tag0,tag1,tag2", {
            "foo": {
                "class": "File",
                "location": "keep:99999999999999999999999999999991+99/foo.txt",
                "size": 3,
                "basename": "foo.txt"
            },
            "bar": {
                "class": "File",
                "location": "keep:99999999999999999999999999999992+99/bar.txt",
                "basename": "baz.txt",
                "size": 4
            }
        })

        final.copy.assert_has_calls([mock.call('bar.txt', 'baz.txt', overwrite=False, source_collection=readermock)])
        final.copy.assert_has_calls([mock.call('foo.txt', 'foo.txt', overwrite=False, source_collection=readermock)])
        final.save_new.assert_has_calls([mock.call(ensure_unique_name=True, name='Test output', owner_uuid='zzzzz-j7d0g-zzzzzzzzzzzzzzz', storage_classes=['foo'])])
        self.assertEqual("""{
    "bar": {
        "basename": "baz.txt",
        "class": "File",
        "location": "baz.txt",
        "size": 4
    },
    "foo": {
        "basename": "foo.txt",
        "class": "File",
        "location": "foo.txt",
        "size": 3
    }
}""", cwlout.getvalue())

        self.assertIs(final, runner.final_output_collection)
        self.assertIs(final_uuid, runner.final_output_collection.manifest_locator())
        self.api.links().create.assert_has_calls([mock.call(body={"head_uuid": final_uuid, "link_class": "tag", "name": "tag0"}), mock.call().execute(num_retries=num_retries)])
        self.api.links().create.assert_has_calls([mock.call(body={"head_uuid": final_uuid, "link_class": "tag", "name": "tag1"}), mock.call().execute(num_retries=num_retries)])
        self.api.links().create.assert_has_calls([mock.call(body={"head_uuid": final_uuid, "link_class": "tag", "name": "tag2"}), mock.call().execute(num_retries=num_retries)])
