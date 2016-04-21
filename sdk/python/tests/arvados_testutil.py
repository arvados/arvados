#!/usr/bin/env python

import arvados
import errno
import hashlib
import httplib
import httplib2
import io
import mock
import os
import pycurl
import Queue
import shutil
import tempfile
import unittest

# Use this hostname when you want to make sure the traffic will be
# instantly refused.  100::/64 is a dedicated black hole.
TEST_HOST = '100::'

skip_sleep = mock.patch('time.sleep', lambda n: None)  # clown'll eat me

def queue_with(items):
    """Return a thread-safe iterator that yields the given items.

    +items+ can be given as an array or an iterator. If an iterator is
    given, it will be consumed to fill the queue before queue_with()
    returns.
    """
    queue = Queue.Queue()
    for val in items:
        queue.put(val)
    return lambda *args, **kwargs: queue.get(block=False)

# fake_httplib2_response and mock_responses
# mock calls to httplib2.Http.request()
def fake_httplib2_response(code, **headers):
    headers.update(status=str(code),
                   reason=httplib.responses.get(code, "Unknown Response"))
    return httplib2.Response(headers)

def mock_responses(body, *codes, **headers):
    return mock.patch('httplib2.Http.request', side_effect=queue_with((
        (fake_httplib2_response(code, **headers), body) for code in codes)))

def mock_api_responses(api_client, body, codes, headers={}):
    return mock.patch.object(api_client._http, 'request', side_effect=queue_with((
        (fake_httplib2_response(code, **headers), body) for code in codes)))

def str_keep_locator(s):
    return '{}+{}'.format(hashlib.md5(s).hexdigest(), len(s))

class FakeCurl:
    @classmethod
    def make(cls, code, body='', headers={}):
        return mock.Mock(spec=cls, wraps=cls(code, body, headers))

    def __init__(self, code=200, body='', headers={}):
        self._opt = {}
        self._got_url = None
        self._writer = None
        self._headerfunction = None
        self._resp_code = code
        self._resp_body = body
        self._resp_headers = headers

    def getopt(self, opt):
        return self._opt.get(str(opt), None)

    def setopt(self, opt, val):
        self._opt[str(opt)] = val
        if opt == pycurl.WRITEFUNCTION:
            self._writer = val
        elif opt == pycurl.HEADERFUNCTION:
            self._headerfunction = val

    def perform(self):
        if not isinstance(self._resp_code, int):
            raise self._resp_code
        if self.getopt(pycurl.URL) is None:
            raise ValueError
        if self._writer is None:
            raise ValueError
        if self._headerfunction:
            self._headerfunction("HTTP/1.1 {} Status".format(self._resp_code))
            for k, v in self._resp_headers.iteritems():
                self._headerfunction(k + ': ' + str(v))
        if type(self._resp_body) is not bool:
            self._writer(self._resp_body)

    def close(self):
        pass

    def reset(self):
        """Prevent fake UAs from going back into the user agent pool."""
        raise Exception

    def getinfo(self, opt):
        if opt == pycurl.RESPONSE_CODE:
            return self._resp_code
        raise Exception

def mock_keep_responses(body, *codes, **headers):
    """Patch pycurl to return fake responses and raise exceptions.

    body can be a string to return as the response body; an exception
    to raise when perform() is called; or an iterable that returns a
    sequence of such values.
    """
    cm = mock.MagicMock()
    if isinstance(body, tuple):
        codes = list(codes)
        codes.insert(0, body)
        responses = [
            FakeCurl.make(code=code, body=b, headers=headers)
            for b, code in codes
        ]
    else:
        responses = [
            FakeCurl.make(code=code, body=body, headers=headers)
            for code in codes
        ]
    cm.side_effect = queue_with(responses)
    cm.responses = responses
    return mock.patch('pycurl.Curl', cm)


class MockStreamReader(object):
    def __init__(self, name='.', *data):
        self._name = name
        self._data = ''.join(data)
        self._data_locators = [str_keep_locator(d) for d in data]
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
                           service_ssl_flag=False,
                           additional_services=[],
                           read_only=False):
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
                'read_only': read_only,
            } for i in range(0, count)] + additional_services
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
