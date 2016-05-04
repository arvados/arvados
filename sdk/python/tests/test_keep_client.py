import hashlib
import mock
import os
import pycurl
import random
import re
import socket
import threading
import time
import unittest
import urlparse

import arvados
import arvados.retry
import arvados_testutil as tutil
import keepstub
import run_test_server

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
        self.assertEqual(0, self.keep_client.upload_counter.get())
        foo_locator = self.keep_client.put('foo')
        self.assertRegexpMatches(
            foo_locator,
            '^acbd18db4cc2f85cedef654fccc4a4d8\+3',
            'wrong md5 hash from Keep.put("foo"): ' + foo_locator)

        # 6 bytes because uploaded 2 copies
        self.assertEqual(6, self.keep_client.upload_counter.get())

        self.assertEqual(0, self.keep_client.download_counter.get())
        self.assertEqual(self.keep_client.get(foo_locator),
                         'foo',
                         'wrong content from Keep.get(md5("foo"))')
        self.assertEqual(3, self.keep_client.download_counter.get())

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

    @unittest.skip("unreliable test - please fix and close #8752")
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

    def test_KeepEmptyCollectionTest(self):
        blob_locator = self.keep_client.put('', copies=1)
        self.assertRegexpMatches(
            blob_locator,
            '^d41d8cd98f00b204e9800998ecf8427e\+0',
            ('wrong locator from Keep.put(""): ' + blob_locator))

    def test_unicode_must_be_ascii(self):
        # If unicode type, must only consist of valid ASCII
        foo_locator = self.keep_client.put(u'foo')
        self.assertRegexpMatches(
            foo_locator,
            '^acbd18db4cc2f85cedef654fccc4a4d8\+3',
            'wrong md5 hash from Keep.put("foo"): ' + foo_locator)

        with self.assertRaises(UnicodeEncodeError):
            # Error if it is not ASCII
            self.keep_client.put(u'\xe2')

        with self.assertRaises(arvados.errors.ArgumentError):
            # Must be a string type
            self.keep_client.put({})

    def test_KeepHeadTest(self):
        locator = self.keep_client.put('test_head')
        self.assertRegexpMatches(
            locator,
            '^b9a772c7049325feb7130fff1f8333e9\+9',
            'wrong md5 hash from Keep.put for "test_head": ' + locator)
        self.assertEqual(True, self.keep_client.head(locator))
        self.assertEqual(self.keep_client.get(locator),
                         'test_head',
                         'wrong content from Keep.get for "test_head"')

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
    KEEP_PROXY_SERVER = {}

    @classmethod
    def setUpClass(cls):
        super(KeepProxyTestCase, cls).setUpClass()
        run_test_server.authorize_with('active')
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


