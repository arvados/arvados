# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import absolute_import
from builtins import hex
from builtins import str
from builtins import range
from builtins import object
import bz2
import datetime
import gzip
import io
import mock
import os
import unittest
import time

import arvados
from arvados._ranges import Range
from arvados.keep import KeepLocator
from arvados.collection import Collection, CollectionReader
from arvados.arvfile import ArvadosFile, ArvadosFileReader

from . import arvados_testutil as tutil
from .test_stream import StreamFileReaderTestCase, StreamRetryTestMixin

class ArvadosFileWriterTestCase(unittest.TestCase):
    class MockKeep(object):
        def __init__(self, blocks):
            self.blocks = blocks
            self.requests = []
        def get(self, locator, num_retries=0):
            self.requests.append(locator)
            return self.blocks.get(locator)
        def get_from_cache(self, locator):
            self.requests.append(locator)
            return self.blocks.get(locator)
        def put(self, data, num_retries=None, copies=None):
            pdh = tutil.str_keep_locator(data)
            self.blocks[pdh] = bytes(data)
            return pdh

    class MockApi(object):
        def __init__(self, b, r):
            self.body = b
            self.response = r
            self._schema = ArvadosFileWriterTestCase.MockApi.MockSchema()
            self._rootDesc = {}
        class MockSchema(object):
            def __init__(self):
                self.schemas = {'Collection': {'properties': {'replication_desired': {'type':'integer'}}}}
        class MockCollections(object):
            def __init__(self, b, r):
                self.body = b
                self.response = r
            class Execute(object):
                def __init__(self, r):
                    self.response = r
                def execute(self, num_retries=None):
                    return self.response
            def create(self, ensure_unique_name=False, body=None):
                if body != self.body:
                    raise Exception("Body %s does not match expectation %s" % (body, self.body))
                return ArvadosFileWriterTestCase.MockApi.MockCollections.Execute(self.response)
            def update(self, uuid=None, body=None):
                return ArvadosFileWriterTestCase.MockApi.MockCollections.Execute(self.response)
        def collections(self):
            return ArvadosFileWriterTestCase.MockApi.MockCollections(self.body, self.response)


    def test_truncate(self):
        keep = ArvadosFileWriterTestCase.MockKeep({
            "781e5e245d69b566979b86e28d23f2c7+10": b"0123456789",
        })
        api = ArvadosFileWriterTestCase.MockApi({
            "name": "test_truncate",
            "manifest_text": ". 781e5e245d69b566979b86e28d23f2c7+10 0:8:count.txt\n",
            "replication_desired": None,
        }, {
            "uuid": "zzzzz-4zz18-mockcollection0",
            "manifest_text": ". 781e5e245d69b566979b86e28d23f2c7+10 0:8:count.txt\n",
            "portable_data_hash":"7fcd0eaac3aad4c31a6a0e756475da92+52",
        })
        with Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt\n',
                        api_client=api, keep_client=keep) as c:
            writer = c.open("count.txt", "rb+")
            self.assertEqual(writer.size(), 10)
            self.assertEqual(b"0123456789", writer.read(12))

            writer.truncate(8)

            # Make sure reading off the end doesn't break
            self.assertEqual(b"", writer.read(12))

            self.assertEqual(writer.size(), 8)
            writer.seek(0, os.SEEK_SET)
            self.assertEqual(b"01234567", writer.read(12))

            self.assertIsNone(c.manifest_locator())
            self.assertTrue(c.modified())
            c.save_new("test_truncate")
            self.assertEqual("zzzzz-4zz18-mockcollection0", c.manifest_locator())
            self.assertFalse(c.modified())


    def test_truncate2(self):
        keep = ArvadosFileWriterTestCase.MockKeep({
            "781e5e245d69b566979b86e28d23f2c7+10": b"0123456789",
        })
        api = ArvadosFileWriterTestCase.MockApi({
            "name": "test_truncate2",
            "manifest_text": ". 781e5e245d69b566979b86e28d23f2c7+10 7f614da9329cd3aebf59b91aadc30bf0+67108864 0:12:count.txt\n",
            "replication_desired": None,
        }, {
            "uuid": "zzzzz-4zz18-mockcollection0",
            "manifest_text": ". 781e5e245d69b566979b86e28d23f2c7+10 7f614da9329cd3aebf59b91aadc30bf0+67108864 0:12:count.txt\n",
            "portable_data_hash": "272da898abdf86ddc71994835e3155f8+95",
        })
        with Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt\n',
                        api_client=api, keep_client=keep) as c:
            writer = c.open("count.txt", "rb+")
            self.assertEqual(writer.size(), 10)
            self.assertEqual(b"0123456789", writer.read(12))

            # extend file size
            writer.truncate(12)

            self.assertEqual(writer.size(), 12)
            writer.seek(0, os.SEEK_SET)
            self.assertEqual(b"0123456789\x00\x00", writer.read(12))

            self.assertIsNone(c.manifest_locator())
            self.assertTrue(c.modified())
            c.save_new("test_truncate2")
            self.assertEqual("zzzzz-4zz18-mockcollection0", c.manifest_locator())
            self.assertFalse(c.modified())

    def test_truncate3(self):
        keep = ArvadosFileWriterTestCase.MockKeep({
            "781e5e245d69b566979b86e28d23f2c7+10": b"0123456789",
            "a925576942e94b2ef57a066101b48876+10": b"abcdefghij",
        })
        api = ArvadosFileWriterTestCase.MockApi({
            "name": "test_truncate",
            "manifest_text": ". 781e5e245d69b566979b86e28d23f2c7+10 0:8:count.txt\n",
            "replication_desired": None,
        }, {
            "uuid": "zzzzz-4zz18-mockcollection0",
            "manifest_text": ". 781e5e245d69b566979b86e28d23f2c7+10 0:8:count.txt\n",
            "portable_data_hash": "7fcd0eaac3aad4c31a6a0e756475da92+52",
        })
        with Collection('. 781e5e245d69b566979b86e28d23f2c7+10 a925576942e94b2ef57a066101b48876+10 0:20:count.txt\n',
                        api_client=api, keep_client=keep) as c:
            writer = c.open("count.txt", "rb+")
            self.assertEqual(writer.size(), 20)
            self.assertEqual(b"0123456789ab", writer.read(12))
            self.assertEqual(12, writer.tell())

            writer.truncate(8)

            # Make sure reading off the end doesn't break
            self.assertEqual(12, writer.tell())
            self.assertEqual(b"", writer.read(12))

            self.assertEqual(writer.size(), 8)
            self.assertEqual(2, writer.seek(-10, os.SEEK_CUR))
            self.assertEqual(b"234567", writer.read(12))

            self.assertIsNone(c.manifest_locator())
            self.assertTrue(c.modified())
            c.save_new("test_truncate")
            self.assertEqual("zzzzz-4zz18-mockcollection0", c.manifest_locator())
            self.assertFalse(c.modified())



    def test_write_to_end(self):
        keep = ArvadosFileWriterTestCase.MockKeep({
            "781e5e245d69b566979b86e28d23f2c7+10": b"0123456789",
        })
        api = ArvadosFileWriterTestCase.MockApi({
            "name": "test_append",
            "manifest_text": ". 781e5e245d69b566979b86e28d23f2c7+10 acbd18db4cc2f85cedef654fccc4a4d8+3 0:13:count.txt\n",
            "replication_desired": None,
        }, {
            "uuid": "zzzzz-4zz18-mockcollection0",
            "manifest_text": ". 781e5e245d69b566979b86e28d23f2c7+10 acbd18db4cc2f85cedef654fccc4a4d8+3 0:13:count.txt\n",
            "portable_data_hash": "c5c3af76565c8efb6a806546bcf073f3+88",
        })
        with Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt\n',
                             api_client=api, keep_client=keep) as c:
            writer = c.open("count.txt", "rb+")
            self.assertEqual(writer.size(), 10)

            self.assertEqual(5, writer.seek(5, os.SEEK_SET))
            self.assertEqual(b"56789", writer.read(8))

            writer.seek(10, os.SEEK_SET)
            writer.write("foo")
            self.assertEqual(writer.size(), 13)

            writer.seek(5, os.SEEK_SET)
            self.assertEqual(b"56789foo", writer.read(8))

            self.assertIsNone(c.manifest_locator())
            self.assertTrue(c.modified())
            self.assertIsNone(keep.get("acbd18db4cc2f85cedef654fccc4a4d8+3"))

            c.save_new("test_append")
            self.assertEqual("zzzzz-4zz18-mockcollection0", c.manifest_locator())
            self.assertFalse(c.modified())
            self.assertEqual(b"foo", keep.get("acbd18db4cc2f85cedef654fccc4a4d8+3"))


    def test_append(self):
        keep = ArvadosFileWriterTestCase.MockKeep({
            "781e5e245d69b566979b86e28d23f2c7+10": b"0123456789",
        })
        c = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt\n', keep_client=keep)
        writer = c.open("count.txt", "ab+")
        self.assertEqual(writer.read(20), b"0123456789")

        writer.seek(0, os.SEEK_SET)
        writer.write("hello")
        self.assertEqual(writer.read(), b"")
        writer.seek(-5, os.SEEK_CUR)
        self.assertEqual(writer.read(3), b"hel")
        self.assertEqual(writer.read(), b"lo")
        writer.seek(0, os.SEEK_SET)
        self.assertEqual(writer.read(), b"0123456789hello")

        writer.seek(0)
        writer.write("world")
        self.assertEqual(writer.read(), b"")
        writer.seek(0)
        self.assertEqual(writer.read(), b"0123456789helloworld")

        self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 fc5e038d38a57032085441e7fe7010b0+10 0:20:count.txt\n", c.portable_manifest_text())

    def test_write_at_beginning(self):
        keep = ArvadosFileWriterTestCase.MockKeep({
            "781e5e245d69b566979b86e28d23f2c7+10": b"0123456789",
        })
        with Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt\n',
                             keep_client=keep) as c:
            writer = c.open("count.txt", "rb+")
            self.assertEqual(b"0123456789", writer.readfrom(0, 13))
            writer.seek(0, os.SEEK_SET)
            writer.write("foo")
            self.assertEqual(writer.size(), 10)
            self.assertEqual(b"foo3456789", writer.readfrom(0, 13))
            self.assertEqual(". acbd18db4cc2f85cedef654fccc4a4d8+3 781e5e245d69b566979b86e28d23f2c7+10 0:3:count.txt 6:7:count.txt\n", c.portable_manifest_text())

    def test_write_empty(self):
        keep = ArvadosFileWriterTestCase.MockKeep({})
        with Collection(keep_client=keep) as c:
            writer = c.open("count.txt", "wb")
            self.assertEqual(writer.size(), 0)
            self.assertEqual(". d41d8cd98f00b204e9800998ecf8427e+0 0:0:count.txt\n", c.portable_manifest_text())

    def test_save_manifest_text(self):
        keep = ArvadosFileWriterTestCase.MockKeep({})
        with Collection(keep_client=keep) as c:
            writer = c.open("count.txt", "wb")
            writer.write(b"0123456789")
            self.assertEqual('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt\n', c.portable_manifest_text())
            self.assertNotIn('781e5e245d69b566979b86e28d23f2c7+10', keep.blocks)

            self.assertEqual('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt\n', c.save_new(create_collection_record=False))
            self.assertIn('781e5e245d69b566979b86e28d23f2c7+10', keep.blocks)

    def test_get_manifest_text_commits(self):
         keep = ArvadosFileWriterTestCase.MockKeep({})
         with Collection(keep_client=keep) as c:
             writer = c.open("count.txt", "wb")
             writer.write("0123456789")
             self.assertEqual('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt\n', c.portable_manifest_text())
             self.assertNotIn('781e5e245d69b566979b86e28d23f2c7+10', keep.blocks)
             self.assertEqual('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt\n', c.manifest_text())
             self.assertIn('781e5e245d69b566979b86e28d23f2c7+10', keep.blocks)


    def test_write_in_middle(self):
        keep = ArvadosFileWriterTestCase.MockKeep({"781e5e245d69b566979b86e28d23f2c7+10": b"0123456789"})
        with Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt\n',
                             keep_client=keep) as c:
            writer = c.open("count.txt", "rb+")
            self.assertEqual(b"0123456789", writer.readfrom(0, 13))
            writer.seek(3, os.SEEK_SET)
            writer.write("foo")
            self.assertEqual(writer.size(), 10)
            self.assertEqual(b"012foo6789", writer.readfrom(0, 13))
            self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:count.txt 10:3:count.txt 6:4:count.txt\n", c.portable_manifest_text())

    def test_write_at_end(self):
        keep = ArvadosFileWriterTestCase.MockKeep({"781e5e245d69b566979b86e28d23f2c7+10": b"0123456789"})
        with Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt\n',
                             keep_client=keep) as c:
            writer = c.open("count.txt", "rb+")
            self.assertEqual(b"0123456789", writer.readfrom(0, 13))
            writer.seek(7, os.SEEK_SET)
            writer.write("foo")
            self.assertEqual(writer.size(), 10)
            self.assertEqual(b"0123456foo", writer.readfrom(0, 13))
            self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 acbd18db4cc2f85cedef654fccc4a4d8+3 0:7:count.txt 10:3:count.txt\n", c.portable_manifest_text())

    def test_write_across_segment_boundary(self):
        keep = ArvadosFileWriterTestCase.MockKeep({"781e5e245d69b566979b86e28d23f2c7+10": b"0123456789"})
        with Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt 0:10:count.txt\n',
                             keep_client=keep) as c:
            writer = c.open("count.txt", "rb+")
            self.assertEqual(b"012345678901234", writer.readfrom(0, 15))
            writer.seek(7, os.SEEK_SET)
            writer.write("foobar")
            self.assertEqual(writer.size(), 20)
            self.assertEqual(b"0123456foobar34", writer.readfrom(0, 15))
            self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 3858f62230ac3c915f300c664312c63f+6 0:7:count.txt 10:6:count.txt 3:7:count.txt\n", c.portable_manifest_text())

    def test_write_across_several_segments(self):
        keep = ArvadosFileWriterTestCase.MockKeep({"781e5e245d69b566979b86e28d23f2c7+10": b"0123456789"})
        with Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:4:count.txt 0:4:count.txt 0:4:count.txt',
                             keep_client=keep) as c:
            writer = c.open("count.txt", "rb+")
            self.assertEqual(b"012301230123", writer.readfrom(0, 15))
            writer.seek(2, os.SEEK_SET)
            writer.write("abcdefg")
            self.assertEqual(writer.size(), 12)
            self.assertEqual(b"01abcdefg123", writer.readfrom(0, 15))
            self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 7ac66c0f148de9519b8bd264312c4d64+7 0:2:count.txt 10:7:count.txt 1:3:count.txt\n", c.portable_manifest_text())

    def test_write_large(self):
        keep = ArvadosFileWriterTestCase.MockKeep({})
        api = ArvadosFileWriterTestCase.MockApi({"name":"test_write_large",
                                                 "manifest_text": ". a5de24f4417cfba9d5825eadc2f4ca49+67108000 598cc1a4ccaef8ab6e4724d87e675d78+32892000 0:100000000:count.txt\n",
                                                 "replication_desired":None},
                                                {"uuid":"zzzzz-4zz18-mockcollection0",
                                                 "manifest_text": ". a5de24f4417cfba9d5825eadc2f4ca49+67108000 598cc1a4ccaef8ab6e4724d87e675d78+32892000 0:100000000:count.txt\n",
                                                 "portable_data_hash":"9132ca8e3f671c76103a38f5bc24328c+108"})
        with Collection('. ' + arvados.config.EMPTY_BLOCK_LOCATOR + ' 0:0:count.txt',
                             api_client=api, keep_client=keep) as c:
            writer = c.open("count.txt", "rb+")
            text = "0123456789" * 100
            for b in range(0, 100000):
                writer.write(text)
            self.assertEqual(writer.size(), 100000000)

            self.assertIsNone(c.manifest_locator())
            self.assertTrue(c.modified())
            c.save_new("test_write_large")
            self.assertEqual("zzzzz-4zz18-mockcollection0", c.manifest_locator())
            self.assertFalse(c.modified())


    def test_large_write(self):
        keep = ArvadosFileWriterTestCase.MockKeep({})
        api = ArvadosFileWriterTestCase.MockApi({}, {})
        with Collection('. ' + arvados.config.EMPTY_BLOCK_LOCATOR + ' 0:0:count.txt',
                             api_client=api, keep_client=keep) as c:
            writer = c.open("count.txt", "rb+")
            self.assertEqual(writer.size(), 0)

            text = "0123456789"
            writer.write(text)
            text = "0123456789" * 9999999
            writer.write(text)
            self.assertEqual(writer.size(), 100000000)

            self.assertEqual(c.manifest_text(), ". 781e5e245d69b566979b86e28d23f2c7+10 48dd23ea1645fd47d789804d71b5bb8e+67108864 77c57dc6ac5a10bb2205caaa73187994+32891126 0:100000000:count.txt\n")

    def test_sparse_write(self):
        keep = ArvadosFileWriterTestCase.MockKeep({})
        api = ArvadosFileWriterTestCase.MockApi({}, {})
        with Collection('. ' + arvados.config.EMPTY_BLOCK_LOCATOR + ' 0:0:count.txt',
                             api_client=api, keep_client=keep) as c:
            writer = c.open("count.txt", "rb+")
            self.assertEqual(writer.size(), 0)

            text = b"0123456789"
            writer.seek(2)
            writer.write(text)
            self.assertEqual(writer.size(), 12)
            writer.seek(0, os.SEEK_SET)
            self.assertEqual(writer.read(), b"\x00\x00"+text)

            self.assertEqual(c.manifest_text(), ". 7f614da9329cd3aebf59b91aadc30bf0+67108864 781e5e245d69b566979b86e28d23f2c7+10 0:2:count.txt 67108864:10:count.txt\n")


    def test_sparse_write2(self):
        keep = ArvadosFileWriterTestCase.MockKeep({})
        api = ArvadosFileWriterTestCase.MockApi({}, {})
        with Collection('. ' + arvados.config.EMPTY_BLOCK_LOCATOR + ' 0:0:count.txt',
                             api_client=api, keep_client=keep) as c:
            writer = c.open("count.txt", "rb+")
            self.assertEqual(writer.size(), 0)

            text = "0123456789"
            writer.seek((arvados.config.KEEP_BLOCK_SIZE*2) + 2)
            writer.write(text)
            self.assertEqual(writer.size(), (arvados.config.KEEP_BLOCK_SIZE*2) + 12)
            writer.seek(0, os.SEEK_SET)

            self.assertEqual(c.manifest_text(), ". 7f614da9329cd3aebf59b91aadc30bf0+67108864 781e5e245d69b566979b86e28d23f2c7+10 0:67108864:count.txt 0:67108864:count.txt 0:2:count.txt 67108864:10:count.txt\n")


    def test_sparse_write3(self):
        keep = ArvadosFileWriterTestCase.MockKeep({})
        api = ArvadosFileWriterTestCase.MockApi({}, {})
        for r in [[0, 1, 2, 3, 4], [4, 3, 2, 1, 0], [3, 2, 0, 4, 1]]:
            with Collection() as c:
                writer = c.open("count.txt", "rb+")
                self.assertEqual(writer.size(), 0)

                for i in r:
                    w = ("%s" % i) * 10
                    writer.seek(i*10)
                    writer.write(w.encode())
                writer.seek(0)
                self.assertEqual(writer.read(), b"00000000001111111111222222222233333333334444444444")

    def test_sparse_write4(self):
        keep = ArvadosFileWriterTestCase.MockKeep({})
        api = ArvadosFileWriterTestCase.MockApi({}, {})
        for r in [[0, 1, 2, 4], [4, 2, 1, 0], [2, 0, 4, 1]]:
            with Collection() as c:
                writer = c.open("count.txt", "rb+")
                self.assertEqual(writer.size(), 0)

                for i in r:
                    w = ("%s" % i) * 10
                    writer.seek(i*10)
                    writer.write(w.encode())
                writer.seek(0)
                self.assertEqual(writer.read(), b"000000000011111111112222222222\x00\x00\x00\x00\x00\x00\x00\x00\x00\x004444444444")


    def test_rewrite_on_empty_file(self):
        keep = ArvadosFileWriterTestCase.MockKeep({})
        with Collection('. ' + arvados.config.EMPTY_BLOCK_LOCATOR + ' 0:0:count.txt',
                             keep_client=keep) as c:
            writer = c.open("count.txt", "rb+")
            for b in range(0, 10):
                writer.seek(0, os.SEEK_SET)
                writer.write("0123456789")

            self.assertEqual(writer.size(), 10)
            self.assertEqual(b"0123456789", writer.readfrom(0, 20))
            self.assertEqual(". 7a08b07e84641703e5f2c836aa59a170+100 90:10:count.txt\n", c.portable_manifest_text())
            writer.flush()
            self.assertEqual(writer.size(), 10)
            self.assertEqual(b"0123456789", writer.readfrom(0, 20))
            self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt\n", c.portable_manifest_text())

    def test_rewrite_append_existing_file(self):
        keep = ArvadosFileWriterTestCase.MockKeep({
            "781e5e245d69b566979b86e28d23f2c7+10": b"0123456789",
        })
        with Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt',
                             keep_client=keep) as c:
            writer = c.open("count.txt", "rb+")
            for b in range(0, 10):
                writer.seek(10, os.SEEK_SET)
                writer.write("abcdefghij")

            self.assertEqual(writer.size(), 20)
            self.assertEqual(b"0123456789abcdefghij", writer.readfrom(0, 20))
            self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 ae5f43bab79cf0be33f025fa97ae7398+100 0:10:count.txt 100:10:count.txt\n", c.portable_manifest_text())

            writer.arvadosfile.flush()
            self.assertEqual(writer.size(), 20)
            self.assertEqual(b"0123456789abcdefghij", writer.readfrom(0, 20))
            self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 a925576942e94b2ef57a066101b48876+10 0:20:count.txt\n", c.portable_manifest_text())

    def test_rewrite_over_existing_file(self):
        keep = ArvadosFileWriterTestCase.MockKeep({
            "781e5e245d69b566979b86e28d23f2c7+10": b"0123456789",
        })
        with Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt',
                             keep_client=keep) as c:
            writer = c.open("count.txt", "rb+")
            for b in range(0, 10):
                writer.seek(5, os.SEEK_SET)
                writer.write("abcdefghij")

            self.assertEqual(writer.size(), 15)
            self.assertEqual(b"01234abcdefghij", writer.readfrom(0, 20))
            self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 ae5f43bab79cf0be33f025fa97ae7398+100 0:5:count.txt 100:10:count.txt\n", c.portable_manifest_text())

            writer.arvadosfile.flush()

            self.assertEqual(writer.size(), 15)
            self.assertEqual(b"01234abcdefghij", writer.readfrom(0, 20))
            self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 a925576942e94b2ef57a066101b48876+10 0:5:count.txt 10:10:count.txt\n", c.portable_manifest_text())

    def test_write_large_rewrite(self):
        keep = ArvadosFileWriterTestCase.MockKeep({})
        api = ArvadosFileWriterTestCase.MockApi({"name":"test_write_large",
                                                 "manifest_text": ". 3dc0d4bc21f48060bedcb2c91af4f906+32892003 a5de24f4417cfba9d5825eadc2f4ca49+67108000 0:3:count.txt 32892006:67107997:count.txt 3:32892000:count.txt\n",
                                                 "replication_desired":None},
                                                {"uuid":"zzzzz-4zz18-mockcollection0",
                                                 "manifest_text": ". 3dc0d4bc21f48060bedcb2c91af4f906+32892003 a5de24f4417cfba9d5825eadc2f4ca49+67108000 0:3:count.txt 32892006:67107997:count.txt 3:32892000:count.txt\n",
                                                 "portable_data_hash":"217665c6b713e1b78dfba7ebd42344db+156"})
        with Collection('. ' + arvados.config.EMPTY_BLOCK_LOCATOR + ' 0:0:count.txt',
                             api_client=api, keep_client=keep) as c:
            writer = c.open("count.txt", "rb+")
            text = b''.join([b"0123456789" for a in range(0, 100)])
            for b in range(0, 100000):
                writer.write(text)
            writer.seek(0, os.SEEK_SET)
            writer.write("foo")
            self.assertEqual(writer.size(), 100000000)

            self.assertIsNone(c.manifest_locator())
            self.assertTrue(c.modified())
            c.save_new("test_write_large")
            self.assertEqual("zzzzz-4zz18-mockcollection0", c.manifest_locator())
            self.assertFalse(c.modified())

    def test_create(self):
        keep = ArvadosFileWriterTestCase.MockKeep({})
        api = ArvadosFileWriterTestCase.MockApi({
            "name":"test_create",
            "manifest_text":". 2e9ec317e197819358fbc43afca7d837+8 0:8:count.txt\n",
            "replication_desired":None,
        }, {
            "uuid":"zzzzz-4zz18-mockcollection0",
            "manifest_text":". 2e9ec317e197819358fbc43afca7d837+8 0:8:count.txt\n",
            "portable_data_hash":"7a461a8c58601798f690f8b368ac4423+51",
        })
        with Collection(api_client=api, keep_client=keep) as c:
            writer = c.open("count.txt", "wb+")
            self.assertEqual(writer.size(), 0)
            writer.write("01234567")
            self.assertEqual(writer.size(), 8)

            self.assertIsNone(c.manifest_locator())
            self.assertTrue(c.modified())
            self.assertIsNone(keep.get("2e9ec317e197819358fbc43afca7d837+8"))
            c.save_new("test_create")
            self.assertEqual("zzzzz-4zz18-mockcollection0", c.manifest_locator())
            self.assertFalse(c.modified())
            self.assertEqual(b"01234567", keep.get("2e9ec317e197819358fbc43afca7d837+8"))


    def test_create_subdir(self):
        keep = ArvadosFileWriterTestCase.MockKeep({})
        api = ArvadosFileWriterTestCase.MockApi({"name":"test_create",
                                                 "manifest_text":"./foo/bar 2e9ec317e197819358fbc43afca7d837+8 0:8:count.txt\n",
                                                 "replication_desired":None},
                                                {"uuid":"zzzzz-4zz18-mockcollection0",
                                                 "manifest_text":"./foo/bar 2e9ec317e197819358fbc43afca7d837+8 0:8:count.txt\n",
                                                 "portable_data_hash":"1b02aaa62528d28a5be41651cbb9d7c7+59"})
        with Collection(api_client=api, keep_client=keep) as c:
            self.assertIsNone(c.api_response())
            writer = c.open("foo/bar/count.txt", "wb+")
            writer.write("01234567")
            self.assertFalse(c.committed())
            c.save_new("test_create")
            self.assertTrue(c.committed())
            self.assertEqual(c.api_response(), api.response)

    def test_overwrite(self):
        keep = ArvadosFileWriterTestCase.MockKeep({"781e5e245d69b566979b86e28d23f2c7+10": "0123456789"})
        api = ArvadosFileWriterTestCase.MockApi({"name":"test_overwrite",
                                                 "manifest_text":". 2e9ec317e197819358fbc43afca7d837+8 0:8:count.txt\n",
                                                 "replication_desired":None},
                                                {"uuid":"zzzzz-4zz18-mockcollection0",
                                                 "manifest_text":". 2e9ec317e197819358fbc43afca7d837+8 0:8:count.txt\n",
                                                 "portable_data_hash":"7a461a8c58601798f690f8b368ac4423+51"})
        with Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt\n',
                             api_client=api, keep_client=keep) as c:
            writer = c.open("count.txt", "wb+")
            self.assertEqual(writer.size(), 0)
            writer.write("01234567")
            self.assertEqual(writer.size(), 8)

            self.assertIsNone(c.manifest_locator())
            self.assertTrue(c.modified())
            c.save_new("test_overwrite")
            self.assertEqual("zzzzz-4zz18-mockcollection0", c.manifest_locator())
            self.assertFalse(c.modified())

    def test_file_not_found(self):
        with Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt\n') as c:
            with self.assertRaises(IOError):
                writer = c.open("nocount.txt", "rb")

    def test_cannot_open_directory(self):
        with Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt\n') as c:
            with self.assertRaises(IOError):
                writer = c.open(".", "rb")

    def test_create_multiple(self):
        keep = ArvadosFileWriterTestCase.MockKeep({})
        api = ArvadosFileWriterTestCase.MockApi({"name":"test_create_multiple",
                                                 "manifest_text":". 2e9ec317e197819358fbc43afca7d837+8 e8dc4081b13434b45189a720b77b6818+8 0:8:count1.txt 8:8:count2.txt\n",
                                                 "replication_desired":None},
                                                {"uuid":"zzzzz-4zz18-mockcollection0",
                                                 "manifest_text":". 2e9ec317e197819358fbc43afca7d837+8 e8dc4081b13434b45189a720b77b6818+8 0:8:count1.txt 8:8:count2.txt\n",
                                                 "portable_data_hash":"71e7bb6c00d31fc2b4364199fd97be08+102"})
        with Collection(api_client=api, keep_client=keep) as c:
            w1 = c.open("count1.txt", "wb")
            w2 = c.open("count2.txt", "wb")
            w1.write("01234567")
            w2.write("abcdefgh")
            self.assertEqual(w1.size(), 8)
            self.assertEqual(w2.size(), 8)

            self.assertIsNone(c.manifest_locator())
            self.assertTrue(c.modified())
            self.assertIsNone(keep.get("2e9ec317e197819358fbc43afca7d837+8"))
            c.save_new("test_create_multiple")
            self.assertEqual("zzzzz-4zz18-mockcollection0", c.manifest_locator())
            self.assertFalse(c.modified())
            self.assertEqual(b"01234567", keep.get("2e9ec317e197819358fbc43afca7d837+8"))


