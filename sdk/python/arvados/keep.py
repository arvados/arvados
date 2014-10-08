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
import datetime
import ssl
import socket

_logger = logging.getLogger('arvados.keep')
global_client_object = None

import arvados
import arvados.config as config
import arvados.errors
import arvados.retry as retry
import arvados.util

class KeepLocator(object):
    EPOCH_DATETIME = datetime.datetime.utcfromtimestamp(0)
    HINT_RE = re.compile(r'^[A-Z][A-Za-z0-9@_-]+$')

    def __init__(self, locator_str):
        self.hints = []
        self._perm_sig = None
        self._perm_expiry = None
        pieces = iter(locator_str.split('+'))
        self.md5sum = next(pieces)
        try:
            self.size = int(next(pieces))
        except StopIteration:
            self.size = None
        for hint in pieces:
            if self.HINT_RE.match(hint) is None:
                raise ValueError("unrecognized hint data {}".format(hint))
            elif hint.startswith('A'):
                self.parse_permission_hint(hint)
            else:
                self.hints.append(hint)

    def __str__(self):
        return '+'.join(
            str(s) for s in [self.md5sum, self.size,
                             self.permission_hint()] + self.hints
            if s is not None)

    def _make_hex_prop(name, length):
        # Build and return a new property with the given name that
        # must be a hex string of the given length.
        data_name = '_{}'.format(name)
        def getter(self):
            return getattr(self, data_name)
        def setter(self, hex_str):
            if not arvados.util.is_hex(hex_str, length):
                raise ValueError("{} must be a {}-digit hex string: {}".
                                 format(name, length, hex_str))
            setattr(self, data_name, hex_str)
        return property(getter, setter)

    md5sum = _make_hex_prop('md5sum', 32)
    perm_sig = _make_hex_prop('perm_sig', 40)

    @property
    def perm_expiry(self):
        return self._perm_expiry

    @perm_expiry.setter
    def perm_expiry(self, value):
        if not arvados.util.is_hex(value, 1, 8):
            raise ValueError(
                "permission timestamp must be a hex Unix timestamp: {}".
                format(value))
        self._perm_expiry = datetime.datetime.utcfromtimestamp(int(value, 16))

    def permission_hint(self):
        data = [self.perm_sig, self.perm_expiry]
        if None in data:
            return None
        data[1] = int((data[1] - self.EPOCH_DATETIME).total_seconds())
        return "A{}@{:08x}".format(*data)

    def parse_permission_hint(self, s):
        try:
            self.perm_sig, self.perm_expiry = s[1:].split('@', 1)
        except IndexError:
            raise ValueError("bad permission hint {}".format(s))

    def permission_expired(self, as_of_dt=None):
        if self.perm_expiry is None:
            return False
        elif as_of_dt is None:
            as_of_dt = datetime.datetime.now()
        return self.perm_expiry <= as_of_dt


class Keep(object):
    """Simple interface to a global KeepClient object.

    THIS CLASS IS DEPRECATED.  Please instantiate your own KeepClient with your
    own API client.  The global KeepClient will build an API client from the
    current Arvados configuration, which may not match the one you built.
    """
    _last_key = None

    @classmethod
    def global_client_object(cls):
        global global_client_object
        # Previously, KeepClient would change its behavior at runtime based
        # on these configuration settings.  We simulate that behavior here
        # by checking the values and returning a new KeepClient if any of
        # them have changed.
        key = (config.get('ARVADOS_API_HOST'),
               config.get('ARVADOS_API_TOKEN'),
               config.flag_is_true('ARVADOS_API_HOST_INSECURE'),
               config.get('ARVADOS_KEEP_PROXY'),
               config.get('ARVADOS_EXTERNAL_CLIENT') == 'true',
               os.environ.get('KEEP_LOCAL_STORE'))
        if (global_client_object is None) or (cls._last_key != key):
            global_client_object = KeepClient()
            cls._last_key = key
        return global_client_object

    @staticmethod
    def get(locator, **kwargs):
        return Keep.global_client_object().get(locator, **kwargs)

    @staticmethod
    def put(data, **kwargs):
        return Keep.global_client_object().put(data, **kwargs)

