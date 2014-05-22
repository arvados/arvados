# usage example:
#
# ARVADOS_API_TOKEN=abc ARVADOS_API_HOST=arvados.local python -m unittest discover

import unittest
import arvados
import os
import run_test_server

class KeepTestCase(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        try:
            del os.environ['KEEP_LOCAL_STORE']
        except KeyError:
            pass
        run_test_server.run()
        run_test_server.run_keep()

    @classmethod
    def tearDownClass(cls):
        run_test_server.stop()
        run_test_server.stop_keep()

    def test_KeepBasicRWTest(self):
        foo_locator = arvados.Keep.put('foo')
        self.assertEqual(foo_locator,
                         'acbd18db4cc2f85cedef654fccc4a4d8+3',
                         'wrong md5 hash from Keep.put("foo"): ' + foo_locator)
        self.assertEqual(arvados.Keep.get(foo_locator),
                         'foo',
                         'wrong content from Keep.get(md5("foo"))')

    def test_KeepBinaryRWTest(self):
        blob_str = '\xff\xfe\xf7\x00\x01\x02'
        blob_locator = arvados.Keep.put(blob_str)
        self.assertEqual(blob_locator,
                         '7fc7c53b45e53926ba52821140fef396+6',
                         ('wrong locator from Keep.put(<binarydata>):' +
                          blob_locator))
        self.assertEqual(arvados.Keep.get(blob_locator),
                         blob_str,
                         'wrong content from Keep.get(md5(<binarydata>))')

    def test_KeepLongBinaryRWTest(self):
        blob_str = '\xff\xfe\xfd\xfc\x00\x01\x02\x03'
        for i in range(0,23):
            blob_str = blob_str + blob_str
        blob_locator = arvados.Keep.put(blob_str)
        self.assertEqual(blob_locator,
                         '84d90fc0d8175dd5dcfab04b999bc956+67108864',
                         ('wrong locator from Keep.put(<binarydata>): ' +
                          blob_locator))
        self.assertEqual(arvados.Keep.get(blob_locator),
                         blob_str,
                         'wrong content from Keep.get(md5(<binarydata>))')

    def test_KeepSingleCopyRWTest(self):
        blob_str = '\xff\xfe\xfd\xfc\x00\x01\x02\x03'
        blob_locator = arvados.Keep.put(blob_str, copies=1)
        self.assertEqual(blob_locator,
                         'c902006bc98a3eb4a3663b65ab4a6fab+8',
                         ('wrong locator from Keep.put(<binarydata>): ' +
                          blob_locator))
        self.assertEqual(arvados.Keep.get(blob_locator),
                         blob_str,
                         'wrong content from Keep.get(md5(<binarydata>))')
