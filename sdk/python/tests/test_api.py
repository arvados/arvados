#!/usr/bin/env python

import apiclient.errors
import arvados
import httplib2
import json
import mimetypes
import unittest

from apiclient.http import RequestMockBuilder
from httplib import responses as HTTP_RESPONSES

if not mimetypes.inited:
    mimetypes.init()

class ArvadosApiClientTest(unittest.TestCase):
    @classmethod
    def response_from_code(cls, code):
        return httplib2.Response(
            {'status': code,
             'reason': HTTP_RESPONSES.get(code, "Unknown Response"),
             'Content-Type': mimetypes.types_map['.json']})

    @classmethod
    def api_error_response(cls, code, *errors):
        return (cls.response_from_code(code),
                json.dumps({'errors': errors,
                            'error_token': '1234567890+12345678'}))

    @classmethod
    def setUpClass(cls):
        # The apiclient library has support for mocking requests for
        # testing, but it doesn't extend to the discovery document
        # itself.  Point it at a known stable discovery document for now.
        # FIXME: Figure out a better way to stub this out.
        cls.orig_api_host = arvados.config.get('ARVADOS_API_HOST')
        arvados.config.settings()['ARVADOS_API_HOST'] = 'qr1hi.arvadosapi.com'
        mock_responses = {
            'arvados.humans.delete': (cls.response_from_code(500), ""),
            'arvados.humans.get': cls.api_error_response(
                422, "Bad UUID format", "Bad output format"),
            'arvados.humans.list': (None, json.dumps(
                    {'items_available': 0, 'items': []})),
            }
        req_builder = RequestMockBuilder(mock_responses)
        cls.api = arvados.api('v1', False, requestBuilder=req_builder)

    @classmethod
    def tearDownClass(cls):
        if cls.orig_api_host is None:
            del arvados.config.settings()['ARVADOS_API_HOST']
        else:
            arvados.config.settings()['ARVADOS_API_HOST'] = cls.orig_api_host
        # Prevent other tests from using our mocked API client.
        arvados.uncache_api('v1')

    def test_basic_list(self):
        answer = self.api.humans().list(
            filters=[['uuid', 'is', None]]).execute()
        self.assertEqual(answer['items_available'], len(answer['items']))

    def test_exceptions_include_errors(self):
        with self.assertRaises(apiclient.errors.HttpError) as err_ctx:
            self.api.humans().get(uuid='xyz-xyz-abcdef').execute()
        err_s = str(err_ctx.exception)
        for msg in ["Bad UUID format", "Bad output format"]:
            self.assertIn(msg, err_s)

    def test_exceptions_without_errors_have_basic_info(self):
        with self.assertRaises(apiclient.errors.HttpError) as err_ctx:
            self.api.humans().delete(uuid='xyz-xyz-abcdef').execute()
        self.assertIn("500", str(err_ctx.exception))


if __name__ == '__main__':
    unittest.main()