class KeepClientServiceTestCase(unittest.TestCase, tutil.ApiClientMock):
    def get_service_roots(self, api_client):
        keep_client = arvados.KeepClient(api_client=api_client)
        services = keep_client.weighted_service_roots(arvados.KeepLocator('0'*32))
        return [urlparse.urlparse(url) for url in sorted(services)]

    def test_ssl_flag_respected_in_roots(self):
        for ssl_flag in [False, True]:
            services = self.get_service_roots(self.mock_keep_services(
                service_ssl_flag=ssl_flag))
            self.assertEqual(
                ('https' if ssl_flag else 'http'), services[0].scheme)

    def test_correct_ports_with_ipv6_addresses(self):
        service = self.get_service_roots(self.mock_keep_services(
            service_type='proxy', service_host='100::1', service_port=10, count=1))[0]
        self.assertEqual('100::1', service.hostname)
        self.assertEqual(10, service.port)

    # test_*_timeout verify that KeepClient instructs pycurl to use
    # the appropriate connection and read timeouts. They don't care
    # whether pycurl actually exhibits the expected timeout behavior
    # -- those tests are in the KeepClientTimeout test class.

    def test_get_timeout(self):
        api_client = self.mock_keep_services(count=1)
        force_timeout = socket.timeout("timed out")
        with tutil.mock_keep_responses(force_timeout, 0) as mock:
            keep_client = arvados.KeepClient(api_client=api_client)
            with self.assertRaises(arvados.errors.KeepReadError):
                keep_client.get('ffffffffffffffffffffffffffffffff')
            self.assertEqual(
                mock.responses[0].getopt(pycurl.CONNECTTIMEOUT_MS),
                int(arvados.KeepClient.DEFAULT_TIMEOUT[0]*1000))
            self.assertEqual(
                mock.responses[0].getopt(pycurl.LOW_SPEED_TIME),
                int(arvados.KeepClient.DEFAULT_TIMEOUT[1]))
            self.assertEqual(
                mock.responses[0].getopt(pycurl.LOW_SPEED_LIMIT),
                int(arvados.KeepClient.DEFAULT_TIMEOUT[2]))

    def test_put_timeout(self):
        api_client = self.mock_keep_services(count=1)
        force_timeout = socket.timeout("timed out")
        with tutil.mock_keep_responses(force_timeout, 0) as mock:
            keep_client = arvados.KeepClient(api_client=api_client)
            with self.assertRaises(arvados.errors.KeepWriteError):
                keep_client.put('foo')
            self.assertEqual(
                mock.responses[0].getopt(pycurl.CONNECTTIMEOUT_MS),
                int(arvados.KeepClient.DEFAULT_TIMEOUT[0]*1000))
            self.assertEqual(
                mock.responses[0].getopt(pycurl.LOW_SPEED_TIME),
                int(arvados.KeepClient.DEFAULT_TIMEOUT[1]))
            self.assertEqual(
                mock.responses[0].getopt(pycurl.LOW_SPEED_LIMIT),
                int(arvados.KeepClient.DEFAULT_TIMEOUT[2]))

    def test_head_timeout(self):
        api_client = self.mock_keep_services(count=1)
        force_timeout = socket.timeout("timed out")
        with tutil.mock_keep_responses(force_timeout, 0) as mock:
            keep_client = arvados.KeepClient(api_client=api_client)
            with self.assertRaises(arvados.errors.KeepReadError):
                keep_client.head('ffffffffffffffffffffffffffffffff')
            self.assertEqual(
                mock.responses[0].getopt(pycurl.CONNECTTIMEOUT_MS),
                int(arvados.KeepClient.DEFAULT_TIMEOUT[0]*1000))
            self.assertEqual(
                mock.responses[0].getopt(pycurl.LOW_SPEED_TIME),
                int(arvados.KeepClient.DEFAULT_TIMEOUT[1]))
            self.assertEqual(
                mock.responses[0].getopt(pycurl.LOW_SPEED_LIMIT),
                int(arvados.KeepClient.DEFAULT_TIMEOUT[2]))

    def test_proxy_get_timeout(self):
        api_client = self.mock_keep_services(service_type='proxy', count=1)
        force_timeout = socket.timeout("timed out")
        with tutil.mock_keep_responses(force_timeout, 0) as mock:
            keep_client = arvados.KeepClient(api_client=api_client)
            with self.assertRaises(arvados.errors.KeepReadError):
                keep_client.get('ffffffffffffffffffffffffffffffff')
            self.assertEqual(
                mock.responses[0].getopt(pycurl.CONNECTTIMEOUT_MS),
                int(arvados.KeepClient.DEFAULT_PROXY_TIMEOUT[0]*1000))
            self.assertEqual(
                mock.responses[0].getopt(pycurl.LOW_SPEED_TIME),
                int(arvados.KeepClient.DEFAULT_PROXY_TIMEOUT[1]))
            self.assertEqual(
                mock.responses[0].getopt(pycurl.LOW_SPEED_LIMIT),
                int(arvados.KeepClient.DEFAULT_PROXY_TIMEOUT[2]))

    def test_proxy_head_timeout(self):
        api_client = self.mock_keep_services(service_type='proxy', count=1)
        force_timeout = socket.timeout("timed out")
        with tutil.mock_keep_responses(force_timeout, 0) as mock:
            keep_client = arvados.KeepClient(api_client=api_client)
            with self.assertRaises(arvados.errors.KeepReadError):
                keep_client.head('ffffffffffffffffffffffffffffffff')
            self.assertEqual(
                mock.responses[0].getopt(pycurl.CONNECTTIMEOUT_MS),
                int(arvados.KeepClient.DEFAULT_PROXY_TIMEOUT[0]*1000))
            self.assertEqual(
                mock.responses[0].getopt(pycurl.LOW_SPEED_TIME),
                int(arvados.KeepClient.DEFAULT_PROXY_TIMEOUT[1]))
            self.assertEqual(
                mock.responses[0].getopt(pycurl.LOW_SPEED_LIMIT),
                int(arvados.KeepClient.DEFAULT_PROXY_TIMEOUT[2]))

    def test_proxy_put_timeout(self):
        api_client = self.mock_keep_services(service_type='proxy', count=1)
        force_timeout = socket.timeout("timed out")
        with tutil.mock_keep_responses(force_timeout, 0) as mock:
            keep_client = arvados.KeepClient(api_client=api_client)
            with self.assertRaises(arvados.errors.KeepWriteError):
                keep_client.put('foo')
            self.assertEqual(
                mock.responses[0].getopt(pycurl.CONNECTTIMEOUT_MS),
                int(arvados.KeepClient.DEFAULT_PROXY_TIMEOUT[0]*1000))
            self.assertEqual(
                mock.responses[0].getopt(pycurl.LOW_SPEED_TIME),
                int(arvados.KeepClient.DEFAULT_PROXY_TIMEOUT[1]))
            self.assertEqual(
                mock.responses[0].getopt(pycurl.LOW_SPEED_LIMIT),
                int(arvados.KeepClient.DEFAULT_PROXY_TIMEOUT[2]))

    def check_no_services_error(self, verb, exc_class):
        api_client = mock.MagicMock(name='api_client')
        api_client.keep_services().accessible().execute.side_effect = (
            arvados.errors.ApiError)
        keep_client = arvados.KeepClient(api_client=api_client)
        with self.assertRaises(exc_class) as err_check:
            getattr(keep_client, verb)('d41d8cd98f00b204e9800998ecf8427e+0')
        self.assertEqual(0, len(err_check.exception.request_errors()))

    def test_get_error_with_no_services(self):
        self.check_no_services_error('get', arvados.errors.KeepReadError)

    def test_head_error_with_no_services(self):
        self.check_no_services_error('head', arvados.errors.KeepReadError)

    def test_put_error_with_no_services(self):
        self.check_no_services_error('put', arvados.errors.KeepWriteError)

    def check_errors_from_last_retry(self, verb, exc_class):
        api_client = self.mock_keep_services(count=2)
        req_mock = tutil.mock_keep_responses(
            "retry error reporting test", 500, 500, 403, 403)
        with req_mock, tutil.skip_sleep, \
                self.assertRaises(exc_class) as err_check:
            keep_client = arvados.KeepClient(api_client=api_client)
            getattr(keep_client, verb)('d41d8cd98f00b204e9800998ecf8427e+0',
                                       num_retries=3)
        self.assertEqual([403, 403], [
                getattr(error, 'status_code', None)
                for error in err_check.exception.request_errors().itervalues()])

    def test_get_error_reflects_last_retry(self):
        self.check_errors_from_last_retry('get', arvados.errors.KeepReadError)

    def test_head_error_reflects_last_retry(self):
        self.check_errors_from_last_retry('head', arvados.errors.KeepReadError)

    def test_put_error_reflects_last_retry(self):
        self.check_errors_from_last_retry('put', arvados.errors.KeepWriteError)

    def test_put_error_does_not_include_successful_puts(self):
        data = 'partial failure test'
        data_loc = tutil.str_keep_locator(data)
        api_client = self.mock_keep_services(count=3)
        with tutil.mock_keep_responses(data_loc, 200, 500, 500) as req_mock, \
                self.assertRaises(arvados.errors.KeepWriteError) as exc_check:
            keep_client = arvados.KeepClient(api_client=api_client)
            keep_client.put(data)
        self.assertEqual(2, len(exc_check.exception.request_errors()))

    def test_proxy_put_with_no_writable_services(self):
        data = 'test with no writable services'
        data_loc = tutil.str_keep_locator(data)
        api_client = self.mock_keep_services(service_type='proxy', read_only=True, count=1)
        with tutil.mock_keep_responses(data_loc, 200, 500, 500) as req_mock, \
                self.assertRaises(arvados.errors.KeepWriteError) as exc_check:
          keep_client = arvados.KeepClient(api_client=api_client)
          keep_client.put(data)
        self.assertEqual(True, ("no Keep services available" in str(exc_check.exception)))
        self.assertEqual(0, len(exc_check.exception.request_errors()))

    def test_oddball_service_get(self):
        body = 'oddball service get'
        api_client = self.mock_keep_services(service_type='fancynewblobstore')
        with tutil.mock_keep_responses(body, 200):
            keep_client = arvados.KeepClient(api_client=api_client)
            actual = keep_client.get(tutil.str_keep_locator(body))
        self.assertEqual(body, actual)

    def test_oddball_service_put(self):
        body = 'oddball service put'
        pdh = tutil.str_keep_locator(body)
        api_client = self.mock_keep_services(service_type='fancynewblobstore')
        with tutil.mock_keep_responses(pdh, 200):
            keep_client = arvados.KeepClient(api_client=api_client)
            actual = keep_client.put(body, copies=1)
        self.assertEqual(pdh, actual)

    def test_oddball_service_writer_count(self):
        body = 'oddball service writer count'
        pdh = tutil.str_keep_locator(body)
        api_client = self.mock_keep_services(service_type='fancynewblobstore',
                                             count=4)
        headers = {'x-keep-replicas-stored': 3}
        with tutil.mock_keep_responses(pdh, 200, 418, 418, 418,
                                       **headers) as req_mock:
            keep_client = arvados.KeepClient(api_client=api_client)
            actual = keep_client.put(body, copies=2)
        self.assertEqual(pdh, actual)
        self.assertEqual(1, req_mock.call_count)


