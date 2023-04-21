# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from future import standard_library
standard_library.install_aliases()

import copy
import io
import functools
import hashlib
import json
import logging
import mock
import sys
import unittest
import datetime

import arvados
import arvados.collection
import arvados.keep
import pycurl

from arvados.http_to_keep import http_to_keep

import ruamel.yaml as yaml

# Turns out there was already "FakeCurl" that serves the same purpose, but
# I wrote this before I knew that.  Whoops.
class CurlMock:
    def __init__(self, headers = {}):
        self.perform_was_called = False
        self.headers = headers
        self.get_response = 200
        self.head_response = 200
        self.req_headers = []

    def setopt(self, op, *args):
        if op == pycurl.URL:
            self.url = args[0]
        if op == pycurl.WRITEFUNCTION:
            self.writefn = args[0]
        if op == pycurl.HEADERFUNCTION:
            self.headerfn = args[0]
        if op == pycurl.NOBODY:
            self.head = True
        if op == pycurl.HTTPGET:
            self.head = False
        if op == pycurl.HTTPHEADER:
            self.req_headers = args[0]

    def getinfo(self, op):
        if op == pycurl.RESPONSE_CODE:
            if self.head:
                return self.head_response
            else:
                return self.get_response

    def perform(self):
        self.perform_was_called = True

        if self.head:
            self.headerfn("HTTP/1.1 {} Status\r\n".format(self.head_response))
        else:
            self.headerfn("HTTP/1.1 {} Status\r\n".format(self.get_response))

        for k,v in self.headers.items():
            self.headerfn("%s: %s" % (k,v))

        if not self.head and self.get_response == 200:
            self.writefn(self.chunk)


