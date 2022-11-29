# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
"""Arvados API client

The code in this module builds Arvados API client objects you can use to submit
Arvados API requests. This includes extending the underlying HTTP client with
niceties such as caching, X-Request-Id header for tracking, and more. The main
client constructors are `api` and `api_from_config`.
"""

from __future__ import absolute_import
from future import standard_library
standard_library.install_aliases()
from builtins import range
import collections
import http.client
import httplib2
import json
import logging
import os
import re
import socket
import ssl
import sys
import time
import types

import apiclient
from apiclient import discovery as apiclient_discovery
from apiclient import errors as apiclient_errors
from . import config
from . import errors
from . import util
from . import cache

_logger = logging.getLogger('arvados.api')

MAX_IDLE_CONNECTION_DURATION = 30
RETRY_DELAY_INITIAL = 2
RETRY_DELAY_BACKOFF = 2
RETRY_COUNT = 2

if sys.version_info >= (3,):
    httplib2.SSLHandshakeError = None

class OrderedJsonModel(apiclient.model.JsonModel):
    """Model class for JSON that preserves the contents' order.

    API clients that care about preserving the order of fields in API
    server responses can use this model to do so, like this:

        from arvados.api import OrderedJsonModel
        client = arvados.api('v1', ..., model=OrderedJsonModel())
    """

    def deserialize(self, content):
        # This is a very slightly modified version of the parent class'
        # implementation.  Copyright (c) 2010 Google.
        content = content.decode('utf-8')
        body = json.loads(content, object_pairs_hook=collections.OrderedDict)
        if self._data_wrapper and isinstance(body, dict) and 'data' in body:
            body = body['data']
        return body


def _intercept_http_request(self, uri, method="GET", headers={}, **kwargs):
    if not headers.get('X-Request-Id'):
        headers['X-Request-Id'] = self._request_id()
    try:
        if (self.max_request_size and
            kwargs.get('body') and
            self.max_request_size < len(kwargs['body'])):
            raise apiclient_errors.MediaUploadSizeError("Request size %i bytes exceeds published limit of %i bytes" % (len(kwargs['body']), self.max_request_size))

        headers['Authorization'] = 'OAuth2 %s' % self.arvados_api_token

        retryable = method in [
            'DELETE', 'GET', 'HEAD', 'OPTIONS', 'PUT']
        retry_count = self._retry_count if retryable else 0

        if (not retryable and
            time.time() - self._last_request_time > self._max_keepalive_idle):
            # High probability of failure due to connection atrophy. Make
            # sure this request [re]opens a new connection by closing and
            # forgetting all cached connections first.
            for conn in self.connections.values():
                conn.close()
            self.connections.clear()

        delay = self._retry_delay_initial
        for _ in range(retry_count):
            self._last_request_time = time.time()
            try:
                return self.orig_http_request(uri, method, headers=headers, **kwargs)
            except http.client.HTTPException:
                _logger.debug("[%s] Retrying API request in %d s after HTTP error",
                              headers['X-Request-Id'], delay, exc_info=True)
            except ssl.SSLCertVerificationError as e:
                raise ssl.SSLCertVerificationError(e.args[0], "Could not connect to %s\n%s\nPossible causes: remote SSL/TLS certificate expired, or was issued by an untrusted certificate authority." % (uri, e)) from None
            except socket.error:
                # This is the one case where httplib2 doesn't close the
                # underlying connection first.  Close all open
                # connections, expecting this object only has the one
                # connection to the API server.  This is safe because
                # httplib2 reopens connections when needed.
                _logger.debug("[%s] Retrying API request in %d s after socket error",
                              headers['X-Request-Id'], delay, exc_info=True)
                for conn in self.connections.values():
                    conn.close()

            time.sleep(delay)
            delay = delay * self._retry_delay_backoff

        self._last_request_time = time.time()
        return self.orig_http_request(uri, method, headers=headers, **kwargs)
    except Exception as e:
        # Prepend "[request_id] " to the error message, which we
        # assume is the first string argument passed to the exception
        # constructor.
        for i in range(len(e.args or ())):
            if type(e.args[i]) == type(""):
                e.args = e.args[:i] + ("[{}] {}".format(headers['X-Request-Id'], e.args[i]),) + e.args[i+1:]
                raise type(e)(*e.args)
        raise

def _patch_http_request(http, api_token):
    http.arvados_api_token = api_token
    http.max_request_size = 0
    http.orig_http_request = http.request
    http.request = types.MethodType(_intercept_http_request, http)
    http._last_request_time = 0
    http._max_keepalive_idle = MAX_IDLE_CONNECTION_DURATION
    http._retry_delay_initial = RETRY_DELAY_INITIAL
    http._retry_delay_backoff = RETRY_DELAY_BACKOFF
    http._retry_count = RETRY_COUNT
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

def http_cache(data_type):
    homedir = os.environ.get('HOME')
    if not homedir or len(homedir) == 0:
        return None
    path = homedir + '/.cache/arvados/' + data_type
    try:
        util.mkdir_dash_p(path)
    except OSError:
        return None
    return cache.SafeHTTPCache(path, max_age=60*60*24*2)

