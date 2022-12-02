# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import absolute_import

from builtins import object
import arvados
import copy
import mock
import os
import random
import re
import sys
import datetime
import ciso8601
import time
import unittest
import parameterized

from . import run_test_server
from arvados._ranges import Range, LocatorAndRange
from arvados.collection import Collection, CollectionReader
from . import arvados_testutil as tutil
from .arvados_testutil import make_block_cache

class TestResumableWriter(arvados.ResumableCollectionWriter):
    KEEP_BLOCK_SIZE = 1024  # PUT to Keep every 1K.

    def current_state(self):
        return self.dump_state(copy.deepcopy)

@parameterized.parameterized_class([{"disk_cache": True}, {"disk_cache": False}])
class ArvadosCollectionsTest(run_test_server.TestCaseWithServers,
                             tutil.ArvadosBaseTestCase):
    disk_cache = False
    MAIN_SERVER = {}

    @classmethod
    def setUpClass(cls):
        super(ArvadosCollectionsTest, cls).setUpClass()
        # need admin privileges to make collections with unsigned blocks
        run_test_server.authorize_with('admin')
        cls.api_client = arvados.api('v1')
        cls.keep_client = arvados.KeepClient(api_client=cls.api_client,
                                             local_store=cls.local_store,
                                             block_cache=make_block_cache(cls.disk_cache))

    def write_foo_bar_baz(self):
        cw = arvados.CollectionWriter(self.api_client)
        self.assertEqual(cw.current_stream_name(), '.',
                         'current_stream_name() should be "." now')
        cw.set_current_file_name('foo.txt')
        cw.write(b'foo')
        self.assertEqual(cw.current_file_name(), 'foo.txt',
                         'current_file_name() should be foo.txt now')
        cw.start_new_file('bar.txt')
        cw.write(b'bar')
        cw.start_new_stream('baz')
        cw.write(b'baz')
        cw.set_current_file_name('baz.txt')
        self.assertEqual(cw.manifest_text(),
                         ". 3858f62230ac3c915f300c664312c63f+6 0:3:foo.txt 3:3:bar.txt\n" +
                         "./baz 73feffa4b7f6bb68e44cf984c85f6e88+3 0:3:baz.txt\n",
                         "wrong manifest: got {}".format(cw.manifest_text()))
        cw.save_new()
        return cw.portable_data_hash()

    def test_pdh_is_native_str(self):
        pdh = self.write_foo_bar_baz()
        self.assertEqual(type(''), type(pdh))

    def test_keep_local_store(self):
        self.assertEqual(self.keep_client.put(b'foo'), 'acbd18db4cc2f85cedef654fccc4a4d8+3', 'wrong md5 hash from Keep.put')
        self.assertEqual(self.keep_client.get('acbd18db4cc2f85cedef654fccc4a4d8+3'), b'foo', 'wrong data from Keep.get')

    def test_local_collection_writer(self):
        self.assertEqual(self.write_foo_bar_baz(),
                         '23ca013983d6239e98931cc779e68426+114',
                         'wrong locator hash: ' + self.write_foo_bar_baz())

    def test_local_collection_reader(self):
        foobarbaz = self.write_foo_bar_baz()
        cr = arvados.CollectionReader(
            foobarbaz + '+Xzizzle', self.api_client)
        got = []
        for s in cr.all_streams():
            for f in s.all_files():
                got += [[f.size(), f.stream_name(), f.name(), f.read(2**26)]]
        expected = [[3, '.', 'foo.txt', b'foo'],
                    [3, '.', 'bar.txt', b'bar'],
                    [3, './baz', 'baz.txt', b'baz']]
        self.assertEqual(got,
                         expected)
        stream0 = cr.all_streams()[0]
        self.assertEqual(stream0.readfrom(0, 0),
                         b'',
                         'reading zero bytes should have returned empty string')
        self.assertEqual(stream0.readfrom(0, 2**26),
                         b'foobar',
                         'reading entire stream failed')
        self.assertEqual(stream0.readfrom(2**26, 0),
                         b'',
                         'reading zero bytes should have returned empty string')
        self.assertEqual(3, len(cr))
        self.assertTrue(cr)

    def _test_subset(self, collection, expected):
        cr = arvados.CollectionReader(collection, self.api_client)
        for s in cr.all_streams():
            for ex in expected:
                if ex[0] == s:
                    f = s.files()[ex[2]]
                    got = [f.size(), f.stream_name(), f.name(), "".join(f.readall(2**26))]
                    self.assertEqual(got,
                                     ex,
                                     'all_files|as_manifest did not preserve manifest contents: got %s expected %s' % (got, ex))

    def test_collection_manifest_subset(self):
        foobarbaz = self.write_foo_bar_baz()
        self._test_subset(foobarbaz,
                          [[3, '.',     'bar.txt', b'bar'],
                           [3, '.',     'foo.txt', b'foo'],
                           [3, './baz', 'baz.txt', b'baz']])
        self._test_subset((". %s %s 0:3:foo.txt 3:3:bar.txt\n" %
                           (self.keep_client.put(b"foo"),
                            self.keep_client.put(b"bar"))),
                          [[3, '.', 'bar.txt', b'bar'],
                           [3, '.', 'foo.txt', b'foo']])
        self._test_subset((". %s %s 0:2:fo.txt 2:4:obar.txt\n" %
                           (self.keep_client.put(b"foo"),
                            self.keep_client.put(b"bar"))),
                          [[2, '.', 'fo.txt', b'fo'],
                           [4, '.', 'obar.txt', b'obar']])
        self._test_subset((". %s %s 0:2:fo.txt 2:0:zero.txt 2:2:ob.txt 4:2:ar.txt\n" %
                           (self.keep_client.put(b"foo"),
                            self.keep_client.put(b"bar"))),
                          [[2, '.', 'ar.txt', b'ar'],
                           [2, '.', 'fo.txt', b'fo'],
                           [2, '.', 'ob.txt', b'ob'],
                           [0, '.', 'zero.txt', b'']])

    def test_collection_empty_file(self):
        cw = arvados.CollectionWriter(self.api_client)
        cw.start_new_file('zero.txt')
        cw.write(b'')

        self.assertEqual(cw.manifest_text(), ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:zero.txt\n")
        self.check_manifest_file_sizes(cw.manifest_text(), [0])
        cw = arvados.CollectionWriter(self.api_client)
        cw.start_new_file('zero.txt')
        cw.write(b'')
        cw.start_new_file('one.txt')
        cw.write(b'1')
        cw.start_new_stream('foo')
        cw.start_new_file('zero.txt')
        cw.write(b'')
        self.check_manifest_file_sizes(cw.manifest_text(), [0,1,0])

    def test_no_implicit_normalize(self):
        cw = arvados.CollectionWriter(self.api_client)
        cw.start_new_file('b')
        cw.write(b'b')
        cw.start_new_file('a')
        cw.write(b'')
        self.check_manifest_file_sizes(cw.manifest_text(), [1,0])
        self.check_manifest_file_sizes(
            arvados.CollectionReader(
                cw.manifest_text()).manifest_text(normalize=True),
            [0,1])

    def check_manifest_file_sizes(self, manifest_text, expect_sizes):
        cr = arvados.CollectionReader(manifest_text, self.api_client)
        got_sizes = []
        for f in cr.all_files():
            got_sizes += [f.size()]
        self.assertEqual(got_sizes, expect_sizes, "got wrong file sizes %s, expected %s" % (got_sizes, expect_sizes))

    def test_normalized_collection(self):
        m1 = """. 5348b82a029fd9e971a811ce1f71360b+43 0:43:md5sum.txt
. 085c37f02916da1cad16f93c54d899b7+41 0:41:md5sum.txt
. 8b22da26f9f433dea0a10e5ec66d73ba+43 0:43:md5sum.txt
"""
        self.assertEqual(arvados.CollectionReader(m1, self.api_client).manifest_text(normalize=True),
                         """. 5348b82a029fd9e971a811ce1f71360b+43 085c37f02916da1cad16f93c54d899b7+41 8b22da26f9f433dea0a10e5ec66d73ba+43 0:127:md5sum.txt
""")

        m2 = """. 204e43b8a1185621ca55a94839582e6f+67108864 b9677abbac956bd3e86b1deb28dfac03+67108864 fc15aff2a762b13f521baf042140acec+67108864 323d2a3ce20370c4ca1d3462a344f8fd+25885655 0:227212247:var-GS000016015-ASM.tsv.bz2
"""
        self.assertEqual(arvados.CollectionReader(m2, self.api_client).manifest_text(normalize=True), m2)

        m3 = """. 5348b82a029fd9e971a811ce1f71360b+43 3:40:md5sum.txt
. 085c37f02916da1cad16f93c54d899b7+41 0:41:md5sum.txt
. 8b22da26f9f433dea0a10e5ec66d73ba+43 0:43:md5sum.txt
"""
        self.assertEqual(arvados.CollectionReader(m3, self.api_client).manifest_text(normalize=True),
                         """. 5348b82a029fd9e971a811ce1f71360b+43 085c37f02916da1cad16f93c54d899b7+41 8b22da26f9f433dea0a10e5ec66d73ba+43 3:124:md5sum.txt
""")

        m4 = """. 204e43b8a1185621ca55a94839582e6f+67108864 0:3:foo/bar
./zzz 204e43b8a1185621ca55a94839582e6f+67108864 0:999:zzz
./foo 323d2a3ce20370c4ca1d3462a344f8fd+25885655 0:3:bar
"""
        self.assertEqual(arvados.CollectionReader(m4, self.api_client).manifest_text(normalize=True),
                         """./foo 204e43b8a1185621ca55a94839582e6f+67108864 323d2a3ce20370c4ca1d3462a344f8fd+25885655 0:3:bar 67108864:3:bar
./zzz 204e43b8a1185621ca55a94839582e6f+67108864 0:999:zzz
""")

        m5 = """. 204e43b8a1185621ca55a94839582e6f+67108864 0:3:foo/bar
./zzz 204e43b8a1185621ca55a94839582e6f+67108864 0:999:zzz
./foo 204e43b8a1185621ca55a94839582e6f+67108864 3:3:bar
"""
        self.assertEqual(arvados.CollectionReader(m5, self.api_client).manifest_text(normalize=True),
                         """./foo 204e43b8a1185621ca55a94839582e6f+67108864 0:6:bar
./zzz 204e43b8a1185621ca55a94839582e6f+67108864 0:999:zzz
""")

        with self.data_file('1000G_ref_manifest') as f6:
            m6 = f6.read()
            self.assertEqual(arvados.CollectionReader(m6, self.api_client).manifest_text(normalize=True), m6)

        with self.data_file('jlake_manifest') as f7:
            m7 = f7.read()
            self.assertEqual(arvados.CollectionReader(m7, self.api_client).manifest_text(normalize=True), m7)

        m8 = """./a\\040b\\040c 59ca0efa9f5633cb0371bbc0355478d8+13 0:13:hello\\040world.txt
"""
        self.assertEqual(arvados.CollectionReader(m8, self.api_client).manifest_text(normalize=True), m8)

    def test_locators_and_ranges(self):
        blocks2 = [Range('a', 0, 10),
                   Range('b', 10, 10),
                   Range('c', 20, 10),
                   Range('d', 30, 10),
                   Range('e', 40, 10),
                   Range('f', 50, 10)]

        self.assertEqual(arvados.locators_and_ranges(blocks2,  2,  2), [LocatorAndRange('a', 10, 2, 2)])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 12, 2), [LocatorAndRange('b', 10, 2, 2)])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 22, 2), [LocatorAndRange('c', 10, 2, 2)])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 32, 2), [LocatorAndRange('d', 10, 2, 2)])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 42, 2), [LocatorAndRange('e', 10, 2, 2)])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 52, 2), [LocatorAndRange('f', 10, 2, 2)])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 62, 2), [])
        self.assertEqual(arvados.locators_and_ranges(blocks2, -2, 2), [])

        self.assertEqual(arvados.locators_and_ranges(blocks2,  0,  2), [LocatorAndRange('a', 10, 0, 2)])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 10, 2), [LocatorAndRange('b', 10, 0, 2)])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 20, 2), [LocatorAndRange('c', 10, 0, 2)])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 30, 2), [LocatorAndRange('d', 10, 0, 2)])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 40, 2), [LocatorAndRange('e', 10, 0, 2)])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 50, 2), [LocatorAndRange('f', 10, 0, 2)])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 60, 2), [])
        self.assertEqual(arvados.locators_and_ranges(blocks2, -2, 2), [])

        self.assertEqual(arvados.locators_and_ranges(blocks2,  9,  2), [LocatorAndRange('a', 10, 9, 1), LocatorAndRange('b', 10, 0, 1)])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 19, 2), [LocatorAndRange('b', 10, 9, 1), LocatorAndRange('c', 10, 0, 1)])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 29, 2), [LocatorAndRange('c', 10, 9, 1), LocatorAndRange('d', 10, 0, 1)])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 39, 2), [LocatorAndRange('d', 10, 9, 1), LocatorAndRange('e', 10, 0, 1)])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 49, 2), [LocatorAndRange('e', 10, 9, 1), LocatorAndRange('f', 10, 0, 1)])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 59, 2), [LocatorAndRange('f', 10, 9, 1)])


        blocks3 = [Range('a', 0, 10),
                  Range('b', 10, 10),
                  Range('c', 20, 10),
                  Range('d', 30, 10),
                  Range('e', 40, 10),
                  Range('f', 50, 10),
                   Range('g', 60, 10)]

        self.assertEqual(arvados.locators_and_ranges(blocks3,  2,  2), [LocatorAndRange('a', 10, 2, 2)])
        self.assertEqual(arvados.locators_and_ranges(blocks3, 12, 2), [LocatorAndRange('b', 10, 2, 2)])
        self.assertEqual(arvados.locators_and_ranges(blocks3, 22, 2), [LocatorAndRange('c', 10, 2, 2)])
        self.assertEqual(arvados.locators_and_ranges(blocks3, 32, 2), [LocatorAndRange('d', 10, 2, 2)])
        self.assertEqual(arvados.locators_and_ranges(blocks3, 42, 2), [LocatorAndRange('e', 10, 2, 2)])
        self.assertEqual(arvados.locators_and_ranges(blocks3, 52, 2), [LocatorAndRange('f', 10, 2, 2)])
        self.assertEqual(arvados.locators_and_ranges(blocks3, 62, 2), [LocatorAndRange('g', 10, 2, 2)])


        blocks = [Range('a', 0, 10),
                  Range('b', 10, 15),
                  Range('c', 25, 5)]
        self.assertEqual(arvados.locators_and_ranges(blocks, 1, 0), [])
        self.assertEqual(arvados.locators_and_ranges(blocks, 0, 5), [LocatorAndRange('a', 10, 0, 5)])
        self.assertEqual(arvados.locators_and_ranges(blocks, 3, 5), [LocatorAndRange('a', 10, 3, 5)])
        self.assertEqual(arvados.locators_and_ranges(blocks, 0, 10), [LocatorAndRange('a', 10, 0, 10)])

        self.assertEqual(arvados.locators_and_ranges(blocks, 0, 11), [LocatorAndRange('a', 10, 0, 10),
                                                                      LocatorAndRange('b', 15, 0, 1)])
        self.assertEqual(arvados.locators_and_ranges(blocks, 1, 11), [LocatorAndRange('a', 10, 1, 9),
                                                                      LocatorAndRange('b', 15, 0, 2)])
        self.assertEqual(arvados.locators_and_ranges(blocks, 0, 25), [LocatorAndRange('a', 10, 0, 10),
                                                                      LocatorAndRange('b', 15, 0, 15)])

        self.assertEqual(arvados.locators_and_ranges(blocks, 0, 30), [LocatorAndRange('a', 10, 0, 10),
                                                                      LocatorAndRange('b', 15, 0, 15),
                                                                      LocatorAndRange('c', 5, 0, 5)])
        self.assertEqual(arvados.locators_and_ranges(blocks, 1, 30), [LocatorAndRange('a', 10, 1, 9),
                                                                      LocatorAndRange('b', 15, 0, 15),
                                                                      LocatorAndRange('c', 5, 0, 5)])
        self.assertEqual(arvados.locators_and_ranges(blocks, 0, 31), [LocatorAndRange('a', 10, 0, 10),
                                                                      LocatorAndRange('b', 15, 0, 15),
                                                                      LocatorAndRange('c', 5, 0, 5)])

        self.assertEqual(arvados.locators_and_ranges(blocks, 15, 5), [LocatorAndRange('b', 15, 5, 5)])

        self.assertEqual(arvados.locators_and_ranges(blocks, 8, 17), [LocatorAndRange('a', 10, 8, 2),
                                                                      LocatorAndRange('b', 15, 0, 15)])

        self.assertEqual(arvados.locators_and_ranges(blocks, 8, 20), [LocatorAndRange('a', 10, 8, 2),
                                                                      LocatorAndRange('b', 15, 0, 15),
                                                                      LocatorAndRange('c', 5, 0, 3)])

        self.assertEqual(arvados.locators_and_ranges(blocks, 26, 2), [LocatorAndRange('c', 5, 1, 2)])

        self.assertEqual(arvados.locators_and_ranges(blocks, 9, 15), [LocatorAndRange('a', 10, 9, 1),
                                                                      LocatorAndRange('b', 15, 0, 14)])
        self.assertEqual(arvados.locators_and_ranges(blocks, 10, 15), [LocatorAndRange('b', 15, 0, 15)])
        self.assertEqual(arvados.locators_and_ranges(blocks, 11, 15), [LocatorAndRange('b', 15, 1, 14),
                                                                       LocatorAndRange('c', 5, 0, 1)])

    class MockKeep(object):
        def __init__(self, content, num_retries=0):
            self.content = content

        def get(self, locator, num_retries=0, prefetch=False):
            return self.content[locator]

    def test_stream_reader(self):
        keepblocks = {
            'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa+10': b'abcdefghij',
            'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb+15': b'klmnopqrstuvwxy',
            'cccccccccccccccccccccccccccccccc+5': b'z0123',
        }
        mk = self.MockKeep(keepblocks)

        sr = arvados.StreamReader([".", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa+10", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb+15", "cccccccccccccccccccccccccccccccc+5", "0:30:foo"], mk)

        content = b'abcdefghijklmnopqrstuvwxyz0123456789'

        self.assertEqual(sr.readfrom(0, 30), content[0:30])
        self.assertEqual(sr.readfrom(2, 30), content[2:30])

        self.assertEqual(sr.readfrom(2, 8), content[2:10])
        self.assertEqual(sr.readfrom(0, 10), content[0:10])

        self.assertEqual(sr.readfrom(0, 5), content[0:5])
        self.assertEqual(sr.readfrom(5, 5), content[5:10])
        self.assertEqual(sr.readfrom(10, 5), content[10:15])
        self.assertEqual(sr.readfrom(15, 5), content[15:20])
        self.assertEqual(sr.readfrom(20, 5), content[20:25])
        self.assertEqual(sr.readfrom(25, 5), content[25:30])
        self.assertEqual(sr.readfrom(30, 5), b'')

    def test_extract_file(self):
        m1 = """. 5348b82a029fd9e971a811ce1f71360b+43 0:43:md5sum.txt
. 085c37f02916da1cad16f93c54d899b7+41 0:41:md6sum.txt
. 8b22da26f9f433dea0a10e5ec66d73ba+43 0:43:md7sum.txt
. 085c37f02916da1cad16f93c54d899b7+41 5348b82a029fd9e971a811ce1f71360b+43 8b22da26f9f433dea0a10e5ec66d73ba+43 47:80:md8sum.txt
. 085c37f02916da1cad16f93c54d899b7+41 5348b82a029fd9e971a811ce1f71360b+43 8b22da26f9f433dea0a10e5ec66d73ba+43 40:80:md9sum.txt
"""

        m2 = arvados.CollectionReader(m1, self.api_client).manifest_text(normalize=True)

        self.assertEqual(m2,
                         ". 5348b82a029fd9e971a811ce1f71360b+43 085c37f02916da1cad16f93c54d899b7+41 8b22da26f9f433dea0a10e5ec66d73ba+43 0:43:md5sum.txt 43:41:md6sum.txt 84:43:md7sum.txt 6:37:md8sum.txt 84:43:md8sum.txt 83:1:md9sum.txt 0:43:md9sum.txt 84:36:md9sum.txt\n")
        files = arvados.CollectionReader(
            m2, self.api_client).all_streams()[0].files()

        self.assertEqual(files['md5sum.txt'].as_manifest(),
                         ". 5348b82a029fd9e971a811ce1f71360b+43 0:43:md5sum.txt\n")
        self.assertEqual(files['md6sum.txt'].as_manifest(),
                         ". 085c37f02916da1cad16f93c54d899b7+41 0:41:md6sum.txt\n")
        self.assertEqual(files['md7sum.txt'].as_manifest(),
                         ". 8b22da26f9f433dea0a10e5ec66d73ba+43 0:43:md7sum.txt\n")
        self.assertEqual(files['md9sum.txt'].as_manifest(),
                         ". 085c37f02916da1cad16f93c54d899b7+41 5348b82a029fd9e971a811ce1f71360b+43 8b22da26f9f433dea0a10e5ec66d73ba+43 40:80:md9sum.txt\n")

    def test_write_directory_tree(self):
        cwriter = arvados.CollectionWriter(self.api_client)
        cwriter.write_directory_tree(self.build_directory_tree(
                ['basefile', 'subdir/subfile']))
        self.assertEqual(cwriter.manifest_text(),
                         """. c5110c5ac93202d8e0f9e381f22bac0f+8 0:8:basefile
./subdir 1ca4dec89403084bf282ad31e6cf7972+14 0:14:subfile\n""")

    def test_write_named_directory_tree(self):
        cwriter = arvados.CollectionWriter(self.api_client)
        cwriter.write_directory_tree(self.build_directory_tree(
                ['basefile', 'subdir/subfile']), 'root')
        self.assertEqual(
            cwriter.manifest_text(),
            """./root c5110c5ac93202d8e0f9e381f22bac0f+8 0:8:basefile
./root/subdir 1ca4dec89403084bf282ad31e6cf7972+14 0:14:subfile\n""")

    def test_write_directory_tree_in_one_stream(self):
        cwriter = arvados.CollectionWriter(self.api_client)
        cwriter.write_directory_tree(self.build_directory_tree(
                ['basefile', 'subdir/subfile']), max_manifest_depth=0)
        self.assertEqual(cwriter.manifest_text(),
                         """. 4ace875ffdc6824a04950f06858f4465+22 0:8:basefile 8:14:subdir/subfile\n""")

    def test_write_directory_tree_with_limited_recursion(self):
        cwriter = arvados.CollectionWriter(self.api_client)
        cwriter.write_directory_tree(
            self.build_directory_tree(['f1', 'd1/f2', 'd1/d2/f3']),
            max_manifest_depth=1)
        self.assertEqual(cwriter.manifest_text(),
                         """. bd19836ddb62c11c55ab251ccaca5645+2 0:2:f1
./d1 50170217e5b04312024aa5cd42934494+13 0:8:d2/f3 8:5:f2\n""")

    def test_write_directory_tree_with_zero_recursion(self):
        cwriter = arvados.CollectionWriter(self.api_client)
        content = 'd1/d2/f3d1/f2f1'
        blockhash = tutil.str_keep_locator(content)
        cwriter.write_directory_tree(
            self.build_directory_tree(['f1', 'd1/f2', 'd1/d2/f3']),
            max_manifest_depth=0)
        self.assertEqual(
            cwriter.manifest_text(),
            ". {} 0:8:d1/d2/f3 8:5:d1/f2 13:2:f1\n".format(blockhash))

    def test_write_one_file(self):
        cwriter = arvados.CollectionWriter(self.api_client)
        with self.make_test_file() as testfile:
            cwriter.write_file(testfile.name)
            self.assertEqual(
                cwriter.manifest_text(),
                ". 098f6bcd4621d373cade4e832627b4f6+4 0:4:{}\n".format(
                    os.path.basename(testfile.name)))

    def test_write_named_file(self):
        cwriter = arvados.CollectionWriter(self.api_client)
        with self.make_test_file() as testfile:
            cwriter.write_file(testfile.name, 'foo')
            self.assertEqual(cwriter.manifest_text(),
                             ". 098f6bcd4621d373cade4e832627b4f6+4 0:4:foo\n")

    def test_write_multiple_files(self):
        cwriter = arvados.CollectionWriter(self.api_client)
        for letter in 'ABC':
            with self.make_test_file(letter.encode()) as testfile:
                cwriter.write_file(testfile.name, letter)
        self.assertEqual(
            cwriter.manifest_text(),
            ". 902fbdd2b1df0c4f70b4a5d23525e932+3 0:1:A 1:1:B 2:1:C\n")

    def test_basic_resume(self):
        cwriter = TestResumableWriter()
        with self.make_test_file() as testfile:
            cwriter.write_file(testfile.name, 'test')
            resumed = TestResumableWriter.from_state(cwriter.current_state())
        self.assertEqual(cwriter.manifest_text(), resumed.manifest_text(),
                          "resumed CollectionWriter had different manifest")

    def test_resume_fails_when_missing_dependency(self):
        cwriter = TestResumableWriter()
        with self.make_test_file() as testfile:
            cwriter.write_file(testfile.name, 'test')
        self.assertRaises(arvados.errors.StaleWriterStateError,
                          TestResumableWriter.from_state,
                          cwriter.current_state())

    def test_resume_fails_when_dependency_mtime_changed(self):
        cwriter = TestResumableWriter()
        with self.make_test_file() as testfile:
            cwriter.write_file(testfile.name, 'test')
            os.utime(testfile.name, (0, 0))
            self.assertRaises(arvados.errors.StaleWriterStateError,
                              TestResumableWriter.from_state,
                              cwriter.current_state())

    def test_resume_fails_when_dependency_is_nonfile(self):
        cwriter = TestResumableWriter()
        cwriter.write_file('/dev/null', 'empty')
        self.assertRaises(arvados.errors.StaleWriterStateError,
                          TestResumableWriter.from_state,
                          cwriter.current_state())

    def test_resume_fails_when_dependency_size_changed(self):
        cwriter = TestResumableWriter()
        with self.make_test_file() as testfile:
            cwriter.write_file(testfile.name, 'test')
            orig_mtime = os.fstat(testfile.fileno()).st_mtime
            testfile.write(b'extra')
            testfile.flush()
            os.utime(testfile.name, (orig_mtime, orig_mtime))
            self.assertRaises(arvados.errors.StaleWriterStateError,
                              TestResumableWriter.from_state,
                              cwriter.current_state())

    def test_resume_fails_with_expired_locator(self):
        cwriter = TestResumableWriter()
        state = cwriter.current_state()
        # Add an expired locator to the state.
        state['_current_stream_locators'].append(''.join([
                    'a' * 32, '+1+A', 'b' * 40, '@', '10000000']))
        self.assertRaises(arvados.errors.StaleWriterStateError,
                          TestResumableWriter.from_state, state)

    def test_arbitrary_objects_not_resumable(self):
        cwriter = TestResumableWriter()
        with open('/dev/null') as badfile:
            self.assertRaises(arvados.errors.AssertionError,
                              cwriter.write_file, badfile)

    def test_arbitrary_writes_not_resumable(self):
        cwriter = TestResumableWriter()
        self.assertRaises(arvados.errors.AssertionError,
                          cwriter.write, "badtext")