@tutil.skip_sleep
class KeepClientRendezvousTestCase(unittest.TestCase, tutil.ApiClientMock):

    def setUp(self):
        # expected_order[i] is the probe order for
        # hash=md5(sprintf("%064x",i)) where there are 16 services
        # with uuid sprintf("anything-%015x",j) with j in 0..15. E.g.,
        # the first probe for the block consisting of 64 "0"
        # characters is the service whose uuid is
        # "zzzzz-bi6l4-000000000000003", so expected_order[0][0]=='3'.
        self.services = 16
        self.expected_order = [
            list('3eab2d5fc9681074'),
            list('097dba52e648f1c3'),
            list('c5b4e023f8a7d691'),
            list('9d81c02e76a3bf54'),
            ]
        self.blocks = [
            "{:064x}".format(x)
            for x in range(len(self.expected_order))]
        self.hashes = [
            hashlib.md5(self.blocks[x]).hexdigest()
            for x in range(len(self.expected_order))]
        self.api_client = self.mock_keep_services(count=self.services)
        self.keep_client = arvados.KeepClient(api_client=self.api_client)

    def test_weighted_service_roots_against_reference_set(self):
        # Confirm weighted_service_roots() returns the correct order
        for i, hash in enumerate(self.hashes):
            roots = self.keep_client.weighted_service_roots(arvados.KeepLocator(hash))
            got_order = [
                re.search(r'//\[?keep0x([0-9a-f]+)', root).group(1)
                for root in roots]
            self.assertEqual(self.expected_order[i], got_order)

    def test_get_probe_order_against_reference_set(self):
        self._test_probe_order_against_reference_set(
            lambda i: self.keep_client.get(self.hashes[i], num_retries=1))

    def test_head_probe_order_against_reference_set(self):
        self._test_probe_order_against_reference_set(
            lambda i: self.keep_client.head(self.hashes[i], num_retries=1))

    def test_put_probe_order_against_reference_set(self):
        # copies=1 prevents the test from being sensitive to races
        # between writer threads.
        self._test_probe_order_against_reference_set(
            lambda i: self.keep_client.put(self.blocks[i], num_retries=1, copies=1))

    def _test_probe_order_against_reference_set(self, op):
        for i in range(len(self.blocks)):
            with tutil.mock_keep_responses('', *[500 for _ in range(self.services*2)]) as mock, \
                 self.assertRaises(arvados.errors.KeepRequestError):
                op(i)
            got_order = [
                re.search(r'//\[?keep0x([0-9a-f]+)', resp.getopt(pycurl.URL)).group(1)
                for resp in mock.responses]
            self.assertEqual(self.expected_order[i]*2, got_order)

    def test_put_probe_order_multiple_copies(self):
        for copies in range(2, 4):
            for i in range(len(self.blocks)):
                with tutil.mock_keep_responses('', *[500 for _ in range(self.services*3)]) as mock, \
                     self.assertRaises(arvados.errors.KeepWriteError):
                    self.keep_client.put(self.blocks[i], num_retries=2, copies=copies)
                got_order = [
                    re.search(r'//\[?keep0x([0-9a-f]+)', resp.getopt(pycurl.URL)).group(1)
                    for resp in mock.responses]
                # With T threads racing to make requests, the position
                # of a given server in the sequence of HTTP requests
                # (got_order) cannot be more than T-1 positions
                # earlier than that server's position in the reference
                # probe sequence (expected_order).
                #
                # Loop invariant: we have accounted for +pos+ expected
                # probes, either by seeing them in +got_order+ or by
                # putting them in +pending+ in the hope of seeing them
                # later. As long as +len(pending)<T+, we haven't
                # started a request too early.
                pending = []
                for pos, expected in enumerate(self.expected_order[i]*3):
                    got = got_order[pos-len(pending)]
                    while got in pending:
                        del pending[pending.index(got)]
                        got = got_order[pos-len(pending)]
                    if got != expected:
                        pending.append(expected)
                        self.assertLess(
                            len(pending), copies,
                            "pending={}, with copies={}, got {}, expected {}".format(
                                pending, copies, repr(got_order), repr(self.expected_order[i]*3)))

    def test_probe_waste_adding_one_server(self):
        hashes = [
            hashlib.md5("{:064x}".format(x)).hexdigest() for x in range(100)]
        initial_services = 12
        self.api_client = self.mock_keep_services(count=initial_services)
        self.keep_client = arvados.KeepClient(api_client=self.api_client)
        probes_before = [
            self.keep_client.weighted_service_roots(arvados.KeepLocator(hash)) for hash in hashes]
        for added_services in range(1, 12):
            api_client = self.mock_keep_services(count=initial_services+added_services)
            keep_client = arvados.KeepClient(api_client=api_client)
            total_penalty = 0
            for hash_index in range(len(hashes)):
                probe_after = keep_client.weighted_service_roots(
                    arvados.KeepLocator(hashes[hash_index]))
                penalty = probe_after.index(probes_before[hash_index][0])
                self.assertLessEqual(penalty, added_services)
                total_penalty += penalty
            # Average penalty per block should not exceed
            # N(added)/N(orig) by more than 20%, and should get closer
            # to the ideal as we add data points.
            expect_penalty = (
                added_services *
                len(hashes) / initial_services)
            max_penalty = (
                expect_penalty *
                (120 - added_services)/100)
            min_penalty = (
                expect_penalty * 8/10)
            self.assertTrue(
                min_penalty <= total_penalty <= max_penalty,
                "With {}+{} services, {} blocks, penalty {} but expected {}..{}".format(
                    initial_services,
                    added_services,
                    len(hashes),
                    total_penalty,
                    min_penalty,
                    max_penalty))

    def check_64_zeros_error_order(self, verb, exc_class):
        data = '0' * 64
        if verb == 'get':
            data = tutil.str_keep_locator(data)
        # Arbitrary port number:
        aport = random.randint(1024,65535)
        api_client = self.mock_keep_services(service_port=aport, count=self.services)
        keep_client = arvados.KeepClient(api_client=api_client)
        with mock.patch('pycurl.Curl') as curl_mock, \
             self.assertRaises(exc_class) as err_check:
            curl_mock.return_value.side_effect = socket.timeout
            getattr(keep_client, verb)(data)
        urls = [urlparse.urlparse(url)
                for url in err_check.exception.request_errors()]
        self.assertEqual([('keep0x' + c, aport) for c in '3eab2d5fc9681074'],
                         [(url.hostname, url.port) for url in urls])

    def test_get_error_shows_probe_order(self):
        self.check_64_zeros_error_order('get', arvados.errors.KeepReadError)

    def test_put_error_shows_probe_order(self):
        self.check_64_zeros_error_order('put', arvados.errors.KeepWriteError)


