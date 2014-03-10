# usage example:
#
# ARVADOS_API_TOKEN=abc ARVADOS_API_HOST=arvados.local python -m unittest discover

import unittest
import arvados
import os
import bz2
import sys
import subprocess

class KeepLocalStoreTest(unittest.TestCase):
    def setUp(self):
        os.environ['KEEP_LOCAL_STORE'] = '/tmp'
    def runTest(self):
        self.assertEqual(arvados.Keep.put('foo'), 'acbd18db4cc2f85cedef654fccc4a4d8+3', 'wrong md5 hash from Keep.put')
        self.assertEqual(arvados.Keep.get('acbd18db4cc2f85cedef654fccc4a4d8+3'), 'foo', 'wrong data from Keep.get')

class LocalCollectionWriterTest(unittest.TestCase):
    def setUp(self):
        os.environ['KEEP_LOCAL_STORE'] = '/tmp'
    def runTest(self):
        cw = arvados.CollectionWriter()
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
        hash = cw.finish()
        self.assertEqual(hash,
                         'd6c3b8e571f1b81ebb150a45ed06c884+114',
                         "resulting manifest hash was {0}, expecting d6c3b8e571f1b81ebb150a45ed06c884+114".format(hash))

class LocalCollectionReaderTest(unittest.TestCase):
    def setUp(self):
        os.environ['KEEP_LOCAL_STORE'] = '/tmp'
        LocalCollectionWriterTest().runTest()
    def runTest(self):
        cr = arvados.CollectionReader('d6c3b8e571f1b81ebb150a45ed06c884+114+Xzizzle')
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

class LocalCollectionManifestSubsetTest(unittest.TestCase):
    def setUp(self):
        os.environ['KEEP_LOCAL_STORE'] = '/tmp'
        LocalCollectionWriterTest().runTest()
    def runTest(self):
        self._runTest('d6c3b8e571f1b81ebb150a45ed06c884+114',
                      [[3, '.',     'bar.txt', 'bar'],
                       [3, '.',     'foo.txt', 'foo'],
                       [3, './baz', 'baz.txt', 'baz']])
        self._runTest((". %s %s 0:3:foo.txt 3:3:bar.txt\n" %
                       (arvados.Keep.put("foo"),
                        arvados.Keep.put("bar"))),
                      [[3, '.', 'bar.txt', 'bar'],
                       [3, '.', 'foo.txt', 'foo']])
        self._runTest((". %s %s 0:2:fo.txt 2:4:obar.txt\n" %
                       (arvados.Keep.put("foo"),
                        arvados.Keep.put("bar"))),
                      [[2, '.', 'fo.txt', 'fo'],
                       [4, '.', 'obar.txt', 'obar']])
        self._runTest((". %s %s 0:2:fo.txt 2:0:zero.txt 2:2:ob.txt 4:2:ar.txt\n" %
                       (arvados.Keep.put("foo"),
                        arvados.Keep.put("bar"))),
                      [[2, '.', 'ar.txt', 'ar'],
                       [2, '.', 'fo.txt', 'fo'],                       
                       [2, '.', 'ob.txt', 'ob'],
                       [0, '.', 'zero.txt', '']])

    def _runTest(self, collection, expected):
        cr = arvados.CollectionReader(collection)
        for s in cr.all_streams():
            for ex in expected:
                if ex[0] == s:
                    f = s.files()[ex[2]]
                    got = [f.size(), f.stream_name(), f.name(), "".join(f.readall(2**26))]
                    self.assertEqual(got,
                                     ex,
                                     'all_files|as_manifest did not preserve manifest contents: got %s expected %s' % (got, ex))

class LocalCollectionReadlineTest(unittest.TestCase):
    def setUp(self):
        os.environ['KEEP_LOCAL_STORE'] = '/tmp'
    def _runTest(self, what_in, what_out):
        cw = arvados.CollectionWriter()
        cw.start_new_file('test.txt')
        cw.write(what_in)
        test1 = cw.finish()
        cr = arvados.CollectionReader(test1)
        got = []
        for x in list(cr.all_files())[0].readlines():
            got += [x]
        self.assertEqual(got,
                         what_out,
                         "readlines did not split lines correctly: %s" % got)
    def runTest(self):
        self._runTest("\na\nbcd\n\nefg\nz",
                      ["\n", "a\n", "bcd\n", "\n", "efg\n", "z"])
        self._runTest("ab\ncd\n",
                      ["ab\n", "cd\n"])