class TestHttpToKeep(unittest.TestCase):

    @mock.patch("pycurl.Curl")
    @mock.patch("arvados.collection.Collection")
    def test_http_get(self, collectionmock, curlmock):
        api = mock.MagicMock()

        api.collections().list().execute.return_value = {
            "items": []
        }

        cm = mock.MagicMock()
        cm.manifest_locator.return_value = "zzzzz-4zz18-zzzzzzzzzzzzzz3"
        cm.portable_data_hash.return_value = "99999999999999999999999999999998+99"
        collectionmock.return_value = cm

        mockobj = CurlMock()
        mockobj.chunk = b'abc'
        def init():
            return mockobj
        curlmock.side_effect = init

        utcnow = mock.MagicMock()
        utcnow.return_value = datetime.datetime(2018, 5, 15)

        r = http_to_keep(api, None, "http://example.com/file1.txt", utcnow=utcnow)
        self.assertEqual(r, ("99999999999999999999999999999998+99", "file1.txt"))

        assert mockobj.url == b"http://example.com/file1.txt"
        assert mockobj.perform_was_called is True

        cm.open.assert_called_with("file1.txt", "wb")
        cm.save_new.assert_called_with(name="Downloaded from http%3A%2F%2Fexample.com%2Ffile1.txt",
                                       owner_uuid=None, ensure_unique_name=True)

        api.collections().update.assert_has_calls([
            mock.call(uuid=cm.manifest_locator(),
                      body={"collection":{"properties": {'http://example.com/file1.txt': {'Date': 'Tue, 15 May 2018 00:00:00 GMT'}}}})
        ])


    @mock.patch("pycurl.Curl")
    @mock.patch("arvados.collection.CollectionReader")
    def test_http_expires(self, collectionmock, curlmock):
        api = mock.MagicMock()

        api.collections().list().execute.return_value = {
            "items": [{
                "uuid": "zzzzz-4zz18-zzzzzzzzzzzzzz3",
                "portable_data_hash": "99999999999999999999999999999998+99",
                "properties": {
                    'http://example.com/file1.txt': {
                        'Date': 'Tue, 15 May 2018 00:00:00 GMT',
                        'Expires': 'Tue, 17 May 2018 00:00:00 GMT'
                    }
                }
            }]
        }

        cm = mock.MagicMock()
        cm.manifest_locator.return_value = "zzzzz-4zz18-zzzzzzzzzzzzzz3"
        cm.portable_data_hash.return_value = "99999999999999999999999999999998+99"
        cm.keys.return_value = ["file1.txt"]
        collectionmock.return_value = cm

        mockobj = CurlMock()
        mockobj.chunk = b'abc'
        def init():
            return mockobj
        curlmock.side_effect = init

        utcnow = mock.MagicMock()
        utcnow.return_value = datetime.datetime(2018, 5, 16)

        r = http_to_keep(api, None, "http://example.com/file1.txt", utcnow=utcnow)
        self.assertEqual(r, ("99999999999999999999999999999998+99", "file1.txt"))

        assert mockobj.perform_was_called is False


    @mock.patch("pycurl.Curl")
    @mock.patch("arvados.collection.CollectionReader")
    def test_http_cache_control(self, collectionmock, curlmock):
        api = mock.MagicMock()

        api.collections().list().execute.return_value = {
            "items": [{
                "uuid": "zzzzz-4zz18-zzzzzzzzzzzzzz3",
                "portable_data_hash": "99999999999999999999999999999998+99",
                "properties": {
                    'http://example.com/file1.txt': {
                        'Date': 'Tue, 15 May 2018 00:00:00 GMT',
                        'Cache-Control': 'max-age=172800'
                    }
                }
            }]
        }

        cm = mock.MagicMock()
        cm.manifest_locator.return_value = "zzzzz-4zz18-zzzzzzzzzzzzzz3"
        cm.portable_data_hash.return_value = "99999999999999999999999999999998+99"
        cm.keys.return_value = ["file1.txt"]
        collectionmock.return_value = cm

        mockobj = CurlMock()
        mockobj.chunk = b'abc'
        def init():
            return mockobj
        curlmock.side_effect = init

        utcnow = mock.MagicMock()
        utcnow.return_value = datetime.datetime(2018, 5, 16)

        r = http_to_keep(api, None, "http://example.com/file1.txt", utcnow=utcnow)
        self.assertEqual(r, ("99999999999999999999999999999998+99", "file1.txt"))

        assert mockobj.perform_was_called is False


    @mock.patch("pycurl.Curl")
    @mock.patch("arvados.collection.Collection")
    def test_http_expired(self, collectionmock, curlmock):
        api = mock.MagicMock()

        api.collections().list().execute.return_value = {
            "items": [{
                "uuid": "zzzzz-4zz18-zzzzzzzzzzzzzz3",
                "portable_data_hash": "99999999999999999999999999999998+99",
                "properties": {
                    'http://example.com/file1.txt': {
                        'Date': 'Tue, 15 May 2018 00:00:00 GMT',
                        'Expires': 'Wed, 16 May 2018 00:00:00 GMT'
                    }
                }
            }]
        }

        cm = mock.MagicMock()
        cm.manifest_locator.return_value = "zzzzz-4zz18-zzzzzzzzzzzzzz4"
        cm.portable_data_hash.return_value = "99999999999999999999999999999997+99"
        cm.keys.return_value = ["file1.txt"]
        collectionmock.return_value = cm

        mockobj = CurlMock({'Date': 'Thu, 17 May 2018 00:00:00 GMT'})
        mockobj.chunk = b'def'
        def init():
            return mockobj
        curlmock.side_effect = init

        utcnow = mock.MagicMock()
        utcnow.return_value = datetime.datetime(2018, 5, 17)

        r = http_to_keep(api, None, "http://example.com/file1.txt", utcnow=utcnow)
        self.assertEqual(r, ("99999999999999999999999999999997+99", "file1.txt"))

        assert mockobj.url == b"http://example.com/file1.txt"
        assert mockobj.perform_was_called is True

        cm.open.assert_called_with("file1.txt", "wb")
        cm.save_new.assert_called_with(name="Downloaded from http%3A%2F%2Fexample.com%2Ffile1.txt",
                                       owner_uuid=None, ensure_unique_name=True)

        api.collections().update.assert_has_calls([
            mock.call(uuid=cm.manifest_locator(),
                      body={"collection":{"properties": {'http://example.com/file1.txt': {'Date': 'Thu, 17 May 2018 00:00:00 GMT'}}}})
        ])


    @mock.patch("pycurl.Curl")
    @mock.patch("arvados.collection.CollectionReader")
    def test_http_etag(self, collectionmock, curlmock):
        api = mock.MagicMock()

        api.collections().list().execute.return_value = {
            "items": [{
                "uuid": "zzzzz-4zz18-zzzzzzzzzzzzzz3",
                "portable_data_hash": "99999999999999999999999999999998+99",
                "properties": {
                    'http://example.com/file1.txt': {
                        'Date': 'Tue, 15 May 2018 00:00:00 GMT',
                        'Expires': 'Wed, 16 May 2018 00:00:00 GMT',
                        'Etag': '"123456"'
                    }
                }
            }]
        }

        cm = mock.MagicMock()
        cm.manifest_locator.return_value = "zzzzz-4zz18-zzzzzzzzzzzzzz3"
        cm.portable_data_hash.return_value = "99999999999999999999999999999998+99"
        cm.keys.return_value = ["file1.txt"]
        collectionmock.return_value = cm

        mockobj = CurlMock({
            'Date': 'Thu, 17 May 2018 00:00:00 GMT',
            'Expires': 'Sat, 19 May 2018 00:00:00 GMT',
            'Etag': '"123456"'
        })
        mockobj.chunk = None
        def init():
            return mockobj
        curlmock.side_effect = init

        utcnow = mock.MagicMock()
        utcnow.return_value = datetime.datetime(2018, 5, 17)

        r = http_to_keep(api, None, "http://example.com/file1.txt", utcnow=utcnow)
        self.assertEqual(r, ("99999999999999999999999999999998+99", "file1.txt"))

        cm.open.assert_not_called()

        api.collections().update.assert_has_calls([
            mock.call(uuid=cm.manifest_locator(),
                      body={"collection":{"properties": {'http://example.com/file1.txt': {
                          'Date': 'Thu, 17 May 2018 00:00:00 GMT',
                          'Expires': 'Sat, 19 May 2018 00:00:00 GMT',
                          'Etag': '"123456"'
                      }}}})
                      ])

    @mock.patch("pycurl.Curl")
    @mock.patch("arvados.collection.Collection")
    def test_http_content_disp(self, collectionmock, curlmock):
        api = mock.MagicMock()

        api.collections().list().execute.return_value = {
            "items": []
        }

        cm = mock.MagicMock()
        cm.manifest_locator.return_value = "zzzzz-4zz18-zzzzzzzzzzzzzz3"
        cm.portable_data_hash.return_value = "99999999999999999999999999999998+99"
        collectionmock.return_value = cm

        mockobj = CurlMock({"Content-Disposition": "attachment; filename=file1.txt"})
        mockobj.chunk = "abc"
        def init():
            return mockobj
        curlmock.side_effect = init

        utcnow = mock.MagicMock()
        utcnow.return_value = datetime.datetime(2018, 5, 15)

        r = http_to_keep(api, None, "http://example.com/download?fn=/file1.txt", utcnow=utcnow)
        self.assertEqual(r, ("99999999999999999999999999999998+99", "file1.txt"))

        assert mockobj.url == b"http://example.com/download?fn=/file1.txt"

        cm.open.assert_called_with("file1.txt", "wb")
        cm.save_new.assert_called_with(name="Downloaded from http%3A%2F%2Fexample.com%2Fdownload%3Ffn%3D%2Ffile1.txt",
                                       owner_uuid=None, ensure_unique_name=True)

        api.collections().update.assert_has_calls([
            mock.call(uuid=cm.manifest_locator(),
                      body={"collection":{"properties": {"http://example.com/download?fn=/file1.txt": {'Date': 'Tue, 15 May 2018 00:00:00 GMT'}}}})
        ])

    @mock.patch("pycurl.Curl")
    @mock.patch("arvados.collection.CollectionReader")
    def test_http_etag_if_none_match(self, collectionmock, curlmock):
        api = mock.MagicMock()

        api.collections().list().execute.return_value = {
            "items": [{
                "uuid": "zzzzz-4zz18-zzzzzzzzzzzzzz3",
                "portable_data_hash": "99999999999999999999999999999998+99",
                "properties": {
                    'http://example.com/file1.txt': {
                        'Date': 'Tue, 15 May 2018 00:00:00 GMT',
                        'Expires': 'Tue, 16 May 2018 00:00:00 GMT',
                        'Etag': '"123456"'
                    }
                }
            }]
        }

        cm = mock.MagicMock()
        cm.manifest_locator.return_value = "zzzzz-4zz18-zzzzzzzzzzzzzz3"
        cm.portable_data_hash.return_value = "99999999999999999999999999999998+99"
        cm.keys.return_value = ["file1.txt"]
        collectionmock.return_value = cm

        mockobj = CurlMock({
            'Date': 'Tue, 17 May 2018 00:00:00 GMT',
            'Expires': 'Tue, 19 May 2018 00:00:00 GMT',
            'Etag': '"123456"'
        })
        mockobj.chunk = None
        mockobj.head_response = 403
        mockobj.get_response = 304
        def init():
            return mockobj
        curlmock.side_effect = init

        utcnow = mock.MagicMock()
        utcnow.return_value = datetime.datetime(2018, 5, 17)

        r = http_to_keep(api, None, "http://example.com/file1.txt", utcnow=utcnow)
        self.assertEqual(r, ("99999999999999999999999999999998+99", "file1.txt"))

        print(mockobj.req_headers)
        assert mockobj.req_headers == ["Accept: application/octet-stream", "If-None-Match: \"123456\""]
        cm.open.assert_not_called()

        api.collections().update.assert_has_calls([
            mock.call(uuid=cm.manifest_locator(),
                      body={"collection":{"properties": {'http://example.com/file1.txt': {
                          'Date': 'Tue, 17 May 2018 00:00:00 GMT',
                          'Expires': 'Tue, 19 May 2018 00:00:00 GMT',
                          'Etag': '"123456"'
                      }}}})
                      ])

    @mock.patch("pycurl.Curl")
    @mock.patch("arvados.collection.CollectionReader")
    def test_http_prefer_cached_downloads(self, collectionmock, curlmock):
        api = mock.MagicMock()

        api.collections().list().execute.return_value = {
            "items": [{
                "uuid": "zzzzz-4zz18-zzzzzzzzzzzzzz3",
                "portable_data_hash": "99999999999999999999999999999998+99",
                "properties": {
                    'http://example.com/file1.txt': {
                        'Date': 'Tue, 15 May 2018 00:00:00 GMT',
                        'Expires': 'Tue, 16 May 2018 00:00:00 GMT',
                        'Etag': '"123456"'
                    }
                }
            }]
        }

        cm = mock.MagicMock()
        cm.manifest_locator.return_value = "zzzzz-4zz18-zzzzzzzzzzzzzz3"
        cm.portable_data_hash.return_value = "99999999999999999999999999999998+99"
        cm.keys.return_value = ["file1.txt"]
        collectionmock.return_value = cm

        mockobj = CurlMock()
        def init():
            return mockobj
        curlmock.side_effect = init

        utcnow = mock.MagicMock()
        utcnow.return_value = datetime.datetime(2018, 5, 17)

        r = http_to_keep(api, None, "http://example.com/file1.txt", utcnow=utcnow, prefer_cached_downloads=True)
        self.assertEqual(r, ("99999999999999999999999999999998+99", "file1.txt"))

        assert mockobj.perform_was_called is False
        cm.open.assert_not_called()
        api.collections().update.assert_not_called()

    @mock.patch("pycurl.Curl")
    @mock.patch("arvados.collection.CollectionReader")
    def test_http_varying_url_params(self, collectionmock, curlmock):
        for prurl in ("http://example.com/file1.txt", "http://example.com/file1.txt?KeyId=123&Signature=456&Expires=789"):
            api = mock.MagicMock()

            api.collections().list().execute.return_value = {
                "items": [{
                    "uuid": "zzzzz-4zz18-zzzzzzzzzzzzzz3",
                    "portable_data_hash": "99999999999999999999999999999998+99",
                    "properties": {
                        prurl: {
                            'Date': 'Tue, 15 May 2018 00:00:00 GMT',
                            'Expires': 'Tue, 16 May 2018 00:00:00 GMT',
                            'Etag': '"123456"'
                        }
                    }
                }]
            }

            cm = mock.MagicMock()
            cm.manifest_locator.return_value = "zzzzz-4zz18-zzzzzzzzzzzzzz3"
            cm.portable_data_hash.return_value = "99999999999999999999999999999998+99"
            cm.keys.return_value = ["file1.txt"]
            collectionmock.return_value = cm

            mockobj = CurlMock({
                'Date': 'Tue, 17 May 2018 00:00:00 GMT',
                'Expires': 'Tue, 19 May 2018 00:00:00 GMT',
                'Etag': '"123456"'
            })
            mockobj.chunk = None
            def init():
                return mockobj
            curlmock.side_effect = init

            utcnow = mock.MagicMock()
            utcnow.return_value = datetime.datetime(2018, 5, 17)

            r = http_to_keep(api, None, "http://example.com/file1.txt?KeyId=123&Signature=456&Expires=789",
                                              utcnow=utcnow, varying_url_params="KeyId,Signature,Expires")
            self.assertEqual(r, ("99999999999999999999999999999998+99", "file1.txt"))

            assert mockobj.perform_was_called is True
            cm.open.assert_not_called()

            api.collections().update.assert_has_calls([
                mock.call(uuid=cm.manifest_locator(),
                          body={"collection":{"properties": {'http://example.com/file1.txt': {
                              'Date': 'Tue, 17 May 2018 00:00:00 GMT',
                              'Expires': 'Tue, 19 May 2018 00:00:00 GMT',
                              'Etag': '"123456"'
                          }}}})
                          ])
