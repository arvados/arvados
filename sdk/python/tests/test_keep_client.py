# usage example:
#
# ARVADOS_API_TOKEN=abc ARVADOS_API_HOST=arvados.local python -m unittest discover

import os
import unittest

import arvados
import run_test_server

class KeepTestCase(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        super(KeepTestCase, cls).setUpClass()
        try:
            del os.environ['KEEP_LOCAL_STORE']
        except KeyError:
            pass

        # Make sure these are clear, we want to talk to the Keep servers
        # directly.
        os.environ["ARVADOS_KEEP_PROXY"] = ""
        os.environ["ARVADOS_EXTERNAL_CLIENT"] = ""

        run_test_server.run()
        run_test_server.run_keep()
        arvados.keep.global_client_object = None
        arvados.config._settings = None
        run_test_server.authorize_with("admin")

    @classmethod
    def tearDownClass(cls):
        super(KeepTestCase, cls).tearDownClass()
        run_test_server.stop()
        run_test_server.stop_keep()

    def test_KeepBasicRWTest(self):
        foo_locator = arvados.Keep.put('foo')
        self.assertRegexpMatches(
            foo_locator,
            '^acbd18db4cc2f85cedef654fccc4a4d8\+3',
            'wrong md5 hash from Keep.put("foo"): ' + foo_locator)
        self.assertEqual(arvados.Keep.get(foo_locator),
                         'foo',
                         'wrong content from Keep.get(md5("foo"))')

    def test_KeepBinaryRWTest(self):
        blob_str = '\xff\xfe\xf7\x00\x01\x02'
        blob_locator = arvados.Keep.put(blob_str)
        self.assertRegexpMatches(
            blob_locator,
            '^7fc7c53b45e53926ba52821140fef396\+6',
            ('wrong locator from Keep.put(<binarydata>):' + blob_locator))
        self.assertEqual(arvados.Keep.get(blob_locator),
                         blob_str,
                         'wrong content from Keep.get(md5(<binarydata>))')

    def test_KeepLongBinaryRWTest(self):
        blob_str = '\xff\xfe\xfd\xfc\x00\x01\x02\x03'
        for i in range(0,23):
            blob_str = blob_str + blob_str
        blob_locator = arvados.Keep.put(blob_str)
        self.assertRegexpMatches(
            blob_locator,
            '^84d90fc0d8175dd5dcfab04b999bc956\+67108864',
            ('wrong locator from Keep.put(<binarydata>): ' + blob_locator))
        self.assertEqual(arvados.Keep.get(blob_locator),
                         blob_str,
                         'wrong content from Keep.get(md5(<binarydata>))')

    def test_KeepSingleCopyRWTest(self):
        blob_str = '\xff\xfe\xfd\xfc\x00\x01\x02\x03'
        blob_locator = arvados.Keep.put(blob_str, copies=1)
        self.assertRegexpMatches(
            blob_locator,
            '^c902006bc98a3eb4a3663b65ab4a6fab\+8',
            ('wrong locator from Keep.put(<binarydata>): ' + blob_locator))
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

        # GET with an unsigned locator => NotFound
        bar_locator = arvados.Keep.put('bar')
        unsigned_bar_locator = "37b51d194a7513e45b56f6524f2d51f2+3"
        self.assertRegexpMatches(
            bar_locator,
            r'^37b51d194a7513e45b56f6524f2d51f2\+3\+A[a-f0-9]+@[a-f0-9]+$',
            'invalid locator from Keep.put("bar"): ' + bar_locator)
        self.assertRaises(arvados.errors.NotFoundError,
                          arvados.Keep.get,
                          unsigned_bar_locator)

        # GET from a different user => NotFound
        run_test_server.authorize_with('spectator')
        self.assertRaises(arvados.errors.NotFoundError,
                          arvados.Keep.get,
                          bar_locator)

        # Unauthenticated GET for a signed locator => NotFound
        # Unauthenticated GET for an unsigned locator => NotFound
        arvados.keep.global_client_object.api_token = ''
        self.assertRaises(arvados.errors.NotFoundError,
                          arvados.Keep.get,
                          bar_locator)
        self.assertRaises(arvados.errors.NotFoundError,
                          arvados.Keep.get,
                          unsigned_bar_locator)

# KeepOptionalPermission: starts Keep with --permission-key-file
# but not --enforce-permissions (i.e. generate signatures on PUT
# requests, but do not require them for GET requests)
#
# All of these requests should succeed when permissions are optional:
# * authenticated request, signed locator
# * authenticated request, unsigned locator
# * unauthenticated request, signed locator
# * unauthenticated request, unsigned locator

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

    def test_KeepAuthenticatedSignedTest(self):
        run_test_server.authorize_with('active')
        signed_locator = arvados.Keep.put('foo')
        self.assertRegexpMatches(
            signed_locator,
            r'^acbd18db4cc2f85cedef654fccc4a4d8\+3\+A[a-f0-9]+@[a-f0-9]+$',
            'invalid locator from Keep.put("foo"): ' + signed_locator)
        self.assertEqual(arvados.Keep.get(signed_locator),
                         'foo',
                         'wrong content from Keep.get(md5("foo"))')

    def test_KeepAuthenticatedUnsignedTest(self):
        run_test_server.authorize_with('active')
        signed_locator = arvados.Keep.put('foo')
        self.assertRegexpMatches(
            signed_locator,
            r'^acbd18db4cc2f85cedef654fccc4a4d8\+3\+A[a-f0-9]+@[a-f0-9]+$',
            'invalid locator from Keep.put("foo"): ' + signed_locator)
        self.assertEqual(arvados.Keep.get("acbd18db4cc2f85cedef654fccc4a4d8"),
                         'foo',
                         'wrong content from Keep.get(md5("foo"))')

    def test_KeepUnauthenticatedSignedTest(self):
        # Since --enforce-permissions is not in effect, GET requests
        # need not be authenticated.
        run_test_server.authorize_with('active')
        signed_locator = arvados.Keep.put('foo')
        self.assertRegexpMatches(
            signed_locator,
            r'^acbd18db4cc2f85cedef654fccc4a4d8\+3\+A[a-f0-9]+@[a-f0-9]+$',
            'invalid locator from Keep.put("foo"): ' + signed_locator)

        arvados.keep.global_client_object.api_token = ''
        self.assertEqual(arvados.Keep.get(signed_locator),
                         'foo',
                         'wrong content from Keep.get(md5("foo"))')

    def test_KeepUnauthenticatedUnsignedTest(self):
        # Since --enforce-permissions is not in effect, GET requests
        # need not be authenticated.
        run_test_server.authorize_with('active')
        signed_locator = arvados.Keep.put('foo')
        self.assertRegexpMatches(
            signed_locator,
            r'^acbd18db4cc2f85cedef654fccc4a4d8\+3\+A[a-f0-9]+@[a-f0-9]+$',
            'invalid locator from Keep.put("foo"): ' + signed_locator)

        arvados.keep.global_client_object.api_token = ''
        self.assertEqual(arvados.Keep.get("acbd18db4cc2f85cedef654fccc4a4d8"),
                         'foo',
                         'wrong content from Keep.get(md5("foo"))')


class KeepProxyTestCase(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        super(KeepProxyTestCase, cls).setUpClass()

        try:
            del os.environ['KEEP_LOCAL_STORE']
        except KeyError:
            pass

        os.environ["ARVADOS_KEEP_PROXY"] = ""
        os.environ["ARVADOS_EXTERNAL_CLIENT"] = ""

        run_test_server.run()
        run_test_server.run_keep()
        arvados.keep.global_client_object = None
        arvados.config._settings = None
        run_test_server.run_keep_proxy("admin")
        KeepProxyTestCase.arvados_keep_proxy = arvados.config.get("ARVADOS_KEEP_PROXY")

    @classmethod
    def tearDownClass(cls):
        super(KeepProxyTestCase, cls).tearDownClass()
        run_test_server.stop()
        run_test_server.stop_keep()
        run_test_server.stop_keep_proxy()

    def test_KeepProxyTest1(self):
        # Will use ARVADOS_KEEP_PROXY environment variable that is set by
        # run_keep_proxy() in setUpClass()

        os.environ["ARVADOS_KEEP_PROXY"] = KeepProxyTestCase.arvados_keep_proxy
        os.environ["ARVADOS_EXTERNAL_CLIENT"] = ""
        arvados.keep.global_client_object = None
        arvados.config._settings = None

        baz_locator = arvados.Keep.put('baz')
        self.assertRegexpMatches(
            baz_locator,
            '^73feffa4b7f6bb68e44cf984c85f6e88\+3',
            'wrong md5 hash from Keep.put("baz"): ' + baz_locator)
        self.assertEqual(arvados.Keep.get(baz_locator),
                         'baz',
                         'wrong content from Keep.get(md5("baz"))')

        self.assertEqual(True, arvados.Keep.global_client_object().using_proxy)

    def test_KeepProxyTest2(self):
        # We don't want to use ARVADOS_KEEP_PROXY from run_keep_proxy() in
        # setUpClass(), so clear it and set ARVADOS_EXTERNAL_CLIENT which will
        # contact the API server.
        os.environ["ARVADOS_KEEP_PROXY"] = ""
        os.environ["ARVADOS_EXTERNAL_CLIENT"] = "true"
        arvados.keep.global_client_object = None
        arvados.config._settings = None

        # Will send X-External-Client to server and get back the proxy from
        # keep_services/accessible

        baz_locator = arvados.Keep.put('baz2')
        self.assertRegexpMatches(
            baz_locator,
            '^91f372a266fe2bf2823cb8ec7fda31ce\+4',
            'wrong md5 hash from Keep.put("baz2"): ' + baz_locator)
        self.assertEqual(arvados.Keep.get(baz_locator),
                         'baz2',
                         'wrong content from Keep.get(md5("baz2"))')

        self.assertEqual(True, arvados.Keep.global_client_object().using_proxy)
