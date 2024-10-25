# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
"""Arvados REST API client

This module provides classes and functions to construct an Arvados REST API
client. Most users will want to use one of these constructor functions, in
order of preference:

* `arvados.api.api` provides a high-level interface to construct a client from
  either arguments or user configuration. You can call this module just like
  a function as a shortcut for calling `arvados.api.api`.

* `arvados.api.api_from_config` constructs a client from user configuration in
  a dictionary.

* `arvados.api.api_client` provides a lower-level interface to construct a
  simpler client object that is not threadsafe.

Other classes and functions is this module support creating and customizing
the client for specialized use-cases.

The methods on an Arvados REST API client are generated dynamically at
runtime. The `arvados.api_resources` module documents those methods and
return values for the current version of Arvados. It does not
implement anything so you don't need to import it, but it's a helpful
reference to understand how to use the Arvados REST API client.
"""

import collections
import errno
import hashlib
import httplib2
import json
import logging
import os
import pathlib
import re
import socket
import ssl
import sys
import tempfile
import threading
import time
import types

from typing import (
    Any,
    Dict,
    List,
    Mapping,
    Optional,
)

import apiclient
import apiclient.http
from apiclient import discovery as apiclient_discovery
from apiclient import errors as apiclient_errors
from . import config
from . import errors
from . import keep
from . import retry
from . import util
from ._internal import basedirs
from .logging import GoogleHTTPClientFilter, log_handler

_logger = logging.getLogger('arvados.api')
_googleapiclient_log_lock = threading.Lock()

MAX_IDLE_CONNECTION_DURATION = 30
"""
Number of seconds that API client HTTP connections should be allowed to idle
in keepalive state before they are forced closed. Client code can adjust this
constant, and it will be used for all Arvados API clients constructed after
that point.
"""

# An unused HTTP 5xx status code to request a retry internally.
# See _intercept_http_request. This should not be user-visible.
_RETRY_4XX_STATUS = 545

if sys.version_info >= (3,):
    httplib2.SSLHandshakeError = None

_orig_retry_request = apiclient.http._retry_request
def _retry_request(http, num_retries, *args, **kwargs):
    try:
        num_retries = max(num_retries, http.num_retries)
    except AttributeError:
        # `http` client object does not have a `num_retries` attribute.
        # It apparently hasn't gone through _patch_http_request, possibly
        # because this isn't an Arvados API client. Pass through to
        # avoid interfering with other Google API clients.
        return _orig_retry_request(http, num_retries, *args, **kwargs)
    response, body = _orig_retry_request(http, num_retries, *args, **kwargs)
    # If _intercept_http_request ran out of retries for a 4xx response,
    # restore the original status code.
    if response.status == _RETRY_4XX_STATUS:
        response.status = int(response['status'])
    return (response, body)
apiclient.http._retry_request = _retry_request

def _intercept_http_request(self, uri, method="GET", headers={}, **kwargs):
    if not headers.get('X-Request-Id'):
        headers['X-Request-Id'] = self._request_id()
    try:
        if (self.max_request_size and
            kwargs.get('body') and
            self.max_request_size < len(kwargs['body'])):
            raise apiclient_errors.MediaUploadSizeError("Request size %i bytes exceeds published limit of %i bytes" % (len(kwargs['body']), self.max_request_size))

        headers['Authorization'] = 'Bearer %s' % self.arvados_api_token

        if (time.time() - self._last_request_time) > self._max_keepalive_idle:
            # High probability of failure due to connection atrophy. Make
            # sure this request [re]opens a new connection by closing and
            # forgetting all cached connections first.
            for conn in self.connections.values():
                conn.close()
            self.connections.clear()

        self._last_request_time = time.time()
        try:
            response, body = self.orig_http_request(uri, method, headers=headers, **kwargs)
        except ssl.CertificateError as e:
            raise ssl.CertificateError(e.args[0], "Could not connect to %s\n%s\nPossible causes: remote SSL/TLS certificate expired, or was issued by an untrusted certificate authority." % (uri, e)) from None
        # googleapiclient only retries 403, 429, and 5xx status codes.
        # If we got another 4xx status that we want to retry, convert it into
        # 5xx so googleapiclient handles it the way we want.
        if response.status in retry._HTTP_CAN_RETRY and response.status < 500:
            response.status = _RETRY_4XX_STATUS
        return (response, body)
    except Exception as e:
        # Prepend "[request_id] " to the error message, which we
        # assume is the first string argument passed to the exception
        # constructor.
        for i in range(len(e.args or ())):
            if type(e.args[i]) == type(""):
                e.args = e.args[:i] + ("[{}] {}".format(headers['X-Request-Id'], e.args[i]),) + e.args[i+1:]
                raise type(e)(*e.args)
        raise

