# usage example:
#
# ARVADOS_API_TOKEN=abc ARVADOS_API_HOST=arvados.local python -m unittest discover

import arvados
import bz2
import copy
import os
import pprint
import subprocess
import tempfile
import unittest

import run_test_server
from arvados_testutil import ArvadosBaseTestCase

class TestResumableWriter(arvados.ResumableCollectionWriter):
    KEEP_BLOCK_SIZE = 1024  # PUT to Keep every 1K.

    def current_state(self):
        return self.dump_state(copy.deepcopy)


class ArvadosCollectionsTest(run_test_server.TestCaseWithServers,
                             ArvadosBaseTestCase):
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
        return cw.finish()

    def test_keep_local_store(self):
        self.assertEqual(self.keep_client.put('foo'), 'acbd18db4cc2f85cedef654fccc4a4d8+3', 'wrong md5 hash from Keep.put')
        self.assertEqual(self.keep_client.get('acbd18db4cc2f85cedef654fccc4a4d8+3'), 'foo', 'wrong data from Keep.get')

    def test_local_collection_writer(self):
        self.assertEqual(self.write_foo_bar_baz(),
                         'd6c3b8e571f1b81ebb150a45ed06c884+114',
                         "wrong locator hash for files foo, bar, baz")

    def test_local_collection_reader(self):
        self.write_foo_bar_baz()
        cr = arvados.CollectionReader(
            'd6c3b8e571f1b81ebb150a45ed06c884+114+Xzizzle', self.api_client)
        got = []
        for s in cr.all_streams():
            for f in s.all_files():
                got += [[f.size(), f.stream_name(), f.name(), f.read(2**26)]]
        expected = [[3, '.', 'bar.txt', 'bar'],
                    [3, '.', 'foo.txt', 'foo'],
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
        self.write_foo_bar_baz()
        self._test_subset('d6c3b8e571f1b81ebb150a45ed06c884+114',
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

    def _test_readline(self, what_in, what_out):
        cw = arvados.CollectionWriter(self.api_client)
        cw.start_new_file('test.txt')
        cw.write(what_in)
        test1 = cw.finish()
        cr = arvados.CollectionReader(test1, self.api_client)
        got = []
        for x in list(cr.all_files())[0].readlines():
            got += [x]
        self.assertEqual(got,
                         what_out,
                         "readlines did not split lines correctly: %s" % got)

    def test_collection_readline(self):
        self._test_readline("\na\nbcd\n\nefg\nz",
                            ["\n", "a\n", "bcd\n", "\n", "efg\n", "z"])
        self._test_readline("ab\ncd\n",
                            ["ab\n", "cd\n"])

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
        self.check_manifest_file_sizes(cw.manifest_text(), [1,0,0])

    def check_manifest_file_sizes(self, manifest_text, expect_sizes):
        cr = arvados.CollectionReader(manifest_text, self.api_client)
        got_sizes = []
        for f in cr.all_files():
            got_sizes += [f.size()]
        self.assertEqual(got_sizes, expect_sizes, "got wrong file sizes %s, expected %s" % (got_sizes, expect_sizes))

    def test_collection_bz2_decompression(self):
        n_lines_in = 2**18
        data_in = "abc\n"
        for x in xrange(0, 18):
            data_in += data_in
        compressed_data_in = bz2.compress(data_in)
        cw = arvados.CollectionWriter(self.api_client)
        cw.start_new_file('test.bz2')
        cw.write(compressed_data_in)
        bz2_manifest = cw.manifest_text()

        cr = arvados.CollectionReader(bz2_manifest, self.api_client)

        got = 0
        for x in list(cr.all_files())[0].readlines():
            self.assertEqual(x, "abc\n", "decompression returned wrong data: %s" % x)
            got += 1
        self.assertEqual(got,
                         n_lines_in,
                         "decompression returned %d lines instead of %d" % (got, n_lines_in))

    def test_collection_gzip_decompression(self):
        n_lines_in = 2**18
        data_in = "abc\n"
        for x in xrange(0, 18):
            data_in += data_in
        p = subprocess.Popen(["gzip", "-1cn"],
                             stdout=subprocess.PIPE,
                             stdin=subprocess.PIPE,
                             stderr=subprocess.PIPE,
                             shell=False, close_fds=True)
        compressed_data_in, stderrdata = p.communicate(data_in)

        cw = arvados.CollectionWriter(self.api_client)
        cw.start_new_file('test.gz')
        cw.write(compressed_data_in)
        gzip_manifest = cw.manifest_text()

        cr = arvados.CollectionReader(gzip_manifest, self.api_client)
        got = 0
        for x in list(cr.all_files())[0].readlines():
            self.assertEqual(x, "abc\n", "decompression returned wrong data: %s" % x)
            got += 1
        self.assertEqual(got,
                         n_lines_in,
                         "decompression returned %d lines instead of %d" % (got, n_lines_in))

    def test_normalized_collection(self):
        m1 = """. 5348b82a029fd9e971a811ce1f71360b+43 0:43:md5sum.txt
. 085c37f02916da1cad16f93c54d899b7+41 0:41:md5sum.txt
. 8b22da26f9f433dea0a10e5ec66d73ba+43 0:43:md5sum.txt"""
        self.assertEqual(arvados.CollectionReader(m1, self.api_client).manifest_text(),
                         """. 5348b82a029fd9e971a811ce1f71360b+43 085c37f02916da1cad16f93c54d899b7+41 8b22da26f9f433dea0a10e5ec66d73ba+43 0:127:md5sum.txt
""")

        m2 = """. 204e43b8a1185621ca55a94839582e6f+67108864 b9677abbac956bd3e86b1deb28dfac03+67108864 fc15aff2a762b13f521baf042140acec+67108864 323d2a3ce20370c4ca1d3462a344f8fd+25885655 0:227212247:var-GS000016015-ASM.tsv.bz2
"""
        self.assertEqual(arvados.CollectionReader(m2, self.api_client).manifest_text(), m2)

        m3 = """. 5348b82a029fd9e971a811ce1f71360b+43 3:40:md5sum.txt
. 085c37f02916da1cad16f93c54d899b7+41 0:41:md5sum.txt
. 8b22da26f9f433dea0a10e5ec66d73ba+43 0:43:md5sum.txt"""
        self.assertEqual(arvados.CollectionReader(m3, self.api_client).manifest_text(),
                         """. 5348b82a029fd9e971a811ce1f71360b+43 085c37f02916da1cad16f93c54d899b7+41 8b22da26f9f433dea0a10e5ec66d73ba+43 3:124:md5sum.txt
""")

        m4 = """. 204e43b8a1185621ca55a94839582e6f+67108864 0:3:foo/bar
./zzz 204e43b8a1185621ca55a94839582e6f+67108864 0:999:zzz
./foo 323d2a3ce20370c4ca1d3462a344f8fd+25885655 0:3:bar"""
        self.assertEqual(arvados.CollectionReader(m4, self.api_client).manifest_text(),
                         """./foo 204e43b8a1185621ca55a94839582e6f+67108864 323d2a3ce20370c4ca1d3462a344f8fd+25885655 0:3:bar 67108864:3:bar
./zzz 204e43b8a1185621ca55a94839582e6f+67108864 0:999:zzz
""")

        m5 = """. 204e43b8a1185621ca55a94839582e6f+67108864 0:3:foo/bar
./zzz 204e43b8a1185621ca55a94839582e6f+67108864 0:999:zzz
./foo 204e43b8a1185621ca55a94839582e6f+67108864 3:3:bar"""
        self.assertEqual(arvados.CollectionReader(m5, self.api_client).manifest_text(),
                         """./foo 204e43b8a1185621ca55a94839582e6f+67108864 0:6:bar
./zzz 204e43b8a1185621ca55a94839582e6f+67108864 0:999:zzz
""")

        with self.data_file('1000G_ref_manifest') as f6:
            m6 = f6.read()
            self.assertEqual(arvados.CollectionReader(m6, self.api_client).manifest_text(), m6)

        with self.data_file('jlake_manifest') as f7:
            m7 = f7.read()
            self.assertEqual(arvados.CollectionReader(m7, self.api_client).manifest_text(), m7)

        m8 = """./a\\040b\\040c 59ca0efa9f5633cb0371bbc0355478d8+13 0:13:hello\\040world.txt
"""
        self.assertEqual(arvados.CollectionReader(m8, self.api_client).manifest_text(), m8)

    def test_locators_and_ranges(self):
        blocks2 = [['a', 10, 0],
                  ['b', 10, 10],
                  ['c', 10, 20],
                  ['d', 10, 30],
                  ['e', 10, 40],
                  ['f', 10, 50]]

        self.assertEqual(arvados.locators_and_ranges(blocks2,  2,  2), [['a', 10, 2, 2]])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 12, 2), [['b', 10, 2, 2]])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 22, 2), [['c', 10, 2, 2]])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 32, 2), [['d', 10, 2, 2]])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 42, 2), [['e', 10, 2, 2]])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 52, 2), [['f', 10, 2, 2]])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 62, 2), [])
        self.assertEqual(arvados.locators_and_ranges(blocks2, -2, 2), [])

        self.assertEqual(arvados.locators_and_ranges(blocks2,  0,  2), [['a', 10, 0, 2]])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 10, 2), [['b', 10, 0, 2]])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 20, 2), [['c', 10, 0, 2]])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 30, 2), [['d', 10, 0, 2]])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 40, 2), [['e', 10, 0, 2]])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 50, 2), [['f', 10, 0, 2]])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 60, 2), [])
        self.assertEqual(arvados.locators_and_ranges(blocks2, -2, 2), [])

        self.assertEqual(arvados.locators_and_ranges(blocks2,  9,  2), [['a', 10, 9, 1], ['b', 10, 0, 1]])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 19, 2), [['b', 10, 9, 1], ['c', 10, 0, 1]])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 29, 2), [['c', 10, 9, 1], ['d', 10, 0, 1]])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 39, 2), [['d', 10, 9, 1], ['e', 10, 0, 1]])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 49, 2), [['e', 10, 9, 1], ['f', 10, 0, 1]])
        self.assertEqual(arvados.locators_and_ranges(blocks2, 59, 2), [['f', 10, 9, 1]])


        blocks3 = [['a', 10, 0],
                  ['b', 10, 10],
                  ['c', 10, 20],
                  ['d', 10, 30],
                  ['e', 10, 40],
                  ['f', 10, 50],
                  ['g', 10, 60]]

        self.assertEqual(arvados.locators_and_ranges(blocks3,  2,  2), [['a', 10, 2, 2]])
        self.assertEqual(arvados.locators_and_ranges(blocks3, 12, 2), [['b', 10, 2, 2]])
        self.assertEqual(arvados.locators_and_ranges(blocks3, 22, 2), [['c', 10, 2, 2]])
        self.assertEqual(arvados.locators_and_ranges(blocks3, 32, 2), [['d', 10, 2, 2]])
        self.assertEqual(arvados.locators_and_ranges(blocks3, 42, 2), [['e', 10, 2, 2]])
        self.assertEqual(arvados.locators_and_ranges(blocks3, 52, 2), [['f', 10, 2, 2]])
        self.assertEqual(arvados.locators_and_ranges(blocks3, 62, 2), [['g', 10, 2, 2]])


        blocks = [['a', 10, 0],
                  ['b', 15, 10],
                  ['c', 5, 25]]
        self.assertEqual(arvados.locators_and_ranges(blocks, 1, 0), [])
        self.assertEqual(arvados.locators_and_ranges(blocks, 0, 5), [['a', 10, 0, 5]])
        self.assertEqual(arvados.locators_and_ranges(blocks, 3, 5), [['a', 10, 3, 5]])
        self.assertEqual(arvados.locators_and_ranges(blocks, 0, 10), [['a', 10, 0, 10]])

        self.assertEqual(arvados.locators_and_ranges(blocks, 0, 11), [['a', 10, 0, 10],
                                                                      ['b', 15, 0, 1]])
        self.assertEqual(arvados.locators_and_ranges(blocks, 1, 11), [['a', 10, 1, 9],
                                                                      ['b', 15, 0, 2]])
        self.assertEqual(arvados.locators_and_ranges(blocks, 0, 25), [['a', 10, 0, 10],
                                                                      ['b', 15, 0, 15]])

        self.assertEqual(arvados.locators_and_ranges(blocks, 0, 30), [['a', 10, 0, 10],
                                                                      ['b', 15, 0, 15],
                                                                      ['c', 5, 0, 5]])
        self.assertEqual(arvados.locators_and_ranges(blocks, 1, 30), [['a', 10, 1, 9],
                                                                      ['b', 15, 0, 15],
                                                                      ['c', 5, 0, 5]])
        self.assertEqual(arvados.locators_and_ranges(blocks, 0, 31), [['a', 10, 0, 10],
                                                                      ['b', 15, 0, 15],
                                                                      ['c', 5, 0, 5]])

        self.assertEqual(arvados.locators_and_ranges(blocks, 15, 5), [['b', 15, 5, 5]])

        self.assertEqual(arvados.locators_and_ranges(blocks, 8, 17), [['a', 10, 8, 2],
                                                                      ['b', 15, 0, 15]])

        self.assertEqual(arvados.locators_and_ranges(blocks, 8, 20), [['a', 10, 8, 2],
                                                                      ['b', 15, 0, 15],
                                                                      ['c', 5, 0, 3]])

        self.assertEqual(arvados.locators_and_ranges(blocks, 26, 2), [['c', 5, 1, 2]])

        self.assertEqual(arvados.locators_and_ranges(blocks, 9, 15), [['a', 10, 9, 1],
                                                                      ['b', 15, 0, 14]])
        self.assertEqual(arvados.locators_and_ranges(blocks, 10, 15), [['b', 15, 0, 15]])
        self.assertEqual(arvados.locators_and_ranges(blocks, 11, 15), [['b', 15, 1, 14],
                                                                       ['c', 5, 0, 1]])

    class MockStreamReader(object):
        def __init__(self, content):
            self.content = content

        def readfrom(self, start, size):
            return self.content[start:start+size]

    def test_file_stream(self):
        content = 'abcdefghijklmnopqrstuvwxyz0123456789'
        msr = self.MockStreamReader(content)
        segments = [[0, 10, 0],
                    [10, 15, 10],
                    [25, 5, 25]]

        sfr = arvados.StreamFileReader(msr, segments, "test")

        self.assertEqual(sfr.name(), "test")
        self.assertEqual(sfr.size(), 30)

        self.assertEqual(sfr.readfrom(0, 30), content[0:30])
        self.assertEqual(sfr.readfrom(2, 30), content[2:30])

        self.assertEqual(sfr.readfrom(2, 8), content[2:10])
        self.assertEqual(sfr.readfrom(0, 10), content[0:10])

        self.assertEqual(sfr.tell(), 0)
        self.assertEqual(sfr.read(5), content[0:5])
        self.assertEqual(sfr.tell(), 5)
        self.assertEqual(sfr.read(5), content[5:10])
        self.assertEqual(sfr.tell(), 10)
        self.assertEqual(sfr.read(5), content[10:15])
        self.assertEqual(sfr.tell(), 15)
        self.assertEqual(sfr.read(5), content[15:20])
        self.assertEqual(sfr.tell(), 20)
        self.assertEqual(sfr.read(5), content[20:25])
        self.assertEqual(sfr.tell(), 25)
        self.assertEqual(sfr.read(5), content[25:30])
        self.assertEqual(sfr.tell(), 30)
        self.assertEqual(sfr.read(5), '')
        self.assertEqual(sfr.tell(), 30)

        segments = [[26, 10, 0],
                    [0, 15, 10],
                    [15, 5, 25]]

        sfr = arvados.StreamFileReader(msr, segments, "test")

        self.assertEqual(sfr.size(), 30)

        self.assertEqual(sfr.readfrom(0, 30), content[26:36] + content[0:20])
        self.assertEqual(sfr.readfrom(2, 30), content[28:36] + content[0:20])

        self.assertEqual(sfr.readfrom(2, 8), content[28:36])
        self.assertEqual(sfr.readfrom(0, 10), content[26:36])

        self.assertEqual(sfr.tell(), 0)
        self.assertEqual(sfr.read(5), content[26:31])
        self.assertEqual(sfr.tell(), 5)
        self.assertEqual(sfr.read(5), content[31:36])
        self.assertEqual(sfr.tell(), 10)
        self.assertEqual(sfr.read(5), content[0:5])
        self.assertEqual(sfr.tell(), 15)
        self.assertEqual(sfr.read(5), content[5:10])
        self.assertEqual(sfr.tell(), 20)
        self.assertEqual(sfr.read(5), content[10:15])
        self.assertEqual(sfr.tell(), 25)
        self.assertEqual(sfr.read(5), content[15:20])
        self.assertEqual(sfr.tell(), 30)
        self.assertEqual(sfr.read(5), '')
        self.assertEqual(sfr.tell(), 30)


    class MockKeep(object):
        def __init__(self, content):
            self.content = content

        def get(self, locator):
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

    def test_file_reader(self):
        keepblocks = {'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa+10': 'abcdefghij',
                      'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb+15': 'klmnopqrstuvwxy',
                      'cccccccccccccccccccccccccccccccc+5': 'z0123'}
        mk = self.MockKeep(keepblocks)

        sr = arvados.StreamReader([".", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa+10", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb+15", "cccccccccccccccccccccccccccccccc+5", "0:10:foo", "15:10:foo"], mk)

        content = 'abcdefghijpqrstuvwxy'

        f = sr.files()["foo"]

        f.seek(0)
        self.assertEqual(f.read(20), content[0:20])

        f.seek(0)
        self.assertEqual(f.read(6), content[0:6])
        self.assertEqual(f.read(6), content[6:12])
        self.assertEqual(f.read(6), content[12:18])

    def test_extract_file(self):
        m1 = """. 5348b82a029fd9e971a811ce1f71360b+43 0:43:md5sum.txt
. 085c37f02916da1cad16f93c54d899b7+41 0:41:md6sum.txt
. 8b22da26f9f433dea0a10e5ec66d73ba+43 0:43:md7sum.txt
. 085c37f02916da1cad16f93c54d899b7+41 5348b82a029fd9e971a811ce1f71360b+43 8b22da26f9f433dea0a10e5ec66d73ba+43 47:80:md8sum.txt
. 085c37f02916da1cad16f93c54d899b7+41 5348b82a029fd9e971a811ce1f71360b+43 8b22da26f9f433dea0a10e5ec66d73ba+43 40:80:md9sum.txt"""

        m2 = arvados.CollectionReader(m1, self.api_client)

        self.assertEqual(m2.manifest_text(),
                         ". 5348b82a029fd9e971a811ce1f71360b+43 085c37f02916da1cad16f93c54d899b7+41 8b22da26f9f433dea0a10e5ec66d73ba+43 0:43:md5sum.txt 43:41:md6sum.txt 84:43:md7sum.txt 6:37:md8sum.txt 84:43:md8sum.txt 83:1:md9sum.txt 0:43:md9sum.txt 84:36:md9sum.txt\n")

        self.assertEqual(arvados.CollectionReader(m1, self.api_client).all_streams()[0].files()['md5sum.txt'].as_manifest(),
                         ". 5348b82a029fd9e971a811ce1f71360b+43 0:43:md5sum.txt\n")
        self.assertEqual(arvados.CollectionReader(m1, self.api_client).all_streams()[0].files()['md6sum.txt'].as_manifest(),
                         ". 085c37f02916da1cad16f93c54d899b7+41 0:41:md6sum.txt\n")
        self.assertEqual(arvados.CollectionReader(m1, self.api_client).all_streams()[0].files()['md7sum.txt'].as_manifest(),
                         ". 8b22da26f9f433dea0a10e5ec66d73ba+43 0:43:md7sum.txt\n")
        self.assertEqual(arvados.CollectionReader(m1, self.api_client).all_streams()[0].files()['md9sum.txt'].as_manifest(),
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
                         """. 4ace875ffdc6824a04950f06858f4465+22 0:8:basefile
./subdir 4ace875ffdc6824a04950f06858f4465+22 8:14:subfile\n""")

    def test_write_directory_tree_with_limited_recursion(self):
        cwriter = arvados.CollectionWriter(self.api_client)
        cwriter.write_directory_tree(
            self.build_directory_tree(['f1', 'd1/f2', 'd1/d2/f3']),
            max_manifest_depth=1)
        self.assertEqual(cwriter.manifest_text(),
                         """. bd19836ddb62c11c55ab251ccaca5645+2 0:2:f1
./d1 50170217e5b04312024aa5cd42934494+13 8:5:f2
./d1/d2 50170217e5b04312024aa5cd42934494+13 0:8:f3\n""")

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
        self.assertEquals(cwriter.manifest_text(), resumed.manifest_text(),
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


if __name__ == '__main__':
    unittest.main()
