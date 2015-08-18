#!/usr/bin/env python

from __future__ import absolute_import, print_function

import time
from operator import attrgetter

import libcloud.compute.base as cloud_base
import libcloud.compute.providers as cloud_provider
import libcloud.compute.types as cloud_types

from . import BaseComputeNodeDriver
from .. import arvados_node_fqdn, arvados_timestamp, ARVADOS_TIMEFMT

class ComputeNodeDriver(BaseComputeNodeDriver):

    DEFAULT_DRIVER = cloud_provider.get_driver(cloud_types.Provider.AZURE_ARM)
    SEARCH_CACHE = {}

    def __init__(self, auth_kwargs, list_kwargs, create_kwargs,
                 driver_class=DEFAULT_DRIVER):
        list_kwargs["ex_resource_group"] = create_kwargs["ex_resource_group"]

        self.tags = {key[4:]: value
                     for key, value in create_kwargs.iteritems()
                     if key.startswith('tag_')}
        super(ComputeNodeDriver, self).__init__(
            auth_kwargs, list_kwargs, create_kwargs,
            driver_class)

    def arvados_create_kwargs(self, arvados_node):
        cluster_id, _, node_id = arvados_node['uuid'].split('-')
        name = 'compute-{}-{}'.format(node_id, cluster_id)
        tags = {
            'booted_at': time.strftime(ARVADOS_TIMEFMT, time.gmtime()),
            'arv-ping-url': self._make_ping_url(arvados_node),
        }
        tags.update(self.tags)
        return {
            'name': name,
            'ex_tags': tags,
        }

    def sync_node(self, cloud_node, arvados_node):
        hostname = arvados_node_fqdn(arvados_node)
        self.real.ex_create_tags(cloud_node.id, {"hostname": hostname})

    def _init_image(self, urn):
        return "image", self.list_images(ex_urn=urn)[0]

    def _init_ssh_key(self, filename):
        with open(filename) as ssh_file:
            key = cloud_base.NodeAuthSSHKey(ssh_file.read())
        return 'auth', key

    def list_nodes(self):
        # Azure only supports filtering node lists by resource group.
        # Do our own filtering based on tag.
        return [node for node in
                super(ComputeNodeDriver, self).list_nodes()
                if node.extra["tags"].get("arvados-class") == self.tags["arvados-class"]]

    @classmethod
    def node_fqdn(cls, node):
        return node.extra["tags"].get("hostname")

    @classmethod
    def node_start_time(cls, node):
        return arvados_timestamp(node.extra["tags"].get("booted_at"))
