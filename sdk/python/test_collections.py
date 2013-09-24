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
                         '23ca013983d6239e98931cc779e68426+114',
                         'resulting manifest hash is not what I expected')

class LocalCollectionReaderTest(unittest.TestCase):
    def setUp(self):
        os.environ['KEEP_LOCAL_STORE'] = '/tmp'
        LocalCollectionWriterTest().runTest()
    def runTest(self):
        cr = arvados.CollectionReader('23ca013983d6239e98931cc779e68426+114')
        got = []
        for s in cr.all_streams():
            for f in s.all_files():
                got += [[f.size(), f.stream_name(), f.name(), f.read(2**26)]]
        expected = [[3, '.', 'foo.txt', 'foo'],
                    [3, '.', 'bar.txt', 'bar'],
                    [3, './baz', 'baz.txt', 'baz']]
        self.assertEqual(got,
                         expected,
                         'resulting file list is not what I expected')
        stream0 = cr.all_streams()[0]
        self.assertEqual(stream0.read(0),
                         '',
                         'reading zero bytes should have returned empty string')
        self.assertEqual(stream0.read(2**26),
                         'foobar',
                         'reading entire stream failed')
        self.assertEqual(stream0.read(2**26),
                         None,
                         'reading past end of stream should have returned None')
        self.assertEqual(stream0.read(0),
                         '',
                         'reading zero bytes should have returned empty string')

class LocalCollectionManifestSubsetTest(unittest.TestCase):
    def setUp(self):
        os.environ['KEEP_LOCAL_STORE'] = '/tmp'
        LocalCollectionWriterTest().runTest()
    def runTest(self):
        self._runTest('23ca013983d6239e98931cc779e68426+114',
                      [[3, '.', 'foo.txt', 'foo'],
                       [3, '.', 'bar.txt', 'bar'],
                       [3, './baz', 'baz.txt', 'baz']])
        self._runTest((". %s %s 0:3:foo.txt 3:3:bar.txt\n" %
                       (arvados.Keep.put("foo"),
                        arvados.Keep.put("bar"))),
                      [[3, '.', 'foo.txt', 'foo'],
                       [3, '.', 'bar.txt', 'bar']])
        self._runTest((". %s %s 0:2:fo.txt 2:4:obar.txt\n" %
                       (arvados.Keep.put("foo"),
                        arvados.Keep.put("bar"))),
                      [[2, '.', 'fo.txt', 'fo'],
                       [4, '.', 'obar.txt', 'obar']])
        self._runTest((". %s %s 0:2:fo.txt 2:0:zero.txt 2:2:ob.txt 4:2:ar.txt\n" %
                       (arvados.Keep.put("foo"),
                        arvados.Keep.put("bar"))),
                      [[2, '.', 'fo.txt', 'fo'],
                       [0, '.', 'zero.txt', ''],
                       [2, '.', 'ob.txt', 'ob'],
                       [2, '.', 'ar.txt', 'ar']])
    def _runTest(self, collection, expected):
        cr = arvados.CollectionReader(collection)
        manifest_subsets = []
        for s in cr.all_streams():
            for f in s.all_files():
                manifest_subsets += [f.as_manifest()]
        expect_i = 0
        for m in manifest_subsets:
            cr = arvados.CollectionReader(m)
            for f in cr.all_files():
                got = [f.size(), f.stream_name(), f.name(), "".join(f.readall(2**26))]
                self.assertEqual(got,
                                 expected[expect_i],
                                 'all_files|as_manifest did not preserve manifest contents: got %s expected %s' % (got, expected[expect_i]))
                expect_i += 1

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
        self.check_manifest_file_sizes(cw.manifest_text(), [0])
        cw = arvados.CollectionWriter()
        cw.start_new_file('zero.txt')
        cw.write('')
        cw.start_new_file('one.txt')
        cw.write('1')
        cw.start_new_stream('foo')
        cw.start_new_file('zero.txt')
        cw.write('')
        self.check_manifest_file_sizes(cw.manifest_text(), [0,1,0])
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