def api_client(
        version,
        discoveryServiceUrl,
        token,
        *,
        cache=True,
        http=None,
        insecure=False,
        request_id=None,
        timeout=5*60,
        **kwargs,
):
    """Build an Arvados API client

    This function returns a `googleapiclient.discovery.Resource` object
    constructed from the given arguments. This is a relatively low-level
    interface that requires all the necessary inputs as arguments. Most
    users will prefer to use `api` which can accept more flexible inputs.

    Arguments:

    version: str
    : A string naming the version of the Arvados API to use.

    discoveryServiceUrl: str
    : The URL used to discover APIs passed directly to
      `googleapiclient.discovery.build`.

    token: str
    : The authentication token to send with each API call.

    Keyword-only arguments:

    cache: bool
    : If true, loads the API discovery document from, or saves it to, a cache
      on disk (located at `~/.cache/arvados/discovery`).

    http: httplib2.Http | None
    : The HTTP client object the API client object will use to make requests.
      If not provided, this function will build its own to use. Either way, the
      object will be patched as part of the build process.

    insecure: bool
    : If true, ignore SSL certificate validation errors. Default `False`.

    request_id: str | None
    : Default `X-Request-Id` header value for outgoing requests that
      don't already provide one. If `None` or omitted, generate a random
      ID. When retrying failed requests, the same ID is used on all
      attempts.

    timeout: int
    : A timeout value for HTTP requests in seconds. Default 300 (5 minutes).

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
    http = _patch_http_request(http, token)

    svc = apiclient_discovery.build(
        'arvados', version,
        cache_discovery=False,
        discoveryServiceUrl=discoveryServiceUrl,
        http=http,
        **kwargs,
    )
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
        version=None,
        discoveryServiceUrl=None,
        host=None,
        token=None,
        **kwargs,
):
    """Validate kwargs from `api` and build kwargs for `api_client`

    This method takes high-level keyword arguments passed to the `api`
    constructor and normalizes them into a new dictionary that can be passed
    as keyword arguments to `api_client`. It raises `ValueError` if required
    arguments are missing or conflict.

    Arguments:

    version: str | None
    : A string naming the version of the Arvados API to use. If not specified,
      the code will log a warning and fall back to 'v1'.

    discoveryServiceUrl: str | None
    : The URL used to discover APIs passed directly to
      `googleapiclient.discovery.build`. It is an error to pass both
      `discoveryServiceUrl` and `host`.

    host: str | None
    : The hostname and optional port number of the Arvados API server. Used to
      build `discoveryServiceUrl`. It is an error to pass both
      `discoveryServiceUrl` and `host`.

    token: str
    : The authentication token to send with each API call.

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

def api_kwargs_from_config(version=None, apiconfig=None, **kwargs):
    """Build `api_client` keyword arguments from configuration

    This function accepts a mapping with Arvados configuration settings like
    `ARVADOS_API_HOST` and converts them into a mapping of keyword arguments
    that can be passed to `api_client`. If `ARVADOS_API_HOST` or
    `ARVADOS_API_TOKEN` are not configured, it raises `ValueError`.

    Arguments:

    version: str | None
    : A string naming the version of the Arvados API to use. If not specified,
      the code will log a warning and fall back to 'v1'.

    apiconfig: Mapping[str, str] | None
    : A mapping with entries for `ARVADOS_API_HOST`, `ARVADOS_API_TOKEN`, and
      optionally `ARVADOS_API_HOST_INSECURE`. If not provided, calls
      `arvados.config.settings` to get these parameters from user configuration.

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

def api(version=None, cache=True, host=None, token=None, insecure=False,
        request_id=None, timeout=5*60, *,
        discoveryServiceUrl=None, **kwargs):
    """Dynamically build an Arvados API client

    This function provides a high-level "do what I mean" interface to build an
    Arvados API client object. You can call it with no arguments to build a
    client from user configuration; pass `host` and `token` arguments just
    like you would write in user configuration; or pass additional arguments
    for lower-level control over the client.

    This function returns a `arvados.safeapi.ThreadSafeApiCache`, an
    API-compatible wrapper around `googleapiclient.discovery.Resource`. If
    you're handling concurrency yourself and/or your application is very
    performance-sensitive, consider calling `api_client` directly.

    Arguments:

    version: str | None
    : A string naming the version of the Arvados API to use. If not specified,
      the code will log a warning and fall back to 'v1'.

    host: str | None
    : The hostname and optional port number of the Arvados API server.

    token: str | None
    : The authentication token to send with each API call.

    discoveryServiceUrl: str | None
    : The URL used to discover APIs passed directly to
      `googleapiclient.discovery.build`.

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
    # We do the import here to avoid a circular import at the top level.
    from .safeapi import ThreadSafeApiCache
    return ThreadSafeApiCache({}, {}, kwargs, version)

def api_from_config(version=None, apiconfig=None, **kwargs):
    """Build an Arvados API client from a configuration mapping

    This function builds an Arvados API client from a mapping with user
    configuration. It accepts that mapping as an argument, so you can use a
    configuration that's different from what the user has set up.

    This function returns a `arvados.safeapi.ThreadSafeApiCache`, an
    API-compatible wrapper around `googleapiclient.discovery.Resource`. If
    you're handling concurrency yourself and/or your application is very
    performance-sensitive, consider calling `api_client` directly.

    Arguments:

    version: str | None
    : A string naming the version of the Arvados API to use. If not specified,
      the code will log a warning and fall back to 'v1'.

    apiconfig: Mapping[str, str] | None
    : A mapping with entries for `ARVADOS_API_HOST`, `ARVADOS_API_TOKEN`, and
      optionally `ARVADOS_API_HOST_INSECURE`. If not provided, calls
      `arvados.config.settings` to get these parameters from user configuration.

    Other arguments are passed directly to `api_client`. See that function's
    docstring for more information about their meaning.
    """
    return api(**api_kwargs_from_config(version, apiconfig, **kwargs))
