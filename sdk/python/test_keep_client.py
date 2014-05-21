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

class KeepPermissionTestCase(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        try:
            del os.environ['KEEP_LOCAL_STORE']
        except KeyError:
            pass
        run_test_server.run()
        run_test_server.run_keep(blob_signing_key='abcdefghijk0123456789',
                                 enforce_permissions=True)

    @classmethod
    def tearDownClass(cls):
        run_test_server.stop()
        run_test_server.stop_keep()

    def test_KeepBasicRWTest(self):
        run_test_server.authorize_with('active')
        foo_locator = arvados.Keep.put('foo')
        self.assertRegexpMatches(
            foo_locator,
            r'^acbd18db4cc2f85cedef654fccc4a4d8\+3\+A[a-f0-9]+@[a-f0-9]+$',
            'invalid locator from Keep.put("foo"): ' + foo_locator)
        self.assertEqual(arvados.Keep.get(foo_locator),
                         'foo',
                         'wrong content from Keep.get(md5("foo"))')

        # With Keep permissions enabled, a GET request without a locator will fail.
        bar_locator = arvados.Keep.put('bar')
        self.assertRegexpMatches(
            bar_locator,
            r'^37b51d194a7513e45b56f6524f2d51f2\+3\+A[a-f0-9]+@[a-f0-9]+$',
            'invalid locator from Keep.put("bar"): ' + bar_locator)
        self.assertRaises(arvados.errors.NotFoundError,
                          arvados.Keep.get,
                          "37b51d194a7513e45b56f6524f2d51f2")

        # A request without an API token will also fail.
        del arvados.config.settings()["ARVADOS_API_TOKEN"]
        self.assertRaises(arvados.errors.NotFoundError,
                          arvados.Keep.get,
                          bar_locator)

# KeepOptionalPermission: starts Keep with --permission-key-file
# but not --enforce-permissions (i.e. generate signatures on PUT
# requests, but do not require them for GET requests)
#
class KeepOptionalPermission(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        try:
            del os.environ['KEEP_LOCAL_STORE']
        except KeyError:
            pass
        run_test_server.run()
        run_test_server.run_keep(blob_signing_key='abcdefghijk0123456789',
                                 enforce_permissions=False)

    @classmethod
    def tearDownClass(cls):
        run_test_server.stop()
        run_test_server.stop_keep()

    def test_KeepBasicRWTest(self):
        run_test_server.authorize_with('active')
        foo_locator = arvados.Keep.put('foo')
        self.assertRegexpMatches(
            foo_locator,
            r'^acbd18db4cc2f85cedef654fccc4a4d8\+3\+A[a-f0-9]+@[a-f0-9]+$',
            'invalid locator from Keep.put("foo"): ' + foo_locator)
        self.assertEqual(arvados.Keep.get(foo_locator),
                         'foo',
                         'wrong content from Keep.get(md5("foo"))')

    def test_KeepUnsignedLocatorTest(self):
        # Since --enforce-permissions is not in effect, GET requests
        # do not require signatures.
        run_test_server.authorize_with('active')
        foo_locator = arvados.Keep.put('foo')
        self.assertRegexpMatches(
            foo_locator,
            r'^acbd18db4cc2f85cedef654fccc4a4d8\+3\+A[a-f0-9]+@[a-f0-9]+$',
            'invalid locator from Keep.put("foo"): ' + foo_locator)
        self.assertEqual(arvados.Keep.get("acbd18db4cc2f85cedef654fccc4a4d8"),
                         'foo',
                         'wrong content from Keep.get(md5("foo"))')

    def test_KeepUnauthenticatedTest(self):
        # Since --enforce-permissions is not in effect, GET requests
        # need not be authenticated.
        run_test_server.authorize_with('active')
        foo_locator = arvados.Keep.put('foo')
        self.assertRegexpMatches(
            foo_locator,
            r'^acbd18db4cc2f85cedef654fccc4a4d8\+3\+A[a-f0-9]+@[a-f0-9]+$',
            'invalid locator from Keep.put("foo"): ' + foo_locator)

        del arvados.config.settings()["ARVADOS_API_TOKEN"]
        self.assertEqual(arvados.Keep.get("acbd18db4cc2f85cedef654fccc4a4d8"),
                         'foo',
                         'wrong content from Keep.get(md5("foo"))')