class CollectionTestMixin(tutil.ApiClientMock):
    API_COLLECTIONS = run_test_server.fixture('collections')
    DEFAULT_COLLECTION = API_COLLECTIONS['foo_file']
    DEFAULT_DATA_HASH = DEFAULT_COLLECTION['portable_data_hash']
    DEFAULT_MANIFEST = DEFAULT_COLLECTION['manifest_text']
    DEFAULT_UUID = DEFAULT_COLLECTION['uuid']
    ALT_COLLECTION = API_COLLECTIONS['bar_file']
    ALT_DATA_HASH = ALT_COLLECTION['portable_data_hash']
    ALT_MANIFEST = ALT_COLLECTION['manifest_text']

    def api_client_mock(self, status=200):
        client = super(CollectionTestMixin, self).api_client_mock()
        self.mock_keep_services(client, status=status, service_type='proxy', count=1)
        return client


@tutil.skip_sleep
class CollectionReaderTestCase(unittest.TestCase, CollectionTestMixin):
    def mock_get_collection(self, api_mock, code, fixturename):
        body = self.API_COLLECTIONS.get(fixturename)
        self._mock_api_call(api_mock.collections().get, code, body)

    def api_client_mock(self, status=200):
        client = super(CollectionReaderTestCase, self).api_client_mock()
        self.mock_get_collection(client, status, 'foo_file')
        return client

    def test_init_no_default_retries(self):
        client = self.api_client_mock(200)
        reader = arvados.CollectionReader(self.DEFAULT_UUID, api_client=client)
        reader.manifest_text()
        client.collections().get().execute.assert_called_with(num_retries=0)

    def test_uuid_init_success(self):
        client = self.api_client_mock(200)
        reader = arvados.CollectionReader(self.DEFAULT_UUID, api_client=client,
                                          num_retries=3)
        self.assertEqual(self.DEFAULT_COLLECTION['manifest_text'],
                         reader.manifest_text())
        client.collections().get().execute.assert_called_with(num_retries=3)

    def test_uuid_init_failure_raises_api_error(self):
        client = self.api_client_mock(500)
        with self.assertRaises(arvados.errors.ApiError):
            reader = arvados.CollectionReader(self.DEFAULT_UUID, api_client=client)

    def test_locator_init(self):
        client = self.api_client_mock(200)
        # Ensure Keep will not return anything if asked.
        with tutil.mock_keep_responses(None, 404):
            reader = arvados.CollectionReader(self.DEFAULT_DATA_HASH,
                                              api_client=client)
            self.assertEqual(self.DEFAULT_MANIFEST, reader.manifest_text())

    def test_init_no_fallback_to_keep(self):
        # Do not look up a collection UUID or PDH in Keep.
        for key in [self.DEFAULT_UUID, self.DEFAULT_DATA_HASH]:
            client = self.api_client_mock(404)
            with tutil.mock_keep_responses(self.DEFAULT_MANIFEST, 200):
                with self.assertRaises(arvados.errors.ApiError):
                    reader = arvados.CollectionReader(key, api_client=client)

    def test_init_num_retries_propagated(self):
        # More of an integration test...
        client = self.api_client_mock(200)
        reader = arvados.CollectionReader(self.DEFAULT_UUID, api_client=client,
                                          num_retries=3)
        with tutil.mock_keep_responses('foo', 500, 500, 200):
            self.assertEqual(b'foo',
                             b''.join(f.read(9) for f in reader.all_files()))

    def test_read_nonnormalized_manifest_with_collection_reader(self):
        # client should be able to use CollectionReader on a manifest without normalizing it
        client = self.api_client_mock(500)
        nonnormal = ". acbd18db4cc2f85cedef654fccc4a4d8+3+Aabadbadbee@abeebdee 0:3:foo.txt 1:0:bar.txt 0:3:foo.txt\n"
        reader = arvados.CollectionReader(
            nonnormal,
            api_client=client, num_retries=0)
        # Ensure stripped_manifest() doesn't mangle our manifest in
        # any way other than stripping hints.
        self.assertEqual(
            re.sub('\+[^\d\s\+]+', '', nonnormal),
            reader.stripped_manifest())
        # Ensure stripped_manifest() didn't mutate our reader.
        self.assertEqual(nonnormal, reader.manifest_text())
        # Ensure the files appear in the order given in the manifest.
        self.assertEqual(
            [[6, '.', 'foo.txt'],
             [0, '.', 'bar.txt']],
            [[f.size(), f.stream_name(), f.name()]
             for f in reader.all_streams()[0].all_files()])

    def test_read_empty_collection(self):
        client = self.api_client_mock(200)
        self.mock_get_collection(client, 200, 'empty')
        reader = arvados.CollectionReader('d41d8cd98f00b204e9800998ecf8427e+0',
                                          api_client=client)
        self.assertEqual('', reader.manifest_text())
        self.assertEqual(0, len(reader))
        self.assertFalse(reader)

    def test_api_response(self):
        client = self.api_client_mock()
        reader = arvados.CollectionReader(self.DEFAULT_UUID, api_client=client)
        self.assertEqual(self.DEFAULT_COLLECTION, reader.api_response())

    def check_open_file(self, coll_file, stream_name, file_name, file_size):
        self.assertFalse(coll_file.closed, "returned file is not open")
        self.assertEqual(stream_name, coll_file.stream_name())
        self.assertEqual(file_name, coll_file.name)
        self.assertEqual(file_size, coll_file.size())

    def test_open_collection_file_one_argument(self):
        client = self.api_client_mock(200)
        reader = arvados.CollectionReader(self.DEFAULT_UUID, api_client=client)
        cfile = reader.open('./foo', 'rb')
        self.check_open_file(cfile, '.', 'foo', 3)

    def test_open_deep_file(self):
        coll_name = 'collection_with_files_in_subdir'
        client = self.api_client_mock(200)
        self.mock_get_collection(client, 200, coll_name)
        reader = arvados.CollectionReader(
            self.API_COLLECTIONS[coll_name]['uuid'], api_client=client)
        cfile = reader.open('./subdir2/subdir3/file2_in_subdir3.txt', 'rb')
        self.check_open_file(cfile, './subdir2/subdir3', 'file2_in_subdir3.txt',
                             32)

    def test_open_nonexistent_stream(self):
        client = self.api_client_mock(200)
        reader = arvados.CollectionReader(self.DEFAULT_UUID, api_client=client)
        self.assertRaises(IOError, reader.open, './nonexistent/foo')

    def test_open_nonexistent_file(self):
        client = self.api_client_mock(200)
        reader = arvados.CollectionReader(self.DEFAULT_UUID, api_client=client)
        self.assertRaises(IOError, reader.open, 'nonexistent')


