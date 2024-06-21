# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import arvados
import pycurl

import unittest
import parameterized
from . import arvados_testutil as tutil
from .arvados_testutil import DiskCacheBase

@tutil.skip_sleep
@parameterized.parameterized_class([{"disk_cache": True}, {"disk_cache": False}])
class KeepStorageClassesTestCase(unittest.TestCase, tutil.ApiClientMock, DiskCacheBase):
    disk_cache = False

    def setUp(self):
        self.api_client = self.mock_keep_services(count=2)
        self.keep_client = arvados.KeepClient(api_client=self.api_client, block_cache=self.make_block_cache(self.disk_cache))
        self.data = b'xyzzy'
        self.locator = '1271ed5ef305aadabc605b1609e24c52'

    def tearDown(self):
        DiskCacheBase.tearDown(self)

    def test_multiple_default_storage_classes_req_header(self):
        api_mock = self.api_client_mock()
        api_mock.config.return_value = {
            'StorageClasses': {
                'foo': { 'Default': True },
                'bar': { 'Default': True },
                'baz': { 'Default': False }
            }
        }
        api_client = self.mock_keep_services(api_mock=api_mock, count=2)
        keep_client = arvados.KeepClient(api_client=api_client, block_cache=self.make_block_cache(self.disk_cache))
        resp_hdr = {
            'x-keep-storage-classes-confirmed': 'foo=1, bar=1',
            'x-keep-replicas-stored': 1
        }
        with tutil.mock_keep_responses(self.locator, 200, **resp_hdr) as mock:
            keep_client.put(self.data, copies=1)
            req_hdr = mock.responses[0]
            self.assertIn(
                'X-Keep-Storage-Classes: bar, foo', req_hdr.getopt(pycurl.HTTPHEADER))

    def test_storage_classes_req_header(self):
        self.assertEqual(
            self.api_client.config()['StorageClasses'],
            {'default': {'Default': True}})
        cases = [
            # requested, expected
            [['foo'], 'X-Keep-Storage-Classes: foo'],
            [['bar', 'foo'], 'X-Keep-Storage-Classes: bar, foo'],
            [[], 'X-Keep-Storage-Classes: default'],
            [None, 'X-Keep-Storage-Classes: default'],
        ]
        for req_classes, expected_header in cases:
            headers = {'x-keep-replicas-stored': 1}
            if req_classes is None or len(req_classes) == 0:
                confirmed_hdr = 'default=1'
            elif len(req_classes) > 0:
                confirmed_hdr = ', '.join(["{}=1".format(cls) for cls in req_classes])
            headers.update({'x-keep-storage-classes-confirmed': confirmed_hdr})
            with tutil.mock_keep_responses(self.locator, 200, **headers) as mock:
                self.keep_client.put(self.data, copies=1, classes=req_classes)
                req_hdr = mock.responses[0]
                self.assertIn(expected_header, req_hdr.getopt(pycurl.HTTPHEADER))

    def test_partial_storage_classes_put(self):
        headers = {
            'x-keep-replicas-stored': 1,
            'x-keep-storage-classes-confirmed': 'foo=1'}
        with tutil.mock_keep_responses(self.locator, 200, 503, **headers) as mock:
            with self.assertRaises(arvados.errors.KeepWriteError):
                self.keep_client.put(self.data, copies=1, classes=['foo', 'bar'], num_retries=0)
            # 1st request, both classes pending
            req1_headers = mock.responses[0].getopt(pycurl.HTTPHEADER)
            self.assertIn('X-Keep-Storage-Classes: bar, foo', req1_headers)
            # 2nd try, 'foo' class already satisfied
            req2_headers = mock.responses[1].getopt(pycurl.HTTPHEADER)
            self.assertIn('X-Keep-Storage-Classes: bar', req2_headers)

    def test_successful_storage_classes_put_requests(self):
        cases = [
            # wanted_copies, wanted_classes, confirmed_copies, confirmed_classes, expected_requests
            [ 1, ['foo'], 1, 'foo=1', 1],
            [ 1, ['foo'], 2, 'foo=2', 1],
            [ 2, ['foo'], 2, 'foo=2', 1],
            [ 2, ['foo'], 1, 'foo=1', 2],
            [ 1, ['foo', 'bar'], 1, 'foo=1, bar=1', 1],
            [ 1, ['foo', 'bar'], 2, 'foo=2, bar=2', 1],
            [ 2, ['foo', 'bar'], 2, 'foo=2, bar=2', 1],
            [ 2, ['foo', 'bar'], 1, 'foo=1, bar=1', 2],
            [ 1, ['foo', 'bar'], 1, None, 1],
            [ 1, ['foo'], 1, None, 1],
            [ 2, ['foo'], 2, None, 1],
            [ 2, ['foo'], 1, None, 2],
        ]
        for w_copies, w_classes, c_copies, c_classes, e_reqs in cases:
            headers = {'x-keep-replicas-stored': c_copies}
            if c_classes is not None:
                headers.update({'x-keep-storage-classes-confirmed': c_classes})
            with tutil.mock_keep_responses(self.locator, 200, 200, **headers) as mock:
                case_desc = 'wanted_copies={}, wanted_classes="{}", confirmed_copies={}, confirmed_classes="{}", expected_requests={}'.format(w_copies, ', '.join(w_classes), c_copies, c_classes, e_reqs)
                self.assertEqual(self.locator,
                    self.keep_client.put(self.data, copies=w_copies, classes=w_classes),
                    case_desc)
                self.assertEqual(e_reqs, mock.call_count, case_desc)

    def test_failed_storage_classes_put_requests(self):
        cases = [
            # wanted_copies, wanted_classes, confirmed_copies, confirmed_classes, return_code
            [ 1, ['foo'], 1, 'bar=1', 200],
            [ 1, ['foo'], 1, None, 503],
            [ 2, ['foo'], 1, 'bar=1, foo=0', 200],
            [ 3, ['foo'], 1, 'bar=1, foo=1', 200],
            [ 3, ['foo', 'bar'], 1, 'bar=2, foo=1', 200],
        ]
        for w_copies, w_classes, c_copies, c_classes, return_code in cases:
            headers = {'x-keep-replicas-stored': c_copies}
            if c_classes is not None:
                headers.update({'x-keep-storage-classes-confirmed': c_classes})
            with tutil.mock_keep_responses(self.locator, return_code, return_code, **headers):
                case_desc = 'wanted_copies={}, wanted_classes="{}", confirmed_copies={}, confirmed_classes="{}"'.format(w_copies, ', '.join(w_classes), c_copies, c_classes)
                with self.assertRaises(arvados.errors.KeepWriteError, msg=case_desc):
                    self.keep_client.put(self.data, copies=w_copies, classes=w_classes, num_retries=0)