class KeepClientTimeout(unittest.TestCase, tutil.ApiClientMock):
    # BANDWIDTH_LOW_LIM must be less than len(DATA) so we can transfer
    # 1s worth of data and then trigger bandwidth errors before running
    # out of data.
    DATA = 'x'*2**11
    BANDWIDTH_LOW_LIM = 1024
    TIMEOUT_TIME = 1.0

    class assertTakesBetween(unittest.TestCase):
        def __init__(self, tmin, tmax):
            self.tmin = tmin
            self.tmax = tmax

        def __enter__(self):
            self.t0 = time.time()

        def __exit__(self, *args, **kwargs):
            # Round times to milliseconds, like CURL. Otherwise, we
            # fail when CURL reaches a 1s timeout at 0.9998s.
            delta = round(time.time() - self.t0, 3)
            self.assertGreaterEqual(delta, self.tmin)
            self.assertLessEqual(delta, self.tmax)

    class assertTakesGreater(unittest.TestCase):
        def __init__(self, tmin):
            self.tmin = tmin

        def __enter__(self):
            self.t0 = time.time()

        def __exit__(self, *args, **kwargs):
            delta = round(time.time() - self.t0, 3)
            self.assertGreaterEqual(delta, self.tmin)

    def setUp(self):
        sock = socket.socket()
        sock.bind(('0.0.0.0', 0))
        self.port = sock.getsockname()[1]
        sock.close()
        self.server = keepstub.Server(('0.0.0.0', self.port), keepstub.Handler)
        self.thread = threading.Thread(target=self.server.serve_forever)
        self.thread.daemon = True # Exit thread if main proc exits
        self.thread.start()
        self.api_client = self.mock_keep_services(
            count=1,
            service_host='localhost',
            service_port=self.port,
        )

    def tearDown(self):
        self.server.shutdown()

    def keepClient(self, timeouts=(0.1, TIMEOUT_TIME, BANDWIDTH_LOW_LIM)):
        return arvados.KeepClient(
            api_client=self.api_client,
            timeout=timeouts)

    def test_timeout_slow_connect(self):
        # Can't simulate TCP delays with our own socket. Leave our
        # stub server running uselessly, and try to connect to an
        # unroutable IP address instead.
        self.api_client = self.mock_keep_services(
            count=1,
            service_host='240.0.0.0',
        )
        with self.assertTakesBetween(0.1, 0.5):
            with self.assertRaises(arvados.errors.KeepWriteError):
                self.keepClient().put(self.DATA, copies=1, num_retries=0)

    def test_low_bandwidth_no_delays_success(self):
        self.server.setbandwidth(2*self.BANDWIDTH_LOW_LIM)
        kc = self.keepClient()
        loc = kc.put(self.DATA, copies=1, num_retries=0)
        self.assertEqual(self.DATA, kc.get(loc, num_retries=0))

    def test_too_low_bandwidth_no_delays_failure(self):
        # Check that lessening bandwidth corresponds to failing
        kc = self.keepClient()
        loc = kc.put(self.DATA, copies=1, num_retries=0)
        self.server.setbandwidth(0.5*self.BANDWIDTH_LOW_LIM)
        with self.assertTakesGreater(self.TIMEOUT_TIME):
            with self.assertRaises(arvados.errors.KeepReadError) as e:
                kc.get(loc, num_retries=0)
        with self.assertTakesGreater(self.TIMEOUT_TIME):
            with self.assertRaises(arvados.errors.KeepWriteError):
                kc.put(self.DATA, copies=1, num_retries=0)

    def test_low_bandwidth_with_server_response_delay_failure(self):
        kc = self.keepClient()
        loc = kc.put(self.DATA, copies=1, num_retries=0)
        self.server.setbandwidth(self.BANDWIDTH_LOW_LIM)
        self.server.setdelays(response=self.TIMEOUT_TIME)
        with self.assertTakesGreater(self.TIMEOUT_TIME):
            with self.assertRaises(arvados.errors.KeepReadError) as e:
                kc.get(loc, num_retries=0)
        with self.assertTakesGreater(self.TIMEOUT_TIME):
            with self.assertRaises(arvados.errors.KeepWriteError):
                kc.put(self.DATA, copies=1, num_retries=0)
        with self.assertTakesGreater(self.TIMEOUT_TIME):
            with self.assertRaises(arvados.errors.KeepReadError) as e:
                kc.head(loc, num_retries=0)

    def test_low_bandwidth_with_server_mid_delay_failure(self):
        kc = self.keepClient()
        loc = kc.put(self.DATA, copies=1, num_retries=0)
        self.server.setbandwidth(self.BANDWIDTH_LOW_LIM)
        self.server.setdelays(mid_write=self.TIMEOUT_TIME, mid_read=self.TIMEOUT_TIME)
        with self.assertTakesGreater(self.TIMEOUT_TIME):
            with self.assertRaises(arvados.errors.KeepReadError) as e:
                kc.get(loc, num_retries=0)
        with self.assertTakesGreater(self.TIMEOUT_TIME):
            with self.assertRaises(arvados.errors.KeepWriteError):
                kc.put(self.DATA, copies=1, num_retries=0)

    def test_timeout_slow_request(self):
        loc = self.keepClient().put(self.DATA, copies=1, num_retries=0)
        self.server.setdelays(request=.2)
        self._test_connect_timeout_under_200ms(loc)
        self.server.setdelays(request=2)
        self._test_response_timeout_under_2s(loc)

    def test_timeout_slow_response(self):
        loc = self.keepClient().put(self.DATA, copies=1, num_retries=0)
        self.server.setdelays(response=.2)
        self._test_connect_timeout_under_200ms(loc)
        self.server.setdelays(response=2)
        self._test_response_timeout_under_2s(loc)

    def test_timeout_slow_response_body(self):
        loc = self.keepClient().put(self.DATA, copies=1, num_retries=0)
        self.server.setdelays(response_body=.2)
        self._test_connect_timeout_under_200ms(loc)
        self.server.setdelays(response_body=2)
        self._test_response_timeout_under_2s(loc)

    def _test_connect_timeout_under_200ms(self, loc):
        # Allow 100ms to connect, then 1s for response. Everything
        # should work, and everything should take at least 200ms to
        # return.
        kc = self.keepClient(timeouts=(.1, 1))
        with self.assertTakesBetween(.2, .3):
            kc.put(self.DATA, copies=1, num_retries=0)
        with self.assertTakesBetween(.2, .3):
            self.assertEqual(self.DATA, kc.get(loc, num_retries=0))

    def _test_response_timeout_under_2s(self, loc):
        # Allow 10s to connect, then 1s for response. Nothing should
        # work, and everything should take at least 1s to return.
        kc = self.keepClient(timeouts=(10, 1))
        with self.assertTakesBetween(1, 1.9):
            with self.assertRaises(arvados.errors.KeepReadError):
                kc.get(loc, num_retries=0)
        with self.assertTakesBetween(1, 1.9):
            with self.assertRaises(arvados.errors.KeepWriteError):
                kc.put(self.DATA, copies=1, num_retries=0)