@tutil.skip_sleep
class CollectionWriterTestCase(unittest.TestCase, CollectionTestMixin):
    def mock_keep(self, body, *codes, **headers):
        headers.setdefault('x-keep-replicas-stored', 2)
        return tutil.mock_keep_responses(body, *codes, **headers)

    def foo_writer(self, **kwargs):
        kwargs.setdefault('api_client', self.api_client_mock())
        writer = arvados.CollectionWriter(**kwargs)
        writer.start_new_file('foo')
        writer.write(b'foo')
        return writer

    def test_write_whole_collection(self):
        writer = self.foo_writer()
        with self.mock_keep(self.DEFAULT_DATA_HASH, 200, 200):
            self.assertEqual(self.DEFAULT_DATA_HASH, writer.finish())

    def test_write_no_default(self):
        writer = self.foo_writer()
        with self.mock_keep(None, 500):
            with self.assertRaises(arvados.errors.KeepWriteError):
                writer.finish()

    def test_write_insufficient_replicas_via_proxy(self):
        writer = self.foo_writer(replication=3)
        with self.mock_keep(None, 200, **{'x-keep-replicas-stored': 2}):
            with self.assertRaises(arvados.errors.KeepWriteError):
                writer.manifest_text()

    def test_write_insufficient_replicas_via_disks(self):
        client = mock.MagicMock(name='api_client')
        with self.mock_keep(
                None, 200, 200,
                **{'x-keep-replicas-stored': 1}) as keepmock:
            self.mock_keep_services(client, status=200, service_type='disk', count=2)
            writer = self.foo_writer(api_client=client, replication=3)
            with self.assertRaises(arvados.errors.KeepWriteError):
                writer.manifest_text()

    def test_write_three_replicas(self):
        client = mock.MagicMock(name='api_client')
        with self.mock_keep(
                "", 500, 500, 500, 200, 200, 200,
                **{'x-keep-replicas-stored': 1}) as keepmock:
            self.mock_keep_services(client, status=200, service_type='disk', count=6)
            writer = self.foo_writer(api_client=client, replication=3)
            writer.manifest_text()
            self.assertEqual(6, keepmock.call_count)

    def test_write_whole_collection_through_retries(self):
        writer = self.foo_writer(num_retries=2)
        with self.mock_keep(self.DEFAULT_DATA_HASH,
                            500, 500, 200, 500, 500, 200):
            self.assertEqual(self.DEFAULT_DATA_HASH, writer.finish())

    def test_flush_data_retries(self):
        writer = self.foo_writer(num_retries=2)
        foo_hash = self.DEFAULT_MANIFEST.split()[1]
        with self.mock_keep(foo_hash, 500, 200):
            writer.flush_data()
        self.assertEqual(self.DEFAULT_MANIFEST, writer.manifest_text())

    def test_one_open(self):
        client = self.api_client_mock()
        writer = arvados.CollectionWriter(client)
        with writer.open('out') as out_file:
            self.assertEqual('.', writer.current_stream_name())
            self.assertEqual('out', writer.current_file_name())
            out_file.write(b'test data')
            data_loc = tutil.str_keep_locator('test data')
        self.assertTrue(out_file.closed, "writer file not closed after context")
        self.assertRaises(ValueError, out_file.write, 'extra text')
        with self.mock_keep(data_loc, 200) as keep_mock:
            self.assertEqual(". {} 0:9:out\n".format(data_loc),
                             writer.manifest_text())

    def test_open_writelines(self):
        client = self.api_client_mock()
        writer = arvados.CollectionWriter(client)
        with writer.open('six') as out_file:
            out_file.writelines(['12', '34', '56'])
            data_loc = tutil.str_keep_locator('123456')
        with self.mock_keep(data_loc, 200) as keep_mock:
            self.assertEqual(". {} 0:6:six\n".format(data_loc),
                             writer.manifest_text())

    def test_open_flush(self):
        client = self.api_client_mock()
        data_loc1 = tutil.str_keep_locator('flush1')
        data_loc2 = tutil.str_keep_locator('flush2')
        with self.mock_keep((data_loc1, 200), (data_loc2, 200)) as keep_mock:
            writer = arvados.CollectionWriter(client)
            with writer.open('flush_test') as out_file:
                out_file.write(b'flush1')
                out_file.flush()
                out_file.write(b'flush2')
            self.assertEqual(". {} {} 0:12:flush_test\n".format(data_loc1,
                                                                data_loc2),
                             writer.manifest_text())

    def test_two_opens_same_stream(self):
        client = self.api_client_mock()
        writer = arvados.CollectionWriter(client)
        with writer.open('.', '1') as out_file:
            out_file.write(b'1st')
        with writer.open('.', '2') as out_file:
            out_file.write(b'2nd')
        data_loc = tutil.str_keep_locator('1st2nd')
        with self.mock_keep(data_loc, 200) as keep_mock:
            self.assertEqual(". {} 0:3:1 3:3:2\n".format(data_loc),
                             writer.manifest_text())

    def test_two_opens_two_streams(self):
        client = self.api_client_mock()
        data_loc1 = tutil.str_keep_locator('file')
        data_loc2 = tutil.str_keep_locator('indir')
        with self.mock_keep((data_loc1, 200), (data_loc2, 200)) as keep_mock:
            writer = arvados.CollectionWriter(client)
            with writer.open('file') as out_file:
                out_file.write(b'file')
            with writer.open('./dir', 'indir') as out_file:
                out_file.write(b'indir')
            expected = ". {} 0:4:file\n./dir {} 0:5:indir\n".format(
                data_loc1, data_loc2)
            self.assertEqual(expected, writer.manifest_text())

    def test_dup_open_fails(self):
        client = self.api_client_mock()
        writer = arvados.CollectionWriter(client)
        file1 = writer.open('one')
        self.assertRaises(arvados.errors.AssertionError, writer.open, 'two')


