#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import, print_function

import functools
import json
import time
import datetime
import filelock 


import libcloud.compute.providers as cloud_provider
import libcloud.compute.types as cloud_types

from . import BaseComputeNodeDriver
from .. import arvados_node_fqdn, arvados_timestamp, ARVADOS_TIMEFMT

cloud_init_config = """
#cloud-config
write_files:
-   content: {}
    owner: root:root
    path: /var/tmp/arv-node-data/arv-ping-url
    permissions: '0400'
"""
LOCKFILE_NAME = 'lockfile'
DOMAIN_NAME = 'domain'
DOMAIN_NAME_VALUE = None

class ComputeNodeDriver(BaseComputeNodeDriver):

    DEFAULT_DRIVER = cloud_provider.get_driver(cloud_types.Provider.OPENSTACK)
    SEARCH_CACHE = {}
    
    """
    Compute node driver for OpenStack. It heavily uses libcloud drivers.
    This class must implement the methods:

    arvados_create_kwargs, sync_node, node_fqdn, node_start_time,
    create_cloud_name
    """

    def __init__(self, auth_kwargs, list_kwargs, create_kwargs,
                 driver_class=DEFAULT_DRIVER):
        global DOMAIN_NAME_VALUE
        if LOCKFILE_NAME not in create_kwargs:
            raise Exception("'{}' not set in config. Cannot continue!".format(LOCKFILE_NAME))
        if DOMAIN_NAME not in create_kwargs:
            raise Exception("'{}' not set in config. Cannot continue!".format(DOMAIN_NAME))
        self._lockfile = create_kwargs[LOCKFILE_NAME]
        if DOMAIN_NAME_VALUE is None:
            DOMAIN_NAME_VALUE = create_kwargs[DOMAIN_NAME]
        list_kwargs = list_kwargs.copy()
        create_kwargs = { key: value for (key, value) in create_kwargs.items() if key != LOCKFILE_NAME and key != DOMAIN_NAME}
        
        super(ComputeNodeDriver, self).__init__(
            auth_kwargs, list_kwargs, create_kwargs,
            driver_class)
        # Check that the create_kwargs contains all the required parameters
        for required_value in ['image_id', 'flavor_id', 'network_id', 'security_group_id']:
            if required_value not in self.create_kwargs:
                raise Exception("FATAL: [Cloud Create] Configuration does not specify " + required_value)     
        
        # Select the arguments used for spawning new nodes
        self._create_node_image = self._select_by_id(self.create_kwargs['image_id'], self.real.list_images())
        self._create_node_flavor = self._select_by_id(self.create_kwargs['flavor_id'], self.real.list_sizes())
        self._create_node_networks = [self._select_by_id(self.create_kwargs['network_id'], self.real.ex_list_networks())]
        self._create_node_security_groups = [self._select_by_id(self.create_kwargs['security_group_id'], self.real.ex_list_security_groups())]
        
    def _select_by_id(self, id_value, items):
        id_value = str(id_value)
        selected_items = [x for x in items if str(x.id) == id_value]
        N = len(selected_items)
        if N == 0:
            raise Exception("FATAL: There is no item with the ID " + id_value + " in the provided list" )
        if N > 1:
            raise Exception("FATAL: The selected ID " + id_value + " is ambiguous.")
        return selected_items[0] 
    
    def arvados_create_kwargs(self, size, arvados_node):
        """Return dynamic keyword arguments for create_node.

        It is required that this class implements this method.
        It should return a dictionary of keyword arguments to pass to the
        libcloud driver's create_node method.  These arguments will extend the
        static arguments in create_kwargs.
        Arguments:

        * size: The node size that will be created (libcloud NodeSize object)
        * arvados_node: The Arvados node record that will be associated
          with this cloud node, as returned from the API server.
        """
        return {
            'name': self.create_cloud_name(arvados_node),
            'size': self._create_node_flavor,
            'ex_userdata': cloud_init_config.format(self._make_ping_url(arvados_node)),
            'ex_config_drive': True,
            'ex_metadata': self.list_kwargs,
            'image': self._create_node_image,
            'networks': self._create_node_networks,
            'ex_security_groups': self._create_node_security_groups
        }

    def sync_node(self, cloud_node, arvados_node):
        return True

    def node_fqdn(cls, node):
        """
        This method should return the FQDN of the node object argument.
        Different clouds store this in different places.
        """
        global DOMAIN_NAME_VALUE
        return (node.name + "." + DOMAIN_NAME_VALUE)

    @classmethod
    def node_start_time(cls, node):
        """ This method should return the time the node was started, in
        seconds since the epoch UTC."""
        if node.created_at:
            return (node.created_at.replace(tzinfo=None) - datetime.datetime(1970, 1, 1).replace(tzinfo=None)).total_seconds()

        if node.extra and node.extra['tags'] and node.extra['tags'].get("booted_at"):
            return arvados_timestamp(node.extra["tags"].get("booted_at"))
        try:
            return arvados_timestamp(cls._get_metadata(
                node.extra['metadata']['items'], 'booted_at'))
        except KeyError:
            return 0
        return 0

    def create_cloud_name(self, arvados_node):
        nodes = [ str(x.name) for x in self.list_nodes() ]
        for i in range(0, 1000):
	    name = 'compute-{}'.format(i)
            if name not in nodes:
                return name 
    
    def list_nodes(self):
        nodes = self.real.list_nodes()
        res = []
	for n in nodes:
            n_metadata = n.extra['metadata']
            append = True
            for (key, value) in self.list_kwargs.items():
                if key not in n_metadata or n_metadata[key] != value:
                    append = False
                    break
            if append:
                n.size = self.sizes[n.extra['flavorId']]
                res.append(n)
        return res

    def create_node(self, size, arvados_node):
        with filelock.FileLock(self._lockfile):
             return super(ComputeNodeDriver, self).create_node(size, arvados_node)

    @classmethod
    def node_id(cls, node):
        global DOMAIN_NAME_VALUE
        return (node.name + "." + DOMAIN_NAME_VALUE)

