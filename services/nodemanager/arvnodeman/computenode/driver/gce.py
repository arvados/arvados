#!/usr/bin/env python

from __future__ import absolute_import, print_function

import functools
import json
import time

import libcloud.compute.base as cloud_base
import libcloud.compute.providers as cloud_provider
import libcloud.compute.types as cloud_types
from libcloud.compute.drivers import gce

from . import BaseComputeNodeDriver
from .. import arvados_node_fqdn

class ComputeNodeDriver(BaseComputeNodeDriver):
    """Compute node driver wrapper for GCE

    This translates cloud driver requests to GCE's specific parameters.
    """
    DEFAULT_DRIVER = cloud_provider.get_driver(cloud_types.Provider.GCE)
    SEARCH_CACHE = {}
    ssh_key = None
    service_accounts = None

    def __init__(self, auth_kwargs, list_kwargs, create_kwargs,
                 driver_class=DEFAULT_DRIVER):
        super(ComputeNodeDriver, self).__init__(
            auth_kwargs, list_kwargs, create_kwargs,
            driver_class)

        for key in self.create_kwargs.keys():
            init_method = getattr(self, '_init_' + key, None)
            if init_method is not None:
                new_pair = init_method(self.create_kwargs.pop(key))
                if new_pair is not None:
                    self.create_kwargs[new_pair[0]] = new_pair[1]

    def _init_image_id(self, image_id):
        return 'image', self.search_for(image_id, 'list_images')

    def _init_ping_host(self, ping_host):
        self.ping_host = ping_host

    def _init_service_accounts(self, service_accounts_str):
        self.service_accounts = json.loads(service_accounts_str)

    def _init_network_id(self, subnet_id):
        return 'ex_network', self.search_for(subnet_id, 'ex_list_networks')

    def _init_ssh_key(self, filename):
        with open(filename) as ssh_file:
            self.ssh_key = ssh_file.read().strip()

    def arvados_create_kwargs(self, arvados_node):
        result = {'ex_metadata': self.list_kwargs.copy() }
        ping_secret = arvados_node['info'].get('ping_secret')
        if ping_secret is not None:
            ping_url = ('https://{}/arvados/v1/nodes/{}/ping?ping_secret={}'.
                        format(self.ping_host, arvados_node['uuid'],
                               ping_secret))
            result['ex_userdata'] = ping_url
        if self.service_accounts is not None:
            result['ex_service_accounts'] = self.service_accounts

        # SSH keys are delivered to GCE nodes via ex_metadata: see
        # http://stackoverflow.com/questions/26752617/creating-sshkeys-for-gce-instance-using-libcloud
        if self.ssh_key is not None:
            result['ex_metadata']['sshKeys'] = 'root:{}'.format(self.ssh_key)
        return result

    # When an Arvados node is synced with a GCE node, the Arvados hostname
    # is forwarded in a GCE tag 'hostname-foo'.
    # TODO(twp): implement an ex_set_metadata method (at least until
    # libcloud supports the API setMetadata method) so we can pass this
    # sensibly in the node metadata.
    def sync_node(self, cloud_node, arvados_node):
        tags = ['hostname-{}'.format(arvados_node_fqdn(arvados_node))]
        self.real.ex_set_node_tags(cloud_node, tags)

    @classmethod
    def node_start_time(cls, node):
        time_str = node.extra['launch_time'].split('.', 2)[0] + 'UTC'
        return time.mktime(time.strptime(
                time_str,'%Y-%m-%dT%H:%M:%S%Z')) - time.timezone