class CollectionMethods(run_test_server.TestCaseWithServers):

    def test_keys_values_items_support_indexing(self):
        c = Collection()
        with c.open('foo', 'wb') as f:
            f.write(b'foo')
        with c.open('bar', 'wb') as f:
            f.write(b'bar')
        self.assertEqual(2, len(c.keys()))
        if sys.version_info < (3, 0):
            # keys() supports indexing only for python2 callers.
            fn0 = c.keys()[0]
            fn1 = c.keys()[1]
        else:
            fn0, fn1 = c.keys()
        self.assertEqual(2, len(c.values()))
        f0 = c.values()[0]
        f1 = c.values()[1]
        self.assertEqual(2, len(c.items()))
        self.assertEqual(fn0, c.items()[0][0])
        self.assertEqual(fn1, c.items()[1][0])

    def test_get_properties(self):
        c = Collection()
        self.assertEqual(c.get_properties(), {})
        c.save_new(properties={"foo":"bar"})
        self.assertEqual(c.get_properties(), {"foo":"bar"})

    def test_get_trash_at(self):
        c = Collection()
        self.assertEqual(c.get_trash_at(), None)
        c.save_new(trash_at=datetime.datetime(2111, 1, 1, 11, 11, 11, 111111))
        self.assertEqual(c.get_trash_at(), ciso8601.parse_datetime('2111-01-01T11:11:11.111111000Z'))


