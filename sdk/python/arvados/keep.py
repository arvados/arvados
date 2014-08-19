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

_logger = logging.getLogger('arvados.keep')
global_client_object = None

import arvados
import arvados.config as config
import arvados.errors
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


    class KeepWriterThread(threading.Thread):
        """
        Write a blob of data to the given Keep server. On success, call
        save_response() of the given ThreadLimiter to save the returned
        locator.
        """
        def __init__(self, api_token, **kwargs):
            super(KeepClient.KeepWriterThread, self).__init__()
            self._api_token = api_token
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
            _logger.debug("KeepWriterThread %s proceeding %s %s",
                          str(threading.current_thread()),
                          self.args['data_hash'],
                          self.args['service_root'])
            h = httplib2.Http(timeout=self.args.get('timeout', None))
            url = self.args['service_root'] + self.args['data_hash']
            headers = {'Authorization': "OAuth2 %s" % (self._api_token,)}

            if self.args['using_proxy']:
                # We're using a proxy, so tell the proxy how many copies we
                # want it to store
                headers['X-Keep-Desired-Replication'] = str(self.args['want_copies'])

            try:
                _logger.debug("Uploading to {}".format(url))
                resp, content = h.request(url.encode('utf-8'), 'PUT',
                                          headers=headers,
                                          body=self.args['data'])
                if (resp['status'] == '401' and
                    re.match(r'Timestamp verification failed', content)):
                    body = KeepClient.sign_for_old_server(
                        self.args['data_hash'],
                        self.args['data'])
                    h = httplib2.Http(timeout=self.args.get('timeout', None))
                    resp, content = h.request(url.encode('utf-8'), 'PUT',
                                              headers=headers,
                                              body=body)
                if re.match(r'^2\d\d$', resp['status']):
                    self._success = True
                    _logger.debug("KeepWriterThread %s succeeded %s %s",
                                  str(threading.current_thread()),
                                  self.args['data_hash'],
                                  self.args['service_root'])
                    replicas_stored = 1
                    if 'x-keep-replicas-stored' in resp:
                        # Tick the 'done' counter for the number of replica
                        # reported stored by the server, for the case that
                        # we're talking to a proxy or other backend that
                        # stores to multiple copies for us.
                        try:
                            replicas_stored = int(resp['x-keep-replicas-stored'])
                        except ValueError:
                            pass
                    limiter.save_response(content.strip(), replicas_stored)
                else:
                    _logger.debug("Request fail: PUT %s => %s %s",
                                    url, resp['status'], content)
            except (httplib2.HttpLib2Error,
                    httplib.HTTPException,
                    ssl.SSLError) as e:
                # When using https, timeouts look like ssl.SSLError from here.
                # "SSLError: The write operation timed out"
                _logger.debug("Request fail: PUT %s => %s: %s",
                                url, type(e), str(e))


    def __init__(self, api_client=None, proxy=None, timeout=60,
                 api_token=None, local_store=None):
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
          60.
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
        """
        self.lock = threading.Lock()
        if proxy is None:
            proxy = config.get('ARVADOS_KEEP_PROXY')
        if api_token is None:
            api_token = config.get('ARVADOS_API_TOKEN')
        elif api_client is not None:
            raise ValueError(
                "can't build KeepClient with both API client and token")
        if local_store is None:
            local_store = os.environ.get('KEEP_LOCAL_STORE')

        if local_store:
            self.local_store = local_store
            self.get = self.local_store_get
            self.put = self.local_store_put
        else:
            self.timeout = timeout
            self.cache_max = 256 * 1024 * 1024  # Cache is 256MiB
            self._cache = []
            self._cache_lock = threading.Lock()
            if proxy:
                if not proxy.endswith('/'):
                    proxy += '/'
                self.api_token = api_token
                self.service_roots = [proxy]
                self.using_proxy = True
            else:
                # It's important to avoid instantiating an API client
                # unless we actually need one, for testing's sake.
                if api_client is None:
                    api_client = arvados.api('v1')
                self.api_client = api_client
                self.api_token = api_client.api_token
                self.service_roots = None
                self.using_proxy = None

    def shuffled_service_roots(self, hash):
        if self.service_roots is None:
            with self.lock:
                try:
                    keep_services = self.api_client.keep_services().accessible()
                except Exception:  # API server predates Keep services.
                    keep_services = self.api_client.keep_disks().list()

                keep_services = keep_services.execute().get('items')
                if not keep_services:
                    raise arvados.errors.NoKeepServersError()

                self.using_proxy = (keep_services[0].get('service_type') ==
                                    'proxy')

                roots = (("http%s://%s:%d/" %
                          ('s' if f['service_ssl_flag'] else '',
                           f['service_host'],
                           f['service_port']))
                         for f in keep_services)
                self.service_roots = sorted(set(roots))
                _logger.debug(str(self.service_roots))

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

    def get(self, loc_s):
        if ',' in loc_s:
            return ''.join(self.get(x) for x in loc_s.split(','))
        locator = KeepLocator(loc_s)
        expect_hash = locator.md5sum

        slot, first = self.reserve_cache(expect_hash)

        if not first:
            v = slot.get()
            return v

        try:
            for service_root in self.shuffled_service_roots(expect_hash):
                url = service_root + loc_s
                headers = {'Authorization': "OAuth2 %s" % (self.api_token,),
                           'Accept': 'application/octet-stream'}
                blob = self.get_url(url, headers, expect_hash)
                if blob:
                    slot.set(blob)
                    self.cap_cache()
                    return blob

            for hint in locator.hints:
                if not hint.startswith('K@'):
                    continue
                url = 'http://keep.' + hint[2:] + '.arvadosapi.com/' + loc_s
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
            _logger.debug("Request: GET %s", url)
            with timer.Timer() as t:
                resp, content = h.request(url.encode('utf-8'), 'GET',
                                          headers=headers)
            _logger.info("Received %s bytes in %s msec (%s MiB/sec)",
                         len(content), t.msecs,
                         (len(content)/(1024*1024))/t.secs)
            if re.match(r'^2\d\d$', resp['status']):
                md5 = hashlib.md5(content).hexdigest()
                if md5 == expect_hash:
                    return content
                _logger.warning("Checksum fail: md5(%s) = %s", url, md5)
        except Exception as e:
            _logger.debug("Request fail: GET %s => %s: %s",
                         url, type(e), str(e))
        return None

    def put(self, data, copies=2):
        data_hash = hashlib.md5(data).hexdigest()
        have_copies = 0
        want_copies = copies
        if not (want_copies > 0):
            return data_hash
        threads = []
        thread_limiter = KeepClient.ThreadLimiter(want_copies)
        for service_root in self.shuffled_service_roots(data_hash):
            t = KeepClient.KeepWriterThread(
                self.api_token,
                data=data,
                data_hash=data_hash,
                service_root=service_root,
                thread_limiter=thread_limiter,
                timeout=self.timeout,
                using_proxy=self.using_proxy,
                want_copies=(want_copies if self.using_proxy else 1))
            t.start()
            threads += [t]
        for t in threads:
            t.join()
        if thread_limiter.done() < want_copies:
            # Retry the threads (i.e., services) that failed the first
            # time around.
            threads_retry = []
            for t in threads:
                if not t.success():
                    _logger.debug("Retrying: PUT %s %s",
                                    t.args['service_root'],
                                    t.args['data_hash'])
                    retry_with_args = t.args.copy()
                    t_retry = KeepClient.KeepWriterThread(self.api_token,
                                                          **retry_with_args)
                    t_retry.start()
                    threads_retry += [t_retry]
            for t in threads_retry:
                t.join()
        have_copies = thread_limiter.done()
        # If we're done, return the response from Keep
        if have_copies >= want_copies:
            return thread_limiter.response()
        raise arvados.errors.KeepWriteError(
            "Write fail for %s: wanted %d but wrote %d" %
            (data_hash, want_copies, have_copies))

    @staticmethod
    def sign_for_old_server(data_hash, data):
        return (("-----BEGIN PGP SIGNED MESSAGE-----\n\n\n%d %s\n-----BEGIN PGP SIGNATURE-----\n\n-----END PGP SIGNATURE-----\n" % (int(time.time()), data_hash)) + data)

    def local_store_put(self, data):
        md5 = hashlib.md5(data).hexdigest()
        locator = '%s+%d' % (md5, len(data))
        with open(os.path.join(self.local_store, md5 + '.tmp'), 'w') as f:
            f.write(data)
        os.rename(os.path.join(self.local_store, md5 + '.tmp'),
                  os.path.join(self.local_store, md5))
        return locator

    def local_store_get(self, loc_s):
        try:
            locator = KeepLocator(loc_s)
        except ValueError:
            raise arvados.errors.NotFoundError(
                "Invalid data locator: '%s'" % loc_s)
        if locator.md5sum == config.EMPTY_BLOCK_LOCATOR.split('+')[0]:
            return ''
        with open(os.path.join(self.local_store, locator.md5sum), 'r') as f:
            return f.read()
