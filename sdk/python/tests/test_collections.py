# usage example:
#
# ARVADOS_API_TOKEN=abc ARVADOS_API_HOST=arvados.local python -m unittest discover

import arvados
import copy
import hashlib
import mock
import os
import pprint
import re
import tempfile
import unittest

import run_test_server
from arvados._ranges import Range, LocatorAndRange
from arvados.collection import Collection, CollectionReader
import arvados_testutil as tutil

class TestResumableWriter(arvados.ResumableCollectionWriter):
    KEEP_BLOCK_SIZE = 1024  # PUT to Keep every 1K.

    def current_state(self):
        return self.dump_state(copy.deepcopy)


class ArvadosCollectionsTest(run_test_server.TestCaseWithServers,
                             tutil.ArvadosBaseTestCase):
    MAIN_SERVER = {}

    @classmethod
    def setUpClass(cls):
        super(ArvadosCollectionsTest, cls).setUpClass()
        run_test_server.authorize_with('active')
        cls.api_client = arvados.api('v1')
        cls.keep_client = arvados.KeepClient(api_client=cls.api_client,
                                             local_store=cls.local_store)

    def write_foo_bar_baz(self):
        cw = arvados.CollectionWriter(self.api_client)
        self.assertEqual(cw.current_stream_name(), '.',
                         'current_stream_name() should be "." now')
        cw.set_current_file_name('foo.txt')
        cw.write('foo')
        self.assertEqual(cw.current_file_name(), 'foo.txt',
                         'current_file_name() should be foo.txt now')
        cw.start_new_file('bar.txt')
        cw.write('bar')
        cw.start_new_stream('baz')
        cw.write('baz')
        cw.set_current_file_name('baz.txt')
        self.assertEqual(cw.manifest_text(),
                         ". 3858f62230ac3c915f300c664312c63f+6 0:3:foo.txt 3:3:bar.txt\n" +
                         "./baz 73feffa4b7f6bb68e44cf984c85f6e88+3 0:3:baz.txt\n",
                         "wrong manifest: got {}".format(cw.manifest_text()))
        cw.finish()
        return cw.portable_data_hash()

    def test_keep_local_store(self):
        self.assertEqual(self.keep_client.put('foo'), 'acbd18db4cc2f85cedef654fccc4a4d8+3', 'wrong md5 hash from Keep.put')
        self.assertEqual(self.keep_client.get('acbd18db4cc2f85cedef654fccc4a4d8+3'), 'foo', 'wrong data from Keep.get')

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
        expected = [[3, '.', 'foo.txt', 'foo'],
                    [3, '.', 'bar.txt', 'bar'],
                    [3, './baz', 'baz.txt', 'baz']]
        self.assertEqual(got,
                         expected)
        stream0 = cr.all_streams()[0]
        self.assertEqual(stream0.readfrom(0, 0),
                         '',
                         'reading zero bytes should have returned empty string')
        self.assertEqual(stream0.readfrom(0, 2**26),
                         'foobar',
                         'reading entire stream failed')
        self.assertEqual(stream0.readfrom(2**26, 0),
                         '',
                         'reading zero bytes should have returned empty string')

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
                          [[3, '.',     'bar.txt', 'bar'],
                           [3, '.',     'foo.txt', 'foo'],
                           [3, './baz', 'baz.txt', 'baz']])
        self._test_subset((". %s %s 0:3:foo.txt 3:3:bar.txt\n" %
                           (self.keep_client.put("foo"),
                            self.keep_client.put("bar"))),
                          [[3, '.', 'bar.txt', 'bar'],
                           [3, '.', 'foo.txt', 'foo']])
        self._test_subset((". %s %s 0:2:fo.txt 2:4:obar.txt\n" %
                           (self.keep_client.put("foo"),
                            self.keep_client.put("bar"))),
                          [[2, '.', 'fo.txt', 'fo'],
                           [4, '.', 'obar.txt', 'obar']])
        self._test_subset((". %s %s 0:2:fo.txt 2:0:zero.txt 2:2:ob.txt 4:2:ar.txt\n" %
                           (self.keep_client.put("foo"),
                            self.keep_client.put("bar"))),
                          [[2, '.', 'ar.txt', 'ar'],
                           [2, '.', 'fo.txt', 'fo'],
                           [2, '.', 'ob.txt', 'ob'],
                           [0, '.', 'zero.txt', '']])

    def test_collection_empty_file(self):
        cw = arvados.CollectionWriter(self.api_client)
        cw.start_new_file('zero.txt')
        cw.write('')

        self.assertEqual(cw.manifest_text(), ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:zero.txt\n")
        self.check_manifest_file_sizes(cw.manifest_text(), [0])
        cw = arvados.CollectionWriter(self.api_client)
        cw.start_new_file('zero.txt')
        cw.write('')
        cw.start_new_file('one.txt')
        cw.write('1')
        cw.start_new_stream('foo')
        cw.start_new_file('zero.txt')
        cw.write('')
        self.check_manifest_file_sizes(cw.manifest_text(), [0,1,0])

    def test_no_implicit_normalize(self):
        cw = arvados.CollectionWriter(self.api_client)
        cw.start_new_file('b')
        cw.write('b')
        cw.start_new_file('a')
        cw.write('')
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

        def get(self, locator, num_retries=0):
            return self.content[locator]

    def test_stream_reader(self):
        keepblocks = {'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa+10': 'abcdefghij',
                      'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb+15': 'klmnopqrstuvwxy',
                      'cccccccccccccccccccccccccccccccc+5': 'z0123'}
        mk = self.MockKeep(keepblocks)

        sr = arvados.StreamReader([".", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa+10", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb+15", "cccccccccccccccccccccccccccccccc+5", "0:30:foo"], mk)

        content = 'abcdefghijklmnopqrstuvwxyz0123456789'

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
        self.assertEqual(sr.readfrom(30, 5), '')

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
        blockhash = hashlib.md5(content).hexdigest() + '+' + str(len(content))
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
            with self.make_test_file(letter) as testfile:
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
            testfile.write('extra')
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

    def test_read_arbitrary_data_with_collection_reader(self):
        # arv-get relies on this to do "arv-get {keep-locator} -".
        self.write_foo_bar_baz()
        self.assertEqual(
            'foobar',
            arvados.CollectionReader(
                '3858f62230ac3c915f300c664312c63f+6'
                ).manifest_text())


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
    def mock_get_collection(self, api_mock, code, body):
        body = self.API_COLLECTIONS.get(body)
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
        with tutil.mock_get_responses(None, 404):
            reader = arvados.CollectionReader(self.DEFAULT_DATA_HASH,
                                              api_client=client)
            self.assertEqual(self.DEFAULT_MANIFEST, reader.manifest_text())

    def test_locator_init_fallback_to_keep(self):
        # crunch-job needs this to read manifests that have only ever
        # been written to Keep.
        client = self.api_client_mock(200)
        self.mock_get_collection(client, 404, None)
        with tutil.mock_get_responses(self.DEFAULT_MANIFEST, 200):
            reader = arvados.CollectionReader(self.DEFAULT_DATA_HASH,
                                              api_client=client)
            self.assertEqual(self.DEFAULT_MANIFEST, reader.manifest_text())

    def test_uuid_init_no_fallback_to_keep(self):
        # Do not look up a collection UUID in Keep.
        client = self.api_client_mock(404)
        with tutil.mock_get_responses(self.DEFAULT_MANIFEST, 200):
            with self.assertRaises(arvados.errors.ApiError):
                reader = arvados.CollectionReader(self.DEFAULT_UUID,
                                                  api_client=client)

    def test_try_keep_first_if_permission_hint(self):
        # To verify that CollectionReader tries Keep first here, we
        # mock API server to return the wrong data.
        client = self.api_client_mock(200)
        with tutil.mock_get_responses(self.ALT_MANIFEST, 200):
            self.assertEqual(
                self.ALT_MANIFEST,
                arvados.CollectionReader(
                    self.ALT_DATA_HASH + '+Affffffffffffffffffffffffffffffffffffffff@fedcba98',
                    api_client=client).manifest_text())

    def test_init_num_retries_propagated(self):
        # More of an integration test...
        client = self.api_client_mock(200)
        reader = arvados.CollectionReader(self.DEFAULT_UUID, api_client=client,
                                          num_retries=3)
        with tutil.mock_get_responses('foo', 500, 500, 200):
            self.assertEqual('foo',
                             ''.join(f.read(9) for f in reader.all_files()))

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

    def test_api_response(self):
        client = self.api_client_mock()
        reader = arvados.CollectionReader(self.DEFAULT_UUID, api_client=client)
        self.assertEqual(self.DEFAULT_COLLECTION, reader.api_response())

    def test_api_response_with_collection_from_keep(self):
        client = self.api_client_mock()
        self.mock_get_collection(client, 404, 'foo')
        with tutil.mock_get_responses(self.DEFAULT_MANIFEST, 200):
            reader = arvados.CollectionReader(self.DEFAULT_DATA_HASH,
                                              api_client=client)
            api_response = reader.api_response()
        self.assertIsNone(api_response)

    def check_open_file(self, coll_file, stream_name, file_name, file_size):
        self.assertFalse(coll_file.closed, "returned file is not open")
        self.assertEqual(stream_name, coll_file.stream_name())
        self.assertEqual(file_name, coll_file.name)
        self.assertEqual(file_size, coll_file.size())

    def test_open_collection_file_one_argument(self):
        client = self.api_client_mock(200)
        reader = arvados.CollectionReader(self.DEFAULT_UUID, api_client=client)
        cfile = reader.open('./foo')
        self.check_open_file(cfile, '.', 'foo', 3)

    def test_open_deep_file(self):
        coll_name = 'collection_with_files_in_subdir'
        client = self.api_client_mock(200)
        self.mock_get_collection(client, 200, coll_name)
        reader = arvados.CollectionReader(
            self.API_COLLECTIONS[coll_name]['uuid'], api_client=client)
        cfile = reader.open('./subdir2/subdir3/file2_in_subdir3.txt')
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
        return tutil.mock_put_responses(body, *codes, **headers)

    def foo_writer(self, **kwargs):
        kwargs.setdefault('api_client', self.api_client_mock())
        writer = arvados.CollectionWriter(**kwargs)
        writer.start_new_file('foo')
        writer.write('foo')
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
        with self.mock_keep(None, 200, headers={'x-keep-replicas-stored': 2}):
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
                None, 500, 500, 500, 200, 200, 200,
                **{'x-keep-replicas-stored': 1}) as keepmock:
            self.mock_keep_services(client, status=200, service_type='disk', count=6)
            writer = self.foo_writer(api_client=client, replication=3)
            writer.manifest_text()
            # keepmock is the mock session constructor; keepmock.return_value
            # is the mock session object, and keepmock.return_value.put is the
            # actual mock method of interest.
            self.assertEqual(6, keepmock.return_value.put.call_count)

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
            out_file.write('test data')
            data_loc = hashlib.md5('test data').hexdigest() + '+9'
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
            data_loc = hashlib.md5('123456').hexdigest() + '+6'
        with self.mock_keep(data_loc, 200) as keep_mock:
            self.assertEqual(". {} 0:6:six\n".format(data_loc),
                             writer.manifest_text())

    def test_open_flush(self):
        client = self.api_client_mock()
        data_loc1 = hashlib.md5('flush1').hexdigest() + '+6'
        data_loc2 = hashlib.md5('flush2').hexdigest() + '+6'
        with self.mock_keep((data_loc1, 200), (data_loc2, 200)) as keep_mock:
            writer = arvados.CollectionWriter(client)
            with writer.open('flush_test') as out_file:
                out_file.write('flush1')
                out_file.flush()
                out_file.write('flush2')
            self.assertEqual(". {} {} 0:12:flush_test\n".format(data_loc1,
                                                                data_loc2),
                             writer.manifest_text())

    def test_two_opens_same_stream(self):
        client = self.api_client_mock()
        writer = arvados.CollectionWriter(client)
        with writer.open('.', '1') as out_file:
            out_file.write('1st')
        with writer.open('.', '2') as out_file:
            out_file.write('2nd')
        data_loc = hashlib.md5('1st2nd').hexdigest() + '+6'
        with self.mock_keep(data_loc, 200) as keep_mock:
            self.assertEqual(". {} 0:3:1 3:3:2\n".format(data_loc),
                             writer.manifest_text())

    def test_two_opens_two_streams(self):
        client = self.api_client_mock()
        data_loc1 = hashlib.md5('file').hexdigest() + '+4'
        data_loc2 = hashlib.md5('indir').hexdigest() + '+5'
        with self.mock_keep((data_loc1, 200), (data_loc2, 200)) as keep_mock:
            writer = arvados.CollectionWriter(client)
            with writer.open('file') as out_file:
                out_file.write('file')
            with writer.open('./dir', 'indir') as out_file:
                out_file.write('indir')
            expected = ". {} 0:4:file\n./dir {} 0:5:indir\n".format(
                data_loc1, data_loc2)
            self.assertEqual(expected, writer.manifest_text())

    def test_dup_open_fails(self):
        client = self.api_client_mock()
        writer = arvados.CollectionWriter(client)
        file1 = writer.open('one')
        self.assertRaises(arvados.errors.AssertionError, writer.open, 'two')