class CollectionOpenModes(run_test_server.TestCaseWithServers):

    def test_open_binary_modes(self):
        c = Collection()
        for mode in ['wb', 'wb+', 'ab', 'ab+']:
            with c.open('foo', mode) as f:
                f.write(b'foo')

    def test_open_invalid_modes(self):
        c = Collection()
        for mode in ['+r', 'aa', '++', 'r+b', 'beer', '', None]:
            with self.assertRaises(Exception):
                c.open('foo', mode)

    def test_open_text_modes(self):
        c = Collection()
        with c.open('foo', 'wb') as f:
            f.write('foo')
        for mode in ['r', 'rt', 'r+', 'rt+', 'w', 'wt', 'a', 'at']:
            with c.open('foo', mode) as f:
                if mode[0] == 'r' and '+' not in mode:
                    self.assertEqual('foo', f.read(3))
                else:
                    f.write('bar')
                    f.seek(0, os.SEEK_SET)
                    self.assertEqual('bar', f.read(3))


class TextModes(run_test_server.TestCaseWithServers):

    def setUp(self):
        arvados.config.KEEP_BLOCK_SIZE = 4
        if sys.version_info < (3, 0):
            import unicodedata
            self.sailboat = unicodedata.lookup('SAILBOAT')
            self.snowman = unicodedata.lookup('SNOWMAN')
        else:
            self.sailboat = '\N{SAILBOAT}'
            self.snowman = '\N{SNOWMAN}'

    def tearDown(self):
        arvados.config.KEEP_BLOCK_SIZE = 2 ** 26

    def test_read_sailboat_across_block_boundary(self):
        c = Collection()
        f = c.open('sailboats', 'wb')
        data = self.sailboat.encode('utf-8')
        f.write(data)
        f.write(data[:1])
        f.write(data[1:])
        f.write(b'\n')
        f.close()
        self.assertRegex(c.portable_manifest_text(), r'\+4 .*\+3 ')

        f = c.open('sailboats', 'r')
        string = f.readline()
        self.assertEqual(string, self.sailboat+self.sailboat+'\n')
        f.close()

    def test_write_snowman_across_block_boundary(self):
        c = Collection()
        f = c.open('snowmany', 'w')
        data = self.snowman
        f.write(data+data+'\n'+data+'\n')
        f.close()
        self.assertRegex(c.portable_manifest_text(), r'\+4 .*\+4 .*\+3 ')

        f = c.open('snowmany', 'r')
        self.assertEqual(f.readline(), self.snowman+self.snowman+'\n')
        self.assertEqual(f.readline(), self.snowman+'\n')
        f.close()


