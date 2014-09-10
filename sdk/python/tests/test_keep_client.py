import mock
import os
import socket
import unittest

import arvados
import arvados.retry
import run_test_server
from arvados_testutil import fake_httplib2_response

class KeepTestCase(run_test_server.TestCaseWithServers):
    MAIN_SERVER = {}
    KEEP_SERVER = {}

    @classmethod
    def setUpClass(cls):
        super(KeepTestCase, cls).setUpClass()
        run_test_server.authorize_with("admin")
        cls.api_client = arvados.api('v1')
        cls.keep_client = arvados.KeepClient(api_client=cls.api_client,
                                             proxy='', local_store='')

    def test_KeepBasicRWTest(self):
        foo_locator = self.keep_client.put('foo')
        self.assertRegexpMatches(
            foo_locator,
            '^acbd18db4cc2f85cedef654fccc4a4d8\+3',
            'wrong md5 hash from Keep.put("foo"): ' + foo_locator)
        self.assertEqual(self.keep_client.get(foo_locator),
                         'foo',
                         'wrong content from Keep.get(md5("foo"))')

    def test_KeepBinaryRWTest(self):
        blob_str = '\xff\xfe\xf7\x00\x01\x02'
        blob_locator = self.keep_client.put(blob_str)
        self.assertRegexpMatches(
            blob_locator,
            '^7fc7c53b45e53926ba52821140fef396\+6',
            ('wrong locator from Keep.put(<binarydata>):' + blob_locator))
        self.assertEqual(self.keep_client.get(blob_locator),
                         blob_str,
                         'wrong content from Keep.get(md5(<binarydata>))')

    def test_KeepLongBinaryRWTest(self):
        blob_str = '\xff\xfe\xfd\xfc\x00\x01\x02\x03'
        for i in range(0,23):
            blob_str = blob_str + blob_str
        blob_locator = self.keep_client.put(blob_str)
        self.assertRegexpMatches(
            blob_locator,
            '^84d90fc0d8175dd5dcfab04b999bc956\+67108864',
            ('wrong locator from Keep.put(<binarydata>): ' + blob_locator))
        self.assertEqual(self.keep_client.get(blob_locator),
                         blob_str,
                         'wrong content from Keep.get(md5(<binarydata>))')

    def test_KeepSingleCopyRWTest(self):
        blob_str = '\xff\xfe\xfd\xfc\x00\x01\x02\x03'
        blob_locator = self.keep_client.put(blob_str, copies=1)
        self.assertRegexpMatches(
            blob_locator,
            '^c902006bc98a3eb4a3663b65ab4a6fab\+8',
            ('wrong locator from Keep.put(<binarydata>): ' + blob_locator))
        self.assertEqual(self.keep_client.get(blob_locator),
                         blob_str,
                         'wrong content from Keep.get(md5(<binarydata>))')


