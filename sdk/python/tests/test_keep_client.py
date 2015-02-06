import hashlib
import mock
import os
import re
import socket
import unittest
import urlparse

import arvados
import arvados.retry
import arvados_testutil as tutil
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

    def test_KeepEmptyCollectionTest(self):
        blob_locator = self.keep_client.put('', copies=1)
        self.assertRegexpMatches(
            blob_locator,
            '^d41d8cd98f00b204e9800998ecf8427e\+0',
            ('wrong locator from Keep.put(""): ' + blob_locator))


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


class KeepClientServiceTestCase(unittest.TestCase):
    def mock_keep_services(self, *services):
        api_client = mock.MagicMock(name='api_client')
        api_client.keep_services().accessible().execute.return_value = {
            'items_available': len(services),
            'items': [{
                    'uuid': 'zzzzz-bi6l4-{:015x}'.format(index),
                    'owner_uuid': 'zzzzz-tpzed-000000000000000',
                    'service_host': host,
                    'service_port': port,
                    'service_ssl_flag': ssl,
                    'service_type': servtype,
                    } for index, (host, port, ssl, servtype)
                      in enumerate(services)],
            }
        return api_client

    def mock_n_keep_disks(self, service_count):
        return self.mock_keep_services(
            *[("keep0x{:x}".format(index), 80, False, 'disk')
              for index in range(service_count)])

    def get_service_roots(self, *services):
        api_client = self.mock_keep_services(*services)
        keep_client = arvados.KeepClient(api_client=api_client)
        services = keep_client.weighted_service_roots('000000')
        return [urlparse.urlparse(url) for url in sorted(services)]

    def test_ssl_flag_respected_in_roots(self):
        services = self.get_service_roots(('keep', 10, False, 'disk'),
                                          ('keep', 20, True, 'disk'))
        self.assertEqual(10, services[0].port)
        self.assertEqual('http', services[0].scheme)
        self.assertEqual(20, services[1].port)
        self.assertEqual('https', services[1].scheme)

    def test_correct_ports_with_ipv6_addresses(self):
        service = self.get_service_roots(('100::1', 10, True, 'proxy'))[0]
        self.assertEqual('100::1', service.hostname)
        self.assertEqual(10, service.port)

    # test_get_timeout and test_put_timeout test that
    # KeepClient.get and KeepClient.put use the appropriate timeouts
    # when connected directly to a Keep server (i.e. non-proxy timeout)

    def test_get_timeout(self):
        api_client = self.mock_keep_services(('keep', 10, False, 'disk'))
        keep_client = arvados.KeepClient(api_client=api_client)
        force_timeout = [socket.timeout("timed out")]
        with mock.patch('requests.get', side_effect=force_timeout) as mock_request:
            with self.assertRaises(arvados.errors.KeepReadError):
                keep_client.get('ffffffffffffffffffffffffffffffff')
            self.assertTrue(mock_request.called)
            self.assertEqual(
                arvados.KeepClient.DEFAULT_TIMEOUT,
                mock_request.call_args[1]['timeout'])

    def test_put_timeout(self):
        api_client = self.mock_keep_services(('keep', 10, False, 'disk'))
        keep_client = arvados.KeepClient(api_client=api_client)
        force_timeout = [socket.timeout("timed out")]
        with mock.patch('requests.put', side_effect=force_timeout) as mock_request:
            with self.assertRaises(arvados.errors.KeepWriteError):
                keep_client.put('foo')
            self.assertTrue(mock_request.called)
            self.assertEqual(
                arvados.KeepClient.DEFAULT_TIMEOUT,
                mock_request.call_args[1]['timeout'])

    def test_proxy_get_timeout(self):
        # Force a timeout, verifying that the requests.get or
        # requests.put method was called with the proxy_timeout
        # setting rather than the default timeout.
        api_client = self.mock_keep_services(('keep', 10, False, 'proxy'))
        keep_client = arvados.KeepClient(api_client=api_client)
        force_timeout = [socket.timeout("timed out")]
        with mock.patch('requests.get', side_effect=force_timeout) as mock_request:
            with self.assertRaises(arvados.errors.KeepReadError):
                keep_client.get('ffffffffffffffffffffffffffffffff')
            self.assertTrue(mock_request.called)
            self.assertEqual(
                arvados.KeepClient.DEFAULT_PROXY_TIMEOUT,
                mock_request.call_args[1]['timeout'])

    def test_proxy_put_timeout(self):
        # Force a timeout, verifying that the requests.get or
        # requests.put method was called with the proxy_timeout
        # setting rather than the default timeout.
        api_client = self.mock_keep_services(('keep', 10, False, 'proxy'))
        keep_client = arvados.KeepClient(api_client=api_client)
        force_timeout = [socket.timeout("timed out")]
        with mock.patch('requests.put', side_effect=force_timeout) as mock_request:
            with self.assertRaises(arvados.errors.KeepWriteError):
                keep_client.put('foo')
            self.assertTrue(mock_request.called)
            self.assertEqual(
                arvados.KeepClient.DEFAULT_PROXY_TIMEOUT,
                mock_request.call_args[1]['timeout'])

    def test_probe_order_reference_set(self):
        # expected_order[i] is the probe order for
        # hash=md5(sprintf("%064x",i)) where there are 16 services
        # with uuid sprintf("anything-%015x",j) with j in 0..15. E.g.,
        # the first probe for the block consisting of 64 "0"
        # characters is the service whose uuid is
        # "zzzzz-bi6l4-000000000000003", so expected_order[0][0]=='3'.
        expected_order = [
            list('3eab2d5fc9681074'),
            list('097dba52e648f1c3'),
            list('c5b4e023f8a7d691'),
            list('9d81c02e76a3bf54'),
            ]
        hashes = [
            hashlib.md5("{:064x}".format(x)).hexdigest()
            for x in range(len(expected_order))]
        api_client = self.mock_n_keep_disks(16)
        keep_client = arvados.KeepClient(api_client=api_client)
        for i, hash in enumerate(hashes):
            roots = keep_client.weighted_service_roots(hash)
            got_order = [
                re.search(r'//\[?keep0x([0-9a-f]+)', root).group(1)
                for root in roots]
            self.assertEqual(expected_order[i], got_order)

    def test_probe_waste_adding_one_server(self):
        hashes = [
            hashlib.md5("{:064x}".format(x)).hexdigest() for x in range(100)]
        initial_services = 12
        api_client = self.mock_n_keep_disks(initial_services)
        keep_client = arvados.KeepClient(api_client=api_client)
        probes_before = [
            keep_client.weighted_service_roots(hash) for hash in hashes]
        for added_services in range(1, 12):
            api_client = self.mock_n_keep_disks(initial_services+added_services)
            keep_client = arvados.KeepClient(api_client=api_client)
            total_penalty = 0
            for hash_index in range(len(hashes)):
                probe_after = keep_client.weighted_service_roots(
                    hashes[hash_index])
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
            data = hashlib.md5(data).hexdigest() + '+1234'
        api_client = self.mock_n_keep_disks(16)
        keep_client = arvados.KeepClient(api_client=api_client)
        with mock.patch('requests.' + verb,
                        side_effect=socket.timeout) as req_mock, \
                self.assertRaises(exc_class) as err_check:
            getattr(keep_client, verb)(data)
        urls = [urlparse.urlparse(url)
                for url in err_check.exception.service_errors()]
        self.assertEqual([('keep0x' + c, 80) for c in '3eab2d5fc9681074'],
                         [(url.hostname, url.port) for url in urls])

    def test_get_error_shows_probe_order(self):
        self.check_64_zeros_error_order('get', arvados.errors.KeepReadError)

    def test_put_error_shows_probe_order(self):
        self.check_64_zeros_error_order('put', arvados.errors.KeepWriteError)

    def check_no_services_error(self, verb, exc_class):
        api_client = mock.MagicMock(name='api_client')
        api_client.keep_services().accessible().execute.side_effect = (
            arvados.errors.ApiError)
        keep_client = arvados.KeepClient(api_client=api_client)
        with self.assertRaises(exc_class) as err_check:
            getattr(keep_client, verb)('d41d8cd98f00b204e9800998ecf8427e+0')
        self.assertEqual(0, len(err_check.exception.service_errors()))

    def test_get_error_with_no_services(self):
        self.check_no_services_error('get', arvados.errors.KeepReadError)

    def test_put_error_with_no_services(self):
        self.check_no_services_error('put', arvados.errors.KeepWriteError)

    def check_errors_from_last_retry(self, verb, exc_class):
        api_client = self.mock_n_keep_disks(2)
        keep_client = arvados.KeepClient(api_client=api_client)
        req_mock = getattr(tutil, 'mock_{}_responses'.format(verb))(
            "retry error reporting test", 500, 500, 403, 403)
        with req_mock, tutil.skip_sleep, \
                self.assertRaises(exc_class) as err_check:
            getattr(keep_client, verb)('d41d8cd98f00b204e9800998ecf8427e+0',
                                       num_retries=3)
        self.assertEqual([403, 403], [
                getattr(error, 'status_code', None)
                for error in err_check.exception.service_errors().itervalues()])

    def test_get_error_reflects_last_retry(self):
        self.check_errors_from_last_retry('get', arvados.errors.KeepReadError)

    def test_put_error_reflects_last_retry(self):
        self.check_errors_from_last_retry('put', arvados.errors.KeepWriteError)

    def test_put_error_does_not_include_successful_puts(self):
        data = 'partial failure test'
        data_loc = '{}+{}'.format(hashlib.md5(data).hexdigest(), len(data))
        api_client = self.mock_n_keep_disks(3)
        keep_client = arvados.KeepClient(api_client=api_client)
        with tutil.mock_put_responses(data_loc, 200, 500, 500) as req_mock, \
                self.assertRaises(arvados.errors.KeepWriteError) as exc_check:
            keep_client.put(data)
        self.assertEqual(2, len(exc_check.exception.service_errors()))


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
    TEST_PATCHER = staticmethod(tutil.mock_get_responses)

    def run_method(self, locator=KeepClientRetryTestMixin.TEST_LOCATOR,
                   *args, **kwargs):
        return self.new_client().get(locator, *args, **kwargs)

    def test_specific_exception_when_not_found(self):
        with tutil.mock_get_responses(self.DEFAULT_EXPECT, 404, 200):
            self.check_exception(arvados.errors.NotFoundError, num_retries=3)

    def test_general_exception_with_mixed_errors(self):
        # get should raise a NotFoundError if no server returns the block,
        # and a high threshold of servers report that it's not found.
        # This test rigs up 50/50 disagreement between two servers, and
        # checks that it does not become a NotFoundError.
        client = self.new_client()
        with tutil.mock_get_responses(self.DEFAULT_EXPECT, 404, 500):
            with self.assertRaises(arvados.errors.KeepReadError) as exc_check:
                client.get(self.HINTED_LOCATOR)
            self.assertNotIsInstance(
                exc_check.exception, arvados.errors.NotFoundError,
                "mixed errors raised NotFoundError")

    def test_hint_server_can_succeed_without_retries(self):
        with tutil.mock_get_responses(self.DEFAULT_EXPECT, 404, 200, 500):
            self.check_success(locator=self.HINTED_LOCATOR)

    def test_try_next_server_after_timeout(self):
        side_effects = [
            socket.timeout("timed out"),
            tutil.fake_requests_response(200, self.DEFAULT_EXPECT)]
        with mock.patch('requests.get',
                        side_effect=iter(side_effects)):
            self.check_success(locator=self.HINTED_LOCATOR)

    def test_retry_data_with_wrong_checksum(self):
        side_effects = (tutil.fake_requests_response(200, s)
                        for s in ['baddata', self.TEST_DATA])
        with mock.patch('requests.get', side_effect=side_effects):
            self.check_success(locator=self.HINTED_LOCATOR)


@tutil.skip_sleep
class KeepClientRetryPutTestCase(KeepClientRetryTestMixin, unittest.TestCase):
    DEFAULT_EXPECT = KeepClientRetryTestMixin.TEST_LOCATOR
    DEFAULT_EXCEPTION = arvados.errors.KeepWriteError
    TEST_PATCHER = staticmethod(tutil.mock_put_responses)

    def run_method(self, data=KeepClientRetryTestMixin.TEST_DATA,
                   copies=1, *args, **kwargs):
        return self.new_client().put(data, copies, *args, **kwargs)

    def test_do_not_send_multiple_copies_to_same_server(self):
        with tutil.mock_put_responses(self.DEFAULT_EXPECT, 200):
            self.check_exception(copies=2, num_retries=3)
