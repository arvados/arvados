# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

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

class OrderedJsonModel(apiclient.model.JsonModel):
    """Model class for JSON that preserves the contents' order.

    API clients that care about preserving the order of fields in API
    server responses can use this model to do so, like this::

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
    if (self.max_request_size and
        kwargs.get('body') and
        self.max_request_size < len(kwargs['body'])):
        raise apiclient_errors.MediaUploadSizeError("Request size %i bytes exceeds published limit of %i bytes" % (len(kwargs['body']), self.max_request_size))

    if config.get("ARVADOS_EXTERNAL_CLIENT", "") == "true":
        headers['X-External-Client'] = '1'

    headers['Authorization'] = 'OAuth2 %s' % self.arvados_api_token
    if not headers.get('X-Request-Id'):
        headers['X-Request-Id'] = self._request_id()

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
            _logger.debug("Retrying API request in %d s after HTTP error",
                          delay, exc_info=True)
        except socket.error:
            # This is the one case where httplib2 doesn't close the
            # underlying connection first.  Close all open
            # connections, expecting this object only has the one
            # connection to the API server.  This is safe because
            # httplib2 reopens connections when needed.
            _logger.debug("Retrying API request in %d s after socket error",
                          delay, exc_info=True)
            for conn in self.connections.values():
                conn.close()
        except httplib2.SSLHandshakeError as e:
            # Intercept and re-raise with a better error message.
            raise httplib2.SSLHandshakeError("Could not connect to %s\n%s\nPossible causes: remote SSL/TLS certificate expired, or was issued by an untrusted certificate authority." % (uri, e))

        time.sleep(delay)
        delay = delay * self._retry_delay_backoff

    self._last_request_time = time.time()
    return self.orig_http_request(uri, method, headers=headers, **kwargs)

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

def api(version=None, cache=True, host=None, token=None, insecure=False,
        request_id=None, timeout=5*60, **kwargs):
    """Return an apiclient Resources object for an Arvados instance.

    :version:
      A string naming the version of the Arvados API to use (for
      example, 'v1').

    :cache:
      Use a cache (~/.cache/arvados/discovery) for the discovery
      document.

    :host:
      The Arvados API server host (and optional :port) to connect to.

    :token:
      The authentication token to send with each API call.

    :insecure:
      If True, ignore SSL certificate validation errors.

    :timeout:
      A timeout value for http requests.

    :request_id:
      Default X-Request-Id header value for outgoing requests that
      don't already provide one. If None or omitted, generate a random
      ID. When retrying failed requests, the same ID is used on all
      attempts.

    Additional keyword arguments will be passed directly to
    `apiclient_discovery.build` if a new Resource object is created.
    If the `discoveryServiceUrl` or `http` keyword arguments are
    missing, this function will set default values for them, based on
    the current Arvados configuration settings.

    """

    if not version:
        version = 'v1'
        _logger.info("Using default API version. " +
                     "Call arvados.api('%s') instead." %
                     version)
    if 'discoveryServiceUrl' in kwargs:
        if host:
            raise ValueError("both discoveryServiceUrl and host provided")
        # Here we can't use a token from environment, config file,
        # etc. Those probably have nothing to do with the host
        # provided by the caller.
        if not token:
            raise ValueError("discoveryServiceUrl provided, but token missing")
    elif host and token:
        pass
    elif not host and not token:
        return api_from_config(
            version=version, cache=cache, request_id=request_id, **kwargs)
    else:
        # Caller provided one but not the other
        if not host:
            raise ValueError("token argument provided, but host missing.")
        else:
            raise ValueError("host argument provided, but token missing.")

    if host:
        # Caller wants us to build the discoveryServiceUrl
        kwargs['discoveryServiceUrl'] = (
            'https://%s/discovery/v1/apis/{api}/{apiVersion}/rest' % (host,))

    if 'http' not in kwargs:
        http_kwargs = {'ca_certs': util.ca_certs_path()}
        if cache:
            http_kwargs['cache'] = http_cache('discovery')
        if insecure:
            http_kwargs['disable_ssl_certificate_validation'] = True
        kwargs['http'] = httplib2.Http(**http_kwargs)

    if kwargs['http'].timeout is None:
        kwargs['http'].timeout = timeout

    kwargs['http'] = _patch_http_request(kwargs['http'], token)

    svc = apiclient_discovery.build('arvados', version, cache_discovery=False, **kwargs)
    svc.api_token = token
    svc.insecure = insecure
    svc.request_id = request_id
    kwargs['http'].max_request_size = svc._rootDesc.get('maxRequestSize', 0)
    kwargs['http'].cache = None
    kwargs['http']._request_id = lambda: svc.request_id or util.new_request_id()
    return svc

def api_from_config(version=None, apiconfig=None, **kwargs):
    """Return an apiclient Resources object enabling access to an Arvados server
    instance.

    :version:
      A string naming the version of the Arvados REST API to use (for
      example, 'v1').

    :apiconfig:
      If provided, this should be a dict-like object (must support the get()
      method) with entries for ARVADOS_API_HOST, ARVADOS_API_TOKEN, and
      optionally ARVADOS_API_HOST_INSECURE.  If not provided, use
      arvados.config (which gets these parameters from the environment by
      default.)

    Other keyword arguments such as `cache` will be passed along `api()`

    """
    # Load from user configuration or environment
    if apiconfig is None:
        apiconfig = config.settings()

    errors = []
    for x in ['ARVADOS_API_HOST', 'ARVADOS_API_TOKEN']:
        if x not in apiconfig:
            errors.append(x)
    if errors:
        raise ValueError(" and ".join(errors)+" not set.\nPlease set in %s or export environment variable." % config.default_config_file)
    host = apiconfig.get('ARVADOS_API_HOST')
    token = apiconfig.get('ARVADOS_API_TOKEN')
    insecure = config.flag_is_true('ARVADOS_API_HOST_INSECURE', apiconfig)

    return api(version=version, host=host, token=token, insecure=insecure, **kwargs)
