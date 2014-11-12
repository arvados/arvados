#!/usr/bin/env python

import mock
import unittest

import arvados
from arvados import StreamReader, StreamFileReader

import arvados_testutil as tutil
import run_test_server

class StreamRetryTestMixin(object):
    # Define reader_for(coll_name, **kwargs)
    # and read_for_test(reader, size, **kwargs).
    API_COLLECTIONS = run_test_server.fixture('collections')

    def keep_client(self):
        return arvados.KeepClient(proxy='http://[%s]:1' % (tutil.TEST_HOST,),
                                  local_store='')

    def manifest_for(self, coll_name):
        return self.API_COLLECTIONS[coll_name]['manifest_text']

    @tutil.skip_sleep
    def test_success_without_retries(self):
        reader = self.reader_for('bar_file')
        with tutil.mock_get_responses('bar', 200):
            self.assertEqual('bar', self.read_for_test(reader, 3))

    @tutil.skip_sleep
    def test_read_no_default_retry(self):
        reader = self.reader_for('user_agreement')
        with tutil.mock_get_responses('', 500):
            with self.assertRaises(arvados.errors.KeepReadError):
                self.read_for_test(reader, 10)

    @tutil.skip_sleep
    def test_read_with_instance_retries(self):
        reader = self.reader_for('foo_file', num_retries=3)
        with tutil.mock_get_responses('foo', 500, 200):
            self.assertEqual('foo', self.read_for_test(reader, 3))

    @tutil.skip_sleep
    def test_read_with_method_retries(self):
        reader = self.reader_for('foo_file')
        with tutil.mock_get_responses('foo', 500, 200):
            self.assertEqual('foo',
                             self.read_for_test(reader, 3, num_retries=3))

    @tutil.skip_sleep
    def test_read_instance_retries_exhausted(self):
        reader = self.reader_for('bar_file', num_retries=3)
        with tutil.mock_get_responses('bar', 500, 500, 500, 500, 200):
            with self.assertRaises(arvados.errors.KeepReadError):
                self.read_for_test(reader, 3)

    @tutil.skip_sleep
    def test_read_method_retries_exhausted(self):
        reader = self.reader_for('bar_file')
        with tutil.mock_get_responses('bar', 500, 500, 500, 500, 200):
            with self.assertRaises(arvados.errors.KeepReadError):
                self.read_for_test(reader, 3, num_retries=3)

    @tutil.skip_sleep
    def test_method_retries_take_precedence(self):
        reader = self.reader_for('user_agreement', num_retries=10)
        with tutil.mock_get_responses('', 500, 500, 500, 200):
            with self.assertRaises(arvados.errors.KeepReadError):
                self.read_for_test(reader, 10, num_retries=1)


class StreamReaderTestCase(unittest.TestCase, StreamRetryTestMixin):
    def reader_for(self, coll_name, **kwargs):
        return StreamReader(self.manifest_for(coll_name).split(),
                            self.keep_client(), **kwargs)

    def read_for_test(self, reader, byte_count, **kwargs):
        return reader.readfrom(0, byte_count, **kwargs)

    def test_manifest_text_without_keep_client(self):
        mtext = self.manifest_for('multilevel_collection_1')
        for line in mtext.rstrip('\n').split('\n'):
            reader = StreamReader(line.split())
            self.assertEqual(line + '\n', reader.manifest_text())


class StreamFileReadTestCase(unittest.TestCase, StreamRetryTestMixin):
    def reader_for(self, coll_name, **kwargs):
        return StreamReader(self.manifest_for(coll_name).split(),
                            self.keep_client(), **kwargs).all_files()[0]

    def read_for_test(self, reader, byte_count, **kwargs):
        return reader.read(byte_count, **kwargs)


class StreamFileReadFromTestCase(StreamFileReadTestCase):
    def read_for_test(self, reader, byte_count, **kwargs):
        return reader.readfrom(0, byte_count, **kwargs)


class StreamFileReadAllTestCase(StreamFileReadTestCase):
    def read_for_test(self, reader, byte_count, **kwargs):
        return ''.join(reader.readall(**kwargs))


class StreamFileReadAllDecompressedTestCase(StreamFileReadTestCase):
    def read_for_test(self, reader, byte_count, **kwargs):
        return ''.join(reader.readall_decompressed(**kwargs))


class StreamFileReadlinesTestCase(StreamFileReadTestCase):
    def read_for_test(self, reader, byte_count, **kwargs):
        return ''.join(reader.readlines(**kwargs))


if __name__ == '__main__':
    unittest.main()