def _patch_http_request(http, api_token, num_retries):
    http.arvados_api_token = api_token
    http.max_request_size = 0
    http.num_retries = num_retries
    http.orig_http_request = http.request
    http.request = types.MethodType(_intercept_http_request, http)
    http._last_request_time = 0
    http._max_keepalive_idle = MAX_IDLE_CONNECTION_DURATION
    http._request_id = util.new_request_id
    return http

def _close_connections(self):
    for conn in self._http.connections.values():
        conn.close()

# Monkey patch discovery._cast() so objects and arrays get serialized
# with json.dumps() instead of str().
_cast_orig = apiclient_discovery._cast
def _cast_objects_too(value, schema_type):
    global _cast_orig
    if (type(value) != type('') and
        type(value) != type(b'') and
        (schema_type == 'object' or schema_type == 'array')):
        return json.dumps(value)
    else:
        return _cast_orig(value, schema_type)
apiclient_discovery._cast = _cast_objects_too

# Convert apiclient's HttpErrors into our own API error subclass for better
# error reporting.
# Reassigning apiclient_errors.HttpError is not sufficient because most of the
# apiclient submodules import the class into their own namespace.
def _new_http_error(cls, *args, **kwargs):
    return super(apiclient_errors.HttpError, cls).__new__(
        errors.ApiError, *args, **kwargs)
apiclient_errors.HttpError.__new__ = staticmethod(_new_http_error)

class ThreadSafeHTTPCache:
    """Thread-safe replacement for `httplib2.FileCache`

    `arvados.api.http_cache` is the preferred way to construct this object.
    Refer to that function's docstring for details.
    """

    def __init__(self, path=None, max_age=None):
        self._dir = path
        if max_age is not None:
            try:
                self._clean(threshold=time.time() - max_age)
            except:
                pass

    def _clean(self, threshold=0):
        for ent in os.listdir(self._dir):
            fnm = os.path.join(self._dir, ent)
            if os.path.isdir(fnm) or not fnm.endswith('.tmp'):
                continue
            stat = os.lstat(fnm)
            if stat.st_mtime < threshold:
                try:
                    os.unlink(fnm)
                except OSError as err:
                    if err.errno != errno.ENOENT:
                        raise

    def __str__(self):
        return self._dir

    def _filename(self, url):
        return os.path.join(self._dir, hashlib.md5(url.encode('utf-8')).hexdigest()+'.tmp')

    def get(self, url):
        filename = self._filename(url)
        try:
            with open(filename, 'rb') as f:
                return f.read()
        except (IOError, OSError):
            return None

    def set(self, url, content):
        try:
            fd, tempname = tempfile.mkstemp(dir=self._dir)
        except:
            return None
        try:
            try:
                f = os.fdopen(fd, 'wb')
            except:
                os.close(fd)
                raise
            try:
                f.write(content)
            finally:
                f.close()
            os.rename(tempname, self._filename(url))
            tempname = None
        finally:
            if tempname:
                os.unlink(tempname)

    def delete(self, url):
        try:
            os.unlink(self._filename(url))
        except OSError as err:
            if err.errno != errno.ENOENT:
                raise


