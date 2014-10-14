#!/usr/bin/env python

from __future__ import absolute_import, print_function

import functools
import logging
import time

import pykka

from . import computenode as cnode
from .config import actor_class

class _ComputeNodeRecord(object):
    def __init__(self, actor=None, cloud_node=None, arvados_node=None,
                 assignment_time=float('-inf')):
        self.actor = actor
        self.cloud_node = cloud_node
        self.arvados_node = arvados_node
        self.assignment_time = assignment_time


class _BaseNodeTracker(object):
    def __init__(self):
        self.nodes = {}
        self.orphans = {}

    def __getitem__(self, key):
        return self.nodes[key]

    def __len__(self):
        return len(self.nodes)

    def get(self, key, default=None):
        return self.nodes.get(key, default)

    def record_key(self, record):
        return self.item_key(getattr(record, self.RECORD_ATTR))

    def add(self, record):
        self.nodes[self.record_key(record)] = record

    def update_record(self, key, item):
        setattr(self.nodes[key], self.RECORD_ATTR, item)

    def update_from(self, response):
        unseen = set(self.nodes.iterkeys())
        for item in response:
            key = self.item_key(item)
            if key in unseen:
                unseen.remove(key)
                self.update_record(key, item)
            else:
                yield key, item
        self.orphans = {key: self.nodes.pop(key) for key in unseen}

    def unpaired(self):
        return (record for record in self.nodes.itervalues()
                if getattr(record, self.PAIR_ATTR) is None)


class _CloudNodeTracker(_BaseNodeTracker):
    RECORD_ATTR = 'cloud_node'
    PAIR_ATTR = 'arvados_node'
    item_key = staticmethod(lambda cloud_node: cloud_node.id)


class _ArvadosNodeTracker(_BaseNodeTracker):
    RECORD_ATTR = 'arvados_node'
    PAIR_ATTR = 'cloud_node'
    item_key = staticmethod(lambda arvados_node: arvados_node['uuid'])

    def find_stale_node(self, stale_time):
        for record in self.nodes.itervalues():
            node = record.arvados_node
            if (not cnode.timestamp_fresh(cnode.arvados_node_mtime(node),
                                          stale_time) and
                  not cnode.timestamp_fresh(record.assignment_time,
                                            stale_time)):
                return node
        return None


