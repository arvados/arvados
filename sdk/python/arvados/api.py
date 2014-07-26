import httplib2
import json
import logging
import os
import re
import types
import hashlib

import apiclient
import apiclient.discovery
import config
import errors
import util

services = {}

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

def http_cache(data_type):
    path = os.environ['HOME'] + '/.cache/arvados/' + data_type
    try:
        util.mkdir_dash_p(path)
    except OSError:
        path = None
    return path

def api(version=None, cache=True, host=None, token=None, insecure=False):
    global services

    if 'ARVADOS_DEBUG' in config.settings():
        logging.basicConfig(level=logging.DEBUG)

    if not version:
        version = 'v1'
        logging.info("Using default API version. " +
                     "Call arvados.api('%s') instead." %
                     version)
    if host and token:
        # Provided by caller
        pass
    elif not host and not token:
        # Load from user configuration or environment
        for x in ['ARVADOS_API_HOST', 'ARVADOS_API_TOKEN']:
            if x not in config.settings():
                raise Exception("%s is not set. Aborting." % x)
        host = config.get('ARVADOS_API_HOST')
        token = config.get('ARVADOS_API_TOKEN')
        apiinsecure = re.match(r'(?i)^(true|1|yes)$',
                               config.get('ARVADOS_API_HOST_INSECURE', 'no'))
    else:
        # Caller provided one but not the other
        if not host:
            raise Exception("token argument provided, but host missing.")
        else:
            raise Exception("host argument provided, but token missing.")

    connprofile = hashlib.sha1(' '.join([
        version, host, token, ('y' if apiinsecure else 'n')
    ])).hexdigest()

    if not cache or not services.get(connprofile):
        url = 'https://%s/discovery/v1/apis/{api}/{apiVersion}/rest' % host
        credentials = CredentialsFromToken(api_token=token)

        # Use system's CA certificates (if we find them) instead of httplib2's
        ca_certs = '/etc/ssl/certs/ca-certificates.crt'
        if not os.path.exists(ca_certs):
            ca_certs = None             # use httplib2 default

        http = httplib2.Http(ca_certs=ca_certs,
                             cache=(http_cache('discovery') if cache else None))
        http = credentials.authorize(http)
        if apiinsecure:
            http.disable_ssl_certificate_validation = True
        services[connprofile] = apiclient.discovery.build(
            'arvados', version, http=http, discoveryServiceUrl=url)
        http.cache = None

    return services[connprofile]