class NewCollectionTestCase(unittest.TestCase, CollectionTestMixin):

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

    def test_find(self):
        c = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt 0:10:count2.txt\n')
        self.assertIs(c.find("."), c)
        self.assertIs(c.find("./count1.txt"), c["count1.txt"])
        self.assertIs(c.find("count1.txt"), c["count1.txt"])
        with self.assertRaises(IOError):
            c.find("/.")
        with self.assertRaises(arvados.errors.ArgumentError):
            c.find("")

    def test_remove_in_subdir(self):
        c = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n./foo 781e5e245d69b566979b86e28d23f2c7+10 0:10:count2.txt\n')
        c.remove("foo/count2.txt")
        self.assertEqual(". 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n", c.portable_manifest_text())

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
        self.assertEqual(d, [('del', './count2.txt', c2["count2.txt"]),
                             ('add', './count1.txt', c1["count1.txt"])])
        d = c1.diff(c2)
        self.assertEqual(d, [('del', './count1.txt', c1["count1.txt"]),
                             ('add', './count2.txt', c2["count2.txt"])])
        self.assertNotEqual(c1.portable_manifest_text(), c2.portable_manifest_text())
        c1.apply(d)
        self.assertEqual(c1.portable_manifest_text(), c2.portable_manifest_text())

    def test_diff_same(self):
        c1 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n')
        c2 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n')
        d = c2.diff(c1)
        self.assertEqual(d, [])
        d = c1.diff(c2)
        self.assertEqual(d, [])

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
        self.assertEqual(d, [('del', './count2.txt', c2["count2.txt"])])
        d = c1.diff(c2)
        self.assertEqual(d, [('add', './count2.txt', c2["count2.txt"])])

        self.assertNotEqual(c1.portable_manifest_text(), c2.portable_manifest_text())
        c1.apply(d)
        self.assertEqual(c1.portable_manifest_text(), c2.portable_manifest_text())

    def test_diff_add_in_subcollection(self):
        c1 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n')
        c2 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n./foo 5348b82a029fd9e971a811ce1f71360b+43 0:10:count2.txt\n')
        d = c2.diff(c1)
        self.assertEqual(d, [('del', './foo', c2["foo"])])
        d = c1.diff(c2)
        self.assertEqual(d, [('add', './foo', c2["foo"])])

        self.assertNotEqual(c1.portable_manifest_text(), c2.portable_manifest_text())
        c1.apply(d)
        self.assertEqual(c1.portable_manifest_text(), c2.portable_manifest_text())

    def test_diff_del_add_in_subcollection(self):
        c1 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n./foo 5348b82a029fd9e971a811ce1f71360b+43 0:10:count2.txt\n')
        c2 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n./foo 5348b82a029fd9e971a811ce1f71360b+43 0:3:count3.txt\n')

        d = c2.diff(c1)
        self.assertEqual(d, [('del', './foo/count3.txt', c2.find("foo/count3.txt")),
                             ('add', './foo/count2.txt', c1.find("foo/count2.txt"))])
        d = c1.diff(c2)
        self.assertEqual(d, [('del', './foo/count2.txt', c1.find("foo/count2.txt")),
                             ('add', './foo/count3.txt', c2.find("foo/count3.txt"))])

        self.assertNotEqual(c1.portable_manifest_text(), c2.portable_manifest_text())
        c1.apply(d)
        self.assertEqual(c1.portable_manifest_text(), c2.portable_manifest_text())

    def test_diff_mod_in_subcollection(self):
        c1 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n./foo 5348b82a029fd9e971a811ce1f71360b+43 0:10:count2.txt\n')
        c2 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt 0:3:foo\n')
        d = c2.diff(c1)
        self.assertEqual(d, [('mod', './foo', c2["foo"], c1["foo"])])
        d = c1.diff(c2)
        self.assertEqual(d, [('mod', './foo', c1["foo"], c2["foo"])])

        self.assertNotEqual(c1.portable_manifest_text(), c2.portable_manifest_text())
        c1.apply(d)
        self.assertEqual(c1.portable_manifest_text(), c2.portable_manifest_text())

    def test_conflict_keep_local_change(self):
        c1 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n')
        c2 = Collection('. 5348b82a029fd9e971a811ce1f71360b+43 0:10:count2.txt\n')
        d = c1.diff(c2)
        self.assertEqual(d, [('del', './count1.txt', c1["count1.txt"]),
                             ('add', './count2.txt', c2["count2.txt"])])
        f = c1.open("count1.txt", "w")
        f.write("zzzzz")

        # c1 changed, so it should not be deleted.
        c1.apply(d)
        self.assertEqual(c1.portable_manifest_text(), ". 95ebc3c7b3b9f1d2c40fec14415d3cb8+5 5348b82a029fd9e971a811ce1f71360b+43 0:5:count1.txt 5:10:count2.txt\n")

    def test_conflict_mod(self):
        c1 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt')
        c2 = Collection('. 5348b82a029fd9e971a811ce1f71360b+43 0:10:count1.txt')
        d = c1.diff(c2)
        self.assertEqual(d, [('mod', './count1.txt', c1["count1.txt"], c2["count1.txt"])])
        f = c1.open("count1.txt", "w")
        f.write("zzzzz")

        # c1 changed, so c2 mod will go to a conflict file
        c1.apply(d)
        self.assertRegexpMatches(c1.portable_manifest_text(), r"\. 95ebc3c7b3b9f1d2c40fec14415d3cb8\+5 5348b82a029fd9e971a811ce1f71360b\+43 0:5:count1\.txt 5:10:count1\.txt~conflict-\d\d\d\d-\d\d-\d\d-\d\d:\d\d:\d\d~$")

    def test_conflict_add(self):
        c1 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count2.txt\n')
        c2 = Collection('. 5348b82a029fd9e971a811ce1f71360b+43 0:10:count1.txt\n')
        d = c1.diff(c2)
        self.assertEqual(d, [('del', './count2.txt', c1["count2.txt"]),
                             ('add', './count1.txt', c2["count1.txt"])])
        f = c1.open("count1.txt", "w")
        f.write("zzzzz")

        # c1 added count1.txt, so c2 add will go to a conflict file
        c1.apply(d)
        self.assertRegexpMatches(c1.portable_manifest_text(), r"\. 95ebc3c7b3b9f1d2c40fec14415d3cb8\+5 5348b82a029fd9e971a811ce1f71360b\+43 0:5:count1\.txt 5:10:count1\.txt~conflict-\d\d\d\d-\d\d-\d\d-\d\d:\d\d:\d\d~$")

    def test_conflict_del(self):
        c1 = Collection('. 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt')
        c2 = Collection('. 5348b82a029fd9e971a811ce1f71360b+43 0:10:count1.txt')
        d = c1.diff(c2)
        self.assertEqual(d, [('mod', './count1.txt', c1["count1.txt"], c2["count1.txt"])])
        c1.remove("count1.txt")

        # c1 deleted, so c2 mod will go to a conflict file
        c1.apply(d)
        self.assertRegexpMatches(c1.portable_manifest_text(), r"\. 5348b82a029fd9e971a811ce1f71360b\+43 0:10:count1\.txt~conflict-\d\d\d\d-\d\d-\d\d-\d\d:\d\d:\d\d~$")

    def test_notify(self):
        c1 = Collection()
        events = []
        c1.subscribe(lambda event, collection, name, item: events.append((event, collection, name, item)))
        f = c1.open("foo.txt", "w")
        self.assertEqual(events[0], (arvados.collection.ADD, c1, "foo.txt", f.arvadosfile))

    def test_open_w(self):
        c1 = Collection(". 781e5e245d69b566979b86e28d23f2c7+10 0:10:count1.txt\n")
        self.assertEqual(c1["count1.txt"].size(), 10)
        c1.open("count1.txt", "w").close()
        self.assertEqual(c1["count1.txt"].size(), 0)


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

        with c.open("count.txt", "w") as f:
            f.write("0123456789")

        self.assertEqual(c.portable_manifest_text(), ". 781e5e245d69b566979b86e28d23f2c7+10 0:10:count.txt\n")

        return c

    def test_create_and_save(self):
        c = self.create_count_txt()
        c.save()
        self.assertRegexpMatches(c.manifest_text(), r"^\. 781e5e245d69b566979b86e28d23f2c7\+10\+A[a-f0-9]{40}@[a-f0-9]{8} 0:10:count\.txt$",)

    def test_create_and_save_new(self):
        c = self.create_count_txt()
        c.save_new()
        self.assertRegexpMatches(c.manifest_text(), r"^\. 781e5e245d69b566979b86e28d23f2c7\+10\+A[a-f0-9]{40}@[a-f0-9]{8} 0:10:count\.txt$",)

    def test_create_diff_apply(self):
        c1 = self.create_count_txt()
        c1.save()

        c2 = Collection(c1.manifest_locator())
        with c2.open("count.txt", "w") as f:
            f.write("abcdefg")

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
        with c2.open("count.txt", "w") as f:
            f.write("abcdefg")

        c2.save()

        self.assertNotEqual(c1.portable_data_hash(), c2.portable_data_hash())
        c1.update()
        self.assertEqual(c1.portable_data_hash(), c2.portable_data_hash())


    def test_create_and_update_with_conflict(self):
        c1 = self.create_count_txt()
        c1.save()

        with c1.open("count.txt", "w") as f:
            f.write("XYZ")

        c2 = arvados.collection.Collection(c1.manifest_locator())
        with c2.open("count.txt", "w") as f:
            f.write("abcdefg")

        c2.save()

        c1.update()
        self.assertRegexpMatches(c1.manifest_text(), r"\. e65075d550f9b5bf9992fa1d71a131be\+3 7ac66c0f148de9519b8bd264312c4d64\+7\+A[a-f0-9]{40}@[a-f0-9]{8} 0:3:count\.txt 3:7:count\.txt~conflict-\d\d\d\d-\d\d-\d\d-\d\d:\d\d:\d\d~$")


if __name__ == '__main__':
    unittest.main()
