import logging
import mock
import unittest
import os
import functools
import json
import StringIO

import arvados_cwl

class TestMakeOutput(unittest.TestCase):
    @mock.patch("arvados.collection.Collection")
    @mock.patch("arvados.collection.CollectionReader")
    def test_make_output_collection(self, reader, col):
        api = mock.MagicMock()
        keep_client = mock.MagicMock()
        runner = arvados_cwl.ArvCwlRunner(api, keep_client=keep_client)
        runner.project_uuid = 'zzzzz-j7d0g-zzzzzzzzzzzzzzz'

        final = mock.MagicMock()
        col.return_value = final
        readermock = mock.MagicMock()
        reader.return_value = readermock

        cwlout = StringIO.StringIO()
        openmock = mock.MagicMock()
        final.open.return_value = openmock
        openmock.__enter__.return_value = cwlout

        runner.make_output_collection("Test output", {
            "foo": {
                "class": "File",
                "location": "keep:99999999999999999999999999999991+99/foo.txt",
                "size": 3,
                "basename": "foo.txt"
            },
            "bar": {
                "class": "File",
                "location": "keep:99999999999999999999999999999992+99/bar.txt",
                "basename": "baz.txt"
            }
        })

        final.copy.assert_has_calls([mock.call('bar.txt', 'baz.txt', overwrite=False, source_collection=readermock)])
        final.copy.assert_has_calls([mock.call('foo.txt', 'foo.txt', overwrite=False, source_collection=readermock)])
        final.save_new.assert_has_calls([mock.call(ensure_unique_name=True, name='Test output', owner_uuid='zzzzz-j7d0g-zzzzzzzzzzzzzzz')])
        self.assertEqual("""{
    "bar": {
        "class": "File",
        "location": "baz.txt"
    },
    "foo": {
        "class": "File",
        "location": "foo.txt"
    }
}""", cwlout.getvalue())

        self.assertIs(final, runner.final_output_collection)