class LocalCollectionEmptyFileTest(unittest.TestCase):
    def setUp(self):
        os.environ['KEEP_LOCAL_STORE'] = '/tmp'
    def runTest(self):
        cw = arvados.CollectionWriter()
        cw.start_new_file('zero.txt')
        cw.write('')

        self.assertEqual(cw.manifest_text(), ". 0:0:zero.txt\n")
        self.check_manifest_file_sizes(cw.manifest_text(), [0])
        cw = arvados.CollectionWriter()
        cw.start_new_file('zero.txt')
        cw.write('')
        cw.start_new_file('one.txt')
        cw.write('1')
        cw.start_new_stream('foo')
        cw.start_new_file('zero.txt')
        cw.write('')
        self.check_manifest_file_sizes(cw.manifest_text(), [1,0,0])

    def check_manifest_file_sizes(self, manifest_text, expect_sizes):
        cr = arvados.CollectionReader(manifest_text)
        got_sizes = []
        for f in cr.all_files():
            got_sizes += [f.size()]
        self.assertEqual(got_sizes, expect_sizes, "got wrong file sizes %s, expected %s" % (got_sizes, expect_sizes))

class LocalCollectionBZ2DecompressionTest(unittest.TestCase):
    def setUp(self):
        os.environ['KEEP_LOCAL_STORE'] = '/tmp'
    def runTest(self):
        n_lines_in = 2**18
        data_in = "abc\n"
        for x in xrange(0, 18):
            data_in += data_in
        compressed_data_in = bz2.compress(data_in)
        cw = arvados.CollectionWriter()
        cw.start_new_file('test.bz2')
        cw.write(compressed_data_in)
        bz2_manifest = cw.manifest_text()

        cr = arvados.CollectionReader(bz2_manifest)

        got = 0
        for x in list(cr.all_files())[0].readlines():
            self.assertEqual(x, "abc\n", "decompression returned wrong data: %s" % x)
            got += 1
        self.assertEqual(got,
                         n_lines_in,
                         "decompression returned %d lines instead of %d" % (got, n_lines_in))

class LocalCollectionGzipDecompressionTest(unittest.TestCase):
    def setUp(self):
        os.environ['KEEP_LOCAL_STORE'] = '/tmp'
    def runTest(self):
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

        cw = arvados.CollectionWriter()
        cw.start_new_file('test.gz')
        cw.write(compressed_data_in)
        gzip_manifest = cw.manifest_text()

        cr = arvados.CollectionReader(gzip_manifest)
        got = 0
        for x in list(cr.all_files())[0].readlines():
            self.assertEqual(x, "abc\n", "decompression returned wrong data: %s" % x)
            got += 1
        self.assertEqual(got,
                         n_lines_in,
                         "decompression returned %d lines instead of %d" % (got, n_lines_in))

class NormalizedCollectionTest(unittest.TestCase):
    def runTest(self):
        m1 = """. 5348b82a029fd9e971a811ce1f71360b+43 0:43:md5sum.txt
. 085c37f02916da1cad16f93c54d899b7+41 0:41:md5sum.txt
. 8b22da26f9f433dea0a10e5ec66d73ba+43 0:43:md5sum.txt"""
        self.assertEqual(arvados.CollectionReader(m1).manifest_text(),
                         """. 5348b82a029fd9e971a811ce1f71360b+43 085c37f02916da1cad16f93c54d899b7+41 8b22da26f9f433dea0a10e5ec66d73ba+43 0:127:md5sum.txt
""")

        m2 = """. 204e43b8a1185621ca55a94839582e6f+67108864 b9677abbac956bd3e86b1deb28dfac03+67108864 fc15aff2a762b13f521baf042140acec+67108864 323d2a3ce20370c4ca1d3462a344f8fd+25885655 0:227212247:var-GS000016015-ASM.tsv.bz2
"""
        self.assertEqual(arvados.CollectionReader(m2).manifest_text(), m2)

        m3 = """. 5348b82a029fd9e971a811ce1f71360b+43 3:40:md5sum.txt
. 085c37f02916da1cad16f93c54d899b7+41 0:41:md5sum.txt
. 8b22da26f9f433dea0a10e5ec66d73ba+43 0:43:md5sum.txt"""
        self.assertEqual(arvados.CollectionReader(m3).manifest_text(),
                         """. 5348b82a029fd9e971a811ce1f71360b+43 085c37f02916da1cad16f93c54d899b7+41 8b22da26f9f433dea0a10e5ec66d73ba+43 3:124:md5sum.txt
""")

        m4 = """. 204e43b8a1185621ca55a94839582e6f+67108864 0:3:foo/bar
./zzz 204e43b8a1185621ca55a94839582e6f+67108864 0:999:zzz
./foo 323d2a3ce20370c4ca1d3462a344f8fd+25885655 0:3:bar"""
        self.assertEqual(arvados.CollectionReader(m4).manifest_text(),
                         """./foo 204e43b8a1185621ca55a94839582e6f+67108864 323d2a3ce20370c4ca1d3462a344f8fd+25885655 0:3:bar 67108864:3:bar
./zzz 204e43b8a1185621ca55a94839582e6f+67108864 0:999:zzz
""")

        m5 = """. 204e43b8a1185621ca55a94839582e6f+67108864 0:3:foo/bar
./zzz 204e43b8a1185621ca55a94839582e6f+67108864 0:999:zzz
./foo 204e43b8a1185621ca55a94839582e6f+67108864 3:3:bar"""
        self.assertEqual(arvados.CollectionReader(m5).manifest_text(),
                         """./foo 204e43b8a1185621ca55a94839582e6f+67108864 0:6:bar
./zzz 204e43b8a1185621ca55a94839582e6f+67108864 0:999:zzz
""")

        with open('testdata/1000G_ref_manifest') as f6:
            m6 = f6.read()
            self.assertEqual(arvados.CollectionReader(m6).manifest_text(), m6)

        with open('testdata/jlake_manifest') as f7:
            m7 = f7.read()
            self.assertEqual(arvados.CollectionReader(m7).manifest_text(), m7)

        m8 = """./a\\040b\\040c 59ca0efa9f5633cb0371bbc0355478d8+13 0:13:hello\\040world.txt
"""
        self.assertEqual(arvados.CollectionReader(m8).manifest_text(), m8)

