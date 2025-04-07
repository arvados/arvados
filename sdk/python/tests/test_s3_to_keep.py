# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import copy
import io
import functools
import hashlib
import json
import logging
import sys
import unittest
import datetime

from unittest import mock

import arvados
import arvados.collection
import arvados.keep

from arvados._internal import s3_to_keep
import boto3.s3.transfer

class TestS3ToKeep(unittest.TestCase):

    @mock.patch("arvados.collection.Collection")
    def test_s3_get(self, collectionmock):
        api = mock.MagicMock()

        api.collections().list().execute.return_value = {
            "items": []
        }

        cm = mock.MagicMock()
        cm.manifest_locator.return_value = "zzzzz-4zz18-zzzzzzzzzzzzzz3"
        cm.portable_data_hash.return_value = "99999999999999999999999999999998+99"
        collectionmock.return_value = cm

        mockfile = mock.MagicMock()
        cm.open.return_value = mockfile

        mockboto = mock.MagicMock()
        mockbotoclient = mock.MagicMock()
        mockboto.client.return_value = mockbotoclient

        mockbotoclient.head_object.return_value = {
            'ResponseMetadata': {
                'HTTPStatusCode': 200,
                'HTTPHeaders': {
                    "Content-Length": 123
                }
            }
        }

        utcnow = mock.MagicMock()
        utcnow.return_value = datetime.datetime(2018, 5, 15)

        r = s3_to_keep.s3_to_keep(api, mockboto, None, "s3://examplebucket/file1.txt", utcnow=utcnow)
        self.assertEqual(r, ("99999999999999999999999999999998+99", "file1.txt",
                             'zzzzz-4zz18-zzzzzzzzzzzzzz3', 's3://examplebucket/file1.txt',
                             datetime.datetime(2018, 5, 15, 0, 0)))

        cm.open.assert_called_with("file1.txt", "wb")
        cm.save_new.assert_called_with(name="Downloaded from s3%3A%2F%2Fexamplebucket%2Ffile1.txt",
                                       owner_uuid=None, ensure_unique_name=True)

        api.collections().update.assert_has_calls([
            mock.call(uuid=cm.manifest_locator(),
                      body={"collection":{"properties": {'s3://examplebucket/file1.txt': {'Content-Length': 123, 'Date': 'Tue, 15 May 2018 00:00:00 GMT'}}}})
        ])

        kall = mockbotoclient.download_fileobj.call_args
        assert kall.kwargs['Bucket'] == 'examplebucket'
        assert kall.kwargs['Key'] == 'file1.txt'
        assert kall.kwargs['Fileobj'] is mockfile
