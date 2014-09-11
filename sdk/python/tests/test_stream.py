#!/usr/bin/env python

import unittest

import arvados
from arvados import StreamReader, StreamFileReader

import run_test_server

class StreamReaderTestCase(unittest.TestCase):
    API_COLLECTIONS = run_test_server.fixture('collections')

    def manifest_for(self, coll_name):
        return self.API_COLLECTIONS[coll_name]['manifest_text']

    def test_manifest_text_without_keep_client(self):
        mtext = self.manifest_for('multilevel_collection_1')
        for line in mtext.rstrip('\n').split('\n'):
            reader = StreamReader(line.split())
            self.assertEqual(line + '\n', reader.manifest_text())


if __name__ == '__main__':
    unittest.main()
