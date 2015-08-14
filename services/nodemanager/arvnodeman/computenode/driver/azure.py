#!/usr/bin/env python

from __future__ import absolute_import, print_function

import time

import libcloud.compute.base as cloud_base
import libcloud.compute.providers as cloud_provider
import libcloud.compute.types as cloud_types

from . import BaseComputeNodeDriver
from .. import arvados_node_fqdn

class ComputeNodeDriver(BaseComputeNodeDriver):

    DEFAULT_DRIVER = cloud_provider.get_driver(cloud_types.Provider.AZURE)
    SEARCH_CACHE = {}

    def __init__(self, auth_kwargs, list_kwargs, create_kwargs,
                 driver_class=DEFAULT_DRIVER):
        super(ComputeNodeDriver, self).__init__(
            auth_kwargs, list_kwargs, create_kwargs,
            driver_class)

    def arvados_create_kwargs(self, arvados_node):
        return {'name': arvados_node["uuid"]}

    def sync_node(self, cloud_node, arvados_node):
        print("In sync_node")

    def _init_image(self, image):
        return 'image', self.search_for(image, 'list_images')

    def _init_password(self, password):
        return 'auth', cloud_base.NodeAuthPassword(password)

    @classmethod
    def node_fqdn(cls, node):
        return node.name

    @classmethod
    def node_start_time(cls, node):
        pass