class ThreadSafeAPIClient(object):
    """Thread-safe wrapper for an Arvados API client

    This class takes all the arguments necessary to build a lower-level
    Arvados API client `googleapiclient.discovery.Resource`, then
    transparently builds and wraps a unique object per thread. This works
    around the fact that the client's underlying HTTP client object is not
    thread-safe.

    Arguments:

    * apiconfig: Mapping[str, str] | None --- A mapping with entries for
      `ARVADOS_API_HOST`, `ARVADOS_API_TOKEN`, and optionally
      `ARVADOS_API_HOST_INSECURE`. If not provided, uses
      `arvados.config.settings` to get these parameters from user
      configuration.  You can pass an empty mapping to build the client
      solely from `api_params`.

    * keep_params: Mapping[str, Any] --- Keyword arguments used to construct
      an associated `arvados.keep.KeepClient`.

    * api_params: Mapping[str, Any] --- Keyword arguments used to construct
      each thread's API client. These have the same meaning as in the
      `arvados.api.api` function.

    * version: str | None --- A string naming the version of the Arvados API
      to use. If not specified, the code will log a warning and fall back to
      `'v1'`.
    """
    def __init__(
            self,
            apiconfig: Optional[Mapping[str, str]]=None,
            keep_params: Optional[Mapping[str, Any]]={},
            api_params: Optional[Mapping[str, Any]]={},
            version: Optional[str]=None,
    ) -> None:
        if apiconfig or apiconfig is None:
            self._api_kwargs = api_kwargs_from_config(version, apiconfig, **api_params)
        else:
            self._api_kwargs = normalize_api_kwargs(version, **api_params)
        self.api_token = self._api_kwargs['token']
        self.request_id = self._api_kwargs.get('request_id')
        self.local = threading.local()
        self.keep = keep.KeepClient(api_client=self, **keep_params)

    def localapi(self) -> 'googleapiclient.discovery.Resource':
        try:
            client = self.local.api
        except AttributeError:
            client = api_client(**self._api_kwargs)
            client._http._request_id = lambda: self.request_id or util.new_request_id()
            self.local.api = client
        return client

    def __getattr__(self, name: str) -> Any:
        # Proxy nonexistent attributes to the thread-local API client.
        return getattr(self.localapi(), name)


def http_cache(data_type: str) -> Optional[ThreadSafeHTTPCache]:
    """Set up an HTTP file cache

    This function constructs and returns an `arvados.api.ThreadSafeHTTPCache`
    backed by the filesystem under a cache directory from the environment, or
    `None` if the directory cannot be set up. The return value can be passed to
    `httplib2.Http` as the `cache` argument.

    Arguments:

    * data_type: str --- The name of the subdirectory
      where data is cached.
    """
    try:
        path = basedirs.BaseDirectories('CACHE').storage_path(data_type)
    except (OSError, RuntimeError):
        return None
    else:
        return ThreadSafeHTTPCache(str(path), max_age=60*60*24*2)

def api_client(
        version: str,
        discoveryServiceUrl: str,
        token: str,
        *,
        cache: bool=True,
        http: Optional[httplib2.Http]=None,
        insecure: bool=False,
        num_retries: int=10,
        request_id: Optional[str]=None,
        timeout: int=5*60,
        **kwargs: Any,
) -> apiclient_discovery.Resource:
    """Build an Arvados API client

    This function returns a `googleapiclient.discovery.Resource` object
    constructed from the given arguments. This is a relatively low-level
    interface that requires all the necessary inputs as arguments. Most
    users will prefer to use `api` which can accept more flexible inputs.

    Arguments:

    * version: str --- A string naming the version of the Arvados API to use.

    * discoveryServiceUrl: str --- The URL used to discover APIs passed
      directly to `googleapiclient.discovery.build`.

    * token: str --- The authentication token to send with each API call.

    Keyword-only arguments:

    * cache: bool --- If true, loads the API discovery document from, or
      saves it to, a cache on disk.

    * http: httplib2.Http | None --- The HTTP client object the API client
      object will use to make requests.  If not provided, this function will
      build its own to use. Either way, the object will be patched as part
      of the build process.

    * insecure: bool --- If true, ignore SSL certificate validation
      errors. Default `False`.

    * num_retries: int --- The number of times to retry each API request if
      it encounters a temporary failure. Default 10.

    * request_id: str | None --- Default `X-Request-Id` header value for
      outgoing requests that don't already provide one. If `None` or
      omitted, generate a random ID. When retrying failed requests, the same
      ID is used on all attempts.

    * timeout: int --- A timeout value for HTTP requests in seconds. Default
      300 (5 minutes).

    Additional keyword arguments will be passed directly to
    `googleapiclient.discovery.build`.
    """
    if http is None:
        http = httplib2.Http(
            ca_certs=util.ca_certs_path(),
            cache=http_cache('discovery') if cache else None,
            disable_ssl_certificate_validation=bool(insecure),
        )
    if http.timeout is None:
        http.timeout = timeout
    http = _patch_http_request(http, token, num_retries)

    # The first time a client is instantiated, temporarily route
    # googleapiclient.http retry logs if they're not already. These are
    # important because temporary problems fetching the discovery document
    # can cause clients to appear to hang early. This can be removed after
    # we have a more general story for handling googleapiclient logs (#20521).
    client_logger = logging.getLogger('googleapiclient.http')
    # "first time a client is instantiated" = thread that acquires this lock
    # It is never released.
    # googleapiclient sets up its own NullHandler so we detect if logging is
    # configured by looking for a real handler anywhere in the hierarchy.
    client_logger_unconfigured = _googleapiclient_log_lock.acquire(blocking=False) and all(
        isinstance(handler, logging.NullHandler)
        for logger_name in ['', 'googleapiclient', 'googleapiclient.http']
        for handler in logging.getLogger(logger_name).handlers
    )
    if client_logger_unconfigured:
        client_level = client_logger.level
        client_filter = GoogleHTTPClientFilter()
        client_logger.addFilter(client_filter)
        client_logger.addHandler(log_handler)
        if logging.NOTSET < client_level < client_filter.retry_levelno:
            client_logger.setLevel(client_level)
        else:
            client_logger.setLevel(client_filter.retry_levelno)
    try:
        svc = apiclient_discovery.build(
            'arvados', version,
            cache_discovery=False,
            discoveryServiceUrl=discoveryServiceUrl,
            http=http,
            num_retries=num_retries,
            **kwargs,
        )
    finally:
        if client_logger_unconfigured:
            client_logger.removeHandler(log_handler)
            client_logger.removeFilter(client_filter)
            client_logger.setLevel(client_level)
    svc.api_token = token
    svc.insecure = insecure
    svc.request_id = request_id
    svc.config = lambda: util.get_config_once(svc)
    svc.vocabulary = lambda: util.get_vocabulary_once(svc)
    svc.close_connections = types.MethodType(_close_connections, svc)
    http.max_request_size = svc._rootDesc.get('maxRequestSize', 0)
    http.cache = None
    http._request_id = lambda: svc.request_id or util.new_request_id()
    return svc

