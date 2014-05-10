import gflags
import httplib
import httplib2
import logging
import os
import pprint
import sys
import types
import subprocess
import json
import UserDict
import re
import hashlib
import string
import bz2
import zlib
import fcntl
import time
import threading
import timer

global_client_object = None

from api import *
import config
import arvados.errors

class Keep:
    @staticmethod
    def global_client_object():
        global global_client_object
        if global_client_object == None:
            global_client_object = KeepClient()
        return global_client_object

    @staticmethod
    def get(locator, **kwargs):
        return Keep.global_client_object().get(locator, **kwargs)

    @staticmethod
    def put(data, **kwargs):
        return Keep.global_client_object().put(data, **kwargs)

class KeepClient(object):

    class ThreadLimiter(object):
        """
        Limit the number of threads running at a given time to
        {desired successes} minus {successes reported}. When successes
        reported == desired, wake up the remaining threads and tell
        them to quit.

        Should be used in a "with" block.
        """
        def __init__(self, todo):
            self._todo = todo
            self._done = 0
            self._todo_lock = threading.Semaphore(todo)
            self._done_lock = threading.Lock()

        def __enter__(self):
            self._todo_lock.acquire()
            return self

        def __exit__(self, type, value, traceback):
            self._todo_lock.release()

        def shall_i_proceed(self):
            """
            Return true if the current thread should do stuff. Return
            false if the current thread should just stop.
            """
            with self._done_lock:
                return (self._done < self._todo)

        def increment_done(self):
            """
            Report that the current thread was successful.
            """
            with self._done_lock:
                self._done += 1

        def done(self):
            """
            Return how many successes were reported.
            """
            with self._done_lock:
                return self._done

    class KeepWriterThread(threading.Thread):
        """
        Write a blob of data to the given Keep server. Call
        increment_done() of the given ThreadLimiter if the write
        succeeds.
        """
        def __init__(self, **kwargs):
            super(KeepClient.KeepWriterThread, self).__init__()
            self.args = kwargs

        def run(self):
            with self.args['thread_limiter'] as limiter:
                if not limiter.shall_i_proceed():
                    # My turn arrived, but the job has been done without
                    # me.
                    return
                logging.debug("KeepWriterThread %s proceeding %s %s" %
                              (str(threading.current_thread()),
                               self.args['data_hash'],
                               self.args['service_root']))
                h = httplib2.Http()
                url = self.args['service_root'] + self.args['data_hash']
                api_token = config.get('ARVADOS_API_TOKEN')
                headers = {'Authorization': "OAuth2 %s" % api_token}
                try:
                    resp, content = h.request(url.encode('utf-8'), 'PUT',
                                              headers=headers,
                                              body=self.args['data'])
                    if (resp['status'] == '401' and
                        re.match(r'Timestamp verification failed', content)):
                        body = KeepClient.sign_for_old_server(
                            self.args['data_hash'],
                            self.args['data'])
                        h = httplib2.Http()
                        resp, content = h.request(url.encode('utf-8'), 'PUT',
                                                  headers=headers,
                                                  body=body)
                    if re.match(r'^2\d\d$', resp['status']):
                        logging.debug("KeepWriterThread %s succeeded %s %s" %
                                      (str(threading.current_thread()),
                                       self.args['data_hash'],
                                       self.args['service_root']))
                        return limiter.increment_done()
                    logging.warning("Request fail: PUT %s => %s %s" %
                                    (url, resp['status'], content))
                except (httplib2.HttpLib2Error, httplib.HTTPException) as e:
                    logging.warning("Request fail: PUT %s => %s: %s" %
                                    (url, type(e), str(e)))

    def __init__(self):
        self.lock = threading.Lock()
        self.service_roots = None
        self._cache_lock = threading.Lock()
        self._cache = []
        # default 256 megabyte cache
        self.cache_max = 256 * 1024 * 1024

    def shuffled_service_roots(self, hash):
        if self.service_roots == None:
            self.lock.acquire()
            try:
                keep_disks = arvados.api().keep_disks().list().execute()['items']
                roots = (("http%s://%s:%d/" %
                          ('s' if f['service_ssl_flag'] else '',
                           f['service_host'],
                           f['service_port']))
                         for f in keep_disks)
                self.service_roots = sorted(set(roots))
                logging.debug(str(self.service_roots))
            finally:
                self.lock.release()

        # Build an ordering with which to query the Keep servers based on the
        # contents of the hash.
        # "hash" is a hex-encoded number at least 8 digits
        # (32 bits) long

        # seed used to calculate the next keep server from 'pool'
        # to be added to 'pseq'
        seed = hash

        # Keep servers still to be added to the ordering
        pool = self.service_roots[:]

        # output probe sequence
        pseq = []

        # iterate while there are servers left to be assigned
        while len(pool) > 0:
            if len(seed) < 8:
                # ran out of digits in the seed
                if len(pseq) < len(hash) / 4:
                    # the number of servers added to the probe sequence is less
                    # than the number of 4-digit slices in 'hash' so refill the
                    # seed with the last 4 digits and then append the contents
                    # of 'hash'.
                    seed = hash[-4:] + hash
                else:
                    # refill the seed with the contents of 'hash'
                    seed += hash

            # Take the next 8 digits (32 bytes) and interpret as an integer,
            # then modulus with the size of the remaining pool to get the next
            # selected server.
            probe = int(seed[0:8], 16) % len(pool)

            # Append the selected server to the probe sequence and remove it
            # from the pool.
            pseq += [pool[probe]]
            pool = pool[:probe] + pool[probe+1:]

            # Remove the digits just used from the seed
            seed = seed[8:]
        logging.debug(str(pseq))
        return pseq

    class CacheSlot(object):
        def __init__(self, locator):
            self.locator = locator
            self.ready = threading.Event()
            self.content = None

        def get(self):
            self.ready.wait()
            return self.content

        def set(self, value):
            self.content = value
            self.ready.set()

        def size(self):
            if self.content == None:
                return 0
            else:
                return len(self.content)

    def cap_cache(self):
        '''Cap the cache size to self.cache_max'''
        self._cache_lock.acquire()
        try:
            self._cache = filter(lambda c: not (c.ready.is_set() and c.content == None), self._cache)
            sm = sum([slot.size() for slot in self._cache])
            while sm > self.cache_max:
                del self._cache[-1]
                sm = sum([slot.size() for a in self._cache])
        finally:
            self._cache_lock.release()

    def reserve_cache(self, locator):
        '''Reserve a cache slot for the specified locator,
        or return the existing slot.'''
        self._cache_lock.acquire()
        try:
            # Test if the locator is already in the cache
            for i in xrange(0, len(self._cache)):
                if self._cache[i].locator == locator:
                    n = self._cache[i]
                    if i != 0:
                        # move it to the front
                        del self._cache[i]
                        self._cache.insert(0, n)
                    return n, False

            # Add a new cache slot for the locator
            n = KeepClient.CacheSlot(locator)
            self._cache.insert(0, n)
            return n, True
        finally:
            self._cache_lock.release()

    def get(self, locator):
        #logging.debug("Keep.get %s" % (locator))

        if re.search(r',', locator):
            return ''.join(self.get(x) for x in locator.split(','))
        if 'KEEP_LOCAL_STORE' in os.environ:
            return KeepClient.local_store_get(locator)
        expect_hash = re.sub(r'\+.*', '', locator)

        slot, first = self.reserve_cache(expect_hash)
        #logging.debug("%s %s %s" % (slot, first, expect_hash))

        if not first:
            v = slot.get()
            return v

        try:
            for service_root in self.shuffled_service_roots(expect_hash):
                url = service_root + expect_hash
                api_token = config.get('ARVADOS_API_TOKEN')
                headers = {'Authorization': "OAuth2 %s" % api_token,
                           'Accept': 'application/octet-stream'}
                blob = self.get_url(url, headers, expect_hash)
                if blob:
                    slot.set(blob)
                    self.cap_cache()
                    return blob

            for location_hint in re.finditer(r'\+K@([a-z0-9]+)', locator):
                instance = location_hint.group(1)
                url = 'http://keep.' + instance + '.arvadosapi.com/' + expect_hash
                blob = self.get_url(url, {}, expect_hash)
                if blob:
                    slot.set(blob)
                    self.cap_cache()
                    return blob
        except:
            slot.set(None)
            self.cap_cache()
            raise

        slot.set(None)
        self.cap_cache()
        raise arvados.errors.NotFoundError("Block not found: %s" % expect_hash)

    def get_url(self, url, headers, expect_hash):
        h = httplib2.Http()
        try:
            logging.info("Request: GET %s" % (url))
            with timer.Timer() as t:
                resp, content = h.request(url.encode('utf-8'), 'GET',
                                          headers=headers)
            logging.info("Received %s bytes in %s msec (%s MiB/sec)" % (len(content),
                                                                        t.msecs,
                                                                        (len(content)/(1024*1024))/t.secs))
            if re.match(r'^2\d\d$', resp['status']):
                m = hashlib.new('md5')
                m.update(content)
                md5 = m.hexdigest()
                if md5 == expect_hash:
                    return content
                logging.warning("Checksum fail: md5(%s) = %s" % (url, md5))
        except Exception as e:
            logging.info("Request fail: GET %s => %s: %s" %
                         (url, type(e), str(e)))
        return None

    def put(self, data, **kwargs):
        if 'KEEP_LOCAL_STORE' in os.environ:
            return KeepClient.local_store_put(data)
        m = hashlib.new('md5')
        m.update(data)
        data_hash = m.hexdigest()
        have_copies = 0
        want_copies = kwargs.get('copies', 2)
        if not (want_copies > 0):
            return data_hash
        threads = []
        thread_limiter = KeepClient.ThreadLimiter(want_copies)
        for service_root in self.shuffled_service_roots(data_hash):
            t = KeepClient.KeepWriterThread(data=data,
                                            data_hash=data_hash,
                                            service_root=service_root,
                                            thread_limiter=thread_limiter)
            t.start()
            threads += [t]
        for t in threads:
            t.join()
        have_copies = thread_limiter.done()
        if have_copies == want_copies:
            return (data_hash + '+' + str(len(data)))
        raise arvados.errors.KeepWriteError(
            "Write fail for %s: wanted %d but wrote %d" %
            (data_hash, want_copies, have_copies))

    @staticmethod
    def sign_for_old_server(data_hash, data):
        return (("-----BEGIN PGP SIGNED MESSAGE-----\n\n\n%d %s\n-----BEGIN PGP SIGNATURE-----\n\n-----END PGP SIGNATURE-----\n" % (int(time.time()), data_hash)) + data)


    @staticmethod
    def local_store_put(data):
        m = hashlib.new('md5')
        m.update(data)
        md5 = m.hexdigest()
        locator = '%s+%d' % (md5, len(data))
        with open(os.path.join(os.environ['KEEP_LOCAL_STORE'], md5 + '.tmp'), 'w') as f:
            f.write(data)
        os.rename(os.path.join(os.environ['KEEP_LOCAL_STORE'], md5 + '.tmp'),
                  os.path.join(os.environ['KEEP_LOCAL_STORE'], md5))
        return locator

    @staticmethod
    def local_store_get(locator):
        r = re.search('^([0-9a-f]{32,})', locator)
        if not r:
            raise arvados.errors.NotFoundError(
                "Invalid data locator: '%s'" % locator)
        if r.group(0) == config.EMPTY_BLOCK_LOCATOR.split('+')[0]:
            return ''
        with open(os.path.join(os.environ['KEEP_LOCAL_STORE'], r.group(0)), 'r') as f:
            return f.read()
