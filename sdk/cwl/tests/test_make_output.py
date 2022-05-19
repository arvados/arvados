# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from future import standard_library
standard_library.install_aliases()

import functools
import json
import logging
import mock
import os
import io
import unittest

import arvados
import arvados_cwl
import arvados_cwl.executor
from .mock_discovery import get_rootDesc

class TestMakeOutput(unittest.TestCase):
    def setUp(self):
        self.api = mock.MagicMock()
        self.api._rootDesc = get_rootDesc()

    def tearDown(self):
        root_logger = logging.getLogger('')

        # Remove existing RuntimeStatusLoggingHandlers if they exist
        handlers = [h for h in root_logger.handlers if not isinstance(h, arvados_cwl.executor.RuntimeStatusLoggingHandler)]
        root_logger.handlers = handlers

    @mock.patch("arvados.collection.Collection")
    @mock.patch("arvados.collection.CollectionReader")
    def test_make_output_collection(self, reader, col):
        keep_client = mock.MagicMock()
        runner = arvados_cwl.executor.ArvCwlExecutor(self.api, keep_client=keep_client)
        runner.project_uuid = 'zzzzz-j7d0g-zzzzzzzzzzzzzzz'

        final = mock.MagicMock()
        col.return_value = final
        readermock = mock.MagicMock()
        reader.return_value = readermock

        final_uuid = final.manifest_locator()
        num_retries = runner.num_retries

        cwlout = io.StringIO()
        openmock = mock.MagicMock()
        final.open.return_value = openmock
        openmock.__enter__.return_value = cwlout

        _, runner.final_output_collection = runner.make_output_collection("Test output", ["foo"], "tag0,tag1,tag2", {}, {
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
        final.save_new.assert_has_calls([mock.call(ensure_unique_name=True, name='Test output', owner_uuid='zzzzz-j7d0g-zzzzzzzzzzzzzzz', properties={}, storage_classes=['foo'])])
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

    @mock.patch("arvados.collection.Collection")
    @mock.patch("arvados.collection.CollectionReader")
    def test_make_output_for_multiple_file_targets(self, reader, col):
        keep_client = mock.MagicMock()
        runner = arvados_cwl.executor.ArvCwlExecutor(self.api, keep_client=keep_client)
        runner.project_uuid = 'zzzzz-j7d0g-zzzzzzzzzzzzzzz'

        final = mock.MagicMock()
        col.return_value = final
        readermock = mock.MagicMock()
        reader.return_value = readermock

        # This output describes a single file listed in 2 different directories
        _, runner.final_output_collection = runner.make_output_collection("Test output", ["foo"], "", {}, { 'out': [
        {
            'basename': 'testdir1',
            'listing': [
                {
                    'basename': 'test.txt',
                    'nameroot': 'test',
                    'nameext': '.txt',
                    'location': 'keep:99999999999999999999999999999991+99/test.txt',
                    'class': 'File',
                    'size': 16
                }
            ],
            'location': '_:99999999999999999999999999999992+99',
            'class': 'Directory'
        },
        {
            'basename': 'testdir2',
            'listing': [
                {
                    'basename': 'test.txt',
                    'nameroot': 'test',
                    'nameext': '.txt',
                    'location': 'keep:99999999999999999999999999999991+99/test.txt',
                    'class':
                    'File',
                    'size': 16
                }
            ],
            'location': '_:99999999999999999999999999999993+99',
            'class': 'Directory'
        }]})

        # Check that copy is called on the collection for both locations
        final.copy.assert_any_call("test.txt", "testdir1/test.txt", source_collection=mock.ANY, overwrite=mock.ANY)
        final.copy.assert_any_call("test.txt", "testdir2/test.txt", source_collection=mock.ANY, overwrite=mock.ANY)

    @mock.patch("arvados.collection.Collection")
    @mock.patch("arvados.collection.CollectionReader")
    def test_make_output_for_literal_name_conflicts(self, reader, col):
        keep_client = mock.MagicMock()
        runner = arvados_cwl.executor.ArvCwlExecutor(self.api, keep_client=keep_client)
        runner.project_uuid = 'zzzzz-j7d0g-zzzzzzzzzzzzzzz'

        final = mock.MagicMock()
        col.return_value = final
        readermock = mock.MagicMock()
        reader.return_value = readermock

        # This output describes two literals with the same basename
        _, runner.final_output_collection = runner.make_output_collection("Test output", ["foo"], "",  {}, [
        {
            'lit':
            {
                'basename': 'a_file',
                'nameext': '',
                'nameroot': 'a_file',
                'location': '_:f168fc0c-4291-40aa-a04e-366d57390560',
                'class': 'File',
                'contents': 'Hello file literal.'
            }
        },
        {
            'lit':
            {
                'basename': 'a_file',
                'nameext': '',
                'nameroot': 'a_file',
                'location': '_:1728da8f-c64e-4a3e-b2e2-1ee356be7bc8',
                'class': 'File',
                'contents': 'Hello file literal.'
            }
        }])

        # Check that the file name conflict is resolved and open is called for both
        final.open.assert_any_call("a_file", "wb")
        final.open.assert_any_call("a_file_2", "wb")
