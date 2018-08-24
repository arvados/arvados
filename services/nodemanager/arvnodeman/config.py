#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import, print_function

import ConfigParser
import importlib
import logging
import sys

import arvados
import httplib2
import pykka
from apiclient import errors as apierror

from .baseactor import BaseNodeManagerActor

from functools import partial
from libcloud.common.types import LibcloudError
from libcloud.common.exceptions import BaseHTTPError

# IOError is the base class for socket.error, ssl.SSLError, and friends.
# It seems like it hits the sweet spot for operations we want to retry:
# it's low-level, but unlikely to catch code bugs.
NETWORK_ERRORS = (IOError,)
ARVADOS_ERRORS = NETWORK_ERRORS + (apierror.Error,)
CLOUD_ERRORS = NETWORK_ERRORS + (LibcloudError, BaseHTTPError)

actor_class = BaseNodeManagerActor

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
                        'timeout': '15',
                        'jobs_queue': 'yes',
                        'slurm_queue': 'yes'
                    },
            'Daemon': {'min_nodes': '0',
                       'max_nodes': '1',
                       'poll_time': '60',
                       'cloudlist_poll_time': '0',
                       'nodelist_poll_time': '0',
                       'wishlist_poll_time': '0',
                       'max_poll_time': '300',
                       'poll_stale_after': '600',
                       'max_total_price': '0',
                       'boot_fail_after': str(sys.maxint),
                       'node_stale_after': str(60 * 60 * 2),
                       'watchdog': '600',
                       'node_mem_scaling': '0.95',
                       'consecutive_idle_count': '2'},
            'Manage': {'address': '127.0.0.1',
                       'port': '-1',
                       'ManagementToken': ''},
            'Logging': {'file': '/dev/stderr',
                        'level': 'WARNING'}
        }.iteritems():
            if not self.has_section(sec_name):
                self.add_section(sec_name)
            for opt_name, value in settings.iteritems():
                if not self.has_option(sec_name, opt_name):
                    self.set(sec_name, opt_name, value)

    def get_section(self, section, transformers={}, default_transformer=None):
        transformer_map = {
            str: self.get,
            int: self.getint,
            bool: self.getboolean,
            float: self.getfloat,
        }
        result = self._dict()
        for key, value in self.items(section):
            transformer = None
            if transformers.get(key) in transformer_map:
                transformer = partial(transformer_map[transformers[key]], section)
            elif default_transformer in transformer_map:
                transformer = partial(transformer_map[default_transformer], section)
            if transformer is not None:
                try:
                    value = transformer(key)
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
        return arvados.api(version='v1',
                           host=self.get('Arvados', 'host'),
                           token=self.get('Arvados', 'token'),
                           insecure=insecure,
                           http=http)

    def new_cloud_client(self):
        module = importlib.import_module('arvnodeman.computenode.driver.' +
                                         self.get('Cloud', 'provider'))
        driver_class = module.ComputeNodeDriver.DEFAULT_DRIVER
        if self.has_option('Cloud', 'driver_class'):
            d = self.get('Cloud', 'driver_class').split('.')
            mod = '.'.join(d[:-1])
            cls = d[-1]
            driver_class = importlib.import_module(mod).__dict__[cls]
        auth_kwargs = self.get_section('Cloud Credentials')
        if 'timeout' in auth_kwargs:
            auth_kwargs['timeout'] = int(auth_kwargs['timeout'])
        return module.ComputeNodeDriver(auth_kwargs,
                                        self.get_section('Cloud List'),
                                        self.get_section('Cloud Create'),
                                        driver_class=driver_class)

    def node_sizes(self):
        """Finds all acceptable NodeSizes for our installation.

        Returns a list of (NodeSize, kwargs) pairs for each NodeSize object
        returned by libcloud that matches a size listed in our config file.
        """
        all_sizes = self.new_cloud_client().list_sizes()
        size_kwargs = {}
        section_types = {
            'instance_type': str,
            'price': float,
            'preemptible': bool,
        }
        for sec_name in self.sections():
            sec_words = sec_name.split(None, 2)
            if sec_words[0] != 'Size':
                continue
            size_spec = self.get_section(sec_name, section_types, int)
            if 'preemptible' not in size_spec:
                size_spec['preemptible'] = False
            if 'instance_type' not in size_spec:
                # Assume instance type is Size name if missing
                size_spec['instance_type'] = sec_words[1]
            size_spec['id'] = sec_words[1]
            size_kwargs[sec_words[1]] = size_spec
        # EC2 node sizes are identified by id. GCE sizes are identified by name.
        matching_sizes = []
        for size in all_sizes:
            matching_sizes += [
                (size, size_kwargs[s]) for s in size_kwargs
                if size_kwargs[s]['instance_type'] == size.id
                or size_kwargs[s]['instance_type'] == size.name
            ]
        return matching_sizes

    def shutdown_windows(self):
        return [float(n)
                for n in self.get('Cloud', 'shutdown_windows').split(',')]
