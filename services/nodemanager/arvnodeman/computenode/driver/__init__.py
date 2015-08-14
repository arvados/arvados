#!/usr/bin/env python

from __future__ import absolute_import, print_function

from operator import attrgetter

import libcloud.common.types as cloud_types
from libcloud.compute.base import NodeDriver

from ...config import NETWORK_ERRORS

import pprint

class BaseComputeNodeDriver(object):
    """Abstract base class for compute node drivers.

    libcloud drivers abstract away many of the differences between
    cloud providers, but managing compute nodes requires some
    cloud-specific features (e.g., keeping track of node FQDNs and
    boot times).  Compute node drivers are responsible for translating
    the node manager's cloud requests to a specific cloud's
    vocabulary.

    Subclasses must implement arvados_create_kwargs, sync_node,
    node_fqdn, and node_start_time.
    """
    CLOUD_ERRORS = NETWORK_ERRORS + (cloud_types.LibcloudError,)

    def __init__(self, auth_kwargs, list_kwargs, create_kwargs, driver_class):
        """Base initializer for compute node drivers.

        Arguments:
        * auth_kwargs: A dictionary of arguments that are passed into the
          driver_class constructor to instantiate a libcloud driver.
        * list_kwargs: A dictionary of arguments that are passed to the
          libcloud driver's list_nodes method to return the list of compute
          nodes.
        * create_kwargs: A dictionary of arguments that are passed to the
          libcloud driver's create_node method to create a new compute node.
        * driver_class: The class of a libcloud driver to use.
        """
        self.real = driver_class(**auth_kwargs)
        self.list_kwargs = list_kwargs
        self.create_kwargs = create_kwargs
        # Transform entries in create_kwargs.  For each key K, if this class
        # has an _init_K method, remove the entry and call _init_K with the
        # corresponding value.  If _init_K returns None, the entry stays out
        # of the dictionary (we expect we're holding the value somewhere
        # else, like an instance variable).  Otherwise, _init_K returns a
        # key-value tuple pair, and we add that entry to create_kwargs.
        for key in self.create_kwargs.keys():
            init_method = getattr(self, '_init_' + key, None)
            if init_method is not None:
                new_pair = init_method(self.create_kwargs.pop(key))
                if new_pair is not None:
                    self.create_kwargs[new_pair[0]] = new_pair[1]

    def _init_ping_host(self, ping_host):
        self.ping_host = ping_host

    def search_for(self, term, list_method, key=attrgetter('id'), **kwargs):
        """Return one matching item from a list of cloud objects.

        Raises ValueError if the number of matching objects is not exactly 1.

        Arguments:
        * term: The value that identifies a matching item.
        * list_method: A string that names the method to call on this
          instance's libcloud driver for a list of objects.
        * key: A function that accepts a cloud object and returns a
          value search for a `term` match on each item.  Returns the
          object's 'id' attribute by default.
        """
        cache_key = (list_method, term)
        if cache_key not in self.SEARCH_CACHE:
            items = getattr(self.real, list_method)(**kwargs)
            results = [item for item in items
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
        """Return dynamic keyword arguments for create_node.

        Subclasses must override this method.  It should return a dictionary
        of keyword arguments to pass to the libcloud driver's create_node
        method.  These arguments will extend the static arguments in
        create_kwargs.

        Arguments:
        * arvados_node: The Arvados node record that will be associated
          with this cloud node, as returned from the API server.
        """
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
        # This method should return the time the node was started, in
        # seconds since the epoch UTC.
        raise NotImplementedError("BaseComputeNodeDriver.node_start_time")

    @classmethod
    def is_cloud_exception(cls, exception):
        # libcloud compute drivers typically raise bare Exceptions to
        # represent API errors.  Return True for any exception that is
        # exactly an Exception, or a better-known higher-level exception.
        return (isinstance(exception, cls.CLOUD_ERRORS) or
                type(exception) is Exception)

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