class ArvadosFileReaderTestCase(StreamFileReaderTestCase):
    class MockParent(object):
        class MockBlockMgr(object):
            def __init__(self, blocks, nocache):
                self.blocks = blocks
                self.nocache = nocache

            def block_prefetch(self, loc):
                pass

            def get_block_contents(self, loc, num_retries=0, cache_only=False):
                if self.nocache and cache_only:
                    return None
                return self.blocks[loc]

        def __init__(self, blocks, nocache):
            self.blocks = blocks
            self.nocache = nocache
            self.lock = arvados.arvfile.NoopLock()

        def root_collection(self):
            return self

        def _my_block_manager(self):
            return ArvadosFileReaderTestCase.MockParent.MockBlockMgr(self.blocks, self.nocache)


    def make_count_reader(self, nocache=False):
        stream = []
        n = 0
        blocks = {}
        for d in [b'01234', b'34567', b'67890']:
            loc = tutil.str_keep_locator(d)
            blocks[loc] = d
            stream.append(Range(loc, n, len(d)))
            n += len(d)
        af = ArvadosFile(ArvadosFileReaderTestCase.MockParent(blocks, nocache), "count.txt", stream=stream, segments=[Range(1, 0, 3), Range(6, 3, 3), Range(11, 6, 3)])
        return ArvadosFileReader(af, mode="rb")

    def test_read_block_crossing_behavior(self):
        # read() needs to return all the data requested if possible, even if it
        # crosses uncached blocks: https://arvados.org/issues/5856
        sfile = self.make_count_reader(nocache=True)
        self.assertEqual(b'12345678', sfile.read(8))

    def test_successive_reads(self):
        # Override StreamFileReaderTestCase.test_successive_reads
        sfile = self.make_count_reader(nocache=True)
        self.assertEqual(b'1234', sfile.read(4))
        self.assertEqual(b'5678', sfile.read(4))
        self.assertEqual(b'9', sfile.read(4))
        self.assertEqual(b'', sfile.read(4))

    def test_tell_after_block_read(self):
        # Override StreamFileReaderTestCase.test_tell_after_block_read
        sfile = self.make_count_reader(nocache=True)
        self.assertEqual(b'12345678', sfile.read(8))
        self.assertEqual(8, sfile.tell())

    def test_prefetch(self):
        keep = ArvadosFileWriterTestCase.MockKeep({
            "2e9ec317e197819358fbc43afca7d837+8": b"01234567",
            "e8dc4081b13434b45189a720b77b6818+8": b"abcdefgh",
        })
        with Collection(". 2e9ec317e197819358fbc43afca7d837+8 e8dc4081b13434b45189a720b77b6818+8 0:16:count.txt\n", keep_client=keep) as c:
            r = c.open("count.txt", "rb")
            self.assertEqual(b"0123", r.read(4))
        self.assertIn("2e9ec317e197819358fbc43afca7d837+8", keep.requests)
        self.assertIn("e8dc4081b13434b45189a720b77b6818+8", keep.requests)

    def test__eq__from_manifest(self):
        with Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt') as c1:
            with Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt') as c2:
                self.assertTrue(c1["count1.txt"] == c2["count1.txt"])
                self.assertFalse(c1["count1.txt"] != c2["count1.txt"])

    def test__eq__from_writes(self):
        with Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt') as c1:
            with Collection() as c2:
                f = c2.open("count1.txt", "wb")
                f.write("0123456789")

                self.assertTrue(c1["count1.txt"] == c2["count1.txt"])
                self.assertFalse(c1["count1.txt"] != c2["count1.txt"])

    def test__ne__(self):
        with Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt') as c1:
            with Collection() as c2:
                f = c2.open("count1.txt", "wb")
                f.write("1234567890")

                self.assertTrue(c1["count1.txt"] != c2["count1.txt"])
                self.assertFalse(c1["count1.txt"] == c2["count1.txt"])