def normalize_api_kwargs(
        version: Optional[str]=None,
        discoveryServiceUrl: Optional[str]=None,
        host: Optional[str]=None,
        token: Optional[str]=None,
        **kwargs: Any,
) -> Dict[str, Any]:
    """Validate kwargs from `api` and build kwargs for `api_client`

    This method takes high-level keyword arguments passed to the `api`
    constructor and normalizes them into a new dictionary that can be passed
    as keyword arguments to `api_client`. It raises `ValueError` if required
    arguments are missing or conflict.

    Arguments:

    * version: str | None --- A string naming the version of the Arvados API
      to use. If not specified, the code will log a warning and fall back to
      'v1'.

    * discoveryServiceUrl: str | None --- The URL used to discover APIs
      passed directly to `googleapiclient.discovery.build`. It is an error
      to pass both `discoveryServiceUrl` and `host`.

    * host: str | None --- The hostname and optional port number of the
      Arvados API server. Used to build `discoveryServiceUrl`. It is an
      error to pass both `discoveryServiceUrl` and `host`.

    * token: str --- The authentication token to send with each API call.

    Additional keyword arguments will be included in the return value.
    """
    if discoveryServiceUrl and host:
        raise ValueError("both discoveryServiceUrl and host provided")
    elif discoveryServiceUrl:
        url_src = "discoveryServiceUrl"
    elif host:
        url_src = "host argument"
        discoveryServiceUrl = 'https://%s/discovery/v1/apis/{api}/{apiVersion}/rest' % (host,)
    elif token:
        # This specific error message gets priority for backwards compatibility.
        raise ValueError("token argument provided, but host missing.")
    else:
        raise ValueError("neither discoveryServiceUrl nor host provided")
    if not token:
        raise ValueError("%s provided, but token missing" % (url_src,))
    if not version:
        version = 'v1'
        _logger.info(
            "Using default API version. Call arvados.api(%r) instead.",
            version,
        )
    return {
        'discoveryServiceUrl': discoveryServiceUrl,
        'token': token,
        'version': version,
        **kwargs,
    }

