# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import hashlib
import os
import random
import shutil
import sys
import tempfile
import threading
import unittest

import pytest
from unittest import mock

import arvados
import arvados.api
import arvados.util
from arvados._internal import basedirs

from . import run_test_server

def _random(n):
    return bytearray(random.getrandbits(8) for _ in range(n))

class CacheTestThread(threading.Thread):
    def __init__(self, dir):
        super(CacheTestThread, self).__init__()
        self._dir = dir

    def run(self):
        c = arvados.api.ThreadSafeHTTPCache(self._dir)
        url = 'http://example.com/foo'
        self.ok = True
        for x in range(16):
            try:
                data_in = _random(128)
                data_in = hashlib.md5(data_in).hexdigest().encode() + b"\n" + data_in
                c.set(url, data_in)
                data_out = c.get(url)
                digest, _, content = data_out.partition(b"\n")
                if digest != hashlib.md5(content).hexdigest().encode():
                    self.ok = False
            except Exception as err:
                self.ok = False
                print("cache failed: {}: {}".format(type(err), err), file=sys.stderr)
                raise


class TestAPIHTTPCache:
    @pytest.mark.parametrize('data_type', ['discovery', 'keep'])
    def test_good_storage(self, tmp_path, monkeypatch, data_type):
        def storage_path(self, subdir='.', mode=0o700):
            path = tmp_path / subdir
            path.mkdir(mode=mode)
            return path
        monkeypatch.setattr(basedirs.BaseDirectories, 'storage_path', storage_path)
        actual = arvados.http_cache(data_type)
        assert str(actual) == str(tmp_path / data_type)

    @pytest.mark.parametrize('error', [RuntimeError, FileExistsError, PermissionError])
    def test_unwritable_storage(self, monkeypatch, error):
        def fail(self, subdir='.', mode=0o700):
            raise error()
        monkeypatch.setattr(basedirs.BaseDirectories, 'storage_path', fail)
        actual = arvados.http_cache('unwritable')
        assert actual is None


class CacheTest(unittest.TestCase):
    def setUp(self):
        self._dir = tempfile.mkdtemp()

    def tearDown(self):
        shutil.rmtree(self._dir)

    def test_cache_crud(self):
        c = arvados.api.ThreadSafeHTTPCache(self._dir, max_age=0)
        url = 'https://example.com/foo?bar=baz'
        data1 = _random(256)
        data2 = _random(128)
        self.assertEqual(None, c.get(url))
        c.delete(url)
        c.set(url, data1)
        self.assertEqual(data1, c.get(url))
        c.delete(url)
        self.assertEqual(None, c.get(url))
        c.set(url, data1)
        c.set(url, data2)
        self.assertEqual(data2, c.get(url))

    def test_cache_threads(self):
        threads = []
        for _ in range(64):
            t = CacheTestThread(dir=self._dir)
            t.start()
            threads.append(t)
        for t in threads:
            t.join()
            self.assertTrue(t.ok)


class CacheIntegrationTest(run_test_server.TestCaseWithServers):
    MAIN_SERVER = {}

    def test_cache_used_by_default_client(self):
        with mock.patch('arvados.api.ThreadSafeHTTPCache.get') as getter:
            arvados.api('v1')._rootDesc.get('foobar')
            getter.assert_called()
