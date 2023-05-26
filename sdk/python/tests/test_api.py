# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import absolute_import
from builtins import str
from builtins import range
import arvados
import collections
import contextlib
import httplib2
import itertools
import json
import mimetypes
import os
import socket
import string
import sys
import unittest
import urllib.parse as urlparse

import mock
from . import run_test_server

from apiclient import errors as apiclient_errors
from apiclient import http as apiclient_http
from arvados.api import (
    api_client,
    normalize_api_kwargs,
    api_kwargs_from_config,
    OrderedJsonModel,
)
from .arvados_testutil import fake_httplib2_response, mock_api_responses, queue_with

if not mimetypes.inited:
    mimetypes.init()

class ArvadosApiTest(run_test_server.TestCaseWithServers):
    MAIN_SERVER = {}
    ERROR_HEADERS = {'Content-Type': mimetypes.types_map['.json']}
    RETRIED_4XX = frozenset([408, 409, 423])

    def api_error_response(self, code, *errors):
        return (fake_httplib2_response(code, **self.ERROR_HEADERS),
                json.dumps({'errors': errors,
                            'error_token': '1234567890+12345678'}).encode())

    def _config_from_environ(self):
        return {
            key: value
            for key, value in os.environ.items()
            if key.startswith('ARVADOS_API_')
        }

    def _discoveryServiceUrl(
            self,
            host=None,
            path='/discovery/v1/apis/{api}/{apiVersion}/rest',
            scheme='https',
    ):
        if host is None:
            host = os.environ['ARVADOS_API_HOST']
        return urlparse.urlunsplit((scheme, host, path, None, None))

    def test_new_api_objects_with_cache(self):
        clients = [arvados.api('v1', cache=True) for index in [0, 1]]
        self.assertIsNot(*clients)

    def test_empty_list(self):
        answer = arvados.api('v1').humans().list(
            filters=[['uuid', '=', None]]).execute()
        self.assertEqual(answer['items_available'], len(answer['items']))

    def test_nonempty_list(self):
        answer = arvados.api('v1').collections().list().execute()
        self.assertNotEqual(0, answer['items_available'])
        self.assertNotEqual(0, len(answer['items']))

    def test_timestamp_inequality_filter(self):
        api = arvados.api('v1')
        new_item = api.specimens().create(body={}).execute()
        for operator, should_include in [
                ['<', False], ['>', False],
                ['<=', True], ['>=', True], ['=', True]]:
            response = api.specimens().list(filters=[
                ['created_at', operator, new_item['created_at']],
                # Also filter by uuid to ensure (if it matches) it's on page 0
                ['uuid', '=', new_item['uuid']]]).execute()
            uuids = [item['uuid'] for item in response['items']]
            did_include = new_item['uuid'] in uuids
            self.assertEqual(
                did_include, should_include,
                "'%s %s' filter should%s have matched '%s'" % (
                    operator, new_item['created_at'],
                    ('' if should_include else ' not'),
                    new_item['created_at']))

    def test_exceptions_include_errors(self):
        mock_responses = {
            'arvados.humans.get': self.api_error_response(
                422, "Bad UUID format", "Bad output format"),
            }
        req_builder = apiclient_http.RequestMockBuilder(mock_responses)
        api = arvados.api('v1', requestBuilder=req_builder)
        with self.assertRaises(apiclient_errors.HttpError) as err_ctx:
            api.humans().get(uuid='xyz-xyz-abcdef').execute()
        err_s = str(err_ctx.exception)
        for msg in ["Bad UUID format", "Bad output format"]:
            self.assertIn(msg, err_s)

    @mock.patch('time.sleep')
    def test_exceptions_include_request_id(self, sleep):
        api = arvados.api('v1')
        api.request_id='fake-request-id'
        api._http.orig_http_request = mock.MagicMock()
        api._http.orig_http_request.side_effect = socket.error('mock error')
        caught = None
        try:
            api.users().current().execute()
        except Exception as e:
            caught = e
        self.assertRegex(str(caught), r'fake-request-id')

    def test_exceptions_without_errors_have_basic_info(self):
        mock_responses = {
            'arvados.humans.delete': (
                fake_httplib2_response(500, **self.ERROR_HEADERS),
                b"")
            }
        req_builder = apiclient_http.RequestMockBuilder(mock_responses)
        api = arvados.api('v1', requestBuilder=req_builder)
        with self.assertRaises(apiclient_errors.HttpError) as err_ctx:
            api.humans().delete(uuid='xyz-xyz-abcdef').execute()
        self.assertIn("500", str(err_ctx.exception))

    def test_request_too_large(self):
        api = arvados.api('v1')
        maxsize = api._rootDesc.get('maxRequestSize', 0)
        with self.assertRaises(apiclient_errors.MediaUploadSizeError):
            text = "X" * maxsize
            arvados.api('v1').collections().create(body={"manifest_text": text}).execute()

    def test_default_request_timeout(self):
        api = arvados.api('v1')
        self.assertEqual(api._http.timeout, 300,
            "Default timeout value should be 300")

    def test_custom_request_timeout(self):
        api = arvados.api('v1', timeout=1234)
        self.assertEqual(api._http.timeout, 1234,
            "Requested timeout value was 1234")

    def test_4xx_retried(self):
        client = arvados.api('v1')
        for code in self.RETRIED_4XX:
            name = f'retried #{code}'
            with self.subTest(name), mock.patch('time.sleep'):
                expected = {'username': name}
                with mock_api_responses(
                        client,
                        json.dumps(expected),
                        [code, code, 200],
                        self.ERROR_HEADERS,
                        'orig_http_request',
                ):
                    actual = client.users().current().execute()
                self.assertEqual(actual, expected)

    def test_4xx_not_retried(self):
        client = arvados.api('v1', num_retries=3)
        # Note that googleapiclient does retry 403 *if* the response JSON
        # includes flags that say the request was denied by rate limiting.
        # An empty JSON response like we use here should not be retried.
        for code in [400, 401, 403, 404, 422]:
            with self.subTest(f'error {code}'), mock.patch('time.sleep'):
                with mock_api_responses(
                        client,
                        b'{}',
                        [code, 200],
                        self.ERROR_HEADERS,
                        'orig_http_request',
                ), self.assertRaises(arvados.errors.ApiError) as exc_check:
                    client.users().current().execute()
                response = exc_check.exception.args[0]
                self.assertEqual(response.status, code)
                self.assertEqual(response.get('status'), str(code))

    def test_4xx_raised_after_retry_exhaustion(self):
        client = arvados.api('v1', num_retries=1)
        for code in self.RETRIED_4XX:
            with self.subTest(f'failed {code}'), mock.patch('time.sleep'):
                with mock_api_responses(
                        client,
                        b'{}',
                        [code, code, code, 200],
                        self.ERROR_HEADERS,
                        'orig_http_request',
                ), self.assertRaises(arvados.errors.ApiError) as exc_check:
                    client.users().current().execute()
                response = exc_check.exception.args[0]
                self.assertEqual(response.status, code)
                self.assertEqual(response.get('status'), str(code))

    def test_ordered_json_model(self):
        mock_responses = {
            'arvados.humans.get': (
                None,
                json.dumps(collections.OrderedDict(
                    (c, int(c, 16)) for c in string.hexdigits
                )).encode(),
            ),
        }
        req_builder = apiclient_http.RequestMockBuilder(mock_responses)
        api = arvados.api('v1',
                          requestBuilder=req_builder, model=OrderedJsonModel())
        result = api.humans().get(uuid='test').execute()
        self.assertEqual(string.hexdigits, ''.join(list(result.keys())))

    def test_api_is_threadsafe(self):
        api_kwargs = {
            'host': os.environ['ARVADOS_API_HOST'],
            'token': os.environ['ARVADOS_API_TOKEN'],
            'insecure': True,
        }
        config_kwargs = {'apiconfig': os.environ}
        for api_constructor, kwargs in [
                (arvados.api, {}),
                (arvados.api, api_kwargs),
                (arvados.api_from_config, {}),
                (arvados.api_from_config, config_kwargs),
        ]:
            sub_kwargs = "kwargs" if kwargs else "no kwargs"
            with self.subTest(f"{api_constructor.__name__} with {sub_kwargs}"):
                api_client = api_constructor('v1', **kwargs)
                self.assertTrue(hasattr(api_client, 'localapi'),
                                f"client missing localapi method")
                self.assertTrue(hasattr(api_client, 'keep'),
                                f"client missing keep attribute")

    def test_api_host_constructor(self):
        cache = True
        insecure = True
        client = arvados.api(
            'v1',
            cache,
            os.environ['ARVADOS_API_HOST'],
            os.environ['ARVADOS_API_TOKEN'],
            insecure,
        )
        self.assertEqual(client.api_token, os.environ['ARVADOS_API_TOKEN'],
                         "client constructed with incorrect token")

    def test_api_url_constructor(self):
        client = arvados.api(
            'v1',
            discoveryServiceUrl=self._discoveryServiceUrl(),
            token=os.environ['ARVADOS_API_TOKEN'],
            insecure=True,
        )
        self.assertEqual(client.api_token, os.environ['ARVADOS_API_TOKEN'],
                         "client constructed with incorrect token")

    def test_api_bad_args(self):
        all_kwargs = {
            'host': os.environ['ARVADOS_API_HOST'],
            'token': os.environ['ARVADOS_API_TOKEN'],
            'discoveryServiceUrl': self._discoveryServiceUrl(),
        }
        for use_keys in [
                # Passing only a single key is missing required info
                *([key] for key in all_kwargs.keys()),
                # Passing all keys is a conflict
                list(all_kwargs.keys()),
        ]:
            kwargs = {key: all_kwargs[key] for key in use_keys}
            kwargs_list = ', '.join(use_keys)
            with self.subTest(f"calling arvados.api with {kwargs_list} fails"), \
                 self.assertRaises(ValueError):
                arvados.api('v1', insecure=True, **kwargs)

    def test_api_bad_url(self):
        for bad_kwargs in [
                {'discoveryServiceUrl': self._discoveryServiceUrl() + '/BadTestURL'},
                {'version': 'BadTestVersion', 'host': os.environ['ARVADOS_API_HOST']},
        ]:
            bad_key = next(iter(bad_kwargs))
            with self.subTest(f"api fails with bad {bad_key}"), \
                 self.assertRaises(apiclient_errors.UnknownApiNameOrVersion):
                arvados.api(**bad_kwargs, token='test_api_bad_url', insecure=True)

    def test_normalize_api_good_args(self):
        for version, discoveryServiceUrl, host in [
                ('Test1', None, os.environ['ARVADOS_API_HOST']),
                (None, self._discoveryServiceUrl(), None)
        ]:
            argname = 'discoveryServiceUrl' if host is None else 'host'
            with self.subTest(f"normalize_api_kwargs with {argname}"):
                actual = normalize_api_kwargs(
                    version,
                    discoveryServiceUrl,
                    host,
                    os.environ['ARVADOS_API_TOKEN'],
                    insecure=True,
                )
                self.assertEqual(actual['discoveryServiceUrl'], self._discoveryServiceUrl())
                self.assertEqual(actual['token'], os.environ['ARVADOS_API_TOKEN'])
                self.assertEqual(actual['version'], version or 'v1')
                self.assertTrue(actual['insecure'])
                self.assertNotIn('host', actual)

    def test_normalize_api_bad_args(self):
        all_args = (
            self._discoveryServiceUrl(),
            os.environ['ARVADOS_API_HOST'],
            os.environ['ARVADOS_API_TOKEN'],
        )
        for arg_index, arg_value in enumerate(all_args):
            args = [None] * len(all_args)
            args[arg_index] = arg_value
            with self.subTest(f"normalize_api_kwargs with only arg #{arg_index + 1}"), \
                 self.assertRaises(ValueError):
                normalize_api_kwargs('v1', *args)
        with self.subTest("normalize_api_kwargs with discoveryServiceUrl and host"), \
             self.assertRaises(ValueError):
            normalize_api_kwargs('v1', *all_args)

    def test_api_from_config_default(self):
        client = arvados.api_from_config('v1')
        self.assertEqual(client.api_token, os.environ['ARVADOS_API_TOKEN'],
                         "client constructed with incorrect token")

    def test_api_from_config_explicit(self):
        config = self._config_from_environ()
        client = arvados.api_from_config('v1', config)
        self.assertEqual(client.api_token, os.environ['ARVADOS_API_TOKEN'],
                         "client constructed with incorrect token")

    def test_api_from_bad_config(self):
        base_config = self._config_from_environ()
        for del_key in ['ARVADOS_API_HOST', 'ARVADOS_API_TOKEN']:
            with self.subTest(f"api_from_config without {del_key} fails"), \
                 self.assertRaises(ValueError):
                config = dict(base_config)
                del config[del_key]
                arvados.api_from_config('v1', config)

    def test_api_kwargs_from_good_config(self):
        for config in [None, self._config_from_environ()]:
            conf_type = 'default' if config is None else 'passed'
            with self.subTest(f"api_kwargs_from_config with {conf_type} config"):
                version = 'Test1' if config else None
                actual = api_kwargs_from_config(version, config)
                self.assertEqual(actual['discoveryServiceUrl'], self._discoveryServiceUrl())
                self.assertEqual(actual['token'], os.environ['ARVADOS_API_TOKEN'])
                self.assertEqual(actual['version'], version or 'v1')
                self.assertTrue(actual['insecure'])
                self.assertNotIn('host', actual)

    def test_api_kwargs_from_bad_config(self):
        base_config = self._config_from_environ()
        for del_key in ['ARVADOS_API_HOST', 'ARVADOS_API_TOKEN']:
            with self.subTest(f"api_kwargs_from_config without {del_key} fails"), \
                 self.assertRaises(ValueError):
                config = dict(base_config)
                del config[del_key]
                api_kwargs_from_config('v1', config)

    def test_api_client_constructor(self):
        client = api_client(
            'v1',
            self._discoveryServiceUrl(),
            os.environ['ARVADOS_API_TOKEN'],
            insecure=True,
        )
        self.assertEqual(client.api_token, os.environ['ARVADOS_API_TOKEN'],
                         "client constructed with incorrect token")
        self.assertFalse(
            hasattr(client, 'localapi'),
            "client has localapi method when it should not be thread-safe",
        )

    def test_api_client_bad_url(self):
        all_args = ('v1', self._discoveryServiceUrl(), 'test_api_client_bad_url')
        for arg_index, arg_value in [
                (0, 'BadTestVersion'),
                (1, all_args[1] + '/BadTestURL'),
        ]:
            with self.subTest(f"api_client fails with {arg_index}={arg_value!r}"), \
                 self.assertRaises(apiclient_errors.UnknownApiNameOrVersion):
                args = list(all_args)
                args[arg_index] = arg_value
                api_client(*args, insecure=True)


