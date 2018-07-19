#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import, print_function

import time

import libcloud.compute.base as cloud_base
import libcloud.compute.providers as cloud_provider
import libcloud.compute.types as cloud_types
from libcloud.compute.drivers import ec2 as cloud_ec2

from . import BaseComputeNodeDriver
from .. import arvados_node_fqdn

### Monkeypatch libcloud to support AWS' new SecurityGroup API.
# These classes can be removed when libcloud support specifying
# security groups with the SecurityGroupId parameter.
class ANMEC2Connection(cloud_ec2.EC2Connection):
    def request(self, *args, **kwargs):
        params = kwargs.get('params')
        if (params is not None) and (params.get('Action') == 'RunInstances'):
            for key in params.keys():
                if key.startswith('SecurityGroup.'):
                    new_key = key.replace('Group.', 'GroupId.', 1)
                    params[new_key] = params.pop(key).id
            kwargs['params'] = params
        return super(ANMEC2Connection, self).request(*args, **kwargs)


class ANMEC2NodeDriver(cloud_ec2.EC2NodeDriver):
    connectionCls = ANMEC2Connection


class ComputeNodeDriver(BaseComputeNodeDriver):
    """Compute node driver wrapper for EC2.

    This translates cloud driver requests to EC2's specific parameters.
    """
    DEFAULT_DRIVER = ANMEC2NodeDriver
### End monkeypatch
    SEARCH_CACHE = {}

    def __init__(self, auth_kwargs, list_kwargs, create_kwargs,
                 driver_class=DEFAULT_DRIVER):
        # We need full lists of keys up front because these loops modify
        # dictionaries in-place.
        for key in list_kwargs.keys():
            list_kwargs[key.replace('_', ':')] = list_kwargs.pop(key)
        self.tags = {key[4:]: value
                     for key, value in list_kwargs.iteritems()
                     if key.startswith('tag:')}
        # Tags are assigned at instance creation time
        create_kwargs.setdefault('ex_metadata', {})
        create_kwargs['ex_metadata'].update(self.tags)
        super(ComputeNodeDriver, self).__init__(
            auth_kwargs, {'ex_filters': list_kwargs}, create_kwargs,
            driver_class)

    def _init_image_id(self, image_id):
        return 'image', self.search_for(image_id, 'list_images', ex_owner='self')

    def _init_security_groups(self, group_names):
        return 'ex_security_groups', [
            self.search_for(gname.strip(), 'ex_get_security_groups')
            for gname in group_names.split(',')]

    def _init_subnet_id(self, subnet_id):
        return 'ex_subnet', self.search_for(subnet_id, 'ex_list_subnets')

    create_cloud_name = staticmethod(arvados_node_fqdn)

    def arvados_create_kwargs(self, size, arvados_node):
        kw = {'name': self.create_cloud_name(arvados_node),
                'ex_userdata': self._make_ping_url(arvados_node)}
        # libcloud/ec2 disk sizes are in GB, Arvados/SLURM "scratch" value is in MB
        scratch = int(size.scratch / 1000) + 1
        if scratch > size.disk:
            volsize = scratch - size.disk
            if volsize > 16384:
                # Must be 1-16384 for General Purpose SSD (gp2) devices
                # https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_EbsBlockDevice.html
                self._logger.warning("Requested EBS volume size %d is too large, capping size request to 16384 GB", volsize)
                volsize = 16384
            kw["ex_blockdevicemappings"] = [{
                "DeviceName": "/dev/xvdt",
                "Ebs": {
                    "DeleteOnTermination": True,
                    "VolumeSize": volsize,
                    "VolumeType": "gp2"
                }}]
        if size.preemptible:
            # Request a Spot instance for this node
            kw['ex_spot_market'] = True
        return kw

    def sync_node(self, cloud_node, arvados_node):
        self.real.ex_create_tags(cloud_node,
                                 {'Name': arvados_node_fqdn(arvados_node)})

    def create_node(self, size, arvados_node):
        # Set up tag indicating the Arvados assigned Cloud Size id.
        self.create_kwargs['ex_metadata'].update({'arvados_node_size': size.id})
        return super(ComputeNodeDriver, self).create_node(size, arvados_node)

    def list_nodes(self):
        # Need to populate Node.size
        nodes = super(ComputeNodeDriver, self).list_nodes()
        for n in nodes:
            if not n.size:
                n.size = self.sizes()[n.extra["instance_type"]]
            n.extra['arvados_node_size'] = n.extra.get('tags', {}).get('arvados_node_size') or n.size.id
        return nodes

    @classmethod
    def node_fqdn(cls, node):
        return node.name

    @classmethod
    def node_start_time(cls, node):
        time_str = node.extra['launch_time'].split('.', 2)[0] + 'UTC'
        return time.mktime(time.strptime(
                time_str,'%Y-%m-%dT%H:%M:%S%Z')) - time.timezone

    @classmethod
    def node_id(cls, node):
        return node.id
