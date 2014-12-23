#!/usr/bin/env python

import bz2
import gzip
import io
import mock
import os
import unittest
import hashlib

import arvados
from arvados import StreamReader, StreamFileReader, Range, import_manifest, export_manifest

import arvados_testutil as tutil


class ArvadosFileWriterTestCase(unittest.TestCase):
    class MockKeep(object):
        def __init__(self, blocks):
            self.blocks = blocks
        def get(self, locator, num_retries=0):
            return self.blocks[locator]
        def put(self, data):
            pdh = "%s+%i" % (hashlib.md5(data).hexdigest(), len(data))
            self.blocks[pdh] = str(data)
            return pdh

    def test_truncate(self):
        c = import_manifest('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt\n',
                              keep=ArvadosFileWriterTestCase.MockKeep({"781e5e245d69b566979b86e28d23f2c7+10": "0123456789"}))
        writer = c.open("count.txt", "r+")
        self.assertEqual(writer.size(), 10)
        writer.seek(5)
        self.assertEqual("56789", writer.read(8))
        writer.truncate(8)
        writer.seek(5, os.SEEK_SET)
        self.assertEqual("567", writer.read(8))

    def test_append(self):
        c = import_manifest('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt\n',
                              keep=ArvadosFileWriterTestCase.MockKeep({"781e5e245d69b566979b86e28d23f2c7+10": "0123456789"}))
        writer = c.open("count.txt", "r+")
        writer.seek(5, os.SEEK_SET)
        self.assertEqual("56789", writer.read(8))
        writer.seek(10, os.SEEK_SET)
        writer.write("foo")
        self.assertEqual(writer.size(), 13)
        writer.seek(5, os.SEEK_SET)
        self.assertEqual("56789foo", writer.read(8))

    def test_write0(self):
        c = import_manifest('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt\n',
                              keep=ArvadosFileWriterTestCase.MockKeep({"781e5e245d69b566979b86e28d23f2c7+10": "0123456789"}))
        writer = c.open("count.txt", "r+")
        self.assertEqual("0123456789", writer.readfrom(0, 13))
        writer.seek(0, os.SEEK_SET)
        writer.write("foo")
        self.assertEqual(writer.size(), 10)
        self.assertEqual("foo3456789", writer.readfrom(0, 13))
        self.assertEqual(". acbd18db4cc2f85cedef654fccc4a4d8+3 781e5e245d69b566979b86e28d23f2c7+10 0:3:count.txt 6:7:count.txt\n", export_manifest(c))

    def test_write1(self):
        c = import_manifest('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt\n',
                              keep=ArvadosFileWriterTestCase.MockKeep({"781e5e245d69b566979b86e28d23f2c7+10": "0123456789"}))
        writer = c.open("count.txt", "r+")
        self.assertEqual("0123456789", writer.readfrom(0, 13))
        writer.seek(3, os.SEEK_SET)
        writer.write("foo")
        self.assertEqual(writer.size(), 10)
        self.assertEqual("012foo6789", writer.readfrom(0, 13))
        self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:count.txt 10:3:count.txt 6:4:count.txt\n", export_manifest(c))

    def test_write2(self):
        c = import_manifest('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt\n',
                              keep=ArvadosFileWriterTestCase.MockKeep({"781e5e245d69b566979b86e28d23f2c7+10": "0123456789"}))
        writer = c.open("count.txt", "r+")
        self.assertEqual("0123456789", writer.readfrom(0, 13))
        writer.seek(7, os.SEEK_SET)
        writer.write("foo")
        self.assertEqual(writer.size(), 10)
        self.assertEqual("0123456foo", writer.readfrom(0, 13))
        self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 acbd18db4cc2f85cedef654fccc4a4d8+3 0:7:count.txt 10:3:count.txt\n", export_manifest(c))

    def test_write3(self):
        c = import_manifest('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt 0:10:count.txt\n',
                              keep=ArvadosFileWriterTestCase.MockKeep({"781e5e245d69b566979b86e28d23f2c7+10": "0123456789"}))
        writer = c.open("count.txt", "r+")
        self.assertEqual("012345678901234", writer.readfrom(0, 15))
        writer.seek(7, os.SEEK_SET)
        writer.write("foobar")
        self.assertEqual(writer.size(), 20)
        self.assertEqual("0123456foobar34", writer.readfrom(0, 15))
        self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 3858f62230ac3c915f300c664312c63f+6 0:7:count.txt 10:6:count.txt 3:7:count.txt\n", export_manifest(c))

    def test_write4(self):
        c = import_manifest('. 781e5e245d69b566979b86e28d23f2c7+10 0:4:count.txt 0:4:count.txt 0:4:count.txt',
                              keep=ArvadosFileWriterTestCase.MockKeep({"781e5e245d69b566979b86e28d23f2c7+10": "0123456789"}))
        writer = c.open("count.txt", "r+")
        self.assertEqual("012301230123", writer.readfrom(0, 15))
        writer.seek(2, os.SEEK_SET)
        writer.write("abcdefg")
        self.assertEqual(writer.size(), 12)
        self.assertEqual("01abcdefg123", writer.readfrom(0, 15))
        self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 7ac66c0f148de9519b8bd264312c4d64+7 0:2:count.txt 10:7:count.txt 1:3:count.txt\n", export_manifest(c))

    def test_write_large(self):
        c = import_manifest('. ' + arvados.config.EMPTY_BLOCK_LOCATOR + ' 0:0:count.txt',
                              keep=ArvadosFileWriterTestCase.MockKeep({}))
        writer = c.open("count.txt", "r+")
        text = ''.join(["0123456789" for a in xrange(0, 100)])
        for b in xrange(0, 100000):
            writer.write(text)
        self.assertEqual(writer.size(), 100000000)
        self.assertEqual(". a5de24f4417cfba9d5825eadc2f4ca49+67108000 598cc1a4ccaef8ab6e4724d87e675d78+32892000 0:100000000:count.txt\n", export_manifest(c))

    def test_write_rewrite0(self):
        c = import_manifest('. ' + arvados.config.EMPTY_BLOCK_LOCATOR + ' 0:0:count.txt',
                              keep=ArvadosFileWriterTestCase.MockKeep({}))
        writer = c.open("count.txt", "r+")
        for b in xrange(0, 10):
            writer.seek(0, os.SEEK_SET)
            writer.write("0123456789")
        writer.arvadosfile._repack_writes()
        self.assertEqual(writer.size(), 10)
        self.assertEqual("0123456789", writer.readfrom(0, 20))
        self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt\n", export_manifest(c))

    def test_write_rewrite1(self):
        c = import_manifest('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt',
                              keep=ArvadosFileWriterTestCase.MockKeep({"781e5e245d69b566979b86e28d23f2c7+10": "0123456789"}))
        writer = c.open("count.txt", "r+")
        for b in xrange(0, 10):
            writer.seek(10, os.SEEK_SET)
            writer.write("abcdefghij")
        writer.arvadosfile._repack_writes()
        self.assertEqual(writer.size(), 20)
        self.assertEqual("0123456789abcdefghij", writer.readfrom(0, 20))
        self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 a925576942e94b2ef57a066101b48876+10 0:20:count.txt\n", export_manifest(c))

    def test_write_rewrite2(self):
        c = import_manifest('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt',
                              keep=ArvadosFileWriterTestCase.MockKeep({"781e5e245d69b566979b86e28d23f2c7+10": "0123456789"}))
        writer = c.open("count.txt", "r+")
        for b in xrange(0, 10):
            writer.seek(5, os.SEEK_SET)
            writer.write("abcdefghij")
        writer.arvadosfile._repack_writes()
        self.assertEqual(writer.size(), 15)
        self.assertEqual("01234abcdefghij", writer.readfrom(0, 20))
        self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 a925576942e94b2ef57a066101b48876+10 0:5:count.txt 10:10:count.txt\n", export_manifest(c))