class NewCollectionTestCase(unittest.TestCase, CollectionTestMixin):

    def test_replication_desired_kept_on_load(self):
        m = '. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt 0:10:count2.txt\n'
        c1 = Collection(m, replication_desired=1)
        c1.save_new()
        loc = c1.manifest_locator()
        c2 = Collection(loc)
        self.assertEqual(c1.manifest_text(strip=True), c2.manifest_text(strip=True))
        self.assertEqual(c1.replication_desired, c2.replication_desired)

    def test_replication_desired_not_loaded_if_provided(self):
        m = '. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt 0:10:count2.txt\n'
        c1 = Collection(m, replication_desired=1)
        c1.save_new()
        loc = c1.manifest_locator()
        c2 = Collection(loc, replication_desired=2)
        self.assertEqual(c1.manifest_text(strip=True), c2.manifest_text(strip=True))
        self.assertNotEqual(c1.replication_desired, c2.replication_desired)

    def test_storage_classes_desired_kept_on_load(self):
        m = '. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt 0:10:count2.txt\n'
        c1 = Collection(m, storage_classes_desired=['archival'])
        c1.save_new()
        loc = c1.manifest_locator()
        c2 = Collection(loc)
        self.assertEqual(c1.manifest_text(strip=True), c2.manifest_text(strip=True))
        self.assertEqual(c1.storage_classes_desired(), c2.storage_classes_desired())

    def test_storage_classes_change_after_save(self):
        m = '. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt 0:10:count2.txt\n'
        c1 = Collection(m, storage_classes_desired=['archival'])
        c1.save_new()
        loc = c1.manifest_locator()
        c2 = Collection(loc)
        self.assertEqual(['archival'], c2.storage_classes_desired())
        c2.save(storage_classes=['highIO'])
        self.assertEqual(['highIO'], c2.storage_classes_desired())
        c3 = Collection(loc)
        self.assertEqual(c1.manifest_text(strip=True), c3.manifest_text(strip=True))
        self.assertEqual(['highIO'], c3.storage_classes_desired())

    def test_storage_classes_desired_not_loaded_if_provided(self):
        m = '. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt 0:10:count2.txt\n'
        c1 = Collection(m, storage_classes_desired=['archival'])
        c1.save_new()
        loc = c1.manifest_locator()
        c2 = Collection(loc, storage_classes_desired=['default'])
        self.assertEqual(c1.manifest_text(strip=True), c2.manifest_text(strip=True))
        self.assertNotEqual(c1.storage_classes_desired(), c2.storage_classes_desired())

    def test_init_manifest(self):
        m1 = """. 5348b82a029fd9e971a811ce1f71360b+43 0:43:md5sum.txt
. 085c37f02916da1cad16f93c54d899b7+41 0:41:md5sum.txt
. 8b22da26f9f433dea0a10e5ec66d73ba+43 0:43:md5sum.txt
"""
        self.assertEqual(m1, CollectionReader(m1).manifest_text(normalize=False))
        self.assertEqual(". 5348b82a029fd9e971a811ce1f71360b+43 085c37f02916da1cad16f93c54d899b7+41 8b22da26f9f433dea0a10e5ec66d73ba+43 0:127:md5sum.txt\n", CollectionReader(m1).manifest_text(normalize=True))

    def test_init_manifest_with_collision(self):
        m1 = """. 5348b82a029fd9e971a811ce1f71360b+43 0:43:md5sum.txt
./md5sum.txt 085c37f02916da1cad16f93c54d899b7+41 0:41:md5sum.txt
"""
        with self.assertRaises(arvados.errors.ArgumentError):
            self.assertEqual(m1, CollectionReader(m1))

    def test_init_manifest_with_error(self):
        m1 = """. 0:43:md5sum.txt"""
        with self.assertRaises(arvados.errors.ArgumentError):
            self.assertEqual(m1, CollectionReader(m1))

    def test_remove(self):
        c = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt 0:10:count2.txt\n')
        self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt 0:10:count2.txt\n", c.portable_manifest_text())
        self.assertIn("count1.txt", c)
        c.remove("count1.txt")
        self.assertNotIn("count1.txt", c)
        self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 0:10:count2.txt\n", c.portable_manifest_text())
        with self.assertRaises(arvados.errors.ArgumentError):
            c.remove("")

    def test_remove_recursive(self):
        c = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:a/b/c/d/efg.txt 0:10:xyz.txt\n')
        self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 0:10:xyz.txt\n./a/b/c/d 781e5e245d69b566979b86e28d23f2c7+10 0:10:efg.txt\n", c.portable_manifest_text())
        self.assertIn("a", c)
        self.assertEqual(1, len(c["a"].keys()))
        # cannot remove non-empty directory with default recursive=False
        with self.assertRaises(OSError):
            c.remove("a/b")
        with self.assertRaises(OSError):
            c.remove("a/b/c/d")
        c.remove("a/b", recursive=True)
        self.assertEqual(0, len(c["a"].keys()))
        self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 0:10:xyz.txt\n./a d41d8cd98f00b204e9800998ecf8427e+0 0:0:\\056\n", c.portable_manifest_text())

    def test_find(self):
        c = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt 0:10:count2.txt\n')
        self.assertIs(c.find("."), c)
        self.assertIs(c.find("./count1.txt"), c["count1.txt"])
        self.assertIs(c.find("count1.txt"), c["count1.txt"])
        with self.assertRaises(IOError):
            c.find("/.")
        with self.assertRaises(arvados.errors.ArgumentError):
            c.find("")
        self.assertIs(c.find("./nonexistant.txt"), None)
        self.assertIs(c.find("./nonexistantsubdir/nonexistant.txt"), None)

    def test_escaped_paths_dont_get_unescaped_on_manifest(self):
        # Dir & file names are literally '\056' (escaped form: \134056)
        manifest = './\\134056\\040Test d41d8cd98f00b204e9800998ecf8427e+0 0:0:\\134056\n'
        c = Collection(manifest)
        self.assertEqual(c.portable_manifest_text(), manifest)

    def test_other_special_chars_on_file_token(self):
        cases = [
            ('\\000', '\0'),
            ('\\011', '\t'),
            ('\\012', '\n'),
            ('\\072', ':'),
            ('\\134400', '\\400'),
        ]
        for encoded, decoded in cases:
            manifest = '. d41d8cd98f00b204e9800998ecf8427e+0 0:0:some%sfile.txt\n' % encoded
            c = Collection(manifest)
            self.assertEqual(c.portable_manifest_text(), manifest)
            self.assertIn('some%sfile.txt' % decoded, c.keys())

    def test_escaped_paths_do_get_unescaped_on_listing(self):
        # Dir & file names are literally '\056' (escaped form: \134056)
        manifest = './\\134056\\040Test d41d8cd98f00b204e9800998ecf8427e+0 0:0:\\134056\n'
        c = Collection(manifest)
        self.assertIn('\\056 Test', c.keys())
        self.assertIn('\\056', c['\\056 Test'].keys())

    def test_make_empty_dir_with_escaped_chars(self):
        c = Collection()
        c.mkdirs('./Empty\\056Dir')
        self.assertEqual(c.portable_manifest_text(),
                         './Empty\\134056Dir d41d8cd98f00b204e9800998ecf8427e+0 0:0:\\056\n')

    def test_make_empty_dir_with_spaces(self):
        c = Collection()
        c.mkdirs('./foo bar/baz waz')
        self.assertEqual(c.portable_manifest_text(),
                         './foo\\040bar/baz\\040waz d41d8cd98f00b204e9800998ecf8427e+0 0:0:\\056\n')

    def test_remove_in_subdir(self):
        c = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n./foo 781e5e245d69b566979b86e28d23f2c7+10 0:10:count2.txt\n')
        c.remove("foo/count2.txt")
        self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n./foo d41d8cd98f00b204e9800998ecf8427e+0 0:0:\\056\n", c.portable_manifest_text())

    def test_remove_empty_subdir(self):
        c = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n./foo 781e5e245d69b566979b86e28d23f2c7+10 0:10:count2.txt\n')
        c.remove("foo/count2.txt")
        c.remove("foo")
        self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n", c.portable_manifest_text())

    def test_remove_nonempty_subdir(self):
        c = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n./foo 781e5e245d69b566979b86e28d23f2c7+10 0:10:count2.txt\n')
        with self.assertRaises(IOError):
            c.remove("foo")
        c.remove("foo", recursive=True)
        self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n", c.portable_manifest_text())

    def test_copy_to_file_in_dir(self):
        c = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n')
        c.copy("count1.txt", "foo/count2.txt")
        self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n./foo 781e5e245d69b566979b86e28d23f2c7+10 0:10:count2.txt\n", c.portable_manifest_text())

    def test_copy_file(self):
        c = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n')
        c.copy("count1.txt", "count2.txt")
        self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt 0:10:count2.txt\n", c.portable_manifest_text())

    def test_copy_to_existing_dir(self):
        c = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n./foo 781e5e245d69b566979b86e28d23f2c7+10 0:10:count2.txt\n')
        c.copy("count1.txt", "foo")
        self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n./foo 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt 0:10:count2.txt\n", c.portable_manifest_text())

    def test_copy_to_new_dir(self):
        c = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n')
        c.copy("count1.txt", "foo/")
        self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n./foo 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n", c.portable_manifest_text())

    def test_rename_file(self):
        c = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n')
        c.rename("count1.txt", "count2.txt")
        self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 0:10:count2.txt\n", c.manifest_text())

    def test_move_file_to_dir(self):
        c = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n')
        c.mkdirs("foo")
        c.rename("count1.txt", "foo/count2.txt")
        self.assertEqual("./foo 781e5e245d69b566979b86e28d23f2c7+10 0:10:count2.txt\n", c.manifest_text())

    def test_move_file_to_other(self):
        c1 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n')
        c2 = Collection()
        c2.rename("count1.txt", "count2.txt", source_collection=c1)
        self.assertEqual("", c1.manifest_text())
        self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 0:10:count2.txt\n", c2.manifest_text())

    def test_clone(self):
        c = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n./foo 781e5e245d69b566979b86e28d23f2c7+10 0:10:count2.txt\n')
        cl = c.clone()
        self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n./foo 781e5e245d69b566979b86e28d23f2c7+10 0:10:count2.txt\n", cl.portable_manifest_text())

    def test_diff_del_add(self):
        c1 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n')
        c2 = Collection('. 5348b82a029fd9e971a811ce1f71360b+43 0:10:count2.txt\n')
        d = c2.diff(c1)
        self.assertEqual(sorted(d), [
            ('add', './count1.txt', c1["count1.txt"]),
            ('del', './count2.txt', c2["count2.txt"]),
        ])
        d = c1.diff(c2)
        self.assertEqual(sorted(d), [
            ('add', './count2.txt', c2["count2.txt"]),
            ('del', './count1.txt', c1["count1.txt"]),
        ])
        self.assertNotEqual(c1.portable_manifest_text(), c2.portable_manifest_text())
        c1.apply(d)
        self.assertEqual(c1.portable_manifest_text(), c2.portable_manifest_text())

    def test_diff_same(self):
        c1 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n')
        c2 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n')
        d = c2.diff(c1)
        self.assertEqual(d, [('tok', './count1.txt', c2["count1.txt"], c1["count1.txt"])])
        d = c1.diff(c2)
        self.assertEqual(d, [('tok', './count1.txt', c2["count1.txt"], c1["count1.txt"])])

        self.assertEqual(c1.portable_manifest_text(), c2.portable_manifest_text())
        c1.apply(d)
        self.assertEqual(c1.portable_manifest_text(), c2.portable_manifest_text())

    def test_diff_mod(self):
        c1 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n')
        c2 = Collection('. 5348b82a029fd9e971a811ce1f71360b+43 0:10:count1.txt\n')
        d = c2.diff(c1)
        self.assertEqual(d, [('mod', './count1.txt', c2["count1.txt"], c1["count1.txt"])])
        d = c1.diff(c2)
        self.assertEqual(d, [('mod', './count1.txt', c1["count1.txt"], c2["count1.txt"])])

        self.assertNotEqual(c1.portable_manifest_text(), c2.portable_manifest_text())
        c1.apply(d)
        self.assertEqual(c1.portable_manifest_text(), c2.portable_manifest_text())

    def test_diff_add(self):
        c1 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n')
        c2 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 5348b82a029fd9e971a811ce1f71360b+43 0:10:count1.txt 10:20:count2.txt\n')
        d = c2.diff(c1)
        self.assertEqual(sorted(d), [
            ('del', './count2.txt', c2["count2.txt"]),
            ('tok', './count1.txt', c2["count1.txt"], c1["count1.txt"]),
        ])
        d = c1.diff(c2)
        self.assertEqual(sorted(d), [
            ('add', './count2.txt', c2["count2.txt"]),
            ('tok', './count1.txt', c2["count1.txt"], c1["count1.txt"]),
        ])

        self.assertNotEqual(c1.portable_manifest_text(), c2.portable_manifest_text())
        c1.apply(d)
        self.assertEqual(c1.portable_manifest_text(), c2.portable_manifest_text())

    def test_diff_add_in_subcollection(self):
        c1 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n')
        c2 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n./foo 5348b82a029fd9e971a811ce1f71360b+43 0:10:count2.txt\n')
        d = c2.diff(c1)
        self.assertEqual(sorted(d), [
            ('del', './foo', c2["foo"]),
            ('tok', './count1.txt', c2["count1.txt"], c1["count1.txt"]),
        ])
        d = c1.diff(c2)
        self.assertEqual(sorted(d), [
            ('add', './foo', c2["foo"]),
            ('tok', './count1.txt', c2["count1.txt"], c1["count1.txt"]),
        ])
        self.assertNotEqual(c1.portable_manifest_text(), c2.portable_manifest_text())
        c1.apply(d)
        self.assertEqual(c1.portable_manifest_text(), c2.portable_manifest_text())

    def test_diff_del_add_in_subcollection(self):
        c1 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n./foo 5348b82a029fd9e971a811ce1f71360b+43 0:10:count2.txt\n')
        c2 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n./foo 5348b82a029fd9e971a811ce1f71360b+43 0:3:count3.txt\n')
        d = c2.diff(c1)
        self.assertEqual(sorted(d), [
            ('add', './foo/count2.txt', c1.find("foo/count2.txt")),
            ('del', './foo/count3.txt', c2.find("foo/count3.txt")),
            ('tok', './count1.txt', c2["count1.txt"], c1["count1.txt"]),
        ])
        d = c1.diff(c2)
        self.assertEqual(sorted(d), [
            ('add', './foo/count3.txt', c2.find("foo/count3.txt")),
            ('del', './foo/count2.txt', c1.find("foo/count2.txt")),
            ('tok', './count1.txt', c2["count1.txt"], c1["count1.txt"]),
        ])

        self.assertNotEqual(c1.portable_manifest_text(), c2.portable_manifest_text())
        c1.apply(d)
        self.assertEqual(c1.portable_manifest_text(), c2.portable_manifest_text())

    def test_diff_mod_in_subcollection(self):
        c1 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n./foo 5348b82a029fd9e971a811ce1f71360b+43 0:10:count2.txt\n')
        c2 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt 0:3:foo\n')
        d = c2.diff(c1)
        self.assertEqual(sorted(d), [
            ('mod', './foo', c2["foo"], c1["foo"]),
            ('tok', './count1.txt', c2["count1.txt"], c1["count1.txt"]),
        ])
        d = c1.diff(c2)
        self.assertEqual(sorted(d), [
            ('mod', './foo', c1["foo"], c2["foo"]),
            ('tok', './count1.txt', c2["count1.txt"], c1["count1.txt"]),
        ])

        self.assertNotEqual(c1.portable_manifest_text(), c2.portable_manifest_text())
        c1.apply(d)
        self.assertEqual(c1.portable_manifest_text(), c2.portable_manifest_text())

    def test_conflict_keep_local_change(self):
        c1 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n')
        c2 = Collection('. 5348b82a029fd9e971a811ce1f71360b+43 0:10:count2.txt\n')
        d = c1.diff(c2)
        self.assertEqual(sorted(d), [
            ('add', './count2.txt', c2["count2.txt"]),
            ('del', './count1.txt', c1["count1.txt"]),
        ])
        f = c1.open("count1.txt", "wb")
        f.write(b"zzzzz")

        # c1 changed, so it should not be deleted.
        c1.apply(d)
        self.assertEqual(c1.portable_manifest_text(), ". 95ebc3c7b3b9f1d2c40fec14415d3cb8+5 5348b82a029fd9e971a811ce1f71360b+43 0:5:count1.txt 5:10:count2.txt\n")

    def test_conflict_mod(self):
        c1 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt')
        c2 = Collection('. 5348b82a029fd9e971a811ce1f71360b+43 0:10:count1.txt')
        d = c1.diff(c2)
        self.assertEqual(d, [('mod', './count1.txt', c1["count1.txt"], c2["count1.txt"])])
        f = c1.open("count1.txt", "wb")
        f.write(b"zzzzz")

        # c1 changed, so c2 mod will go to a conflict file
        c1.apply(d)
        self.assertRegex(
            c1.portable_manifest_text(),
            r"\. 95ebc3c7b3b9f1d2c40fec14415d3cb8\+5 5348b82a029fd9e971a811ce1f71360b\+43 0:5:count1\.txt 5:10:count1\.txt~\d\d\d\d\d\d\d\d-\d\d\d\d\d\d~conflict~$")

    def test_conflict_add(self):
        c1 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count2.txt\n')
        c2 = Collection('. 5348b82a029fd9e971a811ce1f71360b+43 0:10:count1.txt\n')
        d = c1.diff(c2)
        self.assertEqual(sorted(d), [
            ('add', './count1.txt', c2["count1.txt"]),
            ('del', './count2.txt', c1["count2.txt"]),
        ])
        f = c1.open("count1.txt", "wb")
        f.write(b"zzzzz")

        # c1 added count1.txt, so c2 add will go to a conflict file
        c1.apply(d)
        self.assertRegex(
            c1.portable_manifest_text(),
            r"\. 95ebc3c7b3b9f1d2c40fec14415d3cb8\+5 5348b82a029fd9e971a811ce1f71360b\+43 0:5:count1\.txt 5:10:count1\.txt~\d\d\d\d\d\d\d\d-\d\d\d\d\d\d~conflict~$")

    def test_conflict_del(self):
        c1 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt')
        c2 = Collection('. 5348b82a029fd9e971a811ce1f71360b+43 0:10:count1.txt')
        d = c1.diff(c2)
        self.assertEqual(d, [('mod', './count1.txt', c1["count1.txt"], c2["count1.txt"])])
        c1.remove("count1.txt")

        # c1 deleted, so c2 mod will go to a conflict file
        c1.apply(d)
        self.assertRegex(
            c1.portable_manifest_text(),
            r"\. 5348b82a029fd9e971a811ce1f71360b\+43 0:10:count1\.txt~\d\d\d\d\d\d\d\d-\d\d\d\d\d\d~conflict~$")

    def test_notify(self):
        c1 = Collection()
        events = []
        c1.subscribe(lambda event, collection, name, item: events.append((event, collection, name, item)))
        f = c1.open("foo.txt", "wb")
        self.assertEqual(events[0], (arvados.collection.ADD, c1, "foo.txt", f.arvadosfile))

    def test_open_w(self):
        c1 = Collection(". 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n")
        self.assertEqual(c1["count1.txt"].size(), 10)
        c1.open("count1.txt", "wb").close()
        self.assertEqual(c1["count1.txt"].size(), 0)