class KeepClientGatewayTestCase(unittest.TestCase, tutil.ApiClientMock):
    def mock_disks_and_gateways(self, disks=3, gateways=1):
        self.gateways = [{
                'uuid': 'zzzzz-bi6l4-gateway{:08d}'.format(i),
                'owner_uuid': 'zzzzz-tpzed-000000000000000',
                'service_host': 'gatewayhost{}'.format(i),
                'service_port': 12345,
                'service_ssl_flag': True,
                'service_type': 'gateway:test',
        } for i in range(gateways)]
        self.gateway_roots = [
            "https://{service_host}:{service_port}/".format(**gw)
            for gw in self.gateways]
        self.api_client = self.mock_keep_services(
            count=disks, additional_services=self.gateways)
        self.keepClient = arvados.KeepClient(api_client=self.api_client)

    @mock.patch('pycurl.Curl')
    def test_get_with_gateway_hint_first(self, MockCurl):
        MockCurl.return_value = tutil.FakeCurl.make(
            code=200, body='foo', headers={'Content-Length': 3})
        self.mock_disks_and_gateways()
        locator = 'acbd18db4cc2f85cedef654fccc4a4d8+3+K@' + self.gateways[0]['uuid']
        self.assertEqual('foo', self.keepClient.get(locator))
        self.assertEqual(self.gateway_roots[0]+locator,
                         MockCurl.return_value.getopt(pycurl.URL))
        self.assertEqual(True, self.keepClient.head(locator))

    @mock.patch('pycurl.Curl')
    def test_get_with_gateway_hints_in_order(self, MockCurl):
        gateways = 4
        disks = 3
        mocks = [
            tutil.FakeCurl.make(code=404, body='')
            for _ in range(gateways+disks)
        ]
        MockCurl.side_effect = tutil.queue_with(mocks)
        self.mock_disks_and_gateways(gateways=gateways, disks=disks)
        locator = '+'.join(['acbd18db4cc2f85cedef654fccc4a4d8+3'] +
                           ['K@'+gw['uuid'] for gw in self.gateways])
        with self.assertRaises(arvados.errors.NotFoundError):
            self.keepClient.get(locator)
        # Gateways are tried first, in the order given.
        for i, root in enumerate(self.gateway_roots):
            self.assertEqual(root+locator,
                             mocks[i].getopt(pycurl.URL))
        # Disk services are tried next.
        for i in range(gateways, gateways+disks):
            self.assertRegexpMatches(
                mocks[i].getopt(pycurl.URL),
                r'keep0x')

    @mock.patch('pycurl.Curl')
    def test_head_with_gateway_hints_in_order(self, MockCurl):
        gateways = 4
        disks = 3
        mocks = [
            tutil.FakeCurl.make(code=404, body='')
            for _ in range(gateways+disks)
        ]
        MockCurl.side_effect = tutil.queue_with(mocks)
        self.mock_disks_and_gateways(gateways=gateways, disks=disks)
        locator = '+'.join(['acbd18db4cc2f85cedef654fccc4a4d8+3'] +
                           ['K@'+gw['uuid'] for gw in self.gateways])
        with self.assertRaises(arvados.errors.NotFoundError):
            self.keepClient.head(locator)
        # Gateways are tried first, in the order given.
        for i, root in enumerate(self.gateway_roots):
            self.assertEqual(root+locator,
                             mocks[i].getopt(pycurl.URL))
        # Disk services are tried next.
        for i in range(gateways, gateways+disks):
            self.assertRegexpMatches(
                mocks[i].getopt(pycurl.URL),
                r'keep0x')

    @mock.patch('pycurl.Curl')
    def test_get_with_remote_proxy_hint(self, MockCurl):
        MockCurl.return_value = tutil.FakeCurl.make(
            code=200, body='foo', headers={'Content-Length': 3})
        self.mock_disks_and_gateways()
        locator = 'acbd18db4cc2f85cedef654fccc4a4d8+3+K@xyzzy'
        self.assertEqual('foo', self.keepClient.get(locator))
        self.assertEqual('https://keep.xyzzy.arvadosapi.com/'+locator,
                         MockCurl.return_value.getopt(pycurl.URL))

    @mock.patch('pycurl.Curl')
    def test_head_with_remote_proxy_hint(self, MockCurl):
        MockCurl.return_value = tutil.FakeCurl.make(
            code=200, body='foo', headers={'Content-Length': 3})
        self.mock_disks_and_gateways()
        locator = 'acbd18db4cc2f85cedef654fccc4a4d8+3+K@xyzzy'
        self.assertEqual(True, self.keepClient.head(locator))
        self.assertEqual('https://keep.xyzzy.arvadosapi.com/'+locator,
                         MockCurl.return_value.getopt(pycurl.URL))


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
    #
    # Test classes must define TEST_PATCHER to a method that mocks
    # out appropriate methods in the client.

    PROXY_ADDR = 'http://[%s]:65535/' % (tutil.TEST_HOST,)
    TEST_DATA = 'testdata'
    TEST_LOCATOR = 'ef654c40ab4f1747fc699915d4f70902+8'

    def setUp(self):
        self.client_kwargs = {'proxy': self.PROXY_ADDR, 'local_store': ''}

    def new_client(self, **caller_kwargs):
        kwargs = self.client_kwargs.copy()
        kwargs.update(caller_kwargs)
        return arvados.KeepClient(**kwargs)

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
        with self.TEST_PATCHER(self.DEFAULT_EXPECT, 200):
            self.check_success()

    def test_retry_then_success(self):
        with self.TEST_PATCHER(self.DEFAULT_EXPECT, 500, 200):
            self.check_success(num_retries=3)

    def test_exception_then_success(self):
        with self.TEST_PATCHER(self.DEFAULT_EXPECT, Exception('mock err'), 200):
            self.check_success(num_retries=3)

    def test_no_default_retry(self):
        with self.TEST_PATCHER(self.DEFAULT_EXPECT, 500, 200):
            self.check_exception()

    def test_no_retry_after_permanent_error(self):
        with self.TEST_PATCHER(self.DEFAULT_EXPECT, 403, 200):
            self.check_exception(num_retries=3)

    def test_error_after_retries_exhausted(self):
        with self.TEST_PATCHER(self.DEFAULT_EXPECT, 500, 500, 200):
            self.check_exception(num_retries=1)

    def test_num_retries_instance_fallback(self):
        self.client_kwargs['num_retries'] = 3
        with self.TEST_PATCHER(self.DEFAULT_EXPECT, 500, 200):
            self.check_success()