class ConstructNumRetriesTestCase(unittest.TestCase):
    @staticmethod
    def _fake_retry_request(http, num_retries, req_type, sleep, rand, uri, method, *args, **kwargs):
        return http.request(uri, method, *args, **kwargs)

    @contextlib.contextmanager
    def patch_retry(self):
        # We have this dedicated context manager that goes through `sys.modules`
        # instead of just using `mock.patch` because of the unfortunate
        # `arvados.api` name collision.
        orig_func = sys.modules['arvados.api']._orig_retry_request
        expect_name = 'googleapiclient.http._retry_request'
        self.assertEqual(
            '{0.__module__}.{0.__name__}'.format(orig_func), expect_name,
            f"test setup problem: {expect_name} not at arvados.api._orig_retry_request",
        )
        retry_mock = mock.Mock(wraps=self._fake_retry_request)
        sys.modules['arvados.api']._orig_retry_request = retry_mock
        try:
            yield retry_mock
        finally:
            sys.modules['arvados.api']._orig_retry_request = orig_func

    def _iter_num_retries(self, retry_mock):
        for call in retry_mock.call_args_list:
            try:
                yield call.args[1]
            except IndexError:
                yield call.kwargs['num_retries']

    def test_default_num_retries(self):
        with self.patch_retry() as retry_mock:
            client = arvados.api('v1')
        actual = set(self._iter_num_retries(retry_mock))
        self.assertEqual(len(actual), 1)
        self.assertTrue(actual.pop() > 6, "num_retries lower than expected")

    def _test_calls(self, init_arg, call_args, expected):
        with self.patch_retry() as retry_mock:
            client = arvados.api('v1', num_retries=init_arg)
            for num_retries in call_args:
                client.users().current().execute(num_retries=num_retries)
        actual = self._iter_num_retries(retry_mock)
        # The constructor makes two requests with its num_retries argument:
        # one for the discovery document, and one for the config.
        self.assertEqual(next(actual, None), init_arg)
        self.assertEqual(next(actual, None), init_arg)
        self.assertEqual(list(actual), expected)

    def test_discovery_num_retries(self):
        for num_retries in [0, 5, 55]:
            with self.subTest(f"num_retries={num_retries}"):
                self._test_calls(num_retries, [], [])

    def test_num_retries_called_le_init(self):
        for n in [6, 10]:
            with self.subTest(f"init_arg={n}"):
                call_args = [n - 4, n - 2, n]
                expected = [n] * 3
                self._test_calls(n, call_args, expected)

    def test_num_retries_called_ge_init(self):
        for n in [0, 10]:
            with self.subTest(f"init_arg={n}"):
                call_args = [n, n + 4, n + 8]
                self._test_calls(n, call_args, call_args)

    def test_num_retries_called_mixed(self):
        self._test_calls(5, [2, 6, 4, 8], [5, 6, 5, 8])


