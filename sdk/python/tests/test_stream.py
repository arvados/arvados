# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import bz2
import gzip
import io
import os
import unittest
import hashlib

from unittest import mock

import arvados

from . import arvados_testutil as tutil
from . import run_test_server

class StreamFileReaderTestMixin(object):
    def test_read_block_crossing_behavior(self):
        # read() calls will be aligned on block boundaries - see #3663.
        sfile = self.make_count_reader()
        self.assertEqual(b'123', sfile.read(10))

    def test_small_read(self):
        sfile = self.make_count_reader()
        self.assertEqual(b'12', sfile.read(2))

    def test_successive_reads(self):
        sfile = self.make_count_reader()
        for expect in [b'1234', b'5678', b'9', b'']:
            self.assertEqual(expect, sfile.read(4))

    def test_readfrom_spans_blocks(self):
        sfile = self.make_count_reader()
        self.assertEqual(b'6789', sfile.readfrom(5, 12))

    def test_small_readfrom_spanning_blocks(self):
        sfile = self.make_count_reader()
        self.assertEqual(b'2345', sfile.readfrom(1, 4))

    def test_readall(self):
        sfile = self.make_count_reader()
        self.assertEqual(b'123456789', b''.join(sfile.readall()))

    def test_one_arg_seek(self):
        self.test_absolute_seek([])

    def test_absolute_seek(self, args=[os.SEEK_SET]):
        sfile = self.make_count_reader()
        sfile.seek(6, *args)
        self.assertEqual(b'78', sfile.read(2))
        sfile.seek(4, *args)
        self.assertEqual(b'56', sfile.read(2))

    def test_relative_seek(self, args=[os.SEEK_CUR]):
        sfile = self.make_count_reader()
        self.assertEqual(b'12', sfile.read(2))
        sfile.seek(2, *args)
        self.assertEqual(b'56', sfile.read(2))

    def test_end_seek(self):
        sfile = self.make_count_reader()
        sfile.seek(-6, os.SEEK_END)
        self.assertEqual(b'45', sfile.read(2))

    def test_seek_min_zero(self):
        sfile = self.make_count_reader()
        self.assertEqual(0, sfile.tell())
        with self.assertRaises(IOError):
            sfile.seek(-2, os.SEEK_SET)
        self.assertEqual(0, sfile.tell())

    def test_seek_max_size(self):
        sfile = self.make_count_reader()
        sfile.seek(2, os.SEEK_END)
        # POSIX permits seeking past end of file.
        self.assertEqual(11, sfile.tell())

    def test_size(self):
        self.assertEqual(9, self.make_count_reader().size())

    def test_tell_after_small_read(self):
        sfile = self.make_count_reader()
        sfile.read(1)
        self.assertEqual(1, sfile.tell())

    def test_no_read_after_close(self):
        sfile = self.make_count_reader()
        sfile.close()
        self.assertRaises(ValueError, sfile.read, 2)

    def test_context(self):
        with self.make_count_reader() as sfile:
            self.assertFalse(sfile.closed, "reader is closed inside context")
            self.assertEqual(b'12', sfile.read(2))
        self.assertTrue(sfile.closed, "reader is open after context")

    def check_lines(self, actual):
        self.assertEqual(['one\n', 'two\n', '\n', 'three\n', 'four\n', '\n'],
                         actual)

    def test_readline(self):
        reader = self.make_newlines_reader()
        actual = []
        while True:
            data = reader.readline()
            if not data:
                break
            actual.append(data)
        self.check_lines(actual)

    def test_readlines(self):
        self.check_lines(self.make_newlines_reader().readlines())

    def test_iteration(self):
        self.check_lines(list(iter(self.make_newlines_reader())))

    def test_readline_size(self):
        reader = self.make_newlines_reader()
        self.assertEqual('on', reader.readline(2))
        self.assertEqual('e\n', reader.readline(4))
        self.assertEqual('two\n', reader.readline(6))
        self.assertEqual('\n', reader.readline(8))
        self.assertEqual('thre', reader.readline(4))

    def test_readlines_sizehint(self):
        result = self.make_newlines_reader().readlines(8)
        self.assertEqual(['one\n', 'two\n', '\n', 'three\n', 'four\n', '\n'], result)

    def test_name_attribute(self):
        sfile = self.make_file_reader(name='nametest')
        self.assertEqual('nametest', sfile.name)

    def check_decompressed_name(self, filename, expect):
        reader = self.make_file_reader(name=filename)
        self.assertEqual(expect, reader.decompressed_name())

    def test_decompressed_name_uncompressed_file(self):
        self.check_decompressed_name('test.log', 'test.log')

    def test_decompressed_name_gzip_file(self):
        self.check_decompressed_name('test.log.gz', 'test.log')

    def test_decompressed_name_bz2_file(self):
        self.check_decompressed_name('test.log.bz2', 'test.log')

    def check_decompression(self, compress_ext, compress_func):
        test_text = b'decompression\ntest\n'
        test_data = compress_func(test_text)
        reader = self.make_file_reader(name='test.'+compress_ext, data=test_data)
        self.assertEqual(test_text, b''.join(reader.readall_decompressed()))

    @staticmethod
    def gzip_compress(data):
        compressed_data = io.BytesIO()
        with gzip.GzipFile(fileobj=compressed_data, mode='wb') as gzip_file:
            gzip_file.write(data)
        return compressed_data.getvalue()

    def test_no_decompression(self):
        self.check_decompression('log', lambda s: s)

    def test_gzip_decompression(self):
        self.check_decompression('gz', self.gzip_compress)

    def test_bz2_decompression(self):
        self.check_decompression('bz2', bz2.compress)

    def test_readline_then_readlines(self):
        reader = self.make_newlines_reader()
        data = reader.readline()
        self.assertEqual('one\n', data)
        data = reader.readlines()
        self.assertEqual(['two\n', '\n', 'three\n', 'four\n', '\n'], data)

    def test_readline_then_readall(self):
        reader = self.make_newlines_reader()
        data = reader.readline()
        self.assertEqual('one\n', data)
        self.assertEqual(b''.join([b'two\n', b'\n', b'three\n', b'four\n', b'\n']), b''.join(reader.readall()))


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
        with tutil.mock_keep_responses('bar', 200):
            reader = self.reader_for('bar_file')
            self.assertEqual(b'bar', self.read_for_test(reader, 3))

    @tutil.skip_sleep
    def test_read_with_instance_retries(self):
        with tutil.mock_keep_responses('foo', 500, 200):
            reader = self.reader_for('foo_file', num_retries=3)
            self.assertEqual(b'foo', self.read_for_test(reader, 3))

    @tutil.skip_sleep
    def test_read_with_method_retries(self):
        with tutil.mock_keep_responses('foo', 500, 200):
            reader = self.reader_for('foo_file')
            self.assertEqual(b'foo',
                             self.read_for_test(reader, 3, num_retries=3))

    @tutil.skip_sleep
    def test_read_instance_retries_exhausted(self):
        with tutil.mock_keep_responses('bar', 500, 500, 500, 500, 200):
            reader = self.reader_for('bar_file', num_retries=3)
            with self.assertRaises(arvados.errors.KeepReadError):
                self.read_for_test(reader, 3)

    @tutil.skip_sleep
    def test_read_method_retries_exhausted(self):
        with tutil.mock_keep_responses('bar', 500, 500, 500, 500, 200):
            reader = self.reader_for('bar_file')
            with self.assertRaises(arvados.errors.KeepReadError):
                self.read_for_test(reader, 3, num_retries=3)

    @tutil.skip_sleep
    def test_method_retries_take_precedence(self):
        with tutil.mock_keep_responses('', 500, 500, 500, 200):
            reader = self.reader_for('user_agreement', num_retries=10)
            with self.assertRaises(arvados.errors.KeepReadError):
                self.read_for_test(reader, 10, num_retries=1)


if __name__ == '__main__':
    unittest.main()