@tutil.skip_sleep
class KeepClientRetryGetTestCase(KeepClientRetryTestMixin, unittest.TestCase):
    DEFAULT_EXPECT = KeepClientRetryTestMixin.TEST_DATA
    DEFAULT_EXCEPTION = arvados.errors.KeepReadError
    HINTED_LOCATOR = KeepClientRetryTestMixin.TEST_LOCATOR + '+K@xyzzy'
    TEST_PATCHER = staticmethod(tutil.mock_keep_responses)

    def run_method(self, locator=KeepClientRetryTestMixin.TEST_LOCATOR,
                   *args, **kwargs):
        return self.new_client().get(locator, *args, **kwargs)

    def test_specific_exception_when_not_found(self):
        with tutil.mock_keep_responses(self.DEFAULT_EXPECT, 404, 200):
            self.check_exception(arvados.errors.NotFoundError, num_retries=3)

    def test_general_exception_with_mixed_errors(self):
        # get should raise a NotFoundError if no server returns the block,
        # and a high threshold of servers report that it's not found.
        # This test rigs up 50/50 disagreement between two servers, and
        # checks that it does not become a NotFoundError.
        client = self.new_client()
        with tutil.mock_keep_responses(self.DEFAULT_EXPECT, 404, 500):
            with self.assertRaises(arvados.errors.KeepReadError) as exc_check:
                client.get(self.HINTED_LOCATOR)
            self.assertNotIsInstance(
                exc_check.exception, arvados.errors.NotFoundError,
                "mixed errors raised NotFoundError")

    def test_hint_server_can_succeed_without_retries(self):
        with tutil.mock_keep_responses(self.DEFAULT_EXPECT, 404, 200, 500):
            self.check_success(locator=self.HINTED_LOCATOR)

    def test_try_next_server_after_timeout(self):
        with tutil.mock_keep_responses(
                (socket.timeout("timed out"), 200),
                (self.DEFAULT_EXPECT, 200)):
            self.check_success(locator=self.HINTED_LOCATOR)

    def test_retry_data_with_wrong_checksum(self):
        with tutil.mock_keep_responses(
                ('baddata', 200),
                (self.DEFAULT_EXPECT, 200)):
            self.check_success(locator=self.HINTED_LOCATOR)

