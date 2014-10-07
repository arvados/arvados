#!/usr/bin/env python

from __future__ import absolute_import, print_function

import functools
import logging
import time

import pykka

from . import computenode as cnode
from .config import actor_class

class NodeManagerDaemonActor(actor_class):
    """Node Manager daemon.

    This actor subscribes to all information polls about cloud nodes,
    Arvados nodes, and the job queue.  It creates a ComputeNodeActor
    for every cloud node, subscribing them to poll updates
    appropriately, and starts and stops cloud nodes based on job queue
    demand.
    """
    class PairingTracker(object):
        def __init__(self, key_func, paired_items, unpaired_items):
            self.key_func = key_func
            self._paired_items = paired_items
            self._unpaired_items = unpaired_items

        def all_items(self, response):
            self.unseen = set(self._paired_items.iterkeys())
            self.unseen.update(self._unpaired_items.iterkeys())
            for item in response:
                key = self.key_func(item)
                yield key, item
                if key in self.unseen:
                    self.unseen.remove(key)

        def new_items(self, response):
            for key, item in self.all_items(response):
                if key not in self.unseen:
                    yield key, item

        def unpaired_items(self, response):
            for key, item in self.all_items(response):
                if key not in self._paired_items:
                    yield key, item

        def unseen_items(self):
            for key in self.unseen:
                if key in self._paired_items:
                    home_dict = self._paired_items
                else:
                    home_dict = self._unpaired_items
                yield home_dict, key


    def __init__(self, server_wishlist_actor, arvados_nodes_actor,
                 cloud_nodes_actor, timer_actor,
                 arvados_factory, cloud_factory,
                 shutdown_windows, max_nodes,
                 poll_stale_after=600, node_stale_after=7200,
                 node_setup_class=cnode.ComputeNodeSetupActor,
                 node_shutdown_class=cnode.ComputeNodeShutdownActor,
                 node_actor_class=cnode.ComputeNodeActor):
        super(NodeManagerDaemonActor, self).__init__()
        self._node_setup = node_setup_class
        self._node_shutdown = node_shutdown_class
        self._node_actor = node_actor_class
        self._timer = timer_actor
        self._new_arvados = arvados_factory
        self._new_cloud = cloud_factory
        self._cloud_driver = self._new_cloud()
        self._logger = logging.getLogger('arvnodeman.daemon')
        self._later = self.actor_ref.proxy()
        self.shutdown_windows = shutdown_windows
        self.max_nodes = max_nodes
        self.poll_stale_after = poll_stale_after
        self.node_stale_after = node_stale_after
        self.last_polls = {}
        for poll_name in ['server_wishlist', 'arvados_nodes', 'cloud_nodes']:
            poll_actor = locals()[poll_name + '_actor']
            poll_actor.subscribe(getattr(self._later, 'update_' + poll_name))
            setattr(self, '_{}_actor'.format(poll_name), poll_actor)
            self.last_polls[poll_name] = -self.poll_stale_after
        # Map cloud node IDs, or Arvados node UUIDs, to their ComputeNodeActors.
        self.unpaired_clouds = {}
        self.paired_clouds = {}
        self.paired_arv = {}
        self.unpaired_arv = {}  # Arvados node UUIDs to full node data
        self.assigned_arv = {}  # Arvados node UUIDs to assignment timestamps
        self.booting = {}       # Actor IDs to ComputeNodeSetupActors
        self.shutdowns = {}     # Cloud node IDs to ComputeNodeShutdownActors
        self._logger.debug("Daemon initialized")

    def _update_poll_time(self, poll_key):
        self.last_polls[poll_key] = time.time()

    def _pair_nodes(self, cloud_key, arv_key, actor=None):
        if actor is None:
            actor = self.unpaired_clouds[cloud_key]
        self._logger.info("Cloud node %s has associated with Arvados node %s",
                          cloud_key, arv_key)
        self.paired_clouds[cloud_key] = actor
        self.paired_arv[arv_key] = actor
        self._arvados_nodes_actor.subscribe_to(arv_key,
                                               actor.update_arvados_node)
        self.unpaired_clouds.pop(cloud_key, None)
        self.unpaired_arv.pop(arv_key, None)

    def _new_node(self, cloud_node, arvados_node=None):
        start_time = self._cloud_driver.node_start_time(cloud_node)
        shutdown_timer = cnode.ShutdownTimer(start_time,
                                             self.shutdown_windows)
        actor = self._node_actor.start(
            cloud_node=cloud_node,
            cloud_node_start_time=start_time,
            shutdown_timer=shutdown_timer,
            timer_actor=self._timer,
            arvados_node=arvados_node,
            poll_stale_after=self.poll_stale_after,
            node_stale_after=self.node_stale_after).proxy()
        actor.subscribe(self._later.shutdown_offer)
        self._cloud_nodes_actor.subscribe_to(cloud_node.id,
                                             actor.update_cloud_node)
        if arvados_node is not None:
            self._pair_nodes(cloud_node.id, arvados_node['uuid'], actor)
        return actor

    def update_cloud_nodes(self, nodelist):
        self._update_poll_time('cloud_nodes')
        pairs = self.PairingTracker(lambda n: n.id,
                                    self.paired_clouds, self.unpaired_clouds)
        for key, node in pairs.new_items(nodelist):
            actor = self._new_node(node)
            for arv_key, arv_node in self.unpaired_arv.iteritems():
                if actor.offer_arvados_pair(arv_node).get():
                    self._pair_nodes(key, arv_key, actor)
                    break
            else:
                self._logger.info("Registering new cloud node %s", key)
                self.unpaired_clouds[key] = actor
        for source, key in pairs.unseen_items():
            source.pop(key).stop()
            if key in self.shutdowns:
                self.shutdowns.pop(key).stop()

    def update_arvados_nodes(self, nodelist):
        self._update_poll_time('arvados_nodes')
        pairs = self.PairingTracker(lambda n: n['uuid'],
                                    self.paired_arv, self.unpaired_arv)
        for key, node in pairs.unpaired_items(nodelist):
            if key not in self.unpaired_arv:
                self._logger.info("Registering new Arvados node %s", key)
            self.unpaired_arv[key] = node
            for cloud_key, actor in self.unpaired_clouds.iteritems():
                if actor.offer_arvados_pair(node).get():
                    self._pair_nodes(cloud_key, key, actor)
                    break
        for source, key in pairs.unseen_items():
            if source is self.unpaired_arv:
                del self.unpaired_arv[key]

    def _node_count(self):
        up = sum(len(nodelist) for nodelist in
                 [self.paired_clouds, self.unpaired_clouds, self.booting])
        return up - len(self.shutdowns)

    def _nodes_wanted(self):
        return len(self.last_wishlist) - self._node_count()

    def _nodes_excess(self):
        return -self._nodes_wanted()

    def update_server_wishlist(self, wishlist):
        self._update_poll_time('server_wishlist')
        self.last_wishlist = wishlist[:self.max_nodes]
        nodes_wanted = self._nodes_wanted()
        if nodes_wanted > 0:
            self._later.start_node()
        elif (nodes_wanted < 0) and self.booting:
            self._later.stop_booting_node()

    def _check_poll_freshness(orig_func):
        """Decorator to inhibit a method when poll information is stale.

        This decorator checks the timestamps of all the poll information the
        daemon has received.  The decorated method is only called if none
        of the timestamps are considered stale.
        """
        @functools.wraps(orig_func)
        def wrapper(self, *args, **kwargs):
            now = time.time()
            if all(now - t < self.poll_stale_after
                   for t in self.last_polls.itervalues()):
                return orig_func(self, *args, **kwargs)
            else:
                return None
        return wrapper

    def _find_reusable_arvados_node(self):
        for node in self.unpaired_arv.itervalues():
            assigned_at = self.assigned_arv.get(node['uuid'],
                                                -self.node_stale_after)
            if (not cnode.timestamp_fresh(cnode.arvados_node_mtime(node),
                                          self.node_stale_after) and
                not cnode.timestamp_fresh(assigned_at,
                                          self.node_stale_after)):
                return node
        return None

    @_check_poll_freshness
    def start_node(self):
        nodes_wanted = self._nodes_wanted()
        if nodes_wanted < 1:
            return None
        arvados_node = self._find_reusable_arvados_node()
        size = self.last_wishlist[nodes_wanted - 1]
        self._logger.info("Want %s more nodes.  Booting a %s node.",
                          nodes_wanted, size.name)
        new_setup = self._node_setup.start(
            timer_actor=self._timer,
            arvados_client=self._new_arvados(),
            arvados_node=arvados_node,
            cloud_client=self._new_cloud(),
            cloud_size=size).proxy()
        self.booting[new_setup.actor_ref.actor_urn] = new_setup
        if arvados_node is not None:
            self.assigned_arv[arvados_node['uuid']] = time.time()
        new_setup.subscribe(self._later.node_up)
        if nodes_wanted > 1:
            self._later.start_node()

    def _actor_nodes(self, node_actor):
        return pykka.get_all([node_actor.cloud_node, node_actor.arvados_node])

    def node_up(self, setup_proxy):
        cloud_node, arvados_node = self._actor_nodes(setup_proxy)
        self._new_node(cloud_node, arvados_node)
        del self.booting[setup_proxy.actor_ref.actor_urn]
        self.assigned_arv.pop(arvados_node['uuid'], None)
        setup_proxy.stop()

    @_check_poll_freshness
    def stop_booting_node(self):
        nodes_excess = self._nodes_excess()
        if (nodes_excess < 1) or not self.booting:
            return None
        for key, node in self.booting.iteritems():
            node.stop_if_no_cloud_node().get()
            if not node.actor_ref.is_alive():
                del self.booting[key]
                if nodes_excess > 1:
                    self._later.stop_booting_node()
                break

    @_check_poll_freshness
    def shutdown_offer(self, node_actor):
        if self._nodes_excess() < 1:
            return None
        cloud_node, arvados_node = self._actor_nodes(node_actor)
        if cloud_node.id in self.shutdowns:
            return None
        shutdown = self._node_shutdown.start(timer_actor=self._timer,
                                             cloud_client=self._new_cloud(),
                                             cloud_node=cloud_node).proxy()
        self.shutdowns[cloud_node.id] = shutdown

    def shutdown(self):
        self._logger.info("Shutting down after signal.")
        self.poll_stale_after = -1  # Inhibit starting/stopping nodes
        for bootnode in self.booting.itervalues():
            bootnode.stop_if_no_cloud_node()
        self._later.await_shutdown()

    def await_shutdown(self):
        if any(node.actor_ref.is_alive() for node in self.booting.itervalues()):
            self._timer.schedule(time.time() + 1, self._later.await_shutdown)
        else:
            self.stop()
