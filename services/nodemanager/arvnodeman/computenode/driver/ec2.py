#!/usr/bin/env python

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
        super(ComputeNodeDriver, self).__init__(
            auth_kwargs, {'ex_filters': list_kwargs}, create_kwargs,
            driver_class)

    def _init_image_id(self, image_id):
        return 'image', self.search_for(image_id, 'list_images')

    def _init_security_groups(self, group_names):
        return 'ex_security_groups', [
            self.search_for(gname.strip(), 'ex_get_security_groups')
            for gname in group_names.split(',')]

    def _init_subnet_id(self, subnet_id):
        return 'ex_subnet', self.search_for(subnet_id, 'ex_list_subnets')

    def _init_ssh_key(self, filename):
        with open(filename) as ssh_file:
            key = cloud_base.NodeAuthSSHKey(ssh_file.read())
        return 'auth', key

    def arvados_create_kwargs(self, arvados_node):
        return {'name': arvados_node_fqdn(arvados_node),
                'ex_userdata': self._make_ping_url(arvados_node)}

    def post_create_node(self, cloud_node):
        self.real.ex_create_tags(cloud_node, self.tags)

    def sync_node(self, cloud_node, arvados_node):
        self.real.ex_create_tags(cloud_node,
                                 {'Name': arvados_node_fqdn(arvados_node)})

    @classmethod
    def node_start_time(cls, node):
        time_str = node.extra['launch_time'].split('.', 2)[0] + 'UTC'
        return time.mktime(time.strptime(
                time_str,'%Y-%m-%dT%H:%M:%S%Z')) - time.timezone