def api_kwargs_from_config(
        version: Optional[str]=None,
        apiconfig: Optional[Mapping[str, str]]=None,
        **kwargs: Any
) -> Dict[str, Any]:
    """Build `api_client` keyword arguments from configuration

    This function accepts a mapping with Arvados configuration settings like
    `ARVADOS_API_HOST` and converts them into a mapping of keyword arguments
    that can be passed to `api_client`. If `ARVADOS_API_HOST` or
    `ARVADOS_API_TOKEN` are not configured, it raises `ValueError`.

    Arguments:

    * version: str | None --- A string naming the version of the Arvados API
      to use. If not specified, the code will log a warning and fall back to
      'v1'.

    * apiconfig: Mapping[str, str] | None --- A mapping with entries for
      `ARVADOS_API_HOST`, `ARVADOS_API_TOKEN`, and optionally
      `ARVADOS_API_HOST_INSECURE`. If not provided, calls
      `arvados.config.settings` to get these parameters from user
      configuration.

    Additional keyword arguments will be included in the return value.
    """
    if apiconfig is None:
        apiconfig = config.settings()
    missing = " and ".join(
        key
        for key in ['ARVADOS_API_HOST', 'ARVADOS_API_TOKEN']
        if key not in apiconfig
    )
    if missing:
        raise ValueError(
            "%s not set.\nPlease set in %s or export environment variable." %
            (missing, config.default_config_file),
        )
    return normalize_api_kwargs(
        version,
        None,
        apiconfig['ARVADOS_API_HOST'],
        apiconfig['ARVADOS_API_TOKEN'],
        insecure=config.flag_is_true('ARVADOS_API_HOST_INSECURE', apiconfig),
        **kwargs,
    )

def api(
        version: Optional[str]=None,
        cache: bool=True,
        host: Optional[str]=None,
        token: Optional[str]=None,
        insecure: bool=False,
        request_id: Optional[str]=None,
        timeout: int=5*60,
        *,
        discoveryServiceUrl: Optional[str]=None,
        **kwargs: Any,
) -> ThreadSafeAPIClient:
    """Dynamically build an Arvados API client

    This function provides a high-level "do what I mean" interface to build an
    Arvados API client object. You can call it with no arguments to build a
    client from user configuration; pass `host` and `token` arguments just
    like you would write in user configuration; or pass additional arguments
    for lower-level control over the client.

    This function returns a `arvados.api.ThreadSafeAPIClient`, an
    API-compatible wrapper around `googleapiclient.discovery.Resource`. If
    you're handling concurrency yourself and/or your application is very
    performance-sensitive, consider calling `api_client` directly.

    Arguments:

    * version: str | None --- A string naming the version of the Arvados API
      to use. If not specified, the code will log a warning and fall back to
      'v1'.

    * host: str | None --- The hostname and optional port number of the
      Arvados API server.

    * token: str | None --- The authentication token to send with each API
      call.

    * discoveryServiceUrl: str | None --- The URL used to discover APIs
      passed directly to `googleapiclient.discovery.build`.

    If `host`, `token`, and `discoveryServiceUrl` are all omitted, `host` and
    `token` will be loaded from the user's configuration. Otherwise, you must
    pass `token` and one of `host` or `discoveryServiceUrl`. It is an error to
    pass both `host` and `discoveryServiceUrl`.

    Other arguments are passed directly to `api_client`. See that function's
    docstring for more information about their meaning.
    """
    kwargs.update(
        cache=cache,
        insecure=insecure,
        request_id=request_id,
        timeout=timeout,
    )
    if discoveryServiceUrl or host or token:
        kwargs.update(normalize_api_kwargs(version, discoveryServiceUrl, host, token))
    else:
        kwargs.update(api_kwargs_from_config(version))
    version = kwargs.pop('version')
    return ThreadSafeAPIClient({}, {}, kwargs, version)

def api_from_config(
        version: Optional[str]=None,
        apiconfig: Optional[Mapping[str, str]]=None,
        **kwargs: Any
) -> ThreadSafeAPIClient:
    """Build an Arvados API client from a configuration mapping

    This function builds an Arvados API client from a mapping with user
    configuration. It accepts that mapping as an argument, so you can use a
    configuration that's different from what the user has set up.

    This function returns a `arvados.api.ThreadSafeAPIClient`, an
    API-compatible wrapper around `googleapiclient.discovery.Resource`. If
    you're handling concurrency yourself and/or your application is very
    performance-sensitive, consider calling `api_client` directly.

    Arguments:

    * version: str | None --- A string naming the version of the Arvados API
      to use. If not specified, the code will log a warning and fall back to
      'v1'.

    * apiconfig: Mapping[str, str] | None --- A mapping with entries for
      `ARVADOS_API_HOST`, `ARVADOS_API_TOKEN`, and optionally
      `ARVADOS_API_HOST_INSECURE`. If not provided, calls
      `arvados.config.settings` to get these parameters from user
      configuration.

    Other arguments are passed directly to `api_client`. See that function's
    docstring for more information about their meaning.
    """
    return api(**api_kwargs_from_config(version, apiconfig, **kwargs))