class NewCollectionTestCaseWithServersAndTokens(run_test_server.TestCaseWithServers):
    MAIN_SERVER = {}
    KEEP_SERVER = {}
    local_locator_re = r"[0-9a-f]{32}\+\d+\+A[a-f0-9]{40}@[a-f0-9]{8}"
    remote_locator_re = r"[0-9a-f]{32}\+\d+\+R[a-z]{5}-[a-f0-9]{40}@[a-f0-9]{8}"

    def setUp(self):
        self.keep_put = getattr(arvados.keep.KeepClient, 'put')

    @mock.patch('arvados.keep.KeepClient.put', autospec=True)
    def test_storage_classes_desired(self, put_mock):
        put_mock.side_effect = self.keep_put
        c = Collection(storage_classes_desired=['default'])
        with c.open("file.txt", 'wb') as f:
            f.write('content')
        c.save_new()
        _, kwargs = put_mock.call_args
        self.assertEqual(['default'], kwargs['classes'])

    @mock.patch('arvados.keep.KeepClient.put', autospec=True)
    def test_repacked_block_submission_get_permission_token(self, mocked_put):
        '''
        Make sure that those blocks that are committed after repacking small ones,
        get their permission tokens assigned on the collection manifest.
        '''
        def wrapped_keep_put(*args, **kwargs):
            # Simulate slow put operations
            time.sleep(1)
            return self.keep_put(*args, **kwargs)

        mocked_put.side_effect = wrapped_keep_put
        c = Collection()
        # Write 70 files ~1MiB each so we force to produce 1 big block by repacking
        # small ones before finishing the upload.
        for i in range(70):
            f = c.open("file_{}.txt".format(i), 'wb')
            f.write(random.choice('abcdefghijklmnopqrstuvwxyz') * (2**20+i))
            f.close(flush=False)
        # We should get 2 blocks with their tokens
        self.assertEqual(len(re.findall(self.local_locator_re, c.manifest_text())), 2)

    @mock.patch('arvados.keep.KeepClient.refresh_signature')
    def test_copy_remote_blocks_on_save_new(self, rs_mock):
        remote_block_loc = "acbd18db4cc2f85cedef654fccc4a4d8+3+Remote-" + "a" * 40 + "@abcdef01"
        local_block_loc = "acbd18db4cc2f85cedef654fccc4a4d8+3+A" + "b" * 40 + "@abcdef01"
        rs_mock.return_value = local_block_loc
        c = Collection(". " + remote_block_loc + " 0:3:foofile.txt\n")
        self.assertEqual(
            len(re.findall(self.remote_locator_re, c.manifest_text())), 1)
        self.assertEqual(
            len(re.findall(self.local_locator_re, c.manifest_text())), 0)
        c.save_new()
        rs_mock.assert_called()
        self.assertEqual(
            len(re.findall(self.remote_locator_re, c.manifest_text())), 0)
        self.assertEqual(
            len(re.findall(self.local_locator_re, c.manifest_text())), 1)

    @mock.patch('arvados.keep.KeepClient.refresh_signature')
    def test_copy_remote_blocks_on_save(self, rs_mock):
        remote_block_loc = "acbd18db4cc2f85cedef654fccc4a4d8+3+Remote-" + "a" * 40 + "@abcdef01"
        local_block_loc = "acbd18db4cc2f85cedef654fccc4a4d8+3+A" + "b" * 40 + "@abcdef01"
        rs_mock.return_value = local_block_loc
        # Remote collection
        remote_c = Collection(". " + remote_block_loc + " 0:3:foofile.txt\n")
        self.assertEqual(
            len(re.findall(self.remote_locator_re, remote_c.manifest_text())), 1)
        # Local collection
        local_c = Collection()
        with local_c.open('barfile.txt', 'wb') as f:
            f.write('bar')
        local_c.save_new()
        self.assertEqual(
            len(re.findall(self.local_locator_re, local_c.manifest_text())), 1)
        self.assertEqual(
            len(re.findall(self.remote_locator_re, local_c.manifest_text())), 0)
        # Copy remote file to local collection
        local_c.copy('./foofile.txt', './copied/foofile.txt', remote_c)
        self.assertEqual(
            len(re.findall(self.local_locator_re, local_c.manifest_text())), 1)
        self.assertEqual(
            len(re.findall(self.remote_locator_re, local_c.manifest_text())), 1)
        # Save local collection: remote block should be copied
        local_c.save()
        rs_mock.assert_called()
        self.assertEqual(
            len(re.findall(self.local_locator_re, local_c.manifest_text())), 2)
        self.assertEqual(
            len(re.findall(self.remote_locator_re, local_c.manifest_text())), 0)


