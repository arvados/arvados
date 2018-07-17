# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import absolute_import
from __future__ import division
from future import standard_library
from future.utils import native_str
standard_library.install_aliases()
from builtins import next
from builtins import str
from builtins import range
from builtins import object
import collections
import datetime
import hashlib
import io
import logging
import math
import os
import pycurl
import queue
import re
import socket
import ssl
import sys
import threading
from . import timer
import urllib.parse

if sys.version_info >= (3, 0):
    from io import BytesIO
else:
    from cStringIO import StringIO as BytesIO

import arvados
import arvados.config as config
import arvados.errors
import arvados.retry as retry
import arvados.util

_logger = logging.getLogger('arvados.keep')
global_client_object = None


# Monkey patch TCP constants when not available (apple). Values sourced from:
# http://www.opensource.apple.com/source/xnu/xnu-2422.115.4/bsd/netinet/tcp.h
if sys.platform == 'darwin':
    if not hasattr(socket, 'TCP_KEEPALIVE'):
        socket.TCP_KEEPALIVE = 0x010
    if not hasattr(socket, 'TCP_KEEPINTVL'):
        socket.TCP_KEEPINTVL = 0x101
    if not hasattr(socket, 'TCP_KEEPCNT'):
        socket.TCP_KEEPCNT = 0x102


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
                raise ValueError("invalid hint format: {}".format(hint))
            elif hint.startswith('A'):
                self.parse_permission_hint(hint)
            else:
                self.hints.append(hint)

    def __str__(self):
        return '+'.join(
            native_str(s)
            for s in [self.md5sum, self.size,
                      self.permission_hint()] + self.hints
            if s is not None)

    def stripped(self):
        if self.size is not None:
            return "%s+%i" % (self.md5sum, self.size)
        else:
            return self.md5sum

    def _make_hex_prop(name, length):
        # Build and return a new property with the given name that
        # must be a hex string of the given length.
        data_name = '_{}'.format(name)
        def getter(self):
            return getattr(self, data_name)
        def setter(self, hex_str):
            if not arvados.util.is_hex(hex_str, length):
                raise ValueError("{} is not a {}-digit hex string: {!r}".
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
        __slots__ = ("locator", "ready", "content")

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
            if self.content is None:
                return 0
            else:
                return len(self.content)

    def cap_cache(self):
        '''Cap the cache size to self.cache_max'''
        with self._cache_lock:
            # Select all slots except those where ready.is_set() and content is
            # None (that means there was an error reading the block).
            self._cache = [c for c in self._cache if not (c.ready.is_set() and c.content is None)]
            sm = sum([slot.size() for slot in self._cache])
            while len(self._cache) > 0 and sm > self.cache_max:
                for i in range(len(self._cache)-1, -1, -1):
                    if self._cache[i].ready.is_set():
                        del self._cache[i]
                        break
                sm = sum([slot.size() for slot in self._cache])

    def _get(self, locator):
        # Test if the locator is already in the cache
        for i in range(0, len(self._cache)):
            if self._cache[i].locator == locator:
                n = self._cache[i]
                if i != 0:
                    # move it to the front
                    del self._cache[i]
                    self._cache.insert(0, n)
                return n
        return None

    def get(self, locator):
        with self._cache_lock:
            return self._get(locator)

    def reserve_cache(self, locator):
        '''Reserve a cache slot for the specified locator,
        or return the existing slot.'''
        with self._cache_lock:
            n = self._get(locator)
            if n:
                return n, False
            else:
                # Add a new cache slot for the locator
                n = KeepBlockCache.CacheSlot(locator)
                self._cache.insert(0, n)
                return n, True

class Counter(object):
    def __init__(self, v=0):
        self._lk = threading.Lock()
        self._val = v

    def add(self, v):
        with self._lk:
            self._val += v

    def get(self):
        with self._lk:
            return self._val


class KeepClient(object):

    # Default Keep server connection timeout:  2 seconds
    # Default Keep server read timeout:       256 seconds
    # Default Keep server bandwidth minimum:  32768 bytes per second
    # Default Keep proxy connection timeout:  20 seconds
    # Default Keep proxy read timeout:        256 seconds
    # Default Keep proxy bandwidth minimum:   32768 bytes per second
    DEFAULT_TIMEOUT = (2, 256, 32768)
    DEFAULT_PROXY_TIMEOUT = (20, 256, 32768)


    class KeepService(object):
        """Make requests to a single Keep service, and track results.

        A KeepService is intended to last long enough to perform one
        transaction (GET or PUT) against one Keep service. This can
        involve calling either get() or put() multiple times in order
        to retry after transient failures. However, calling both get()
        and put() on a single instance -- or using the same instance
        to access two different Keep services -- will not produce
        sensible behavior.
        """

        HTTP_ERRORS = (
            socket.error,
            ssl.SSLError,
            arvados.errors.HttpError,
        )

        def __init__(self, root, user_agent_pool=queue.LifoQueue(),
                     upload_counter=None,
                     download_counter=None,
                     headers={},
                     insecure=False):
            self.root = root
            self._user_agent_pool = user_agent_pool
            self._result = {'error': None}
            self._usable = True
            self._session = None
            self._socket = None
            self.get_headers = {'Accept': 'application/octet-stream'}
            self.get_headers.update(headers)
            self.put_headers = headers
            self.upload_counter = upload_counter
            self.download_counter = download_counter
            self.insecure = insecure

        def usable(self):
            """Is it worth attempting a request?"""
            return self._usable

        def finished(self):
            """Did the request succeed or encounter permanent failure?"""
            return self._result['error'] == False or not self._usable

        def last_result(self):
            return self._result

        def _get_user_agent(self):
            try:
                return self._user_agent_pool.get(block=False)
            except queue.Empty:
                return pycurl.Curl()

        def _put_user_agent(self, ua):
            try:
                ua.reset()
                self._user_agent_pool.put(ua, block=False)
            except:
                ua.close()

        def _socket_open(self, *args, **kwargs):
            if len(args) + len(kwargs) == 2:
                return self._socket_open_pycurl_7_21_5(*args, **kwargs)
            else:
                return self._socket_open_pycurl_7_19_3(*args, **kwargs)

        def _socket_open_pycurl_7_19_3(self, family, socktype, protocol, address=None):
            return self._socket_open_pycurl_7_21_5(
                purpose=None,
                address=collections.namedtuple(
                    'Address', ['family', 'socktype', 'protocol', 'addr'],
                )(family, socktype, protocol, address))

        def _socket_open_pycurl_7_21_5(self, purpose, address):
            """Because pycurl doesn't have CURLOPT_TCP_KEEPALIVE"""
            s = socket.socket(address.family, address.socktype, address.protocol)
            s.setsockopt(socket.SOL_SOCKET, socket.SO_KEEPALIVE, 1)
            # Will throw invalid protocol error on mac. This test prevents that.
            if hasattr(socket, 'TCP_KEEPIDLE'):
                s.setsockopt(socket.IPPROTO_TCP, socket.TCP_KEEPIDLE, 75)
            s.setsockopt(socket.IPPROTO_TCP, socket.TCP_KEEPINTVL, 75)
            self._socket = s
            return s

        def get(self, locator, method="GET", timeout=None):
            # locator is a KeepLocator object.
            url = self.root + str(locator)
            _logger.debug("Request: %s %s", method, url)
            curl = self._get_user_agent()
            ok = None
            try:
                with timer.Timer() as t:
                    self._headers = {}
                    response_body = BytesIO()
                    curl.setopt(pycurl.NOSIGNAL, 1)
                    curl.setopt(pycurl.OPENSOCKETFUNCTION,
                                lambda *args, **kwargs: self._socket_open(*args, **kwargs))
                    curl.setopt(pycurl.URL, url.encode('utf-8'))
                    curl.setopt(pycurl.HTTPHEADER, [
                        '{}: {}'.format(k,v) for k,v in self.get_headers.items()])
                    curl.setopt(pycurl.WRITEFUNCTION, response_body.write)
                    curl.setopt(pycurl.HEADERFUNCTION, self._headerfunction)
                    if self.insecure:
                        curl.setopt(pycurl.SSL_VERIFYPEER, 0)
                    if method == "HEAD":
                        curl.setopt(pycurl.NOBODY, True)
                    self._setcurltimeouts(curl, timeout)

                    try:
                        curl.perform()
                    except Exception as e:
                        raise arvados.errors.HttpError(0, str(e))
                    finally:
                        if self._socket:
                            self._socket.close()
                            self._socket = None
                    self._result = {
                        'status_code': curl.getinfo(pycurl.RESPONSE_CODE),
                        'body': response_body.getvalue(),
                        'headers': self._headers,
                        'error': False,
                    }

                ok = retry.check_http_response_success(self._result['status_code'])
                if not ok:
                    self._result['error'] = arvados.errors.HttpError(
                        self._result['status_code'],
                        self._headers.get('x-status-line', 'Error'))
            except self.HTTP_ERRORS as e:
                self._result = {
                    'error': e,
                }
            self._usable = ok != False
            if self._result.get('status_code', None):
                # The client worked well enough to get an HTTP status
                # code, so presumably any problems are just on the
                # server side and it's OK to reuse the client.
                self._put_user_agent(curl)
            else:
                # Don't return this client to the pool, in case it's
                # broken.
                curl.close()
            if not ok:
                _logger.debug("Request fail: GET %s => %s: %s",
                              url, type(self._result['error']), str(self._result['error']))
                return None
            if method == "HEAD":
                _logger.info("HEAD %s: %s bytes",
                         self._result['status_code'],
                         self._result.get('content-length'))
                return True

            _logger.info("GET %s: %s bytes in %s msec (%.3f MiB/sec)",
                         self._result['status_code'],
                         len(self._result['body']),
                         t.msecs,
                         1.0*len(self._result['body'])/2**20/t.secs if t.secs > 0 else 0)

            if self.download_counter:
                self.download_counter.add(len(self._result['body']))
            resp_md5 = hashlib.md5(self._result['body']).hexdigest()
            if resp_md5 != locator.md5sum:
                _logger.warning("Checksum fail: md5(%s) = %s",
                                url, resp_md5)
                self._result['error'] = arvados.errors.HttpError(
                    0, 'Checksum fail')
                return None
            return self._result['body']

        def put(self, hash_s, body, timeout=None):
            url = self.root + hash_s
            _logger.debug("Request: PUT %s", url)
            curl = self._get_user_agent()
            ok = None
            try:
                with timer.Timer() as t:
                    self._headers = {}
                    body_reader = BytesIO(body)
                    response_body = BytesIO()
                    curl.setopt(pycurl.NOSIGNAL, 1)
                    curl.setopt(pycurl.OPENSOCKETFUNCTION,
                                lambda *args, **kwargs: self._socket_open(*args, **kwargs))
                    curl.setopt(pycurl.URL, url.encode('utf-8'))
                    # Using UPLOAD tells cURL to wait for a "go ahead" from the
                    # Keep server (in the form of a HTTP/1.1 "100 Continue"
                    # response) instead of sending the request body immediately.
                    # This allows the server to reject the request if the request
                    # is invalid or the server is read-only, without waiting for
                    # the client to send the entire block.
                    curl.setopt(pycurl.UPLOAD, True)
                    curl.setopt(pycurl.INFILESIZE, len(body))
                    curl.setopt(pycurl.READFUNCTION, body_reader.read)
                    curl.setopt(pycurl.HTTPHEADER, [
                        '{}: {}'.format(k,v) for k,v in self.put_headers.items()])
                    curl.setopt(pycurl.WRITEFUNCTION, response_body.write)
                    curl.setopt(pycurl.HEADERFUNCTION, self._headerfunction)
                    if self.insecure:
                        curl.setopt(pycurl.SSL_VERIFYPEER, 0)
                    self._setcurltimeouts(curl, timeout)
                    try:
                        curl.perform()
                    except Exception as e:
                        raise arvados.errors.HttpError(0, str(e))
                    finally:
                        if self._socket:
                            self._socket.close()
                            self._socket = None
                    self._result = {
                        'status_code': curl.getinfo(pycurl.RESPONSE_CODE),
                        'body': response_body.getvalue().decode('utf-8'),
                        'headers': self._headers,
                        'error': False,
                    }
                ok = retry.check_http_response_success(self._result['status_code'])
                if not ok:
                    self._result['error'] = arvados.errors.HttpError(
                        self._result['status_code'],
                        self._headers.get('x-status-line', 'Error'))
            except self.HTTP_ERRORS as e:
                self._result = {
                    'error': e,
                }
            self._usable = ok != False # still usable if ok is True or None
            if self._result.get('status_code', None):
                # Client is functional. See comment in get().
                self._put_user_agent(curl)
            else:
                curl.close()
            if not ok:
                _logger.debug("Request fail: PUT %s => %s: %s",
                              url, type(self._result['error']), str(self._result['error']))
                return False
            _logger.info("PUT %s: %s bytes in %s msec (%.3f MiB/sec)",
                         self._result['status_code'],
                         len(body),
                         t.msecs,
                         1.0*len(body)/2**20/t.secs if t.secs > 0 else 0)
            if self.upload_counter:
                self.upload_counter.add(len(body))
            return True

        def _setcurltimeouts(self, curl, timeouts):
            if not timeouts:
                return
            elif isinstance(timeouts, tuple):
                if len(timeouts) == 2:
                    conn_t, xfer_t = timeouts
                    bandwidth_bps = KeepClient.DEFAULT_TIMEOUT[2]
                else:
                    conn_t, xfer_t, bandwidth_bps = timeouts
            else:
                conn_t, xfer_t = (timeouts, timeouts)
                bandwidth_bps = KeepClient.DEFAULT_TIMEOUT[2]
            curl.setopt(pycurl.CONNECTTIMEOUT_MS, int(conn_t*1000))
            curl.setopt(pycurl.LOW_SPEED_TIME, int(math.ceil(xfer_t)))
            curl.setopt(pycurl.LOW_SPEED_LIMIT, int(math.ceil(bandwidth_bps)))

        def _headerfunction(self, header_line):
            if isinstance(header_line, bytes):
                header_line = header_line.decode('iso-8859-1')
            if ':' in header_line:
                name, value = header_line.split(':', 1)
                name = name.strip().lower()
                value = value.strip()
            elif self._headers:
                name = self._lastheadername
                value = self._headers[name] + ' ' + header_line.strip()
            elif header_line.startswith('HTTP/'):
                name = 'x-status-line'
                value = header_line
            else:
                _logger.error("Unexpected header line: %s", header_line)
                return
            self._lastheadername = name
            self._headers[name] = value
            # Returning None implies all bytes were written


    class KeepWriterQueue(queue.Queue):
        def __init__(self, copies):
            queue.Queue.__init__(self) # Old-style superclass
            self.wanted_copies = copies
            self.successful_copies = 0
            self.response = None
            self.successful_copies_lock = threading.Lock()
            self.pending_tries = copies
            self.pending_tries_notification = threading.Condition()

        def write_success(self, response, replicas_nr):
            with self.successful_copies_lock:
                self.successful_copies += replicas_nr
                self.response = response
            with self.pending_tries_notification:
                self.pending_tries_notification.notify_all()

        def write_fail(self, ks):
            with self.pending_tries_notification:
                self.pending_tries += 1
                self.pending_tries_notification.notify()

        def pending_copies(self):
            with self.successful_copies_lock:
                return self.wanted_copies - self.successful_copies

        def get_next_task(self):
            with self.pending_tries_notification:
                while True:
                    if self.pending_copies() < 1:
                        # This notify_all() is unnecessary --
                        # write_success() already called notify_all()
                        # when pending<1 became true, so it's not
                        # possible for any other thread to be in
                        # wait() now -- but it's cheap insurance
                        # against deadlock so we do it anyway:
                        self.pending_tries_notification.notify_all()
                        # Drain the queue and then raise Queue.Empty
                        while True:
                            self.get_nowait()
                            self.task_done()
                    elif self.pending_tries > 0:
                        service, service_root = self.get_nowait()
                        if service.finished():
                            self.task_done()
                            continue
                        self.pending_tries -= 1
                        return service, service_root
                    elif self.empty():
                        self.pending_tries_notification.notify_all()
                        raise queue.Empty
                    else:
                        self.pending_tries_notification.wait()


    class KeepWriterThreadPool(object):
        def __init__(self, data, data_hash, copies, max_service_replicas, timeout=None):
            self.total_task_nr = 0
            self.wanted_copies = copies
            if (not max_service_replicas) or (max_service_replicas >= copies):
                num_threads = 1
            else:
                num_threads = int(math.ceil(1.0*copies/max_service_replicas))
            _logger.debug("Pool max threads is %d", num_threads)
            self.workers = []
            self.queue = KeepClient.KeepWriterQueue(copies)
            # Create workers
            for _ in range(num_threads):
                w = KeepClient.KeepWriterThread(self.queue, data, data_hash, timeout)
                self.workers.append(w)

        def add_task(self, ks, service_root):
            self.queue.put((ks, service_root))
            self.total_task_nr += 1

        def done(self):
            return self.queue.successful_copies

        def join(self):
            # Start workers
            for worker in self.workers:
                worker.start()
            # Wait for finished work
            self.queue.join()

        def response(self):
            return self.queue.response


    class KeepWriterThread(threading.Thread):
        TaskFailed = RuntimeError()

        def __init__(self, queue, data, data_hash, timeout=None):
            super(KeepClient.KeepWriterThread, self).__init__()
            self.timeout = timeout
            self.queue = queue
            self.data = data
            self.data_hash = data_hash
            self.daemon = True

        def run(self):
            while True:
                try:
                    service, service_root = self.queue.get_next_task()
                except queue.Empty:
                    return
                try:
                    locator, copies = self.do_task(service, service_root)
                except Exception as e:
                    if e is not self.TaskFailed:
                        _logger.exception("Exception in KeepWriterThread")
                    self.queue.write_fail(service)
                else:
                    self.queue.write_success(locator, copies)
                finally:
                    self.queue.task_done()

        def do_task(self, service, service_root):
            success = bool(service.put(self.data_hash,
                                        self.data,
                                        timeout=self.timeout))
            result = service.last_result()

            if not success:
                if result.get('status_code', None):
                    _logger.debug("Request fail: PUT %s => %s %s",
                                  self.data_hash,
                                  result['status_code'],
                                  result['body'])
                raise self.TaskFailed

            _logger.debug("KeepWriterThread %s succeeded %s+%i %s",
                          str(threading.current_thread()),
                          self.data_hash,
                          len(self.data),
                          service_root)
            try:
                replicas_stored = int(result['headers']['x-keep-replicas-stored'])
            except (KeyError, ValueError):
                replicas_stored = 1

            return result['body'].strip(), replicas_stored


    def __init__(self, api_client=None, proxy=None,
                 timeout=DEFAULT_TIMEOUT, proxy_timeout=DEFAULT_PROXY_TIMEOUT,
                 api_token=None, local_store=None, block_cache=None,
                 num_retries=0, session=None):
        """Initialize a new KeepClient.

        Arguments:
        :api_client:
          The API client to use to find Keep services.  If not
          provided, KeepClient will build one from available Arvados
          configuration.

        :proxy:
          If specified, this KeepClient will send requests to this Keep
          proxy.  Otherwise, KeepClient will fall back to the setting of the
          ARVADOS_KEEP_SERVICES or ARVADOS_KEEP_PROXY configuration settings.
          If you want to KeepClient does not use a proxy, pass in an empty
          string.

        :timeout:
          The initial timeout (in seconds) for HTTP requests to Keep
          non-proxy servers.  A tuple of three floats is interpreted as
          (connection_timeout, read_timeout, minimum_bandwidth). A connection
          will be aborted if the average traffic rate falls below
          minimum_bandwidth bytes per second over an interval of read_timeout
          seconds. Because timeouts are often a result of transient server
          load, the actual connection timeout will be increased by a factor
          of two on each retry.
          Default: (2, 256, 32768).

        :proxy_timeout:
          The initial timeout (in seconds) for HTTP requests to
          Keep proxies. A tuple of three floats is interpreted as
          (connection_timeout, read_timeout, minimum_bandwidth). The behavior
          described above for adjusting connection timeouts on retry also
          applies.
          Default: (20, 256, 32768).

        :api_token:
          If you're not using an API client, but only talking
          directly to a Keep proxy, this parameter specifies an API token
          to authenticate Keep requests.  It is an error to specify both
          api_client and api_token.  If you specify neither, KeepClient
          will use one available from the Arvados configuration.

        :local_store:
          If specified, this KeepClient will bypass Keep
          services, and save data to the named directory.  If unspecified,
          KeepClient will fall back to the setting of the $KEEP_LOCAL_STORE
          environment variable.  If you want to ensure KeepClient does not
          use local storage, pass in an empty string.  This is primarily
          intended to mock a server for testing.

        :num_retries:
          The default number of times to retry failed requests.
          This will be used as the default num_retries value when get() and
          put() are called.  Default 0.
        """
        self.lock = threading.Lock()
        if proxy is None:
            if config.get('ARVADOS_KEEP_SERVICES'):
                proxy = config.get('ARVADOS_KEEP_SERVICES')
            else:
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

        if api_client is None:
            self.insecure = config.flag_is_true('ARVADOS_API_HOST_INSECURE')
        else:
            self.insecure = api_client.insecure

        self.block_cache = block_cache if block_cache else KeepBlockCache()
        self.timeout = timeout
        self.proxy_timeout = proxy_timeout
        self._user_agent_pool = queue.LifoQueue()
        self.upload_counter = Counter()
        self.download_counter = Counter()
        self.put_counter = Counter()
        self.get_counter = Counter()
        self.hits_counter = Counter()
        self.misses_counter = Counter()

        if local_store:
            self.local_store = local_store
            self.get = self.local_store_get
            self.put = self.local_store_put
        else:
            self.num_retries = num_retries
            self.max_replicas_per_service = None
            if proxy:
                proxy_uris = proxy.split()
                for i in range(len(proxy_uris)):
                    if not proxy_uris[i].endswith('/'):
                        proxy_uris[i] += '/'
                    # URL validation
                    url = urllib.parse.urlparse(proxy_uris[i])
                    if not (url.scheme and url.netloc):
                        raise arvados.errors.ArgumentError("Invalid proxy URI: {}".format(proxy_uris[i]))
                self.api_token = api_token
                self._gateway_services = {}
                self._keep_services = [{
                    'uuid': "00000-bi6l4-%015d" % idx,
                    'service_type': 'proxy',
                    '_service_root': uri,
                    } for idx, uri in enumerate(proxy_uris)]
                self._writable_services = self._keep_services
                self.using_proxy = True
                self._static_services_list = True
            else:
                # It's important to avoid instantiating an API client
                # unless we actually need one, for testing's sake.
                if api_client is None:
                    api_client = arvados.api('v1')
                self.api_client = api_client
                self.api_token = api_client.api_token
                self._gateway_services = {}
                self._keep_services = None
                self._writable_services = None
                self.using_proxy = None
                self._static_services_list = False

    def current_timeout(self, attempt_number):
        """Return the appropriate timeout to use for this client.

        The proxy timeout setting if the backend service is currently a proxy,
        the regular timeout setting otherwise.  The `attempt_number` indicates
        how many times the operation has been tried already (starting from 0
        for the first try), and scales the connection timeout portion of the
        return value accordingly.

        """
        # TODO(twp): the timeout should be a property of a
        # KeepService, not a KeepClient. See #4488.
        t = self.proxy_timeout if self.using_proxy else self.timeout
        if len(t) == 2:
            return (t[0] * (1 << attempt_number), t[1])
        else:
            return (t[0] * (1 << attempt_number), t[1], t[2])
    def _any_nondisk_services(self, service_list):
        return any(ks.get('service_type', 'disk') != 'disk'
                   for ks in service_list)

    def build_services_list(self, force_rebuild=False):
        if (self._static_services_list or
              (self._keep_services and not force_rebuild)):
            return
        with self.lock:
            try:
                keep_services = self.api_client.keep_services().accessible()
            except Exception:  # API server predates Keep services.
                keep_services = self.api_client.keep_disks().list()

            # Gateway services are only used when specified by UUID,
            # so there's nothing to gain by filtering them by
            # service_type.
            self._gateway_services = {ks['uuid']: ks for ks in
                                      keep_services.execute()['items']}
            if not self._gateway_services:
                raise arvados.errors.NoKeepServersError()

            # Precompute the base URI for each service.
            for r in self._gateway_services.values():
                host = r['service_host']
                if not host.startswith('[') and host.find(':') >= 0:
                    # IPv6 URIs must be formatted like http://[::1]:80/...
                    host = '[' + host + ']'
                r['_service_root'] = "{}://{}:{:d}/".format(
                    'https' if r['service_ssl_flag'] else 'http',
                    host,
                    r['service_port'])

            _logger.debug(str(self._gateway_services))
            self._keep_services = [
                ks for ks in self._gateway_services.values()
                if not ks.get('service_type', '').startswith('gateway:')]
            self._writable_services = [ks for ks in self._keep_services
                                       if not ks.get('read_only')]

            # For disk type services, max_replicas_per_service is 1
            # It is unknown (unlimited) for other service types.
            if self._any_nondisk_services(self._writable_services):
                self.max_replicas_per_service = None
            else:
                self.max_replicas_per_service = 1

    def _service_weight(self, data_hash, service_uuid):
        """Compute the weight of a Keep service endpoint for a data
        block with a known hash.

        The weight is md5(h + u) where u is the last 15 characters of
        the service endpoint's UUID.
        """
        return hashlib.md5((data_hash + service_uuid[-15:]).encode()).hexdigest()

    def weighted_service_roots(self, locator, force_rebuild=False, need_writable=False):
        """Return an array of Keep service endpoints, in the order in
        which they should be probed when reading or writing data with
        the given hash+hints.
        """
        self.build_services_list(force_rebuild)

        sorted_roots = []
        # Use the services indicated by the given +K@... remote
        # service hints, if any are present and can be resolved to a
        # URI.
        for hint in locator.hints:
            if hint.startswith('K@'):
                if len(hint) == 7:
                    sorted_roots.append(
                        "https://keep.{}.arvadosapi.com/".format(hint[2:]))
                elif len(hint) == 29:
                    svc = self._gateway_services.get(hint[2:])
                    if svc:
                        sorted_roots.append(svc['_service_root'])

        # Sort the available local services by weight (heaviest first)
        # for this locator, and return their service_roots (base URIs)
        # in that order.
        use_services = self._keep_services
        if need_writable:
            use_services = self._writable_services
        self.using_proxy = self._any_nondisk_services(use_services)
        sorted_roots.extend([
            svc['_service_root'] for svc in sorted(
                use_services,
                reverse=True,
                key=lambda svc: self._service_weight(locator.md5sum, svc['uuid']))])
        _logger.debug("{}: {}".format(locator, sorted_roots))
        return sorted_roots

    def map_new_services(self, roots_map, locator, force_rebuild, need_writable, headers):
        # roots_map is a dictionary, mapping Keep service root strings
        # to KeepService objects.  Poll for Keep services, and add any
        # new ones to roots_map.  Return the current list of local
        # root strings.
        headers.setdefault('Authorization', "OAuth2 %s" % (self.api_token,))
        local_roots = self.weighted_service_roots(locator, force_rebuild, need_writable)
        for root in local_roots:
            if root not in roots_map:
                roots_map[root] = self.KeepService(
                    root, self._user_agent_pool,
                    upload_counter=self.upload_counter,
                    download_counter=self.download_counter,
                    headers=headers,
                    insecure=self.insecure)
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

    def get_from_cache(self, loc):
        """Fetch a block only if is in the cache, otherwise return None."""
        slot = self.block_cache.get(loc)
        if slot is not None and slot.ready.is_set():
            return slot.get()
        else:
            return None

    @retry.retry_method
    def head(self, loc_s, **kwargs):
        return self._get_or_head(loc_s, method="HEAD", **kwargs)

    @retry.retry_method
    def get(self, loc_s, **kwargs):
        return self._get_or_head(loc_s, method="GET", **kwargs)

    def _get_or_head(self, loc_s, method="GET", num_retries=None, request_id=None):
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

        self.get_counter.add(1)

        slot = None
        blob = None
        try:
            locator = KeepLocator(loc_s)
            if method == "GET":
                slot, first = self.block_cache.reserve_cache(locator.md5sum)
                if not first:
                    self.hits_counter.add(1)
                    blob = slot.get()
                    if blob is None:
                        raise arvados.errors.KeepReadError(
                            "failed to read {}".format(loc_s))
                    return blob

            self.misses_counter.add(1)

            headers = {
                'X-Request-Id': (request_id or
                                 (hasattr(self, 'api_client') and self.api_client.request_id) or
                                 arvados.util.new_request_id()),
            }

            # If the locator has hints specifying a prefix (indicating a
            # remote keepproxy) or the UUID of a local gateway service,
            # read data from the indicated service(s) instead of the usual
            # list of local disk services.
            hint_roots = ['http://keep.{}.arvadosapi.com/'.format(hint[2:])
                          for hint in locator.hints if hint.startswith('K@') and len(hint) == 7]
            hint_roots.extend([self._gateway_services[hint[2:]]['_service_root']
                               for hint in locator.hints if (
                                       hint.startswith('K@') and
                                       len(hint) == 29 and
                                       self._gateway_services.get(hint[2:])
                                       )])
            # Map root URLs to their KeepService objects.
            roots_map = {
                root: self.KeepService(root, self._user_agent_pool,
                                       upload_counter=self.upload_counter,
                                       download_counter=self.download_counter,
                                       headers=headers,
                                       insecure=self.insecure)
                for root in hint_roots
            }

            # See #3147 for a discussion of the loop implementation.  Highlights:
            # * Refresh the list of Keep services after each failure, in case
            #   it's being updated.
            # * Retry until we succeed, we're out of retries, or every available
            #   service has returned permanent failure.
            sorted_roots = []
            roots_map = {}
            loop = retry.RetryLoop(num_retries, self._check_loop_result,
                                   backoff_start=2)
            for tries_left in loop:
                try:
                    sorted_roots = self.map_new_services(
                        roots_map, locator,
                        force_rebuild=(tries_left < num_retries),
                        need_writable=False,
                        headers=headers)
                except Exception as error:
                    loop.save_result(error)
                    continue

                # Query KeepService objects that haven't returned
                # permanent failure, in our specified shuffle order.
                services_to_try = [roots_map[root]
                                   for root in sorted_roots
                                   if roots_map[root].usable()]
                for keep_service in services_to_try:
                    blob = keep_service.get(locator, method=method, timeout=self.current_timeout(num_retries-tries_left))
                    if blob is not None:
                        break
                loop.save_result((blob, len(services_to_try)))

            # Always cache the result, then return it if we succeeded.
            if loop.success():
                if method == "HEAD":
                    return True
                else:
                    return blob
        finally:
            if slot is not None:
                slot.set(blob)
                self.block_cache.cap_cache()

        # Q: Including 403 is necessary for the Keep tests to continue
        # passing, but maybe they should expect KeepReadError instead?
        not_founds = sum(1 for key in sorted_roots
                         if roots_map[key].last_result().get('status_code', None) in {403, 404, 410})
        service_errors = ((key, roots_map[key].last_result()['error'])
                          for key in sorted_roots)
        if not roots_map:
            raise arvados.errors.KeepReadError(
                "failed to read {}: no Keep services available ({})".format(
                    loc_s, loop.last_result()))
        elif not_founds == len(sorted_roots):
            raise arvados.errors.NotFoundError(
                "{} not found".format(loc_s), service_errors)
        else:
            raise arvados.errors.KeepReadError(
                "failed to read {}".format(loc_s), service_errors, label="service")

    @retry.retry_method
    def put(self, data, copies=2, num_retries=None, request_id=None):
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

        if not isinstance(data, bytes):
            data = data.encode()

        self.put_counter.add(1)

        data_hash = hashlib.md5(data).hexdigest()
        loc_s = data_hash + '+' + str(len(data))
        if copies < 1:
            return loc_s
        locator = KeepLocator(loc_s)

        headers = {
            'X-Request-Id': (request_id or
                             (hasattr(self, 'api_client') and self.api_client.request_id) or
                             arvados.util.new_request_id()),
            'X-Keep-Desired-Replicas': str(copies),
        }
        roots_map = {}
        loop = retry.RetryLoop(num_retries, self._check_loop_result,
                               backoff_start=2)
        done = 0
        for tries_left in loop:
            try:
                sorted_roots = self.map_new_services(
                    roots_map, locator,
                    force_rebuild=(tries_left < num_retries),
                    need_writable=True,
                    headers=headers)
            except Exception as error:
                loop.save_result(error)
                continue

            writer_pool = KeepClient.KeepWriterThreadPool(data=data,
                                                        data_hash=data_hash,
                                                        copies=copies - done,
                                                        max_service_replicas=self.max_replicas_per_service,
                                                        timeout=self.current_timeout(num_retries - tries_left))
            for service_root, ks in [(root, roots_map[root])
                                     for root in sorted_roots]:
                if ks.finished():
                    continue
                writer_pool.add_task(ks, service_root)
            writer_pool.join()
            done += writer_pool.done()
            loop.save_result((done >= copies, writer_pool.total_task_nr))

        if loop.success():
            return writer_pool.response()
        if not roots_map:
            raise arvados.errors.KeepWriteError(
                "failed to write {}: no Keep services available ({})".format(
                    data_hash, loop.last_result()))
        else:
            service_errors = ((key, roots_map[key].last_result()['error'])
                              for key in sorted_roots
                              if roots_map[key].last_result()['error'])
            raise arvados.errors.KeepWriteError(
                "failed to write {} (wanted {} copies but wrote {})".format(
                    data_hash, copies, writer_pool.done()), service_errors, label="service")

    def local_store_put(self, data, copies=1, num_retries=None):
        """A stub for put().

        This method is used in place of the real put() method when
        using local storage (see constructor's local_store argument).

        copies and num_retries arguments are ignored: they are here
        only for the sake of offering the same call signature as
        put().

        Data stored this way can be retrieved via local_store_get().
        """
        md5 = hashlib.md5(data).hexdigest()
        locator = '%s+%d' % (md5, len(data))
        with open(os.path.join(self.local_store, md5 + '.tmp'), 'wb') as f:
            f.write(data)
        os.rename(os.path.join(self.local_store, md5 + '.tmp'),
                  os.path.join(self.local_store, md5))
        return locator

    def local_store_get(self, loc_s, num_retries=None):
        """Companion to local_store_put()."""
        try:
            locator = KeepLocator(loc_s)
        except ValueError:
            raise arvados.errors.NotFoundError(
                "Invalid data locator: '%s'" % loc_s)
        if locator.md5sum == config.EMPTY_BLOCK_LOCATOR.split('+')[0]:
            return b''
        with open(os.path.join(self.local_store, locator.md5sum), 'rb') as f:
            return f.read()

    def is_cached(self, locator):
        return self.block_cache.reserve_cache(expect_hash)
