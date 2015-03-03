import httplib2
import json
import logging
import os
import re
import types

import apiclient
from apiclient import discovery as apiclient_discovery
from apiclient import errors as apiclient_errors
import config
import errors
import util

_logger = logging.getLogger('arvados.api')

class CredentialsFromToken(object):
    def __init__(self, api_token):
        self.api_token = api_token

    @staticmethod
    def http_request(self, uri, **kwargs):
        from httplib import BadStatusLine

        if (self.max_request_size and
            kwargs.get('body') and
            self.max_request_size < len(kwargs['body'])):
            raise apiclient_errors.MediaUploadSizeError("Request size %i bytes exceeds published limit of %i bytes" % (len(kwargs['body']), self.max_request_size))

        if 'headers' not in kwargs:
            kwargs['headers'] = {}

        if config.get("ARVADOS_EXTERNAL_CLIENT", "") == "true":
            kwargs['headers']['X-External-Client'] = '1'

        kwargs['headers']['Authorization'] = 'OAuth2 %s' % self.arvados_api_token
        try:
            return self.orig_http_request(uri, **kwargs)
        except BadStatusLine:
            # This is how httplib tells us that it tried to reuse an
            # existing connection but it was already closed by the
            # server. In that case, yes, we would like to retry.
            # Unfortunately, we are not absolutely certain that the
            # previous call did not succeed, so this is slightly
            # risky.
            return self.orig_http_request(uri, **kwargs)

    def authorize(self, http):
        http.arvados_api_token = self.api_token
        http.orig_http_request = http.request
        http.request = types.MethodType(self.http_request, http)
        http.max_request_size = 0
        return http

# Monkey patch discovery._cast() so objects and arrays get serialized
# with json.dumps() instead of str().
_cast_orig = apiclient_discovery._cast
def _cast_objects_too(value, schema_type):
    global _cast_orig
    if (type(value) != type('') and
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
    path = os.environ['HOME'] + '/.cache/arvados/' + data_type
    try:
        util.mkdir_dash_p(path)
    except OSError:
        path = None
    return path

def api(version=None, cache=True, host=None, token=None, insecure=False, **kwargs):
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
        return api_from_config(version=version, cache=cache, **kwargs)
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
        http_kwargs = {}
        # Prefer system's CA certificates (if available) over httplib2's.
        certs_path = '/etc/ssl/certs/ca-certificates.crt'
        if os.path.exists(certs_path):
            http_kwargs['ca_certs'] = certs_path
        if cache:
            http_kwargs['cache'] = http_cache('discovery')
        if insecure:
            http_kwargs['disable_ssl_certificate_validation'] = True
        kwargs['http'] = httplib2.Http(**http_kwargs)

    credentials = CredentialsFromToken(api_token=token)
    kwargs['http'] = credentials.authorize(kwargs['http'])

    svc = apiclient_discovery.build('arvados', version, **kwargs)
    svc.api_token = token
    kwargs['http'].max_request_size = svc._rootDesc.get('maxRequestSize', 0)
    kwargs['http'].cache = None
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

    for x in ['ARVADOS_API_HOST', 'ARVADOS_API_TOKEN']:
        if x not in apiconfig:
            raise ValueError("%s is not set. Aborting." % x)
    host = apiconfig.get('ARVADOS_API_HOST')
    token = apiconfig.get('ARVADOS_API_TOKEN')
    insecure = config.flag_is_true('ARVADOS_API_HOST_INSECURE', apiconfig)

    return api(version=version, host=host, token=token, insecure=insecure, **kwargs)