class ArvadosFileReadTestCase(unittest.TestCase, StreamRetryTestMixin):
    def reader_for(self, coll_name, **kwargs):
        stream = []
        segments = []
        n = 0
        for d in self.manifest_for(coll_name).split():
            try:
                k = KeepLocator(d)
                segments.append(Range(n, n, k.size))
                stream.append(Range(d, n, k.size))
                n += k.size
            except ValueError:
                pass

        blockmanager = arvados.arvfile._BlockManager(self.keep_client())
        blockmanager.prefetch_enabled = False
        col = Collection(keep_client=self.keep_client(), block_manager=blockmanager)
        af = ArvadosFile(col, "test",
                         stream=stream,
                         segments=segments)
        kwargs.setdefault('mode', 'rb')
        return ArvadosFileReader(af, **kwargs)

    def read_for_test(self, reader, byte_count, **kwargs):
        return reader.read(byte_count, **kwargs)


class ArvadosFileReadFromTestCase(ArvadosFileReadTestCase):
    def read_for_test(self, reader, byte_count, **kwargs):
        return reader.readfrom(0, byte_count, **kwargs)


class ArvadosFileReadAllTestCase(ArvadosFileReadTestCase):
    def read_for_test(self, reader, byte_count, **kwargs):
        return b''.join(reader.readall(**kwargs))