@tutil.skip_sleep
class KeepClientRetryHeadTestCase(KeepClientRetryTestMixin, unittest.TestCase):
    DEFAULT_EXPECT = True
    DEFAULT_EXCEPTION = arvados.errors.KeepReadError
    HINTED_LOCATOR = KeepClientRetryTestMixin.TEST_LOCATOR + '+K@xyzzy'
    TEST_PATCHER = staticmethod(tutil.mock_keep_responses)

    def run_method(self, locator=KeepClientRetryTestMixin.TEST_LOCATOR,
                   *args, **kwargs):
        return self.new_client().head(locator, *args, **kwargs)

    def test_specific_exception_when_not_found(self):
        with tutil.mock_keep_responses(self.DEFAULT_EXPECT, 404, 200):
            self.check_exception(arvados.errors.NotFoundError, num_retries=3)

    def test_general_exception_with_mixed_errors(self):
        # head should raise a NotFoundError if no server returns the block,
        # and a high threshold of servers report that it's not found.
        # This test rigs up 50/50 disagreement between two servers, and
        # checks that it does not become a NotFoundError.
        client = self.new_client()
        with tutil.mock_keep_responses(self.DEFAULT_EXPECT, 404, 500):
            with self.assertRaises(arvados.errors.KeepReadError) as exc_check:
                client.head(self.HINTED_LOCATOR)
            self.assertNotIsInstance(
                exc_check.exception, arvados.errors.NotFoundError,
                "mixed errors raised NotFoundError")

    def test_hint_server_can_succeed_without_retries(self):
        with tutil.mock_keep_responses(self.DEFAULT_EXPECT, 404, 200, 500):
            self.check_success(locator=self.HINTED_LOCATOR)

    def test_try_next_server_after_timeout(self):
        with tutil.mock_keep_responses(
                (socket.timeout("timed out"), 200),
                (self.DEFAULT_EXPECT, 200)):
            self.check_success(locator=self.HINTED_LOCATOR)

