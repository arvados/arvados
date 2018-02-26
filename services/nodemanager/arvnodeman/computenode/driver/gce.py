#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import, print_function

import functools
import json
import time

import libcloud.compute.providers as cloud_provider
import libcloud.compute.types as cloud_types

from . import BaseComputeNodeDriver
from .. import arvados_node_fqdn, arvados_timestamp, ARVADOS_TIMEFMT

class ComputeNodeDriver(BaseComputeNodeDriver):
    """Compute node driver wrapper for GCE

    This translates cloud driver requests to GCE's specific parameters.
    """
    DEFAULT_DRIVER = cloud_provider.get_driver(cloud_types.Provider.GCE)
    SEARCH_CACHE = {}

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
        self._project = auth_kwargs.get("project")
        super(ComputeNodeDriver, self).__init__(
            auth_kwargs, list_kwargs, create_kwargs,
            driver_class)
        self._sizes_by_id = {sz.id: sz for sz in self.sizes.itervalues()}
        self._disktype_links = {dt.name: self._object_link(dt)
                                for dt in self.real.ex_list_disktypes()}

    @staticmethod
    def _object_link(cloud_object):
        return cloud_object.extra.get('selfLink')

    def _init_image(self, image_name):
        return 'image', self.search_for(
            image_name, 'list_images', self._name_key, ex_project=self._project)

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

    def create_cloud_name(self, arvados_node):
        uuid_parts = arvados_node['uuid'].split('-', 2)
        return 'compute-{parts[2]}-{parts[0]}'.format(parts=uuid_parts)

    def arvados_create_kwargs(self, size, arvados_node):
        name = self.create_cloud_name(arvados_node)

        if size.scratch > 375000:
            self._logger.warning("Requested %d MB scratch space, but GCE driver currently only supports attaching a single 375 GB disk.", size.scratch)

        disks = [
            {'autoDelete': True,
             'boot': True,
             'deviceName': name,
             'initializeParams':
                 {'diskName': name,
                  'diskType': self._disktype_links['pd-standard'],
                  'sourceImage': self._object_link(self.create_kwargs['image']),
                  },
             'type': 'PERSISTENT',
             },
            {'autoDelete': True,
             'boot': False,
             # Boot images rely on this device name to find the SSD.
             # Any change must be coordinated in the image.
             'deviceName': 'tmp',
             'initializeParams':
                 {'diskType': self._disktype_links['local-ssd'],
                  },
             'type': 'SCRATCH',
             },
            ]
        result = {'name': name,
                  'ex_metadata': self.create_kwargs['ex_metadata'].copy(),
                  'ex_tags': list(self.node_tags),
                  'ex_disks_gce_struct': disks,
                  }
        result['ex_metadata'].update({
                'arv-ping-url': self._make_ping_url(arvados_node),
                'booted_at': time.strftime(ARVADOS_TIMEFMT, time.gmtime()),
                'hostname': arvados_node_fqdn(arvados_node),
                })
        return result


    def list_nodes(self):
        # The GCE libcloud driver only supports filtering node lists by zone.
        # Do our own filtering based on tag list.
        nodelist = [node for node in
                    super(ComputeNodeDriver, self).list_nodes()
                    if self.node_tags.issubset(node.extra.get('tags', []))]
        # As of 0.18, the libcloud GCE driver sets node.size to the size's name.
        # It's supposed to be the actual size object.  Check that it's not,
        # and monkeypatch the results when that's the case.
        if nodelist and not hasattr(nodelist[0].size, 'id'):
            for node in nodelist:
                node.size = self._sizes_by_id[node.size]
        return nodelist

    @classmethod
    def _find_metadata(cls, metadata_items, key):
        # Given a list of two-item metadata dictonaries, return the one with
        # the named key.  Raise KeyError if not found.
        try:
            return next(data_dict for data_dict in metadata_items
                        if data_dict.get('key') == key)
        except StopIteration:
            raise KeyError(key)

    @classmethod
    def _get_metadata(cls, metadata_items, key, *default):
        try:
            return cls._find_metadata(metadata_items, key)['value']
        except KeyError:
            if default:
                return default[0]
            raise

    def sync_node(self, cloud_node, arvados_node):
        # Update the cloud node record to ensure we have the correct metadata
        # fingerprint.
        cloud_node = self.real.ex_get_node(cloud_node.name, cloud_node.extra['zone'])

        # We can't store the FQDN on the name attribute or anything like it,
        # because (a) names are static throughout the node's life (so FQDN
        # isn't available because we don't know it at node creation time) and
        # (b) it can't contain dots.  Instead stash it in metadata.
        hostname = arvados_node_fqdn(arvados_node)
        metadata_req = cloud_node.extra['metadata'].copy()
        metadata_items = metadata_req.setdefault('items', [])
        try:
            self._find_metadata(metadata_items, 'hostname')['value'] = hostname
        except KeyError:
            metadata_items.append({'key': 'hostname', 'value': hostname})

        self.real.ex_set_node_metadata(cloud_node, metadata_items)

    @classmethod
    def node_fqdn(cls, node):
        # See sync_node comment.
        return cls._get_metadata(node.extra['metadata'].get('items', []),
                                 'hostname', '')

    @classmethod
    def node_start_time(cls, node):
        try:
            return arvados_timestamp(cls._get_metadata(
                    node.extra['metadata']['items'], 'booted_at'))
        except KeyError:
            return 0

    @classmethod
    def node_id(cls, node):
        return node.id
