#!/usr/bin/env python

import arvados
import errno
import hashlib
import httplib
import httplib2
import io
import mock
import os
import requests
import shutil
import tempfile
import unittest

# Use this hostname when you want to make sure the traffic will be
# instantly refused.  100::/64 is a dedicated black hole.
TEST_HOST = '100::'

skip_sleep = mock.patch('time.sleep', lambda n: None)  # clown'll eat me

# fake_httplib2_response and mock_responses
# mock calls to httplib2.Http.request()
def fake_httplib2_response(code, **headers):
    headers.update(status=str(code),
                   reason=httplib.responses.get(code, "Unknown Response"))
    return httplib2.Response(headers)

def mock_responses(body, *codes, **headers):
    return mock.patch('httplib2.Http.request', side_effect=(
            (fake_httplib2_response(code, **headers), body) for code in codes))

# fake_requests_response, mock_get_responses and mock_put_responses
# mock calls to requests.get() and requests.put()
def fake_requests_response(code, body, **headers):
    r = requests.Response()
    r.status_code = code
    r.reason = httplib.responses.get(code, "Unknown Response")
    r.headers = headers
    r.raw = io.BytesIO(body)
    return r

def mock_get_responses(body, *codes, **headers):
    return mock.patch('requests.get', side_effect=(
        fake_requests_response(code, body, **headers) for code in codes))

def mock_put_responses(body, *codes, **headers):
    return mock.patch('requests.put', side_effect=(
        fake_requests_response(code, body, **headers) for code in codes))

def mock_requestslib_responses(method, body, *codes, **headers):
    return mock.patch(method, side_effect=(
        fake_requests_response(code, body, **headers) for code in codes))

class MockStreamReader(object):
    def __init__(self, name='.', *data):
        self._name = name
        self._data = ''.join(data)
        self._data_locators = ['{}+{}'.format(hashlib.md5(d).hexdigest(),
                                              len(d)) for d in data]
        self.num_retries = 0

    def name(self):
        return self._name

    def readfrom(self, start, size, num_retries=None):
        return self._data[start:start + size]


class ApiClientMock(object):
    def api_client_mock(self):
        return mock.MagicMock(name='api_client_mock')

    def mock_keep_services(self, api_mock=None, status=200, count=12,
                           service_type='disk',
                           service_host=None,
                           service_port=None,
                           service_ssl_flag=False):
        if api_mock is None:
            api_mock = self.api_client_mock()
        body = {
            'items_available': count,
            'items': [{
                'uuid': 'zzzzz-bi6l4-{:015x}'.format(i),
                'owner_uuid': 'zzzzz-tpzed-000000000000000',
                'service_host': service_host or 'keep0x{:x}'.format(i),
                'service_port': service_port or 65535-i,
                'service_ssl_flag': service_ssl_flag,
                'service_type': service_type,
            } for i in range(0, count)]
        }
        self._mock_api_call(api_mock.keep_services().accessible, status, body)
        return api_mock

    def _mock_api_call(self, mock_method, code, body):
        mock_method = mock_method().execute
        if code == 200:
            mock_method.return_value = body
        else:
            mock_method.side_effect = arvados.errors.ApiError(
                fake_httplib2_response(code), "{}")


class ArvadosBaseTestCase(unittest.TestCase):
    # This class provides common utility functions for our tests.

    def setUp(self):
        self._tempdirs = []

    def tearDown(self):
        for workdir in self._tempdirs:
            shutil.rmtree(workdir, ignore_errors=True)

    def make_tmpdir(self):
        self._tempdirs.append(tempfile.mkdtemp())
        return self._tempdirs[-1]

    def data_file(self, filename):
        try:
            basedir = os.path.dirname(__file__)
        except NameError:
            basedir = '.'
        return open(os.path.join(basedir, 'data', filename))

    def build_directory_tree(self, tree):
        tree_root = self.make_tmpdir()
        for leaf in tree:
            path = os.path.join(tree_root, leaf)
            try:
                os.makedirs(os.path.dirname(path))
            except OSError as error:
                if error.errno != errno.EEXIST:
                    raise
            with open(path, 'w') as tmpfile:
                tmpfile.write(leaf)
        return tree_root

    def make_test_file(self, text="test"):
        testfile = tempfile.NamedTemporaryFile()
        testfile.write(text)
        testfile.flush()
        return testfile
