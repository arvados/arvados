#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import, print_function

import logging
from operator import attrgetter

import libcloud.common.types as cloud_types
from libcloud.compute.base import NodeDriver, NodeAuthSSHKey

from ...config import CLOUD_ERRORS
from .. import RetryMixin

class BaseComputeNodeDriver(RetryMixin):
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


    @RetryMixin._retry()
    def _create_driver(self, driver_class, **auth_kwargs):
        return driver_class(**auth_kwargs)

    @RetryMixin._retry()
    def _set_sizes(self):
        self.sizes = {sz.id: sz for sz in self.real.list_sizes()}

    def __init__(self, auth_kwargs, list_kwargs, create_kwargs,
                 driver_class, retry_wait=1, max_retry_wait=180):
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

        super(BaseComputeNodeDriver, self).__init__(retry_wait, max_retry_wait,
                                         logging.getLogger(self.__class__.__name__),
                                         type(self),
                                         None)
        self.real = self._create_driver(driver_class, **auth_kwargs)
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

        self._set_sizes()

    def _init_ping_host(self, ping_host):
        self.ping_host = ping_host

    def _init_ssh_key(self, filename):
        with open(filename) as ssh_file:
            key = NodeAuthSSHKey(ssh_file.read())
        return 'auth', key

    def search_for_now(self, term, list_method, key=attrgetter('id'), **kwargs):
        """Return one matching item from a list of cloud objects.

        Raises ValueError if the number of matching objects is not exactly 1.

        Arguments:
        * term: The value that identifies a matching item.
        * list_method: A string that names the method to call for a
          list of objects.
        * key: A function that accepts a cloud object and returns a
          value search for a `term` match on each item.  Returns the
          object's 'id' attribute by default.
        """
        try:
            list_func = getattr(self, list_method)
        except AttributeError:
            list_func = getattr(self.real, list_method)
        items = list_func(**kwargs)
        results = [item for item in items if key(item) == term]
        count = len(results)
        if count != 1:
            raise ValueError("{} returned {} results for {!r}".format(
                    list_method, count, term))
        return results[0]

    def search_for(self, term, list_method, key=attrgetter('id'), **kwargs):
        """Return one cached matching item from a list of cloud objects.

        See search_for_now() for details of arguments and exceptions.
        This method caches results, so it's good to find static cloud objects
        like node sizes, regions, etc.
        """
        cache_key = (list_method, term)
        if cache_key not in self.SEARCH_CACHE:
            self.SEARCH_CACHE[cache_key] = self.search_for_now(
                term, list_method, key, **kwargs)
        return self.SEARCH_CACHE[cache_key]

    def list_nodes(self, **kwargs):
        l = self.list_kwargs.copy()
        l.update(kwargs)
        return self.real.list_nodes(**l)

    def create_cloud_name(self, arvados_node):
        """Return a cloud node name for the given Arvados node record.

        Subclasses must override this method.  It should return a string
        that can be used as the name for a newly-created cloud node,
        based on identifying information in the Arvados node record.

        Arguments:
        * arvados_node: This Arvados node record to seed the new cloud node.
        """
        raise NotImplementedError("BaseComputeNodeDriver.create_cloud_name")

    def arvados_create_kwargs(self, size, arvados_node):
        """Return dynamic keyword arguments for create_node.

        Subclasses must override this method.  It should return a dictionary
        of keyword arguments to pass to the libcloud driver's create_node
        method.  These arguments will extend the static arguments in
        create_kwargs.

        Arguments:
        * size: The node size that will be created (libcloud NodeSize object)
        * arvados_node: The Arvados node record that will be associated
          with this cloud node, as returned from the API server.
        """
        raise NotImplementedError("BaseComputeNodeDriver.arvados_create_kwargs")

    def broken(self, cloud_node):
        """Return true if libcloud has indicated the node is in a "broken" state."""
        return False

    def _make_ping_url(self, arvados_node):
        return 'https://{}/arvados/v1/nodes/{}/ping?ping_secret={}'.format(
            self.ping_host, arvados_node['uuid'],
            arvados_node['info']['ping_secret'])

    @staticmethod
    def _name_key(cloud_object):
        return cloud_object.name

    def create_node(self, size, arvados_node):
        try:
            kwargs = self.create_kwargs.copy()
            kwargs.update(self.arvados_create_kwargs(size, arvados_node))
            kwargs['size'] = size
            return self.real.create_node(**kwargs)
        except CLOUD_ERRORS as create_error:
            # Workaround for bug #6702: sometimes the create node request
            # succeeds but times out and raises an exception instead of
            # returning a result.  If this happens, we get stuck in a retry
            # loop forever because subsequent create_node attempts will fail
            # due to node name collision.  So check if the node we intended to
            # create shows up in the cloud node list and return it if found.
            try:
                return self.search_for_now(kwargs['name'], 'list_nodes', self._name_key)
            except ValueError:
                raise create_error

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

    def destroy_node(self, cloud_node):
        try:
            return self.real.destroy_node(cloud_node)
        except CLOUD_ERRORS as destroy_error:
            # Sometimes the destroy node request succeeds but times out and
            # raises an exception instead of returning success.  If this
            # happens, we get a noisy stack trace.  Check if the node is still
            # on the node list.  If it is gone, we can declare victory.
            try:
                self.search_for_now(cloud_node.id, 'list_nodes')
            except ValueError:
                # If we catch ValueError, that means search_for_now didn't find
                # it, which means destroy_node actually succeeded.
                return True
            # The node is still on the list.  Re-raise.
            raise

    # Now that we've defined all our own methods, delegate generic, public
    # attributes of libcloud drivers that we haven't defined ourselves.
    def _delegate_to_real(attr_name):
        return property(
            lambda self: getattr(self.real, attr_name),
            lambda self, value: setattr(self.real, attr_name, value),
            doc=getattr(getattr(NodeDriver, attr_name), '__doc__', None))

    # node id
    @classmethod
    def node_id(cls):
        raise NotImplementedError("BaseComputeNodeDriver.node_id")

    _locals = locals()
    for _attr_name in dir(NodeDriver):
        if (not _attr_name.startswith('_')) and (_attr_name not in _locals):
            _locals[_attr_name] = _delegate_to_real(_attr_name)
