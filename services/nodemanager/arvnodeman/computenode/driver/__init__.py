#!/usr/bin/env python

from __future__ import absolute_import, print_function

import libcloud.common.types as cloud_types
from libcloud.compute.base import NodeDriver

from ...config import NETWORK_ERRORS

class BaseComputeNodeDriver(object):
    """Abstract base class for compute node drivers.

    libcloud abstracts away many of the differences between cloud providers,
    but managing compute nodes requires some cloud-specific features (e.g.,
    on EC2 we use tags to identify compute nodes).  Compute node drivers
    are responsible for translating the node manager's cloud requests to a
    specific cloud's vocabulary.

    Subclasses must implement arvados_create_kwargs (to update node
    creation kwargs with information about the specific Arvados node
    record), sync_node, and node_start_time.
    """
    CLOUD_ERRORS = NETWORK_ERRORS + (cloud_types.LibcloudError,)

    def __init__(self, auth_kwargs, list_kwargs, create_kwargs, driver_class):
        self.real = driver_class(**auth_kwargs)
        self.list_kwargs = list_kwargs
        self.create_kwargs = create_kwargs
        for key in self.create_kwargs.keys():
            init_method = getattr(self, '_init_' + key, None)
            if init_method is not None:
                new_pair = init_method(self.create_kwargs.pop(key))
                if new_pair is not None:
                    self.create_kwargs[new_pair[0]] = new_pair[1]

    def _init_ping_host(self, ping_host):
        self.ping_host = ping_host

    def search_for(self, term, list_method, key=lambda item: item.id):
        cache_key = (list_method, term)
        if cache_key not in self.SEARCH_CACHE:
            results = [item for item in getattr(self.real, list_method)()
                       if key(item) == term]
            count = len(results)
            if count != 1:
                raise ValueError("{} returned {} results for '{}'".format(
                        list_method, count, term))
            self.SEARCH_CACHE[cache_key] = results[0]
        return self.SEARCH_CACHE[cache_key]

    def list_nodes(self):
        return self.real.list_nodes(**self.list_kwargs)

    def arvados_create_kwargs(self, arvados_node):
        raise NotImplementedError("BaseComputeNodeDriver.arvados_create_kwargs")

    def _make_ping_url(self, arvados_node):
        return 'https://{}/arvados/v1/nodes/{}/ping?ping_secret={}'.format(
            self.ping_host, arvados_node['uuid'],
            arvados_node['info']['ping_secret'])

    def create_node(self, size, arvados_node):
        kwargs = self.create_kwargs.copy()
        kwargs.update(self.arvados_create_kwargs(arvados_node))
        kwargs['size'] = size
        return self.real.create_node(**kwargs)

    def post_create_node(self, cloud_node):
        # ComputeNodeSetupActor calls this method after the cloud node is
        # created.  Any setup tasks that need to happen afterward (e.g.,
        # tagging) should be done in this method.
        pass

    def sync_node(self, cloud_node, arvados_node):
        # When a compute node first pings the API server, the API server
        # will automatically assign some attributes on the corresponding
        # node record, like hostname.  This method should propagate that
        # information back to the cloud node appropriately.
        raise NotImplementedError("BaseComputeNodeDriver.sync_node")

    @classmethod
    def node_fqdn(cls, node):
        # This method should return the FQDN of the node object argument.
        # Different clouds store this in different places.
        raise NotImplementedError("BaseComputeNodeDriver.node_fqdn")

    @classmethod
    def node_start_time(cls, node):
        raise NotImplementedError("BaseComputeNodeDriver.node_start_time")

    @classmethod
    def is_cloud_exception(cls, exception):
        # libcloud compute drivers typically raise bare Exceptions to
        # represent API errors.  Return True for any exception that is
        # exactly an Exception, or a better-known higher-level exception.
        return (isinstance(exception, cls.CLOUD_ERRORS) or
                getattr(exception, '__class__', None) is Exception)

    # Now that we've defined all our own methods, delegate generic, public
    # attributes of libcloud drivers that we haven't defined ourselves.
    def _delegate_to_real(attr_name):
        return property(
            lambda self: getattr(self.real, attr_name),
            lambda self, value: setattr(self.real, attr_name, value),
            doc=getattr(getattr(NodeDriver, attr_name), '__doc__', None))

    _locals = locals()
    for _attr_name in dir(NodeDriver):
        if (not _attr_name.startswith('_')) and (_attr_name not in _locals):
            _locals[_attr_name] = _delegate_to_real(_attr_name)
