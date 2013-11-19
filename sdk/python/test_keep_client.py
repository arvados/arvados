# usage example:
#
# ARVADOS_API_TOKEN=abc ARVADOS_API_HOST=arvados.local python -m unittest discover

import unittest
import arvados
import os

class KeepRWTest(unittest.TestCase):
    def setUp(self):
        try:
            del os.environ['KEEP_LOCAL_STORE']
        except KeyError:
            pass
    def runTest(self):
        foo_locator = arvados.Keep.put('foo')
        self.assertEqual(foo_locator,
                         'acbd18db4cc2f85cedef654fccc4a4d8+3',
                         'wrong md5 hash from Keep.put("foo"): ' + foo_locator)
        self.assertEqual(arvados.Keep.get(foo_locator),
                         'foo',
                         'wrong content from Keep.get(md5("foo"))')
        blob_str = '\xff\xfe\xf7\x00\x01\x02'
        blob_locator = arvados.Keep.put(blob_str)
        self.assertEqual(blob_locator,
                         '7fc7c53b45e53926ba52821140fef396+6',
                         'wrong md5 hash from Keep.put(<binarydata>)')
        self.assertEqual(arvados.Keep.get(blob_locator),
                         blob_str,
                         'wrong content from Keep.get(md5(<binarydata>))')
