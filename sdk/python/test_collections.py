# usage example:
#
# ARVADOS_API_TOKEN=abc ARVADOS_API_HOST=arvados.local python -m unittest discover

import unittest
import arvados
import os

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
                         'a4d26dddc10ad8b5eb39347c916de16c+112',
                         'resulting manifest hash is not what I expected')

class LocalCollectionReaderTest(unittest.TestCase):
    def setUp(self):
        os.environ['KEEP_LOCAL_STORE'] = '/tmp'
        LocalCollectionWriterTest().runTest()
    def runTest(self):
        cr = arvados.CollectionReader('a4d26dddc10ad8b5eb39347c916de16c+112')
        got = []
        for s in cr.all_streams():
            for f in s.all_files():
                got += [[f.size(), f.stream_name(), f.name(), f.read(2**26)]]
        expected = [[3, '.', 'foo.txt', 'foo'],
                    [3, '.', 'bar.txt', 'bar'],
                    [3, 'baz', 'baz.txt', 'baz']]
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
        cr = arvados.CollectionReader('a4d26dddc10ad8b5eb39347c916de16c+112')
        manifest_subsets = []
        for s in cr.all_streams():
            for f in s.all_files():
                manifest_subsets += [f.as_manifest()]
        got = []
        for m in manifest_subsets:
            cr = arvados.CollectionReader(m)
            for f in cr.all_files():
                got += [[f.size(), f.stream_name(), f.name(), f.read(2**26)]]
        expected = [[3, '.', 'foo.txt', 'foo'],
                    [3, '.', 'bar.txt', 'bar'],
                    [3, 'baz', 'baz.txt', 'baz']]
        self.assertEqual(got,
                         expected,
                         'all_files|as_manifest did not preserve manifest contents')

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