class PreCloseSocketTestCase(unittest.TestCase):
    def setUp(self):
        self.api = arvados.api('v1')
        self.assertTrue(hasattr(self.api._http, 'orig_http_request'),
                        "test doesn't know how to intercept HTTP requests")
        self.mock_response = {'user': 'person'}
        self.request_success = (fake_httplib2_response(200),
                                json.dumps(self.mock_response))
        self.api._http.orig_http_request = mock.MagicMock()
        # All requests succeed by default. Tests override as needed.
        self.api._http.orig_http_request.return_value = self.request_success

    @mock.patch('time.time', side_effect=[i*2**20 for i in range(99)])
    def test_close_old_connections_non_retryable(self, sleep):
        self._test_connection_close(expect=1)

    @mock.patch('time.time', side_effect=itertools.count())
    def test_no_close_fresh_connections_non_retryable(self, sleep):
        self._test_connection_close(expect=0)

    @mock.patch('time.time', side_effect=itertools.count())
    def test_override_max_idle_time(self, sleep):
        self.api._http._max_keepalive_idle = 0
        self._test_connection_close(expect=1)

    def _test_connection_close(self, expect=0):
        # Do two POST requests. The second one must close all
        # connections +expect+ times.
        self.api.users().create(body={}).execute()
        mock_conns = {str(i): mock.MagicMock() for i in range(2)}
        self.api._http.connections = mock_conns.copy()
        self.api.users().create(body={}).execute()
        for c in mock_conns.values():
            self.assertEqual(c.close.call_count, expect)


if __name__ == '__main__':
    unittest.main()
