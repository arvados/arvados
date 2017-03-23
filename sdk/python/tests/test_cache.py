from __future__ import print_function

import md5
import mock
import shutil
import random
import sys
import tempfile
import threading
import unittest

import arvados.cache
import run_test_server


def _random(n):
    return bytearray(random.getrandbits(8) for _ in xrange(n))


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
                data_in = md5.new(data_in).hexdigest() + "\n" + str(data_in)
                c.set(url, data_in)
                data_out = c.get(url)
                digest, content = data_out.split("\n", 1)
                if digest != md5.new(content).hexdigest():
                    self.ok = False
            except Exception as err:
                self.ok = False
                print("cache failed: {}".format(err), file=sys.stderr)


class CacheTest(unittest.TestCase):
    def setUp(self):
        self._dir = tempfile.mkdtemp()

    def tearDown(self):
        shutil.rmtree(self._dir)

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
