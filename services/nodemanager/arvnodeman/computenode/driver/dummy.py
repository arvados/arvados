#!/usr/bin/env python

from __future__ import absolute_import, print_function

import time

import libcloud.compute.providers as cloud_provider
import libcloud.compute.types as cloud_types

from . import BaseComputeNodeDriver
from .. import arvados_node_fqdn

class ComputeNodeDriver(BaseComputeNodeDriver):
    """Compute node driver wrapper for libcloud's dummy driver.

    This class provides the glue necessary to run the node manager with a
    dummy cloud.  It's useful for testing.
    """
    DEFAULT_DRIVER = cloud_provider.get_driver(cloud_types.Provider.DUMMY)
    DEFAULT_REAL = DEFAULT_DRIVER('ComputeNodeDriver')
    DUMMY_START_TIME = time.time()

    def __init__(self, auth_kwargs, list_kwargs, create_kwargs,
                 driver_class=DEFAULT_DRIVER):
        super(ComputeNodeDriver, self).__init__(
            auth_kwargs, list_kwargs, create_kwargs, driver_class)
        if driver_class is self.DEFAULT_DRIVER:
            self.real = self.DEFAULT_REAL

    def _ensure_private_ip(self, node):
        if not node.private_ips:
            node.private_ips = ['10.10.0.{}'.format(node.id)]

    def arvados_create_kwargs(self, arvados_node):
        return {}

    def list_nodes(self):
        nodelist = super(ComputeNodeDriver, self).list_nodes()
        for node in nodelist:
            self._ensure_private_ip(node)
        return nodelist

    def create_node(self, size, arvados_node):
        node = super(ComputeNodeDriver, self).create_node(size, arvados_node)
        self._ensure_private_ip(node)
        return node

    def sync_node(self, cloud_node, arvados_node):
        cloud_node.name = arvados_node_fqdn(arvados_node)

    @classmethod
    def node_fqdn(cls, node):
        return node.name

    @classmethod
    def node_start_time(cls, node):
        return cls.DUMMY_START_TIME
