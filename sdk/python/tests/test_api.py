#!/usr/bin/env python

import arvados
import httplib2
import json
import mimetypes
import os
import run_test_server
import unittest
from apiclient import errors as apiclient_errors
from apiclient import http as apiclient_http

from arvados_testutil import fake_httplib2_response

if not mimetypes.inited:
    mimetypes.init()

class ArvadosApiClientTest(unittest.TestCase):
    ERROR_HEADERS = {'Content-Type': mimetypes.types_map['.json']}

    @classmethod
    def api_error_response(cls, code, *errors):
        return (fake_httplib2_response(code, **cls.ERROR_HEADERS),
                json.dumps({'errors': errors,
                            'error_token': '1234567890+12345678'}))

    @classmethod
    def setUpClass(cls):
        # The apiclient library has support for mocking requests for
        # testing, but it doesn't extend to the discovery document
        # itself. For now, bring up an API server that will serve
        # a discovery document.
        # FIXME: Figure out a better way to stub this out.
        run_test_server.run()
        mock_responses = {
            'arvados.humans.delete': (
                fake_httplib2_response(500, **cls.ERROR_HEADERS),
                ""),
            'arvados.humans.get': cls.api_error_response(
                422, "Bad UUID format", "Bad output format"),
            'arvados.humans.list': (None, json.dumps(
                    {'items_available': 0, 'items': []})),
            }
        req_builder = apiclient_http.RequestMockBuilder(mock_responses)
        cls.api = arvados.api('v1',
                              host=os.environ['ARVADOS_API_HOST'],
                              token='discovery-doc-only-no-token-needed',
                              insecure=True,
                              requestBuilder=req_builder)

    @classmethod
    def tearDownClass(cls):
        run_test_server.stop()

    def test_basic_list(self):
        answer = self.api.humans().list(
            filters=[['uuid', 'is', None]]).execute()
        self.assertEqual(answer['items_available'], len(answer['items']))

    def test_exceptions_include_errors(self):
        with self.assertRaises(apiclient_errors.HttpError) as err_ctx:
            self.api.humans().get(uuid='xyz-xyz-abcdef').execute()
        err_s = str(err_ctx.exception)
        for msg in ["Bad UUID format", "Bad output format"]:
            self.assertIn(msg, err_s)

    def test_exceptions_without_errors_have_basic_info(self):
        with self.assertRaises(apiclient_errors.HttpError) as err_ctx:
            self.api.humans().delete(uuid='xyz-xyz-abcdef').execute()
        self.assertIn("500", str(err_ctx.exception))


if __name__ == '__main__':
    unittest.main()
