#!/usr/bin/env python

from __future__ import absolute_import, print_function

import functools
import logging
import time

import pykka

from . import computenode as cnode
from .computenode import dispatch
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
        self._blacklist = set()

    # Proxy the methods listed below to self.nodes.
    def _proxy_method(name):
        method = getattr(dict, name)
        @functools.wraps(method, ('__name__', '__doc__'))
        def wrapper(self, *args, **kwargs):
            return method(self.nodes, *args, **kwargs)
        return wrapper

    for _method_name in ['__contains__', '__getitem__', '__len__', 'get']:
        locals()[_method_name] = _proxy_method(_method_name)

    def record_key(self, record):
        return self.item_key(getattr(record, self.RECORD_ATTR))

    def add(self, record):
        self.nodes[self.record_key(record)] = record

    def blacklist(self, key):
        self._blacklist.add(key)

    def update_record(self, key, item):
        setattr(self.nodes[key], self.RECORD_ATTR, item)

    def update_from(self, response):
        unseen = set(self.nodes.iterkeys())
        for item in response:
            key = self.item_key(item)
            if key in self._blacklist:
                continue
            elif key in unseen:
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
                 shutdown_windows, server_calculator,
                 min_nodes, max_nodes,
                 poll_stale_after=600,
                 boot_fail_after=1800,
                 node_stale_after=7200,
                 node_setup_class=dispatch.ComputeNodeSetupActor,
                 node_shutdown_class=dispatch.ComputeNodeShutdownActor,
                 node_actor_class=dispatch.ComputeNodeMonitorActor,
                 max_total_price=0):
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
        self.server_calculator = server_calculator
        self.min_cloud_size = self.server_calculator.cheapest_size()
        self.min_nodes = min_nodes
        self.max_nodes = max_nodes
        self.max_total_price = max_total_price
        self.poll_stale_after = poll_stale_after
        self.boot_fail_after = boot_fail_after
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
        self.booted = {}        # Cloud node IDs to _ComputeNodeRecords
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
            cloud_fqdn_func=self._cloud_driver.node_fqdn,
            update_actor=self._cloud_updater,
            timer_actor=self._timer,
            arvados_node=None,
            poll_stale_after=self.poll_stale_after,
            node_stale_after=self.node_stale_after,
            cloud_client=self._cloud_driver,
            boot_fail_after=self.boot_fail_after).proxy()
        actor.subscribe(self._later.node_can_shutdown)
        self._cloud_nodes_actor.subscribe_to(cloud_node.id,
                                             actor.update_cloud_node)
        record = _ComputeNodeRecord(actor, cloud_node)
        return record

    def update_cloud_nodes(self, nodelist):
        self._update_poll_time('cloud_nodes')
        for key, node in self.cloud_nodes.update_from(nodelist):
            self._logger.info("Registering new cloud node %s", key)
            if key in self.booted:
                record = self.booted.pop(key)
            else:
                record = self._new_node(node)
            self.cloud_nodes.add(record)
            for arv_rec in self.arvados_nodes.unpaired():
                if record.actor.offer_arvados_pair(arv_rec.arvados_node).get():
                    self._pair_nodes(record, arv_rec.arvados_node)
                    break
        for key, record in self.cloud_nodes.orphans.iteritems():
            if key in self.shutdowns:
                try:
                    self.shutdowns[key].stop().get()
                except pykka.ActorDeadError:
                    pass
                del self.shutdowns[key]
            record.actor.stop()
            record.cloud_node = None

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

    def _nodes_up(self, size):
        up = 0
        up += sum(1
                  for c in self.booting.itervalues()
                  if size is None or c.cloud_size.get().id == size.id)
        up += sum(1
                  for i in (self.booted, self.cloud_nodes.nodes)
                  for c in i.itervalues()
                  if size is None or c.cloud_node.size.id == size.id)
        return up

    def _total_price(self):
        cost = 0
        cost += sum(self.server_calculator.find_size(c.cloud_size.get().id).price
                  for c in self.booting.itervalues())
        cost += sum(self.server_calculator.find_size(c.cloud_node.size.id).price
                    for i in (self.booted, self.cloud_nodes.nodes)
                    for c in i.itervalues())
        return cost

    def _nodes_busy(self, size):
        return sum(1 for busy in
                   pykka.get_all(rec.actor.in_state('busy') for rec in
                                 self.cloud_nodes.nodes.itervalues()
                                 if rec.cloud_node.size.id == size.id)
                   if busy)

    def _nodes_missing(self, size):
        return sum(1 for arv_node in
                   pykka.get_all(rec.actor.arvados_node for rec in
                                 self.cloud_nodes.nodes.itervalues()
                                 if rec.cloud_node.size.id == size.id and rec.actor.cloud_node.get().id not in self.shutdowns)
                   if arv_node and cnode.arvados_node_missing(arv_node, self.node_stale_after))

    def _size_wishlist(self, size):
        return sum(1 for c in self.last_wishlist if c.id == size.id)

    def _size_shutdowns(self, size):
        sh = 0
        for c in self.shutdowns.itervalues():
            try:
                if c.cloud_node.get().size.id == size.id:
                    sh += 1
            except pykka.ActorDeadError:
                pass
        return sh

    def _nodes_wanted(self, size):
        total_up_count = self._nodes_up(None)
        under_min = self.min_nodes - total_up_count
        over_max = total_up_count - self.max_nodes
        total_price = self._total_price()

        if over_max >= 0:
            return -over_max
        elif under_min > 0 and size.id == self.min_cloud_size.id:
            return under_min

        up_count = self._nodes_up(size) - (self._size_shutdowns(size) +
                                           self._nodes_busy(size) +
                                           self._nodes_missing(size))

        self._logger.debug("%s: idle nodes %i, wishlist size %i", size.name, up_count, self._size_wishlist(size))

        wanted = self._size_wishlist(size) - up_count
        if wanted > 0 and self.max_total_price and ((total_price + (size.price*wanted)) > self.max_total_price):
            can_boot = int((self.max_total_price - total_price) / size.price)
            if can_boot == 0:
                self._logger.info("Not booting %s (price %s) because with it would exceed max_total_price of %s (current total_price is %s)",
                                  size.name, size.price, self.max_total_price, total_price)
            return can_boot
        else:
            return wanted

    def _nodes_excess(self, size):
        up_count = self._nodes_up(size) - self._size_shutdowns(size)
        if size.id == self.min_cloud_size.id:
            up_count -= self.min_nodes
        return up_count - self._nodes_busy(size) - self._size_wishlist(size)

    def update_server_wishlist(self, wishlist):
        self._update_poll_time('server_wishlist')
        self.last_wishlist = wishlist
        for size in reversed(self.server_calculator.cloud_sizes):
            nodes_wanted = self._nodes_wanted(size)
            if nodes_wanted > 0:
                self._later.start_node(size)
            elif (nodes_wanted < 0) and self.booting:
                self._later.stop_booting_node(size)

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
    def start_node(self, cloud_size):
        nodes_wanted = self._nodes_wanted(cloud_size)
        if nodes_wanted < 1:
            return None
        arvados_node = self.arvados_nodes.find_stale_node(self.node_stale_after)
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
            self._later.start_node(cloud_size)

    def _get_actor_attrs(self, actor, *attr_names):
        return pykka.get_all([getattr(actor, name) for name in attr_names])

    def node_up(self, setup_proxy):
        cloud_node = setup_proxy.cloud_node.get()
        del self.booting[setup_proxy.actor_ref.actor_urn]
        setup_proxy.stop()
        record = self.cloud_nodes.get(cloud_node.id)
        if record is None:
            record = self._new_node(cloud_node)
            self.booted[cloud_node.id] = record
        self._timer.schedule(time.time() + self.boot_fail_after,
                             self._later.shutdown_unpaired_node, cloud_node.id)

    @_check_poll_freshness
    def stop_booting_node(self, size):
        nodes_excess = self._nodes_excess(size)
        if (nodes_excess < 1) or not self.booting:
            return None
        for key, node in self.booting.iteritems():
            if node.cloud_size.get().id == size.id and node.stop_if_no_cloud_node().get():
                del self.booting[key]
                if nodes_excess > 1:
                    self._later.stop_booting_node(size)
                break

    def _begin_node_shutdown(self, node_actor, cancellable):
        cloud_node_id = node_actor.cloud_node.get().id
        if cloud_node_id in self.shutdowns:
            return None
        shutdown = self._node_shutdown.start(
            timer_actor=self._timer, cloud_client=self._new_cloud(),
            arvados_client=self._new_arvados(),
            node_monitor=node_actor.actor_ref, cancellable=cancellable).proxy()
        self.shutdowns[cloud_node_id] = shutdown
        shutdown.subscribe(self._later.node_finished_shutdown)

    @_check_poll_freshness
    def node_can_shutdown(self, node_actor):
        if self._nodes_excess(node_actor.cloud_node.get().size) > 0:
            self._begin_node_shutdown(node_actor, cancellable=True)

    def shutdown_unpaired_node(self, cloud_node_id):
        for record_dict in [self.cloud_nodes, self.booted]:
            if cloud_node_id in record_dict:
                record = record_dict[cloud_node_id]
                break
        else:
            return None
        if not record.actor.in_state('idle', 'busy').get():
            self._begin_node_shutdown(record.actor, cancellable=False)

    def node_finished_shutdown(self, shutdown_actor):
        cloud_node, success, cancel_reason = self._get_actor_attrs(
            shutdown_actor, 'cloud_node', 'success', 'cancel_reason')
        shutdown_actor.stop()
        cloud_node_id = cloud_node.id
        if not success:
            if cancel_reason == self._node_shutdown.NODE_BROKEN:
                self.cloud_nodes.blacklist(cloud_node_id)
            del self.shutdowns[cloud_node_id]
        elif cloud_node_id in self.booted:
            self.booted.pop(cloud_node_id).actor.stop()
            del self.shutdowns[cloud_node_id]

    def shutdown(self):
        self._logger.info("Shutting down after signal.")
        self.poll_stale_after = -1  # Inhibit starting/stopping nodes
        setup_stops = {key: node.stop_if_no_cloud_node()
                       for key, node in self.booting.iteritems()}
        self.booting = {key: self.booting[key]
                        for key in setup_stops if not setup_stops[key].get()}
        self._later.await_shutdown()

    def await_shutdown(self):
        if self.booting:
            self._timer.schedule(time.time() + 1, self._later.await_shutdown)
        else:
            self.stop()