class NodeManagerDaemonActor(actor_class):
    """Node Manager daemon.

    This actor subscribes to all information polls about cloud nodes,
    Arvados nodes, and the job queue.  It creates a ComputeNodeMonitorActor
    for every cloud node, subscribing them to poll updates
    appropriately.  It creates and destroys cloud nodes based on job queue
    demand, and stops the corresponding ComputeNode actors when their work
    is done.
    """
    def __init__(self, server_wishlist_actor, arvados_nodes_actor,
                 cloud_nodes_actor, cloud_update_actor, timer_actor,
                 arvados_factory, cloud_factory,
                 shutdown_windows, max_nodes,
                 poll_stale_after=600, node_stale_after=7200,
                 node_setup_class=cnode.ComputeNodeSetupActor,
                 node_shutdown_class=cnode.ComputeNodeShutdownActor,
                 node_actor_class=cnode.ComputeNodeMonitorActor):
        super(NodeManagerDaemonActor, self).__init__()
        self._node_setup = node_setup_class
        self._node_shutdown = node_shutdown_class
        self._node_actor = node_actor_class
        self._cloud_updater = cloud_update_actor
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
        self.cloud_nodes = _CloudNodeTracker()
        self.arvados_nodes = _ArvadosNodeTracker()
        self.booting = {}       # Actor IDs to ComputeNodeSetupActors
        self.shutdowns = {}     # Cloud node IDs to ComputeNodeShutdownActors
        self._logger.debug("Daemon initialized")

    def _update_poll_time(self, poll_key):
        self.last_polls[poll_key] = time.time()

    def _pair_nodes(self, node_record, arvados_node):
        self._logger.info("Cloud node %s has associated with Arvados node %s",
                          node_record.cloud_node.id, arvados_node['uuid'])
        self._arvados_nodes_actor.subscribe_to(
            arvados_node['uuid'], node_record.actor.update_arvados_node)
        node_record.arvados_node = arvados_node
        self.arvados_nodes.add(node_record)

    def _new_node(self, cloud_node):
        start_time = self._cloud_driver.node_start_time(cloud_node)
        shutdown_timer = cnode.ShutdownTimer(start_time,
                                             self.shutdown_windows)
        actor = self._node_actor.start(
            cloud_node=cloud_node,
            cloud_node_start_time=start_time,
            shutdown_timer=shutdown_timer,
            update_actor=self._cloud_updater,
            timer_actor=self._timer,
            arvados_node=None,
            poll_stale_after=self.poll_stale_after,
            node_stale_after=self.node_stale_after).proxy()
        actor.subscribe(self._later.node_can_shutdown)
        self._cloud_nodes_actor.subscribe_to(cloud_node.id,
                                             actor.update_cloud_node)
        record = _ComputeNodeRecord(actor, cloud_node)
        self.cloud_nodes.add(record)
        return record

    def update_cloud_nodes(self, nodelist):
        self._update_poll_time('cloud_nodes')
        for key, node in self.cloud_nodes.update_from(nodelist):
            self._logger.info("Registering new cloud node %s", key)
            record = self._new_node(node)
            for arv_rec in self.arvados_nodes.unpaired():
                if record.actor.offer_arvados_pair(arv_rec.arvados_node).get():
                    self._pair_nodes(record, arv_rec.arvados_node)
                    break
        for key, record in self.cloud_nodes.orphans.iteritems():
            record.actor.stop()
            if key in self.shutdowns:
                self.shutdowns.pop(key).stop()

    def update_arvados_nodes(self, nodelist):
        self._update_poll_time('arvados_nodes')
        for key, node in self.arvados_nodes.update_from(nodelist):
            self._logger.info("Registering new Arvados node %s", key)
            record = _ComputeNodeRecord(arvados_node=node)
            self.arvados_nodes.add(record)
        for arv_rec in self.arvados_nodes.unpaired():
            arv_node = arv_rec.arvados_node
            for cloud_rec in self.cloud_nodes.unpaired():
                if cloud_rec.actor.offer_arvados_pair(arv_node).get():
                    self._pair_nodes(cloud_rec, arv_node)
                    break

    def _node_count(self):
        up = sum(len(nodelist) for nodelist in [self.cloud_nodes, self.booting])
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

    @_check_poll_freshness
    def start_node(self):
        nodes_wanted = self._nodes_wanted()
        if nodes_wanted < 1:
            return None
        arvados_node = self.arvados_nodes.find_stale_node(self.node_stale_after)
        cloud_size = self.last_wishlist[nodes_wanted - 1]
        self._logger.info("Want %s more nodes.  Booting a %s node.",
                          nodes_wanted, cloud_size.name)
        new_setup = self._node_setup.start(
            timer_actor=self._timer,
            arvados_client=self._new_arvados(),
            arvados_node=arvados_node,
            cloud_client=self._new_cloud(),
            cloud_size=cloud_size).proxy()
        self.booting[new_setup.actor_ref.actor_urn] = new_setup
        if arvados_node is not None:
            self.arvados_nodes[arvados_node['uuid']].assignment_time = (
                time.time())
        new_setup.subscribe(self._later.node_up)
        if nodes_wanted > 1:
            self._later.start_node()

    def _actor_nodes(self, node_actor):
        return pykka.get_all([node_actor.cloud_node, node_actor.arvados_node])

    def node_up(self, setup_proxy):
        cloud_node, arvados_node = self._actor_nodes(setup_proxy)
        del self.booting[setup_proxy.actor_ref.actor_urn]
        setup_proxy.stop()
        record = self.cloud_nodes.get(cloud_node.id)
        if record is None:
            record = self._new_node(cloud_node)
        self._pair_nodes(record, arvados_node)

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
    def node_can_shutdown(self, node_actor):
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
