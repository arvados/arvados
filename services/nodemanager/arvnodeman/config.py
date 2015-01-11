#!/usr/bin/env python

from __future__ import absolute_import, print_function

import ConfigParser
import importlib
import logging
import ssl
import sys

import arvados
import httplib2
import pykka
from apiclient import errors as apierror

# IOError is the base class for socket.error and friends.
# It seems like it hits the sweet spot for operations we want to retry:
# it's low-level, but unlikely to catch code bugs.
NETWORK_ERRORS = (IOError, ssl.SSLError)
ARVADOS_ERRORS = NETWORK_ERRORS + (apierror.Error,)

actor_class = pykka.ThreadingActor

class NodeManagerConfig(ConfigParser.SafeConfigParser):
    """Node Manager Configuration class.

    This a standard Python ConfigParser, with additional helper methods to
    create objects instantiated with configuration information.
    """

    LOGGING_NONLEVELS = frozenset(['file'])

    def __init__(self, *args, **kwargs):
        # Can't use super() because SafeConfigParser is an old-style class.
        ConfigParser.SafeConfigParser.__init__(self, *args, **kwargs)
        for sec_name, settings in {
            'Arvados': {'insecure': 'no',
                        'timeout': '15'},
            'Daemon': {'min_nodes': '0',
                       'max_nodes': '1',
                       'poll_time': '60',
                       'max_poll_time': '300',
                       'poll_stale_after': '600',
                       'boot_fail_after': str(sys.maxint),
                       'node_stale_after': str(60 * 60 * 2)},
            'Logging': {'file': '/dev/stderr',
                        'level': 'WARNING'},
        }.iteritems():
            if not self.has_section(sec_name):
                self.add_section(sec_name)
            for opt_name, value in settings.iteritems():
                if not self.has_option(sec_name, opt_name):
                    self.set(sec_name, opt_name, value)

    def get_section(self, section, transformer=None):
        result = self._dict()
        for key, value in self.items(section):
            if transformer is not None:
                try:
                    value = transformer(value)
                except (TypeError, ValueError):
                    pass
            result[key] = value
        return result

    def log_levels(self):
        return {key: getattr(logging, self.get('Logging', key).upper())
                for key in self.options('Logging')
                if key not in self.LOGGING_NONLEVELS}

    def dispatch_classes(self):
        mod_name = 'arvnodeman.computenode.dispatch'
        if self.has_option('Daemon', 'dispatcher'):
            mod_name = '{}.{}'.format(mod_name,
                                      self.get('Daemon', 'dispatcher'))
        module = importlib.import_module(mod_name)
        return (module.ComputeNodeSetupActor,
                module.ComputeNodeShutdownActor,
                module.ComputeNodeUpdateActor,
                module.ComputeNodeMonitorActor)

    def new_arvados_client(self):
        if self.has_option('Daemon', 'certs_file'):
            certs_file = self.get('Daemon', 'certs_file')
        else:
            certs_file = None
        insecure = self.getboolean('Arvados', 'insecure')
        http = httplib2.Http(timeout=self.getint('Arvados', 'timeout'),
                             ca_certs=certs_file,
                             disable_ssl_certificate_validation=insecure)
        return arvados.api('v1',
                           cache=False,  # Don't reuse an existing client.
                           host=self.get('Arvados', 'host'),
                           token=self.get('Arvados', 'token'),
                           insecure=insecure,
                           http=http)

    def new_cloud_client(self):
        module = importlib.import_module('arvnodeman.computenode.driver.' +
                                         self.get('Cloud', 'provider'))
        auth_kwargs = self.get_section('Cloud Credentials')
        if 'timeout' in auth_kwargs:
            auth_kwargs['timeout'] = int(auth_kwargs['timeout'])
        return module.ComputeNodeDriver(auth_kwargs,
                                        self.get_section('Cloud List'),
                                        self.get_section('Cloud Create'))

    def node_sizes(self, all_sizes):
        size_kwargs = {}
        for sec_name in self.sections():
            sec_words = sec_name.split(None, 2)
            if sec_words[0] != 'Size':
                continue
            size_kwargs[sec_words[1]] = self.get_section(sec_name, int)
        return [(size, size_kwargs[size.id]) for size in all_sizes
                if size.id in size_kwargs]

    def shutdown_windows(self):
        return [int(n)
                for n in self.get('Cloud', 'shutdown_windows').split(',')]
