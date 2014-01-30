import httplib2
import json
import logging
import os
import re
import types

import apiclient
import apiclient.discovery
import config
import errors
import util

services = {}

class CredentialsFromEnv(object):
    @staticmethod
    def http_request(self, uri, **kwargs):
        from httplib import BadStatusLine
        if 'headers' not in kwargs:
            kwargs['headers'] = {}
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

def http_cache(data_type):
    path = os.environ['HOME'] + '/.cache/arvados/' + data_type
    try:
        util.mkdir_dash_p(path)
    except OSError:
        path = None
    return path

def api(version=None):
    global services

    if 'ARVADOS_DEBUG' in config.settings():
        logging.basicConfig(level=logging.DEBUG)

    if not services.get(version):
        apiVersion = version
        if not version:
            apiVersion = 'v1'
            logging.info("Using default API version. " +
                         "Call arvados.api('%s') instead." %
                         apiVersion)
        if 'ARVADOS_API_HOST' not in config.settings():
            raise Exception("ARVADOS_API_HOST is not set. Aborting.")
        url = ('https://%s/discovery/v1/apis/{api}/{apiVersion}/rest' %
               config.get('ARVADOS_API_HOST'))
        credentials = CredentialsFromEnv()

        # Use system's CA certificates (if we find them) instead of httplib2's
        ca_certs = '/etc/ssl/certs/ca-certificates.crt'
        if not os.path.exists(ca_certs):
            ca_certs = None             # use httplib2 default

        http = httplib2.Http(ca_certs=ca_certs,
                             cache=http_cache('discovery'))
        http = credentials.authorize(http)
        if re.match(r'(?i)^(true|1|yes)$',
                    config.get('ARVADOS_API_HOST_INSECURE', 'no')):
            http.disable_ssl_certificate_validation=True
        services[version] = apiclient.discovery.build(
            'arvados', apiVersion, http=http, discoveryServiceUrl=url)
    return services[version]

