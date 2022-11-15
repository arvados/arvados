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
import arvados_cwl
import arvados_cwl.runner
import arvados.keep

from .matcher import JsonDiffMatcher, StripYAMLComments
from .mock_discovery import get_rootDesc

import arvados_cwl.http

import ruamel.yaml as yaml


class TestHttpToKeep(unittest.TestCase):

    @mock.patch("requests.get")
    @mock.patch("arvados.collection.Collection")
    def test_http_get(self, collectionmock, getmock):
        api = mock.MagicMock()

        api.collections().list().execute.return_value = {
            "items": []
        }

        cm = mock.MagicMock()
        cm.manifest_locator.return_value = "zzzzz-4zz18-zzzzzzzzzzzzzz3"
        cm.portable_data_hash.return_value = "99999999999999999999999999999998+99"
        collectionmock.return_value = cm

        req = mock.MagicMock()
        req.status_code = 200
        req.headers = {}
        req.iter_content.return_value = ["abc"]
        getmock.return_value = req

        utcnow = mock.MagicMock()
        utcnow.return_value = datetime.datetime(2018, 5, 15)

        r = arvados_cwl.http.http_to_keep(api, None, "http://example.com/file1.txt", utcnow=utcnow)
        self.assertEqual(r, "keep:99999999999999999999999999999998+99/file1.txt")

        getmock.assert_called_with("http://example.com/file1.txt", stream=True, allow_redirects=True, headers={})

        cm.open.assert_called_with("file1.txt", "wb")
        cm.save_new.assert_called_with(name="Downloaded from http%3A%2F%2Fexample.com%2Ffile1.txt",
                                       owner_uuid=None, ensure_unique_name=True)

        api.collections().update.assert_has_calls([
            mock.call(uuid=cm.manifest_locator(),
                      body={"collection":{"properties": {'http://example.com/file1.txt': {'Date': 'Tue, 15 May 2018 00:00:00 GMT'}}}})
        ])


    @mock.patch("requests.get")
    @mock.patch("arvados.collection.CollectionReader")
    def test_http_expires(self, collectionmock, getmock):
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

        req = mock.MagicMock()
        req.status_code = 200
        req.headers = {}
        req.iter_content.return_value = ["abc"]
        getmock.return_value = req

        utcnow = mock.MagicMock()
        utcnow.return_value = datetime.datetime(2018, 5, 16)

        r = arvados_cwl.http.http_to_keep(api, None, "http://example.com/file1.txt", utcnow=utcnow)
        self.assertEqual(r, "keep:99999999999999999999999999999998+99/file1.txt")

        getmock.assert_not_called()


    @mock.patch("requests.get")
    @mock.patch("arvados.collection.CollectionReader")
    def test_http_cache_control(self, collectionmock, getmock):
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

        req = mock.MagicMock()
        req.status_code = 200
        req.headers = {}
        req.iter_content.return_value = ["abc"]
        getmock.return_value = req

        utcnow = mock.MagicMock()
        utcnow.return_value = datetime.datetime(2018, 5, 16)

        r = arvados_cwl.http.http_to_keep(api, None, "http://example.com/file1.txt", utcnow=utcnow)
        self.assertEqual(r, "keep:99999999999999999999999999999998+99/file1.txt")

        getmock.assert_not_called()


    @mock.patch("requests.get")
    @mock.patch("requests.head")
    @mock.patch("arvados.collection.Collection")
    def test_http_expired(self, collectionmock, headmock, getmock):
        api = mock.MagicMock()

        api.collections().list().execute.return_value = {
            "items": [{
                "uuid": "zzzzz-4zz18-zzzzzzzzzzzzzz3",
                "portable_data_hash": "99999999999999999999999999999998+99",
                "properties": {
                    'http://example.com/file1.txt': {
                        'Date': 'Tue, 15 May 2018 00:00:00 GMT',
                        'Expires': 'Tue, 16 May 2018 00:00:00 GMT'
                    }
                }
            }]
        }

        cm = mock.MagicMock()
        cm.manifest_locator.return_value = "zzzzz-4zz18-zzzzzzzzzzzzzz4"
        cm.portable_data_hash.return_value = "99999999999999999999999999999997+99"
        cm.keys.return_value = ["file1.txt"]
        collectionmock.return_value = cm

        req = mock.MagicMock()
        req.status_code = 200
        req.headers = {'Date': 'Tue, 17 May 2018 00:00:00 GMT'}
        req.iter_content.return_value = ["def"]
        getmock.return_value = req
        headmock.return_value = req

        utcnow = mock.MagicMock()
        utcnow.return_value = datetime.datetime(2018, 5, 17)

        r = arvados_cwl.http.http_to_keep(api, None, "http://example.com/file1.txt", utcnow=utcnow)
        self.assertEqual(r, "keep:99999999999999999999999999999997+99/file1.txt")

        getmock.assert_called_with("http://example.com/file1.txt", stream=True, allow_redirects=True, headers={})

        cm.open.assert_called_with("file1.txt", "wb")
        cm.save_new.assert_called_with(name="Downloaded from http%3A%2F%2Fexample.com%2Ffile1.txt",
                                       owner_uuid=None, ensure_unique_name=True)

        api.collections().update.assert_has_calls([
            mock.call(uuid=cm.manifest_locator(),
                      body={"collection":{"properties": {'http://example.com/file1.txt': {'Date': 'Tue, 17 May 2018 00:00:00 GMT'}}}})
        ])


    @mock.patch("requests.get")
    @mock.patch("requests.head")
    @mock.patch("arvados.collection.CollectionReader")
    def test_http_etag(self, collectionmock, headmock, getmock):
        api = mock.MagicMock()

        api.collections().list().execute.return_value = {
            "items": [{
                "uuid": "zzzzz-4zz18-zzzzzzzzzzzzzz3",
                "portable_data_hash": "99999999999999999999999999999998+99",
                "properties": {
                    'http://example.com/file1.txt': {
                        'Date': 'Tue, 15 May 2018 00:00:00 GMT',
                        'Expires': 'Tue, 16 May 2018 00:00:00 GMT',
                        'ETag': '"123456"'
                    }
                }
            }]
        }

        cm = mock.MagicMock()
        cm.manifest_locator.return_value = "zzzzz-4zz18-zzzzzzzzzzzzzz3"
        cm.portable_data_hash.return_value = "99999999999999999999999999999998+99"
        cm.keys.return_value = ["file1.txt"]
        collectionmock.return_value = cm

        req = mock.MagicMock()
        req.status_code = 200
        req.headers = {
            'Date': 'Tue, 17 May 2018 00:00:00 GMT',
            'Expires': 'Tue, 19 May 2018 00:00:00 GMT',
            'ETag': '"123456"'
        }
        headmock.return_value = req

        utcnow = mock.MagicMock()
        utcnow.return_value = datetime.datetime(2018, 5, 17)

        r = arvados_cwl.http.http_to_keep(api, None, "http://example.com/file1.txt", utcnow=utcnow)
        self.assertEqual(r, "keep:99999999999999999999999999999998+99/file1.txt")

        getmock.assert_not_called()
        cm.open.assert_not_called()

        api.collections().update.assert_has_calls([
            mock.call(uuid=cm.manifest_locator(),
                      body={"collection":{"properties": {'http://example.com/file1.txt': {
                          'Date': 'Tue, 17 May 2018 00:00:00 GMT',
                          'Expires': 'Tue, 19 May 2018 00:00:00 GMT',
                          'ETag': '"123456"'
                      }}}})
                      ])

    @mock.patch("requests.get")
    @mock.patch("arvados.collection.Collection")
    def test_http_content_disp(self, collectionmock, getmock):
        api = mock.MagicMock()

        api.collections().list().execute.return_value = {
            "items": []
        }

        cm = mock.MagicMock()
        cm.manifest_locator.return_value = "zzzzz-4zz18-zzzzzzzzzzzzzz3"
        cm.portable_data_hash.return_value = "99999999999999999999999999999998+99"
        collectionmock.return_value = cm

        req = mock.MagicMock()
        req.status_code = 200
        req.headers = {"Content-Disposition": "attachment; filename=file1.txt"}
        req.iter_content.return_value = ["abc"]
        getmock.return_value = req

        utcnow = mock.MagicMock()
        utcnow.return_value = datetime.datetime(2018, 5, 15)

        r = arvados_cwl.http.http_to_keep(api, None, "http://example.com/download?fn=/file1.txt", utcnow=utcnow)
        self.assertEqual(r, "keep:99999999999999999999999999999998+99/file1.txt")

        getmock.assert_called_with("http://example.com/download?fn=/file1.txt", stream=True, allow_redirects=True, headers={})

        cm.open.assert_called_with("file1.txt", "wb")
        cm.save_new.assert_called_with(name="Downloaded from http%3A%2F%2Fexample.com%2Fdownload%3Ffn%3D%2Ffile1.txt",
                                       owner_uuid=None, ensure_unique_name=True)

        api.collections().update.assert_has_calls([
            mock.call(uuid=cm.manifest_locator(),
                      body={"collection":{"properties": {"http://example.com/download?fn=/file1.txt": {'Date': 'Tue, 15 May 2018 00:00:00 GMT'}}}})
        ])

    @mock.patch("requests.get")
    @mock.patch("requests.head")
    @mock.patch("arvados.collection.CollectionReader")
    def test_http_etag_if_none_match(self, collectionmock, headmock, getmock):
        api = mock.MagicMock()

        api.collections().list().execute.return_value = {
            "items": [{
                "uuid": "zzzzz-4zz18-zzzzzzzzzzzzzz3",
                "portable_data_hash": "99999999999999999999999999999998+99",
                "properties": {
                    'http://example.com/file1.txt': {
                        'Date': 'Tue, 15 May 2018 00:00:00 GMT',
                        'Expires': 'Tue, 16 May 2018 00:00:00 GMT',
                        'ETag': '"123456"'
                    }
                }
            }]
        }

        cm = mock.MagicMock()
        cm.manifest_locator.return_value = "zzzzz-4zz18-zzzzzzzzzzzzzz3"
        cm.portable_data_hash.return_value = "99999999999999999999999999999998+99"
        cm.keys.return_value = ["file1.txt"]
        collectionmock.return_value = cm

        # Head request fails, will try a conditional GET instead
        req = mock.MagicMock()
        req.status_code = 403
        req.headers = {
        }
        headmock.return_value = req

        utcnow = mock.MagicMock()
        utcnow.return_value = datetime.datetime(2018, 5, 17)

        req = mock.MagicMock()
        req.status_code = 304
        req.headers = {
            'Date': 'Tue, 17 May 2018 00:00:00 GMT',
            'Expires': 'Tue, 19 May 2018 00:00:00 GMT',
            'ETag': '"123456"'
        }
        getmock.return_value = req

        r = arvados_cwl.http.http_to_keep(api, None, "http://example.com/file1.txt", utcnow=utcnow)
        self.assertEqual(r, "keep:99999999999999999999999999999998+99/file1.txt")

        getmock.assert_called_with("http://example.com/file1.txt", stream=True, allow_redirects=True, headers={"If-None-Match": '"123456"'})
        cm.open.assert_not_called()

        api.collections().update.assert_has_calls([
            mock.call(uuid=cm.manifest_locator(),
                      body={"collection":{"properties": {'http://example.com/file1.txt': {
                          'Date': 'Tue, 17 May 2018 00:00:00 GMT',
                          'Expires': 'Tue, 19 May 2018 00:00:00 GMT',
                          'ETag': '"123456"'
                      }}}})
                      ])


    @mock.patch("requests.get")
    @mock.patch("requests.head")
    @mock.patch("arvados.collection.CollectionReader")
    def test_http_prefer_cached_downloads(self, collectionmock, headmock, getmock):
        api = mock.MagicMock()

        api.collections().list().execute.return_value = {
            "items": [{
                "uuid": "zzzzz-4zz18-zzzzzzzzzzzzzz3",
                "portable_data_hash": "99999999999999999999999999999998+99",
                "properties": {
                    'http://example.com/file1.txt': {
                        'Date': 'Tue, 15 May 2018 00:00:00 GMT',
                        'Expires': 'Tue, 16 May 2018 00:00:00 GMT',
                        'ETag': '"123456"'
                    }
                }
            }]
        }

        cm = mock.MagicMock()
        cm.manifest_locator.return_value = "zzzzz-4zz18-zzzzzzzzzzzzzz3"
        cm.portable_data_hash.return_value = "99999999999999999999999999999998+99"
        cm.keys.return_value = ["file1.txt"]
        collectionmock.return_value = cm

        utcnow = mock.MagicMock()
        utcnow.return_value = datetime.datetime(2018, 5, 17)

        r = arvados_cwl.http.http_to_keep(api, None, "http://example.com/file1.txt", utcnow=utcnow, prefer_cached_downloads=True)
        self.assertEqual(r, "keep:99999999999999999999999999999998+99/file1.txt")

        headmock.assert_not_called()
        getmock.assert_not_called()
        cm.open.assert_not_called()
        api.collections().update.assert_not_called()

    @mock.patch("requests.get")
    @mock.patch("requests.head")
    @mock.patch("arvados.collection.CollectionReader")
    def test_http_varying_url_params(self, collectionmock, headmock, getmock):
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
                            'ETag': '"123456"'
                        }
                    }
                }]
            }

            cm = mock.MagicMock()
            cm.manifest_locator.return_value = "zzzzz-4zz18-zzzzzzzzzzzzzz3"
            cm.portable_data_hash.return_value = "99999999999999999999999999999998+99"
            cm.keys.return_value = ["file1.txt"]
            collectionmock.return_value = cm

            req = mock.MagicMock()
            req.status_code = 200
            req.headers = {
                'Date': 'Tue, 17 May 2018 00:00:00 GMT',
                'Expires': 'Tue, 19 May 2018 00:00:00 GMT',
                'ETag': '"123456"'
            }
            headmock.return_value = req

            utcnow = mock.MagicMock()
            utcnow.return_value = datetime.datetime(2018, 5, 17)

            r = arvados_cwl.http.http_to_keep(api, None, "http://example.com/file1.txt?KeyId=123&Signature=456&Expires=789",
                                              utcnow=utcnow, varying_url_params="KeyId,Signature,Expires")
            self.assertEqual(r, "keep:99999999999999999999999999999998+99/file1.txt")

            getmock.assert_not_called()
            cm.open.assert_not_called()

            api.collections().update.assert_has_calls([
                mock.call(uuid=cm.manifest_locator(),
                          body={"collection":{"properties": {'http://example.com/file1.txt': {
                              'Date': 'Tue, 17 May 2018 00:00:00 GMT',
                              'Expires': 'Tue, 19 May 2018 00:00:00 GMT',
                              'ETag': '"123456"'
                          }}}})
                          ])