class KeepBlockCache(object):
    # Default RAM cache is 256MiB
    def __init__(self, cache_max=(256 * 1024 * 1024)):
        self.cache_max = cache_max
        self._cache = []
        self._cache_lock = threading.Lock()

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
            # Select all slots except those where ready.is_set() and content is
            # None (that means there was an error reading the block).
            self._cache = [c for c in self._cache if not (c.ready.is_set() and c.content is None)]
            sm = sum([slot.size() for slot in self._cache])
            while len(self._cache) > 0 and sm > self.cache_max:
                for i in xrange(len(self._cache)-1, -1, -1):
                    if self._cache[i].ready.is_set():
                        del self._cache[i]
                        break
                sm = sum([slot.size() for slot in self._cache])
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
            n = KeepBlockCache.CacheSlot(locator)
            self._cache.insert(0, n)
            return n, True
        finally:
            self._cache_lock.release()

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
            self._response = None
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

        def save_response(self, response_body, replicas_stored):
            """
            Records a response body (a locator, possibly signed) returned by
            the Keep server.  It is not necessary to save more than
            one response, since we presume that any locator returned
            in response to a successful request is valid.
            """
            with self._done_lock:
                self._done += replicas_stored
                self._response = response_body

        def response(self):
            """
            Returns the body from the response to a PUT request.
            """
            with self._done_lock:
                return self._response

        def done(self):
            """
            Return how many successes were reported.
            """
            with self._done_lock:
                return self._done


    class KeepService(object):
        # Make requests to a single Keep service, and track results.
        HTTP_ERRORS = (httplib2.HttpLib2Error, httplib.HTTPException,
                       socket.error, ssl.SSLError)

        def __init__(self, root, **headers):
            self.root = root
            self.last_result = None
            self.success_flag = None
            self.get_headers = {'Accept': 'application/octet-stream'}
            self.get_headers.update(headers)
            self.put_headers = headers

        def usable(self):
            return self.success_flag is not False

        def finished(self):
            return self.success_flag is not None

        def last_status(self):
            try:
                return int(self.last_result[0].status)
            except (AttributeError, IndexError, ValueError):
                return None

        def get(self, http, locator):
            # http is an httplib2.Http object.
            # locator is a KeepLocator object.
            url = self.root + str(locator)
            _logger.debug("Request: GET %s", url)
            try:
                with timer.Timer() as t:
                    result = http.request(url.encode('utf-8'), 'GET',
                                          headers=self.get_headers)
            except self.HTTP_ERRORS as e:
                _logger.debug("Request fail: GET %s => %s: %s",
                              url, type(e), str(e))
                self.last_result = e
            else:
                self.last_result = result
                self.success_flag = retry.check_http_response_success(result)
                content = result[1]
                _logger.info("%s response: %s bytes in %s msec (%.3f MiB/sec)",
                             self.last_status(), len(content), t.msecs,
                             (len(content)/(1024.0*1024))/t.secs)
                if self.success_flag:
                    resp_md5 = hashlib.md5(content).hexdigest()
                    if resp_md5 == locator.md5sum:
                        return content
                    _logger.warning("Checksum fail: md5(%s) = %s",
                                    url, resp_md5)
            return None

        def put(self, http, hash_s, body):
            url = self.root + hash_s
            _logger.debug("Request: PUT %s", url)
            try:
                result = http.request(url.encode('utf-8'), 'PUT',
                                      headers=self.put_headers, body=body)
            except self.HTTP_ERRORS as e:
                _logger.debug("Request fail: PUT %s => %s: %s",
                              url, type(e), str(e))
                self.last_result = e
            else:
                self.last_result = result
                self.success_flag = retry.check_http_response_success(result)
            return self.success_flag


    class KeepWriterThread(threading.Thread):
        """
        Write a blob of data to the given Keep server. On success, call
        save_response() of the given ThreadLimiter to save the returned
        locator.
        """
        def __init__(self, keep_service, **kwargs):
            super(KeepClient.KeepWriterThread, self).__init__()
            self.service = keep_service
            self.args = kwargs
            self._success = False

        def success(self):
            return self._success

        def run(self):
            with self.args['thread_limiter'] as limiter:
                if not limiter.shall_i_proceed():
                    # My turn arrived, but the job has been done without
                    # me.
                    return
                self.run_with_limiter(limiter)

        def run_with_limiter(self, limiter):
            if self.service.finished():
                return
            _logger.debug("KeepWriterThread %s proceeding %s %s",
                          str(threading.current_thread()),
                          self.args['data_hash'],
                          self.args['service_root'])
            h = httplib2.Http(timeout=self.args.get('timeout', None))
            self._success = bool(self.service.put(
                    h, self.args['data_hash'], self.args['data']))
            status = self.service.last_status()
            if self._success:
                resp, body = self.service.last_result
                _logger.debug("KeepWriterThread %s succeeded %s %s",
                              str(threading.current_thread()),
                              self.args['data_hash'],
                              self.args['service_root'])
                # Tick the 'done' counter for the number of replica
                # reported stored by the server, for the case that
                # we're talking to a proxy or other backend that
                # stores to multiple copies for us.
                try:
                    replicas_stored = int(resp['x-keep-replicas-stored'])
                except (KeyError, ValueError):
                    replicas_stored = 1
                limiter.save_response(body.strip(), replicas_stored)
            elif status is not None:
                _logger.debug("Request fail: PUT %s => %s %s",
                              self.args['data_hash'], status,
                              self.service.last_result[1])


    def __init__(self, api_client=None, proxy=None, timeout=300,
                 api_token=None, local_store=None, block_cache=None,
                 num_retries=0):
        """Initialize a new KeepClient.

        Arguments:
        * api_client: The API client to use to find Keep services.  If not
          provided, KeepClient will build one from available Arvados
          configuration.
        * proxy: If specified, this KeepClient will send requests to this
          Keep proxy.  Otherwise, KeepClient will fall back to the setting
          of the ARVADOS_KEEP_PROXY configuration setting.  If you want to
          ensure KeepClient does not use a proxy, pass in an empty string.
        * timeout: The timeout for all HTTP requests, in seconds.  Default
          300.
        * api_token: If you're not using an API client, but only talking
          directly to a Keep proxy, this parameter specifies an API token
          to authenticate Keep requests.  It is an error to specify both
          api_client and api_token.  If you specify neither, KeepClient
          will use one available from the Arvados configuration.
        * local_store: If specified, this KeepClient will bypass Keep
          services, and save data to the named directory.  If unspecified,
          KeepClient will fall back to the setting of the $KEEP_LOCAL_STORE
          environment variable.  If you want to ensure KeepClient does not
          use local storage, pass in an empty string.  This is primarily
          intended to mock a server for testing.
        * num_retries: The default number of times to retry failed requests.
          This will be used as the default num_retries value when get() and
          put() are called.  Default 0.
        """
        self.lock = threading.Lock()
        if proxy is None:
            proxy = config.get('ARVADOS_KEEP_PROXY')
        if api_token is None:
            if api_client is None:
                api_token = config.get('ARVADOS_API_TOKEN')
            else:
                api_token = api_client.api_token
        elif api_client is not None:
            raise ValueError(
                "can't build KeepClient with both API client and token")
        if local_store is None:
            local_store = os.environ.get('KEEP_LOCAL_STORE')

        self.block_cache = block_cache if block_cache else KeepBlockCache()

        if local_store:
            self.local_store = local_store
            self.get = self.local_store_get
            self.put = self.local_store_put
        else:
            self.timeout = timeout
            self.num_retries = num_retries
            if proxy:
                if not proxy.endswith('/'):
                    proxy += '/'
                self.api_token = api_token
                self.service_roots = [proxy]
                self.using_proxy = True
                self.static_service_roots = True
            else:
                # It's important to avoid instantiating an API client
                # unless we actually need one, for testing's sake.
                if api_client is None:
                    api_client = arvados.api('v1')
                self.api_client = api_client
                self.api_token = api_client.api_token
                self.service_roots = None
                self.using_proxy = None
                self.static_service_roots = False

    def build_service_roots(self, force_rebuild=False):
        if (self.static_service_roots or
              (self.service_roots and not force_rebuild)):
            return
        with self.lock:
            try:
                keep_services = self.api_client.keep_services().accessible()
            except Exception:  # API server predates Keep services.
                keep_services = self.api_client.keep_disks().list()

            keep_services = keep_services.execute().get('items')
            if not keep_services:
                raise arvados.errors.NoKeepServersError()

            self.using_proxy = any(ks.get('service_type') == 'proxy'
                                   for ks in keep_services)

            roots = ("{}://[{}]:{:d}/".format(
                        'https' if ks['service_ssl_flag'] else 'http',
                         ks['service_host'],
                         ks['service_port'])
                     for ks in keep_services)
            self.service_roots = sorted(set(roots))
            _logger.debug(str(self.service_roots))

    def shuffled_service_roots(self, hash, force_rebuild=False):
        self.build_service_roots(force_rebuild)

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
        _logger.debug(str(pseq))
        return pseq


    def map_new_services(self, roots_map, md5_s, force_rebuild, **headers):
        # roots_map is a dictionary, mapping Keep service root strings
        # to KeepService objects.  Poll for Keep services, and add any
        # new ones to roots_map.  Return the current list of local
        # root strings.
        headers.setdefault('Authorization', "OAuth2 %s" % (self.api_token,))
        local_roots = self.shuffled_service_roots(md5_s, force_rebuild)
        for root in local_roots:
            if root not in roots_map:
                roots_map[root] = self.KeepService(root, **headers)
        return local_roots

    @staticmethod
    def _check_loop_result(result):
        # KeepClient RetryLoops should save results as a 2-tuple: the
        # actual result of the request, and the number of servers available
        # to receive the request this round.
        # This method returns True if there's a real result, False if
        # there are no more servers available, otherwise None.
        if isinstance(result, Exception):
            return None
        result, tried_server_count = result
        if (result is not None) and (result is not False):
            return True
        elif tried_server_count < 1:
            _logger.info("No more Keep services to try; giving up")
            return False
        else:
            return None

    @retry.retry_method
    def get(self, loc_s, num_retries=None):
        """Get data from Keep.

        This method fetches one or more blocks of data from Keep.  It
        sends a request each Keep service registered with the API
        server (or the proxy provided when this client was
        instantiated), then each service named in location hints, in
        sequence.  As soon as one service provides the data, it's
        returned.

        Arguments:
        * loc_s: A string of one or more comma-separated locators to fetch.
          This method returns the concatenation of these blocks.
        * num_retries: The number of times to retry GET requests to
          *each* Keep server if it returns temporary failures, with
          exponential backoff.  Note that, in each loop, the method may try
          to fetch data from every available Keep service, along with any
          that are named in location hints in the locator.  The default value
          is set when the KeepClient is initialized.
        """
        if ',' in loc_s:
            return ''.join(self.get(x) for x in loc_s.split(','))
        locator = KeepLocator(loc_s)
        expect_hash = locator.md5sum

        slot, first = self.block_cache.reserve_cache(expect_hash)
        if not first:
            v = slot.get()
            return v

        # See #3147 for a discussion of the loop implementation.  Highlights:
        # * Refresh the list of Keep services after each failure, in case
        #   it's being updated.
        # * Retry until we succeed, we're out of retries, or every available
        #   service has returned permanent failure.
        hint_roots = ['http://keep.{}.arvadosapi.com/'.format(hint[2:])
                      for hint in locator.hints if hint.startswith('K@')]
        # Map root URLs their KeepService objects.
        roots_map = {root: self.KeepService(root) for root in hint_roots}
        blob = None
        loop = retry.RetryLoop(num_retries, self._check_loop_result,
                               backoff_start=2)
        for tries_left in loop:
            try:
                local_roots = self.map_new_services(
                    roots_map, expect_hash,
                    force_rebuild=(tries_left < num_retries))
            except Exception as error:
                loop.save_result(error)
                continue

            # Query KeepService objects that haven't returned
            # permanent failure, in our specified shuffle order.
            services_to_try = [roots_map[root]
                               for root in (local_roots + hint_roots)
                               if roots_map[root].usable()]
            http = httplib2.Http(timeout=self.timeout)
            for keep_service in services_to_try:
                blob = keep_service.get(http, locator)
                if blob is not None:
                    break
            loop.save_result((blob, len(services_to_try)))

        # Always cache the result, then return it if we succeeded.
        slot.set(blob)
        self.block_cache.cap_cache()
        if loop.success():
            return blob

        # No servers fulfilled the request.  Count how many responded
        # "not found;" if the ratio is high enough (currently 75%), report
        # Not Found; otherwise a generic error.
        # Q: Including 403 is necessary for the Keep tests to continue
        # passing, but maybe they should expect KeepReadError instead?
        not_founds = sum(1 for ks in roots_map.values()
                         if ks.last_status() in set([403, 404, 410]))
        if roots_map and ((float(not_founds) / len(roots_map)) >= .75):
            raise arvados.errors.NotFoundError(loc_s)
        else:
            raise arvados.errors.KeepReadError(loc_s)

    @retry.retry_method
    def put(self, data, copies=2, num_retries=None):
        """Save data in Keep.

        This method will get a list of Keep services from the API server, and
        send the data to each one simultaneously in a new thread.  Once the
        uploads are finished, if enough copies are saved, this method returns
        the most recent HTTP response body.  If requests fail to upload
        enough copies, this method raises KeepWriteError.

        Arguments:
        * data: The string of data to upload.
        * copies: The number of copies that the user requires be saved.
          Default 2.
        * num_retries: The number of times to retry PUT requests to
          *each* Keep server if it returns temporary failures, with
          exponential backoff.  The default value is set when the
          KeepClient is initialized.
        """
        data_hash = hashlib.md5(data).hexdigest()
        if copies < 1:
            return data_hash

        headers = {}
        if self.using_proxy:
            # Tell the proxy how many copies we want it to store
            headers['X-Keep-Desired-Replication'] = str(copies)
        roots_map = {}
        thread_limiter = KeepClient.ThreadLimiter(copies)
        loop = retry.RetryLoop(num_retries, self._check_loop_result,
                               backoff_start=2)
        for tries_left in loop:
            try:
                local_roots = self.map_new_services(
                    roots_map, data_hash,
                    force_rebuild=(tries_left < num_retries), **headers)
            except Exception as error:
                loop.save_result(error)
                continue

            threads = []
            for service_root, ks in roots_map.iteritems():
                if ks.finished():
                    continue
                t = KeepClient.KeepWriterThread(
                    ks,
                    data=data,
                    data_hash=data_hash,
                    service_root=service_root,
                    thread_limiter=thread_limiter,
                    timeout=self.timeout)
                t.start()
                threads.append(t)
            for t in threads:
                t.join()
            loop.save_result((thread_limiter.done() >= copies, len(threads)))

        if loop.success():
            return thread_limiter.response()
        raise arvados.errors.KeepWriteError(
            "Write fail for %s: wanted %d but wrote %d" %
            (data_hash, copies, thread_limiter.done()))

    # Local storage methods need no-op num_retries arguments to keep
    # integration tests happy.  With better isolation they could
    # probably be removed again.
    def local_store_put(self, data, num_retries=0):
        md5 = hashlib.md5(data).hexdigest()
        locator = '%s+%d' % (md5, len(data))
        with open(os.path.join(self.local_store, md5 + '.tmp'), 'w') as f:
            f.write(data)
        os.rename(os.path.join(self.local_store, md5 + '.tmp'),
                  os.path.join(self.local_store, md5))
        return locator

    def local_store_get(self, loc_s, num_retries=0):
        try:
            locator = KeepLocator(loc_s)
        except ValueError:
            raise arvados.errors.NotFoundError(
                "Invalid data locator: '%s'" % loc_s)
        if locator.md5sum == config.EMPTY_BLOCK_LOCATOR.split('+')[0]:
            return ''
        with open(os.path.join(self.local_store, locator.md5sum), 'r') as f:
            return f.read()