class LocatorsAndRangesTest(unittest.TestCase):
    def runTest(self):
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

class FileStreamTest(unittest.TestCase):
    class MockStreamReader(object):
        def __init__(self, content):
            self.content = content

        def readfrom(self, start, size):
            return self.content[start:start+size]

    def runTest(self):
        content = 'abcdefghijklmnopqrstuvwxyz0123456789'
        msr = FileStreamTest.MockStreamReader(content)
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


class StreamReaderTest(unittest.TestCase):

    class MockKeep(object):
        def __init__(self, content):
            self.content = content

        def get(self, locator):
            return self.content[locator]

    def runTest(self):
        keepblocks = {'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa+10': 'abcdefghij', 
                      'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb+15': 'klmnopqrstuvwxy', 
                      'cccccccccccccccccccccccccccccccc+5': 'z0123'}
        mk = StreamReaderTest.MockKeep(keepblocks)

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

class ExtractFileTest(unittest.TestCase):
    def runTest(self):
        m1 = """. 5348b82a029fd9e971a811ce1f71360b+43 0:43:md5sum.txt
. 085c37f02916da1cad16f93c54d899b7+41 0:41:md6sum.txt
. 8b22da26f9f433dea0a10e5ec66d73ba+43 0:43:md7sum.txt
. 085c37f02916da1cad16f93c54d899b7+41 5348b82a029fd9e971a811ce1f71360b+43 8b22da26f9f433dea0a10e5ec66d73ba+43 47:80:md8sum.txt
. 085c37f02916da1cad16f93c54d899b7+41 5348b82a029fd9e971a811ce1f71360b+43 8b22da26f9f433dea0a10e5ec66d73ba+43 40:80:md9sum.txt"""

        m2 = arvados.CollectionReader(m1)

        self.assertEqual(m2.manifest_text(),
                         ". 5348b82a029fd9e971a811ce1f71360b+43 085c37f02916da1cad16f93c54d899b7+41 8b22da26f9f433dea0a10e5ec66d73ba+43 0:43:md5sum.txt 43:41:md6sum.txt 84:43:md7sum.txt 6:37:md8sum.txt 84:43:md8sum.txt 83:1:md9sum.txt 0:43:md9sum.txt 84:36:md9sum.txt\n")

        self.assertEqual(arvados.CollectionReader(m1).all_streams()[0].files()['md5sum.txt'].as_manifest(),
                         ". 5348b82a029fd9e971a811ce1f71360b+43 0:43:md5sum.txt\n")
        self.assertEqual(arvados.CollectionReader(m1).all_streams()[0].files()['md6sum.txt'].as_manifest(),
                         ". 085c37f02916da1cad16f93c54d899b7+41 0:41:md6sum.txt\n")
        self.assertEqual(arvados.CollectionReader(m1).all_streams()[0].files()['md7sum.txt'].as_manifest(),
                         ". 8b22da26f9f433dea0a10e5ec66d73ba+43 0:43:md7sum.txt\n")
        self.assertEqual(arvados.CollectionReader(m1).all_streams()[0].files()['md9sum.txt'].as_manifest(),
                         ". 085c37f02916da1cad16f93c54d899b7+41 5348b82a029fd9e971a811ce1f71360b+43 8b22da26f9f433dea0a10e5ec66d73ba+43 40:80:md9sum.txt\n")
