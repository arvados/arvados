#!/usr/bin/env python

from __future__ import absolute_import, print_function

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
    def __init__(self, auth_kwargs, list_kwargs, create_kwargs, driver_class):
        self.real = driver_class(**auth_kwargs)
        self.list_kwargs = list_kwargs
        self.create_kwargs = create_kwargs

    def __getattr__(self, name):
        # Proxy non-extension methods to the real driver.
        if (not name.startswith('_') and not name.startswith('ex_')
              and hasattr(self.real, name)):
            return getattr(self.real, name)
        else:
            return super(BaseComputeNodeDriver, self).__getattr__(name)

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

    def create_node(self, size, arvados_node):
        kwargs = self.create_kwargs.copy()
        kwargs.update(self.arvados_create_kwargs(arvados_node))
        kwargs['size'] = size
        return self.real.create_node(**kwargs)

    def sync_node(self, cloud_node, arvados_node):
        # When a compute node first pings the API server, the API server
        # will automatically assign some attributes on the corresponding
        # node record, like hostname.  This method should propagate that
        # information back to the cloud node appropriately.
        raise NotImplementedError("BaseComputeNodeDriver.sync_node")

    @classmethod
    def node_start_time(cls, node):
        raise NotImplementedError("BaseComputeNodeDriver.node_start_time")
