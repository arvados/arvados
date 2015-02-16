#!/usr/bin/env python

from __future__ import absolute_import, print_function

import functools
import json
import time

import libcloud.compute.providers as cloud_provider
import libcloud.compute.types as cloud_types

from . import BaseComputeNodeDriver
from .. import arvados_node_fqdn

class ComputeNodeDriver(BaseComputeNodeDriver):
    """Compute node driver wrapper for GCE

    This translates cloud driver requests to GCE's specific parameters.
    """
    DEFAULT_DRIVER = cloud_provider.get_driver(cloud_types.Provider.GCE)
    SEARCH_CACHE = {}
    node_start_times = {}

    def __init__(self, auth_kwargs, list_kwargs, create_kwargs,
                 driver_class=DEFAULT_DRIVER):
        list_kwargs = list_kwargs.copy()
        tags_str = list_kwargs.pop('tags', '')
        if not tags_str.strip():
            self.node_tags = frozenset()
        else:
            self.node_tags = frozenset(t.strip() for t in tags_str.split(','))
        create_kwargs = create_kwargs.copy()
        create_kwargs.setdefault('external_ip', None)
        create_kwargs.setdefault('ex_metadata', {})
        super(ComputeNodeDriver, self).__init__(
            auth_kwargs, list_kwargs, create_kwargs,
            driver_class)

    @staticmethod
    def _name_key(cloud_object):
        return cloud_object.name

    def _init_image(self, image_name):
        return 'image', self.search_for(
            image_name, 'list_images', self._name_key)

    def _init_location(self, location_name):
        return 'location', self.search_for(
            location_name, 'list_locations', self._name_key)

    def _init_network(self, network_name):
        return 'ex_network', self.search_for(
            network_name, 'ex_list_networks', self._name_key)

    def _init_service_accounts(self, service_accounts_str):
        return 'ex_service_accounts', json.loads(service_accounts_str)

    def _init_ssh_key(self, filename):
        # SSH keys are delivered to GCE nodes via ex_metadata: see
        # http://stackoverflow.com/questions/26752617/creating-sshkeys-for-gce-instance-using-libcloud
        with open(filename) as ssh_file:
            self.create_kwargs['ex_metadata']['sshKeys'] = (
                'root:' + ssh_file.read().strip())

    def list_sizes(self):
        return super(ComputeNodeDriver, self).list_sizes(
            self.create_kwargs['location'])

    def arvados_create_kwargs(self, arvados_node):
        cluster_id, _, node_id = arvados_node['uuid'].split('-')
        result = {'name': 'compute-{}-{}'.format(node_id, cluster_id),
                  'ex_metadata': self.create_kwargs['ex_metadata'].copy(),
                  'ex_tags': list(self.node_tags)}
        result['ex_metadata']['hostname'] = arvados_node_fqdn(arvados_node)
        result['ex_metadata']['user-data'] = self._make_ping_url(arvados_node)
        return result

    def list_nodes(self):
        # The GCE libcloud driver only supports filtering node lists by zone.
        # Do our own filtering based on tag list.
        return [node for node in
                super(ComputeNodeDriver, self).list_nodes()
                if self.node_tags.issubset(node.extra.get('tags', []))]

    def sync_node(self, cloud_node, arvados_node):
        hostname = arvados_node_fqdn(arvados_node)
        metadata_req = cloud_node.extra['metadata'].copy()
        for data_dict in metadata_req.setdefault('items', []):
            if data_dict['key'] == 'hostname':
                data_dict['value'] = hostname
                break
        else:
            metadata_req['items'].append({'key': 'hostname', 'value': hostname})
        response = self.real.connection.async_request(
            '/zones/{}/instances/{}/setMetadata'.format(
                cloud_node.extra['zone'].name, cloud_node.name),
            method='POST', data=metadata_req)
        if not response.success():
            raise Exception("setMetadata error: {}".format(response.error))

    def destroy_node(self, node):
        success = super(ComputeNodeDriver, self).destroy_node(node)
        if success:
            self.node_start_times.pop(node.id, None)
        return success

    @classmethod
    def node_start_time(cls, node):
        # Launch time isn't available on GCE node records.  Thankfully that's
        # not too big a deal because they have by-minute billing.
        # Fake an answer based on the first time we see it.
        return cls.node_start_times.setdefault(node.id, time.time())
