# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import absolute_import
from builtins import str
from builtins import range
import arvados
import collections
import httplib2
import itertools
import json
import mimetypes
import os
import socket
import string
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
    RETRY_DELAY_INITIAL,
    RETRY_DELAY_BACKOFF,
    RETRY_COUNT,
)
from .arvados_testutil import fake_httplib2_response, queue_with

if not mimetypes.inited:
    mimetypes.init()

class ArvadosApiTest(run_test_server.TestCaseWithServers):
    MAIN_SERVER = {}
    ERROR_HEADERS = {'Content-Type': mimetypes.types_map['.json']}

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


class RetryREST(unittest.TestCase):
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

    @mock.patch('time.sleep')
    def test_socket_error_retry_get(self, sleep):
        self.api._http.orig_http_request.side_effect = (
            socket.error('mock error'),
            self.request_success,
        )
        self.assertEqual(self.api.users().current().execute(),
                         self.mock_response)
        self.assertGreater(self.api._http.orig_http_request.call_count, 1,
                           "client got the right response without retrying")
        self.assertEqual(sleep.call_args_list,
                         [mock.call(RETRY_DELAY_INITIAL)])

    @mock.patch('time.sleep')
    def test_same_automatic_request_id_on_retry(self, sleep):
        self.api._http.orig_http_request.side_effect = (
            socket.error('mock error'),
            self.request_success,
        )
        self.api.users().current().execute()
        calls = self.api._http.orig_http_request.call_args_list
        self.assertEqual(len(calls), 2)
        self.assertEqual(
            calls[0][1]['headers']['X-Request-Id'],
            calls[1][1]['headers']['X-Request-Id'])
        self.assertRegex(calls[0][1]['headers']['X-Request-Id'], r'^req-[a-z0-9]{20}$')

    @mock.patch('time.sleep')
    def test_provided_request_id_on_retry(self, sleep):
        self.api.request_id='fake-request-id'
        self.api._http.orig_http_request.side_effect = (
            socket.error('mock error'),
            self.request_success,
        )
        self.api.users().current().execute()
        calls = self.api._http.orig_http_request.call_args_list
        self.assertEqual(len(calls), 2)
        for call in calls:
            self.assertEqual(call[1]['headers']['X-Request-Id'], 'fake-request-id')

    @mock.patch('time.sleep')
    def test_socket_error_retry_delay(self, sleep):
        self.api._http.orig_http_request.side_effect = socket.error('mock')
        self.api._http._retry_count = 3
        with self.assertRaises(socket.error):
            self.api.users().current().execute()
        self.assertEqual(self.api._http.orig_http_request.call_count, 4)
        self.assertEqual(sleep.call_args_list, [
            mock.call(RETRY_DELAY_INITIAL),
            mock.call(RETRY_DELAY_INITIAL * RETRY_DELAY_BACKOFF),
            mock.call(RETRY_DELAY_INITIAL * RETRY_DELAY_BACKOFF**2),
        ])

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

    @mock.patch('time.sleep')
    def test_socket_error_no_retry_post(self, sleep):
        self.api._http.orig_http_request.side_effect = (
            socket.error('mock error'),
            self.request_success,
        )
        with self.assertRaises(socket.error):
            self.api.users().create(body={}).execute()
        self.assertEqual(self.api._http.orig_http_request.call_count, 1,
                         "client should try non-retryable method exactly once")
        self.assertEqual(sleep.call_args_list, [])


if __name__ == '__main__':
    unittest.main()