class ArvadosFileReadAllDecompressedTestCase(ArvadosFileReadTestCase):
    def read_for_test(self, reader, byte_count, **kwargs):
        return b''.join(reader.readall_decompressed(**kwargs))


class ArvadosFileReadlinesTestCase(ArvadosFileReadTestCase):
    def read_for_test(self, reader, byte_count, **kwargs):
        return ''.join(reader.readlines(**kwargs)).encode()


class ArvadosFileTestCase(unittest.TestCase):
    def datetime_to_hex(self, dt):
        return hex(int(time.mktime(dt.timetuple())))[2:]

    def test_permission_expired(self):
        base_manifest = ". 781e5e245d69b566979b86e28d23f2c7+10+A715fd31f8111894f717eb1003c1b0216799dd9ec@{} 0:10:count.txt\n"
        now = datetime.datetime.now()
        a_week_ago = now - datetime.timedelta(days=7)
        a_month_ago = now - datetime.timedelta(days=30)
        a_week_from_now = now + datetime.timedelta(days=7)
        with Collection(base_manifest.format(self.datetime_to_hex(a_week_from_now))) as c:
            self.assertFalse(c.find('count.txt').permission_expired())
        with Collection(base_manifest.format(self.datetime_to_hex(a_week_ago))) as c:
            f = c.find('count.txt')
            self.assertTrue(f.permission_expired())
            self.assertTrue(f.permission_expired(a_week_from_now))
            self.assertFalse(f.permission_expired(a_month_ago))