@tutil.skip_sleep
class KeepClientRetryPutTestCase(KeepClientRetryTestMixin, unittest.TestCase):
    DEFAULT_EXPECT = KeepClientRetryTestMixin.TEST_LOCATOR
    DEFAULT_EXCEPTION = arvados.errors.KeepWriteError
    TEST_PATCHER = staticmethod(tutil.mock_keep_responses)

    def run_method(self, data=KeepClientRetryTestMixin.TEST_DATA,
                   copies=1, *args, **kwargs):
        return self.new_client().put(data, copies, *args, **kwargs)

    def test_do_not_send_multiple_copies_to_same_server(self):
        with tutil.mock_keep_responses(self.DEFAULT_EXPECT, 200):
            self.check_exception(copies=2, num_retries=3)


@tutil.skip_sleep
class RetryNeedsMultipleServices(unittest.TestCase, tutil.ApiClientMock):
    # Test put()s that need two distinct servers to succeed, possibly
    # requiring multiple passes through the retry loop.

    def setUp(self):
        self.api_client = self.mock_keep_services(count=2)
        self.keep_client = arvados.KeepClient(api_client=self.api_client)

    def test_success_after_exception(self):
        with tutil.mock_keep_responses(
                'acbd18db4cc2f85cedef654fccc4a4d8+3',
                Exception('mock err'), 200, 200) as req_mock:
            self.keep_client.put('foo', num_retries=1, copies=2)
        self.assertTrue(3, req_mock.call_count)

    def test_success_after_retryable_error(self):
        with tutil.mock_keep_responses(
                'acbd18db4cc2f85cedef654fccc4a4d8+3',
                500, 200, 200) as req_mock:
            self.keep_client.put('foo', num_retries=1, copies=2)
        self.assertTrue(3, req_mock.call_count)

    def test_fail_after_final_error(self):
        # First retry loop gets a 200 (can't achieve replication by
        # storing again on that server) and a 400 (can't retry that
        # server at all), so we shouldn't try a third request.
        with tutil.mock_keep_responses(
                'acbd18db4cc2f85cedef654fccc4a4d8+3',
                200, 400, 200) as req_mock:
            with self.assertRaises(arvados.errors.KeepWriteError):
                self.keep_client.put('foo', num_retries=1, copies=2)
        self.assertTrue(2, req_mock.call_count)