class NewCollectionTestCaseWithServers(run_test_server.TestCaseWithServers):
    def test_preserve_version_on_save(self):
        c = Collection()
        c.save_new(preserve_version=True)
        coll_record = arvados.api().collections().get(uuid=c.manifest_locator()).execute()
        self.assertEqual(coll_record['version'], 1)
        self.assertEqual(coll_record['preserve_version'], True)
        with c.open("foo.txt", "wb") as foo:
            foo.write(b"foo")
        c.save(preserve_version=True)
        coll_record = arvados.api().collections().get(uuid=c.manifest_locator()).execute()
        self.assertEqual(coll_record['version'], 2)
        self.assertEqual(coll_record['preserve_version'], True)
        with c.open("bar.txt", "wb") as foo:
            foo.write(b"bar")
        c.save(preserve_version=False)
        coll_record = arvados.api().collections().get(uuid=c.manifest_locator()).execute()
        self.assertEqual(coll_record['version'], 3)
        self.assertEqual(coll_record['preserve_version'], False)

    def test_get_manifest_text_only_committed(self):
        c = Collection()
        with c.open("count.txt", "wb") as f:
            # One file committed
            with c.open("foo.txt", "wb") as foo:
                foo.write(b"foo")
                foo.flush() # Force block commit
            f.write(b"0123456789")
            # Other file not committed. Block not written to keep yet.
            self.assertEqual(
                c._get_manifest_text(".",
                                     strip=False,
                                     normalize=False,
                                     only_committed=True),
                '. acbd18db4cc2f85cedef654fccc4a4d8+3 0:0:count.txt 0:3:foo.txt\n')
            # And now with the file closed...
            f.flush() # Force block commit
        self.assertEqual(
            c._get_manifest_text(".",
                                 strip=False,
                                 normalize=False,
                                 only_committed=True),
            ". 781e5e245d69b566979b86e28d23f2c7+10 acbd18db4cc2f85cedef654fccc4a4d8+3 0:10:count.txt 10:3:foo.txt\n")

    def test_only_small_blocks_are_packed_together(self):
        c = Collection()
        # Write a couple of small files,
        f = c.open("count.txt", "wb")
        f.write(b"0123456789")
        f.close(flush=False)
        foo = c.open("foo.txt", "wb")
        foo.write(b"foo")
        foo.close(flush=False)
        # Then, write a big file, it shouldn't be packed with the ones above
        big = c.open("bigfile.txt", "wb")
        big.write(b"x" * 1024 * 1024 * 33) # 33 MB > KEEP_BLOCK_SIZE/2
        big.close(flush=False)
        self.assertEqual(
            c.manifest_text("."),
            '. 2d303c138c118af809f39319e5d507e9+34603008 a8430a058b8fbf408e1931b794dbd6fb+13 0:34603008:bigfile.txt 34603008:10:count.txt 34603018:3:foo.txt\n')

    def test_flush_after_small_block_packing(self):
        c = Collection()
        # Write a couple of small files,
        f = c.open("count.txt", "wb")
        f.write(b"0123456789")
        f.close(flush=False)
        foo = c.open("foo.txt", "wb")
        foo.write(b"foo")
        foo.close(flush=False)

        self.assertEqual(
            c.manifest_text(),
            '. a8430a058b8fbf408e1931b794dbd6fb+13 0:10:count.txt 10:3:foo.txt\n')

        f = c.open("count.txt", "rb+")
        f.close(flush=True)

        self.assertEqual(
            c.manifest_text(),
            '. a8430a058b8fbf408e1931b794dbd6fb+13 0:10:count.txt 10:3:foo.txt\n')

    def test_write_after_small_block_packing2(self):
        c = Collection()
        # Write a couple of small files,
        f = c.open("count.txt", "wb")
        f.write(b"0123456789")
        f.close(flush=False)
        foo = c.open("foo.txt", "wb")
        foo.write(b"foo")
        foo.close(flush=False)

        self.assertEqual(
            c.manifest_text(),
            '. a8430a058b8fbf408e1931b794dbd6fb+13 0:10:count.txt 10:3:foo.txt\n')

        f = c.open("count.txt", "rb+")
        f.write(b"abc")
        f.close(flush=False)

        self.assertEqual(
            c.manifest_text(),
            '. 900150983cd24fb0d6963f7d28e17f72+3 a8430a058b8fbf408e1931b794dbd6fb+13 0:3:count.txt 6:7:count.txt 13:3:foo.txt\n')


    def test_small_block_packing_with_overwrite(self):
        c = Collection()
        c.open("b1", "wb").close()
        c["b1"].writeto(0, b"b1", 0)

        c.open("b2", "wb").close()
        c["b2"].writeto(0, b"b2", 0)

        c["b1"].writeto(0, b"1b", 0)

        self.assertEqual(c.manifest_text(), ". ed4f3f67c70b02b29c50ce1ea26666bd+4 0:2:b1 2:2:b2\n")
        self.assertEqual(c["b1"].manifest_text(), ". ed4f3f67c70b02b29c50ce1ea26666bd+4 0:2:b1\n")
        self.assertEqual(c["b2"].manifest_text(), ". ed4f3f67c70b02b29c50ce1ea26666bd+4 2:2:b2\n")


class CollectionCreateUpdateTest(run_test_server.TestCaseWithServers):
    MAIN_SERVER = {}
    KEEP_SERVER = {}

    def create_count_txt(self):
        # Create an empty collection, save it to the API server, then write a
        # file, but don't save it.

        c = Collection()
        c.save_new("CollectionCreateUpdateTest", ensure_unique_name=True)
        self.assertEqual(c.portable_data_hash(), "d41d8cd98f00b204e9800998ecf8427e+0")
        self.assertEqual(c.api_response()["portable_data_hash"], "d41d8cd98f00b204e9800998ecf8427e+0" )

        with c.open("count.txt", "wb") as f:
            f.write(b"0123456789")

        self.assertEqual(c.portable_manifest_text(), ". 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt\n")

        return c

    def test_create_and_save(self):
        c = self.create_count_txt()
        c.save(properties={'type' : 'Intermediate'},
               storage_classes=['archive'],
               trash_at=datetime.datetime(2111, 1, 1, 11, 11, 11, 111111))

        self.assertRegex(
            c.manifest_text(),
            r"^\. 781e5e245d69b566979b86e28d23f2c7\+10\+A[a-f0-9]{40}@[a-f0-9]{8} 0:10:count\.txt$",)
        self.assertEqual(c.api_response()["storage_classes_desired"], ['archive'])
        self.assertEqual(c.api_response()["properties"], {'type' : 'Intermediate'})
        self.assertEqual(c.api_response()["trash_at"], '2111-01-01T11:11:11.111111000Z')


    def test_create_and_save_new(self):
        c = self.create_count_txt()
        c.save_new(properties={'type' : 'Intermediate'},
                   storage_classes=['archive'],
                   trash_at=datetime.datetime(2111, 1, 1, 11, 11, 11, 111111))

        self.assertRegex(
            c.manifest_text(),
            r"^\. 781e5e245d69b566979b86e28d23f2c7\+10\+A[a-f0-9]{40}@[a-f0-9]{8} 0:10:count\.txt$",)
        self.assertEqual(c.api_response()["storage_classes_desired"], ['archive'])
        self.assertEqual(c.api_response()["properties"], {'type' : 'Intermediate'})
        self.assertEqual(c.api_response()["trash_at"], '2111-01-01T11:11:11.111111000Z')

    def test_create_and_save_after_commiting(self):
        c = self.create_count_txt()
        c.save(properties={'type' : 'Intermediate'},
               storage_classes=['hot'],
               trash_at=datetime.datetime(2111, 1, 1, 11, 11, 11, 111111))
        c.save(properties={'type' : 'Output'},
               storage_classes=['cold'],
               trash_at=datetime.datetime(2222, 2, 2, 22, 22, 22, 222222))

        self.assertEqual(c.api_response()["storage_classes_desired"], ['cold'])
        self.assertEqual(c.api_response()["properties"], {'type' : 'Output'})
        self.assertEqual(c.api_response()["trash_at"], '2222-02-02T22:22:22.222222000Z')

    def test_create_diff_apply(self):
        c1 = self.create_count_txt()
        c1.save()

        c2 = Collection(c1.manifest_locator())
        with c2.open("count.txt", "wb") as f:
            f.write(b"abcdefg")

        diff = c1.diff(c2)

        self.assertEqual(diff[0], (arvados.collection.MOD, u'./count.txt', c1["count.txt"], c2["count.txt"]))

        c1.apply(diff)
        self.assertEqual(c1.portable_data_hash(), c2.portable_data_hash())

    def test_diff_apply_with_token(self):
        baseline = CollectionReader(". 781e5e245d69b566979b86e28d23f2c7+10+A715fd31f8111894f717eb1003c1b0216799dd9ec@54f5dd1a 0:10:count.txt\n")
        c = Collection(". 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt\n")
        other = CollectionReader(". 7ac66c0f148de9519b8bd264312c4d64+7+A715fd31f8111894f717eb1003c1b0216799dd9ec@54f5dd1a 0:7:count.txt\n")

        diff = baseline.diff(other)
        self.assertEqual(diff, [('mod', u'./count.txt', c["count.txt"], other["count.txt"])])

        c.apply(diff)

        self.assertEqual(c.manifest_text(), ". 7ac66c0f148de9519b8bd264312c4d64+7+A715fd31f8111894f717eb1003c1b0216799dd9ec@54f5dd1a 0:7:count.txt\n")


    def test_create_and_update(self):
        c1 = self.create_count_txt()
        c1.save()

        c2 = arvados.collection.Collection(c1.manifest_locator())
        with c2.open("count.txt", "wb") as f:
            f.write(b"abcdefg")

        c2.save()

        self.assertNotEqual(c1.portable_data_hash(), c2.portable_data_hash())
        c1.update()
        self.assertEqual(c1.portable_data_hash(), c2.portable_data_hash())


    def test_create_and_update_with_conflict(self):
        c1 = self.create_count_txt()
        c1.save()

        with c1.open("count.txt", "wb") as f:
            f.write(b"XYZ")

        c2 = arvados.collection.Collection(c1.manifest_locator())
        with c2.open("count.txt", "wb") as f:
            f.write(b"abcdefg")

        c2.save()

        c1.update()
        self.assertRegex(
            c1.manifest_text(),
            r"\. e65075d550f9b5bf9992fa1d71a131be\+3\S* 7ac66c0f148de9519b8bd264312c4d64\+7\S* 0:3:count\.txt 3:7:count\.txt~\d\d\d\d\d\d\d\d-\d\d\d\d\d\d~conflict~$")

    def test_pdh_is_native_str(self):
        c1 = self.create_count_txt()
        pdh = c1.portable_data_hash()
        self.assertEqual(type(''), type(pdh))


if __name__ == '__main__':
    unittest.main()