class KeepPermissionTestCase(run_test_server.TestCaseWithServers):
    MAIN_SERVER = {}
    KEEP_SERVER = {'blob_signing_key': 'abcdefghijk0123456789',
                   'enforce_permissions': True}

    def test_KeepBasicRWTest(self):
        run_test_server.authorize_with('active')
        keep_client = arvados.KeepClient()
        foo_locator = keep_client.put('foo')
        self.assertRegexpMatches(
            foo_locator,
            r'^acbd18db4cc2f85cedef654fccc4a4d8\+3\+A[a-f0-9]+@[a-f0-9]+$',
            'invalid locator from Keep.put("foo"): ' + foo_locator)
        self.assertEqual(keep_client.get(foo_locator),
                         'foo',
                         'wrong content from Keep.get(md5("foo"))')

        # GET with an unsigned locator => NotFound
        bar_locator = keep_client.put('bar')
        unsigned_bar_locator = "37b51d194a7513e45b56f6524f2d51f2+3"
        self.assertRegexpMatches(
            bar_locator,
            r'^37b51d194a7513e45b56f6524f2d51f2\+3\+A[a-f0-9]+@[a-f0-9]+$',
            'invalid locator from Keep.put("bar"): ' + bar_locator)
        self.assertRaises(arvados.errors.NotFoundError,
                          keep_client.get,
                          unsigned_bar_locator)

        # GET from a different user => NotFound
        run_test_server.authorize_with('spectator')
        self.assertRaises(arvados.errors.NotFoundError,
                          arvados.Keep.get,
                          bar_locator)

        # Unauthenticated GET for a signed locator => NotFound
        # Unauthenticated GET for an unsigned locator => NotFound
        keep_client.api_token = ''
        self.assertRaises(arvados.errors.NotFoundError,
                          keep_client.get,
                          bar_locator)
        self.assertRaises(arvados.errors.NotFoundError,
                          keep_client.get,
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
class KeepOptionalPermission(run_test_server.TestCaseWithServers):
    MAIN_SERVER = {}
    KEEP_SERVER = {'blob_signing_key': 'abcdefghijk0123456789',
                   'enforce_permissions': False}

    @classmethod
    def setUpClass(cls):
        super(KeepOptionalPermission, cls).setUpClass()
        run_test_server.authorize_with("admin")
        cls.api_client = arvados.api('v1')

    def setUp(self):
        super(KeepOptionalPermission, self).setUp()
        self.keep_client = arvados.KeepClient(api_client=self.api_client,
                                              proxy='', local_store='')

    def _put_foo_and_check(self):
        signed_locator = self.keep_client.put('foo')
        self.assertRegexpMatches(
            signed_locator,
            r'^acbd18db4cc2f85cedef654fccc4a4d8\+3\+A[a-f0-9]+@[a-f0-9]+$',
            'invalid locator from Keep.put("foo"): ' + signed_locator)
        return signed_locator

    def test_KeepAuthenticatedSignedTest(self):
        signed_locator = self._put_foo_and_check()
        self.assertEqual(self.keep_client.get(signed_locator),
                         'foo',
                         'wrong content from Keep.get(md5("foo"))')

    def test_KeepAuthenticatedUnsignedTest(self):
        signed_locator = self._put_foo_and_check()
        self.assertEqual(self.keep_client.get("acbd18db4cc2f85cedef654fccc4a4d8"),
                         'foo',
                         'wrong content from Keep.get(md5("foo"))')

    def test_KeepUnauthenticatedSignedTest(self):
        # Check that signed GET requests work even when permissions
        # enforcement is off.
        signed_locator = self._put_foo_and_check()
        self.keep_client.api_token = ''
        self.assertEqual(self.keep_client.get(signed_locator),
                         'foo',
                         'wrong content from Keep.get(md5("foo"))')

    def test_KeepUnauthenticatedUnsignedTest(self):
        # Since --enforce-permissions is not in effect, GET requests
        # need not be authenticated.
        signed_locator = self._put_foo_and_check()
        self.keep_client.api_token = ''
        self.assertEqual(self.keep_client.get("acbd18db4cc2f85cedef654fccc4a4d8"),
                         'foo',
                         'wrong content from Keep.get(md5("foo"))')


class KeepProxyTestCase(run_test_server.TestCaseWithServers):
    MAIN_SERVER = {}
    KEEP_SERVER = {}
    KEEP_PROXY_SERVER = {'auth': 'admin'}

    @classmethod
    def setUpClass(cls):
        super(KeepProxyTestCase, cls).setUpClass()
        cls.api_client = arvados.api('v1')

    def tearDown(self):
        arvados.config.settings().pop('ARVADOS_EXTERNAL_CLIENT', None)
        super(KeepProxyTestCase, self).tearDown()

    def test_KeepProxyTest1(self):
        # Will use ARVADOS_KEEP_PROXY environment variable that is set by
        # setUpClass().
        keep_client = arvados.KeepClient(api_client=self.api_client,
                                         local_store='')
        baz_locator = keep_client.put('baz')
        self.assertRegexpMatches(
            baz_locator,
            '^73feffa4b7f6bb68e44cf984c85f6e88\+3',
            'wrong md5 hash from Keep.put("baz"): ' + baz_locator)
        self.assertEqual(keep_client.get(baz_locator),
                         'baz',
                         'wrong content from Keep.get(md5("baz"))')
        self.assertTrue(keep_client.using_proxy)

    def test_KeepProxyTest2(self):
        # Don't instantiate the proxy directly, but set the X-External-Client
        # header.  The API server should direct us to the proxy.
        arvados.config.settings()['ARVADOS_EXTERNAL_CLIENT'] = 'true'
        keep_client = arvados.KeepClient(api_client=self.api_client,
                                         proxy='', local_store='')
        baz_locator = keep_client.put('baz2')
        self.assertRegexpMatches(
            baz_locator,
            '^91f372a266fe2bf2823cb8ec7fda31ce\+4',
            'wrong md5 hash from Keep.put("baz2"): ' + baz_locator)
        self.assertEqual(keep_client.get(baz_locator),
                         'baz2',
                         'wrong content from Keep.get(md5("baz2"))')
        self.assertTrue(keep_client.using_proxy)


class KeepClientRetryTestMixin(object):
    # Testing with a local Keep store won't exercise the retry behavior.
    # Instead, our strategy is:
    # * Create a client with one proxy specified (pointed at a black
    #   hole), so there's no need to instantiate an API client, and
    #   all HTTP requests come from one place.
    # * Mock httplib's request method to provide simulated responses.
    # This lets us test the retry logic extensively without relying on any
    # supporting servers, and prevents side effects in case something hiccups.
    # To use this mixin, define DEFAULT_EXPECT, DEFAULT_EXCEPTION, and
    # run_method().
    PROXY_ADDR = 'http://[100::]/'
    TEST_DATA = 'testdata'
    TEST_LOCATOR = 'ef654c40ab4f1747fc699915d4f70902+8'

    @staticmethod
    def mock_responses(body, *codes):
        return mock.patch('httplib2.Http.request', side_effect=(
                (fake_httplib2_response(code), body) for code in codes))

    def new_client(self):
        return arvados.KeepClient(proxy=self.PROXY_ADDR, local_store='')

    def run_method(self, *args, **kwargs):
        raise NotImplementedError("test subclasses must define run_method")

    def check_success(self, expected=None, *args, **kwargs):
        if expected is None:
            expected = self.DEFAULT_EXPECT
        self.assertEqual(expected, self.run_method(*args, **kwargs))

    def check_exception(self, error_class=None, *args, **kwargs):
        if error_class is None:
            error_class = self.DEFAULT_EXCEPTION
        self.assertRaises(error_class, self.run_method, *args, **kwargs)

    def test_immediate_success(self):
        with self.mock_responses(self.DEFAULT_EXPECT, 200):
            self.check_success()

    def test_retry_then_success(self):
        with self.mock_responses(self.DEFAULT_EXPECT, 500, 200):
            self.check_success(num_retries=3)

    def test_no_default_retry(self):
        with self.mock_responses(self.DEFAULT_EXPECT, 500, 200):
            self.check_exception()

    def test_no_retry_after_permanent_error(self):
        with self.mock_responses(self.DEFAULT_EXPECT, 403, 200):
            self.check_exception(num_retries=3)

    def test_error_after_retries_exhausted(self):
        with self.mock_responses(self.DEFAULT_EXPECT, 500, 500, 200):
            self.check_exception(num_retries=1)


# Don't delay from HTTPRetryLoop's exponential backoff.
no_backoff = mock.patch('time.sleep', lambda n: None)
@no_backoff
class KeepClientRetryGetTestCase(unittest.TestCase, KeepClientRetryTestMixin):
    DEFAULT_EXPECT = KeepClientRetryTestMixin.TEST_DATA
    DEFAULT_EXCEPTION = arvados.errors.KeepReadError
    HINTED_LOCATOR = KeepClientRetryTestMixin.TEST_LOCATOR + '+K@xyzzy'

    def run_method(self, locator=KeepClientRetryTestMixin.TEST_LOCATOR,
                   *args, **kwargs):
        return self.new_client().get(locator, *args, **kwargs)

    def test_specific_exception_when_not_found(self):
        with self.mock_responses(self.DEFAULT_EXPECT, 404, 200):
            self.check_exception(arvados.errors.NotFoundError, num_retries=3)

    def test_general_exception_with_mixed_errors(self):
        # get should raise a NotFoundError if no server returns the block,
        # and a high threshold of servers report that it's not found.
        # This test rigs up 50/50 disagreement between two servers, and
        # checks that it does not become a NotFoundError.
        client = self.new_client()
        with self.mock_responses(self.DEFAULT_EXPECT, 404, 500):
            with self.assertRaises(arvados.errors.KeepReadError) as exc_check:
                client.get(self.HINTED_LOCATOR)
            self.assertNotIsInstance(
                exc_check.exception, arvados.errors.NotFoundError,
                "mixed errors raised NotFoundError")

    def test_hint_server_can_succeed_without_retries(self):
        with self.mock_responses(self.DEFAULT_EXPECT, 404, 200, 500):
            self.check_success(locator=self.HINTED_LOCATOR)

    def test_try_next_server_after_timeout(self):
        side_effects = [socket.timeout("timed out"),
                        (fake_httplib2_response(200), self.DEFAULT_EXPECT)]
        with mock.patch('httplib2.Http.request',
                        side_effect=iter(side_effects)):
            self.check_success(locator=self.HINTED_LOCATOR)


@no_backoff
class KeepClientRetryPutTestCase(unittest.TestCase, KeepClientRetryTestMixin):
    DEFAULT_EXPECT = KeepClientRetryTestMixin.TEST_LOCATOR
    DEFAULT_EXCEPTION = arvados.errors.KeepWriteError

    def run_method(self, data=KeepClientRetryTestMixin.TEST_DATA,
                   copies=1, *args, **kwargs):
        return self.new_client().put(data, copies, *args, **kwargs)

    def test_do_not_send_multiple_copies_to_same_server(self):
        with self.mock_responses(self.DEFAULT_EXPECT, 200):
            self.check_exception(copies=2, num_retries=3)
