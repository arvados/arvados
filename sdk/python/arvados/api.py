import httplib2
import json
import logging
import os
import re
import types

import apiclient
import apiclient.discovery
import apiclient.errors
import config
import errors
import util

_logger = logging.getLogger('arvados.api')
conncache = {}

class CredentialsFromToken(object):
    def __init__(self, api_token):
        self.api_token = api_token

    @staticmethod
    def http_request(self, uri, **kwargs):
        from httplib import BadStatusLine
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
        return http

# Monkey patch discovery._cast() so objects and arrays get serialized
# with json.dumps() instead of str().
_cast_orig = apiclient.discovery._cast
def _cast_objects_too(value, schema_type):
    global _cast_orig
    if (type(value) != type('') and
        (schema_type == 'object' or schema_type == 'array')):
        return json.dumps(value)
    else:
        return _cast_orig(value, schema_type)
apiclient.discovery._cast = _cast_objects_too

# Convert apiclient's HttpErrors into our own API error subclass for better
# error reporting.
# Reassigning apiclient.errors.HttpError is not sufficient because most of the
# apiclient submodules import the class into their own namespace.
def _new_http_error(cls, *args, **kwargs):
    return super(apiclient.errors.HttpError, cls).__new__(
        errors.ApiError, *args, **kwargs)
apiclient.errors.HttpError.__new__ = staticmethod(_new_http_error)

def http_cache(data_type):
    path = os.environ['HOME'] + '/.cache/arvados/' + data_type
    try:
        util.mkdir_dash_p(path)
    except OSError:
        path = None
    return path

def api(version=None, cache=True, host=None, token=None, insecure=False, **kwargs):
    """Return an apiclient Resources object for an Arvados instance.

    Arguments:
    * version: A string naming the version of the Arvados API to use (for
      example, 'v1').
    * cache: If True (default), return an existing Resources object if
      one already exists with the same endpoint and credentials. If
      False, create a new one, and do not keep it in the cache (i.e.,
      do not return it from subsequent api(cache=True) calls with
      matching endpoint and credentials).
    * host: The Arvados API server host (and optional :port) to connect to.
    * token: The authentication token to send with each API call.
    * insecure: If True, ignore SSL certificate validation errors.

    Additional keyword arguments will be passed directly to
    `apiclient.discovery.build` if a new Resource object is created.
    If the `discoveryServiceUrl` or `http` keyword arguments are
    missing, this function will set default values for them, based on
    the current Arvados configuration settings.

    """

    if not version:
        version = 'v1'
        logging.info("Using default API version. " +
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
        # Load from user configuration or environment
        for x in ['ARVADOS_API_HOST', 'ARVADOS_API_TOKEN']:
            if x not in config.settings():
                raise ValueError("%s is not set. Aborting." % x)
        host = config.get('ARVADOS_API_HOST')
        token = config.get('ARVADOS_API_TOKEN')
        insecure = config.flag_is_true('ARVADOS_API_HOST_INSECURE')
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

    if cache:
        connprofile = (version, host, token, insecure)
        svc = conncache.get(connprofile)
        if svc:
            return svc

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

    svc = apiclient.discovery.build('arvados', version, **kwargs)
    svc.api_token = token
    kwargs['http'].cache = None
    if cache:
        conncache[connprofile] = svc
    return svc
