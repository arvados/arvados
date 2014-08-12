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
services = {}

class CredentialsFromEnv(object):
    @staticmethod
    def http_request(self, uri, **kwargs):
        from httplib import BadStatusLine
        if 'headers' not in kwargs:
            kwargs['headers'] = {}

        if config.get("ARVADOS_EXTERNAL_CLIENT", "") == "true":
            kwargs['headers']['X-External-Client'] = '1'

        kwargs['headers']['Authorization'] = 'OAuth2 %s' % config.get('ARVADOS_API_TOKEN', 'ARVADOS_API_TOKEN_not_set')
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

def api(version=None, cache=True, **kwargs):
    """Return an apiclient Resources object for an Arvados instance.

    Arguments:
    * version: A string naming the version of the Arvados API to use (for
      example, 'v1').
    * cache: If True (default), return an existing resources object, or use
      a cached discovery document to build one.

    Additional keyword arguments will be passed directly to
    `apiclient.discovery.build`.  If the `discoveryServiceUrl` or `http`
    keyword arguments are missing, this function will set default values for
    them, based on the current Arvados configuration settings."""
    if not cache or not services.get(version):
        if not version:
            version = 'v1'
            _logger.info("Using default API version. " +
                         "Call arvados.api('%s') instead.",
                         version)

        if 'discoveryServiceUrl' not in kwargs:
            api_host = config.get('ARVADOS_API_HOST')
            if not api_host:
                raise ValueError(
                    "No discoveryServiceUrl or ARVADOS_API_HOST set.")
            kwargs['discoveryServiceUrl'] = (
                'https://%s/discovery/v1/apis/{api}/{apiVersion}/rest' %
                (api_host,))

        if 'http' not in kwargs:
            http_kwargs = {}
            # Prefer system's CA certificates (if available) over httplib2's.
            certs_path = '/etc/ssl/certs/ca-certificates.crt'
            if os.path.exists(certs_path):
                http_kwargs['ca_certs'] = certs_path
            if cache:
                http_kwargs['cache'] = http_cache('discovery')
            if (config.get('ARVADOS_API_HOST_INSECURE', '').lower() in
                  ('yes', 'true', '1')):
                http_kwargs['disable_ssl_certificate_validation'] = True
            kwargs['http'] = httplib2.Http(**http_kwargs)

        kwargs['http'] = CredentialsFromEnv().authorize(kwargs['http'])
        services[version] = apiclient.discovery.build('arvados', version,
                                                      **kwargs)
        kwargs['http'].cache = None
    return services[version]

def uncache_api(version):
    if version in services:
        del services[version]
