from __future__ import print_function
from __future__ import absolute_import

from builtins import str
from builtins import range
import hashlib
import mock
import os
import random
import shutil
import sys
import tempfile
import threading
import unittest

import arvados
import arvados.cache
from . import run_test_server


def _random(n):
    return bytearray(random.getrandbits(8) for _ in range(n))


class CacheTestThread(threading.Thread):
    def __init__(self, dir):
        super(CacheTestThread, self).__init__()
        self._dir = dir

    def run(self):
        c = arvados.cache.SafeHTTPCache(self._dir)
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


class CacheTest(unittest.TestCase):
    def setUp(self):
        self._dir = tempfile.mkdtemp()

    def tearDown(self):
        shutil.rmtree(self._dir)

    def test_cache_create_error(self):
        _, filename = tempfile.mkstemp()
        home_was = os.environ['HOME']
        os.environ['HOME'] = filename
        try:
            c = arvados.http_cache('test')
            self.assertEqual(None, c)
        finally:
            os.environ['HOME'] = home_was
            os.unlink(filename)

    def test_cache_crud(self):
        c = arvados.cache.SafeHTTPCache(self._dir, max_age=0)
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
        with mock.patch('arvados.cache.SafeHTTPCache.get') as getter:
            arvados.api('v1')._rootDesc.get('foobar')
            getter.assert_called()