class BlockManagerTest(unittest.TestCase):
    def test_bufferblock_append(self):
        keep = ArvadosFileWriterTestCase.MockKeep({})
        with arvados.arvfile._BlockManager(keep) as blockmanager:
            bufferblock = blockmanager.alloc_bufferblock()
            bufferblock.append("foo")

            self.assertEqual(bufferblock.size(), 3)
            self.assertEqual(bufferblock.buffer_view[0:3], b"foo")
            self.assertEqual(bufferblock.locator(), "acbd18db4cc2f85cedef654fccc4a4d8+3")

            bufferblock.append("bar")

            self.assertEqual(bufferblock.size(), 6)
            self.assertEqual(bufferblock.buffer_view[0:6], b"foobar")
            self.assertEqual(bufferblock.locator(), "3858f62230ac3c915f300c664312c63f+6")

            bufferblock.set_state(arvados.arvfile._BufferBlock.PENDING)
            with self.assertRaises(arvados.errors.AssertionError):
                bufferblock.append("bar")

    def test_bufferblock_dup(self):
        keep = ArvadosFileWriterTestCase.MockKeep({})
        with arvados.arvfile._BlockManager(keep) as blockmanager:
            bufferblock = blockmanager.alloc_bufferblock()
            bufferblock.append("foo")

            self.assertEqual(bufferblock.size(), 3)
            self.assertEqual(bufferblock.buffer_view[0:3], b"foo")
            self.assertEqual(bufferblock.locator(), "acbd18db4cc2f85cedef654fccc4a4d8+3")
            bufferblock.set_state(arvados.arvfile._BufferBlock.PENDING)

            bufferblock2 = blockmanager.dup_block(bufferblock, None)
            self.assertNotEqual(bufferblock.blockid, bufferblock2.blockid)

            bufferblock2.append("bar")

            self.assertEqual(bufferblock2.size(), 6)
            self.assertEqual(bufferblock2.buffer_view[0:6], b"foobar")
            self.assertEqual(bufferblock2.locator(), "3858f62230ac3c915f300c664312c63f+6")

            self.assertEqual(bufferblock.size(), 3)
            self.assertEqual(bufferblock.buffer_view[0:3], b"foo")
            self.assertEqual(bufferblock.locator(), "acbd18db4cc2f85cedef654fccc4a4d8+3")

    def test_bufferblock_get(self):
        keep = ArvadosFileWriterTestCase.MockKeep({
            "781e5e245d69b566979b86e28d23f2c7+10": b"0123456789",
        })
        with arvados.arvfile._BlockManager(keep) as blockmanager:
            bufferblock = blockmanager.alloc_bufferblock()
            bufferblock.append("foo")

            self.assertEqual(blockmanager.get_block_contents("781e5e245d69b566979b86e28d23f2c7+10", 1), b"0123456789")
            self.assertEqual(blockmanager.get_block_contents(bufferblock.blockid, 1), b"foo")

    def test_bufferblock_commit(self):
        mockkeep = mock.MagicMock()
        with arvados.arvfile._BlockManager(mockkeep) as blockmanager:
            bufferblock = blockmanager.alloc_bufferblock()
            bufferblock.owner = mock.MagicMock(spec=arvados.arvfile.ArvadosFile)
            def flush(sync=None):
                blockmanager.commit_bufferblock(bufferblock, sync)
            bufferblock.owner.flush.side_effect = flush
            bufferblock.append("foo")
            blockmanager.commit_all()
            self.assertTrue(bufferblock.owner.flush.called)
            self.assertTrue(mockkeep.put.called)
            self.assertEqual(bufferblock.state(), arvados.arvfile._BufferBlock.COMMITTED)
            self.assertIsNone(bufferblock.buffer_view)

    def test_bufferblock_commit_pending(self):
        # Test for bug #7225
        mockkeep = mock.MagicMock()
        mockkeep.put.side_effect = lambda x: time.sleep(1)
        with arvados.arvfile._BlockManager(mockkeep) as blockmanager:
            bufferblock = blockmanager.alloc_bufferblock()
            bufferblock.append("foo")

            blockmanager.commit_bufferblock(bufferblock, False)
            self.assertEqual(bufferblock.state(), arvados.arvfile._BufferBlock.PENDING)

            blockmanager.commit_bufferblock(bufferblock, True)
            self.assertEqual(bufferblock.state(), arvados.arvfile._BufferBlock.COMMITTED)


    def test_bufferblock_commit_with_error(self):
        mockkeep = mock.MagicMock()
        mockkeep.put.side_effect = arvados.errors.KeepWriteError("fail")
        with arvados.arvfile._BlockManager(mockkeep) as blockmanager:
            bufferblock = blockmanager.alloc_bufferblock()
            bufferblock.owner = mock.MagicMock(spec=arvados.arvfile.ArvadosFile)
            def flush(sync=None):
                blockmanager.commit_bufferblock(bufferblock, sync)
            bufferblock.owner.flush.side_effect = flush
            bufferblock.append("foo")
            with self.assertRaises(arvados.errors.KeepWriteError) as err:
                blockmanager.commit_all()
            self.assertTrue(bufferblock.owner.flush.called)
            self.assertEqual(str(err.exception), "Error writing some blocks: block acbd18db4cc2f85cedef654fccc4a4d8+3 raised KeepWriteError (fail)")
            self.assertEqual(bufferblock.state(), arvados.arvfile._BufferBlock.ERROR)
