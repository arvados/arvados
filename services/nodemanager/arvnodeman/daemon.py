#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import, print_function

import functools
import logging
import time

import pykka

from . import computenode as cnode
from . import status
from .computenode import dispatch
from .config import actor_class

class _ComputeNodeRecord(object):
    def __init__(self, actor=None, cloud_node=None, arvados_node=None,
                 assignment_time=float('-inf')):
        self.actor = actor
        self.cloud_node = cloud_node
        self.arvados_node = arvados_node
        self.assignment_time = assignment_time
        self.shutdown_actor = None

class _BaseNodeTracker(object):
    def __init__(self):
        self.nodes = {}
        self.orphans = {}

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
        # Try to select a stale node record that have an assigned slot first
        for record in sorted(self.nodes.itervalues(),
                             key=lambda r: r.arvados_node['slot_number'],
                             reverse=True):
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
                 max_total_price=0,
                 consecutive_idle_count=1):
        super(NodeManagerDaemonActor, self).__init__()
        self._node_setup = node_setup_class
        self._node_shutdown = node_shutdown_class
        self._node_actor = node_actor_class
        self._cloud_updater = cloud_update_actor
        self._timer = timer_actor
        self._new_arvados = arvados_factory
        self._new_cloud = cloud_factory
        self._cloud_driver = self._new_cloud()
        self._later = self.actor_ref.tell_proxy()
        self.shutdown_windows = shutdown_windows
        self.server_calculator = server_calculator
        self.min_cloud_size = self.server_calculator.cheapest_size()
        self.min_nodes = min_nodes
        self.max_nodes = max_nodes
        self.node_quota = max_nodes
        self.max_total_price = max_total_price
        self.poll_stale_after = poll_stale_after
        self.boot_fail_after = boot_fail_after
        self.node_stale_after = node_stale_after
        self.consecutive_idle_count = consecutive_idle_count
        self.last_polls = {}
        for poll_name in ['server_wishlist', 'arvados_nodes', 'cloud_nodes']:
            poll_actor = locals()[poll_name + '_actor']
            poll_actor.subscribe(getattr(self._later, 'update_' + poll_name))
            setattr(self, '_{}_actor'.format(poll_name), poll_actor)
            self.last_polls[poll_name] = -self.poll_stale_after
        self.cloud_nodes = _CloudNodeTracker()
        self.arvados_nodes = _ArvadosNodeTracker()
        self.booting = {}       # Actor IDs to ComputeNodeSetupActors
        self.sizes_booting = {} # Actor IDs to node size

    def on_start(self):
        self._logger = logging.getLogger("%s.%s" % (self.__class__.__name__, self.actor_urn[33:]))
        self._logger.debug("Daemon started")

    def _update_poll_time(self, poll_key):
        self.last_polls[poll_key] = time.time()

    def _pair_nodes(self, node_record, arvados_node):
        self._logger.info("Cloud node %s is now paired with Arvados node %s with hostname %s",
                          node_record.cloud_node.name, arvados_node['uuid'], arvados_node['hostname'])
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
            node_stale_after=self.node_stale_after,
            cloud_client=self._cloud_driver,
            boot_fail_after=self.boot_fail_after,
            consecutive_idle_count=self.consecutive_idle_count)
        actorTell = actor.tell_proxy()
        actorTell.subscribe(self._later.node_can_shutdown)
        self._cloud_nodes_actor.subscribe_to(cloud_node.id,
                                             actorTell.update_cloud_node)
        record = _ComputeNodeRecord(actor.proxy(), cloud_node)
        return record

    def _register_cloud_node(self, node):
        rec = self.cloud_nodes.get(node.id)
        if rec is None:
            self._logger.info("Registering new cloud node %s", node.id)
            record = self._new_node(node)
            self.cloud_nodes.add(record)
        else:
            rec.cloud_node = node

    def update_cloud_nodes(self, nodelist):
        self._update_poll_time('cloud_nodes')
        for _, node in self.cloud_nodes.update_from(nodelist):
            self._register_cloud_node(node)

        self.try_pairing()

        for record in self.cloud_nodes.orphans.itervalues():
            if record.shutdown_actor:
                try:
                    record.shutdown_actor.stop()
                except pykka.ActorDeadError:
                    pass
                record.shutdown_actor = None

            # A recently booted node is a node that successfully completed the
            # setup actor but has not yet appeared in the cloud node list.
            # This will have the tag _nodemanager_recently_booted on it, which
            # means (if we're not shutting it down) we want to put it back into
            # the cloud node list.  Once it really appears in the cloud list,
            # the object in record.cloud_node will be replaced by a new one
            # that lacks the "_nodemanager_recently_booted" tag.
            if hasattr(record.cloud_node, "_nodemanager_recently_booted"):
                self.cloud_nodes.add(record)
            else:
                # Node disappeared from the cloud node list. If it's paired,
                # remove its idle time counter.
                if record.arvados_node:
                    status.tracker.idle_out(record.arvados_node.get('hostname'))
                # Stop the monitor actor if necessary and forget about the node.
                if record.actor:
                    try:
                        record.actor.stop()
                    except pykka.ActorDeadError:
                        pass
                    record.actor = None
                record.cloud_node = None

    def _register_arvados_node(self, key, arv_node):
        self._logger.info("Registering new Arvados node %s", key)
        record = _ComputeNodeRecord(arvados_node=arv_node)
        self.arvados_nodes.add(record)

    def update_arvados_nodes(self, nodelist):
        self._update_poll_time('arvados_nodes')
        for key, node in self.arvados_nodes.update_from(nodelist):
            self._register_arvados_node(key, node)
        self.try_pairing()

    def try_pairing(self):
        for record in self.cloud_nodes.unpaired():
            for arv_rec in self.arvados_nodes.unpaired():
                if record.actor is not None and record.actor.offer_arvados_pair(arv_rec.arvados_node).get():
                    self._pair_nodes(record, arv_rec.arvados_node)
                    break

    def _nodes_booting(self, size):
        s = sum(1
                for c in self.booting.iterkeys()
                if size is None or self.sizes_booting[c].id == size.id)
        return s

    def _node_states(self, size):
        proxy_states = []
        states = []
        for rec in self.cloud_nodes.nodes.itervalues():
            if size is None or rec.cloud_node.size.id == size.id:
                if rec.shutdown_actor is None and rec.actor is not None:
                    proxy_states.append(rec.actor.get_state())
                else:
                    states.append("shutdown")
        return states + pykka.get_all(proxy_states)

    def _update_tracker(self):
        updates = {
            k: 0
            for k in status.tracker.keys()
            if k.startswith('nodes_')
        }
        for s in self._node_states(size=None):
            updates.setdefault('nodes_'+s, 0)
            updates['nodes_'+s] += 1
        updates['nodes_wish'] = len(self.last_wishlist)
        updates['node_quota'] = self.node_quota
        status.tracker.update(updates)

    def _state_counts(self, size):
        states = self._node_states(size)
        counts = {
            "booting": self._nodes_booting(size),
            "unpaired": 0,
            "busy": 0,
            "idle": 0,
            "fail": 0,
            "down": 0,
            "shutdown": 0
        }
        for s in states:
            counts[s] = counts[s] + 1
        return counts

    def _nodes_up(self, counts):
        up = counts["booting"] + counts["unpaired"] + counts["idle"] + counts["busy"]
        return up

    def _total_price(self):
        cost = 0
        cost += sum(self.sizes_booting[c].price
                    for c in self.booting.iterkeys())
        cost += sum(c.cloud_node.size.price
                    for c in self.cloud_nodes.nodes.itervalues())
        return cost

    def _size_wishlist(self, size):
        return sum(1 for c in self.last_wishlist if c.id == size.id)

    def _nodes_wanted(self, size):
        total_node_count = self._nodes_booting(None) + len(self.cloud_nodes)
        under_min = self.min_nodes - total_node_count
        over_max = total_node_count - self.node_quota
        total_price = self._total_price()

        counts = self._state_counts(size)

        up_count = self._nodes_up(counts)
        busy_count = counts["busy"]
        wishlist_count = self._size_wishlist(size)

        self._logger.info("%s: wishlist %i, up %i (booting %i, unpaired %i, idle %i, busy %i), down %i, shutdown %i", size.id,
                          wishlist_count,
                          up_count,
                          counts["booting"],
                          counts["unpaired"],
                          counts["idle"],
                          busy_count,
                          counts["down"]+counts["fail"],
                          counts["shutdown"])

        if over_max >= 0:
            return -over_max
        elif under_min > 0 and size.id == self.min_cloud_size.id:
            return under_min

        wanted = wishlist_count - (up_count - busy_count)
        if wanted > 0 and self.max_total_price and ((total_price + (size.price*wanted)) > self.max_total_price):
            can_boot = int((self.max_total_price - total_price) / size.price)
            if can_boot == 0:
                self._logger.info("Not booting %s (price %s) because with it would exceed max_total_price of %s (current total_price is %s)",
                                  size.id, size.price, self.max_total_price, total_price)
            return can_boot
        else:
            return wanted

    def _nodes_excess(self, size):
        counts = self._state_counts(size)
        up_count = self._nodes_up(counts)
        if size.id == self.min_cloud_size.id:
            up_count -= self.min_nodes
        return up_count - (counts["busy"] + self._size_wishlist(size))

    def update_server_wishlist(self, wishlist):
        self._update_poll_time('server_wishlist')
        requestable_nodes = self.node_quota - (self._nodes_booting(None) + len(self.cloud_nodes))
        self.last_wishlist = wishlist[:requestable_nodes]
        for size in reversed(self.server_calculator.cloud_sizes):
            try:
                nodes_wanted = self._nodes_wanted(size)
                if nodes_wanted > 0:
                    self._later.start_node(size)
                elif (nodes_wanted < 0) and self.booting:
                    self._later.stop_booting_node(size)
            except Exception:
                self._logger.exception("while calculating nodes wanted for size %s", getattr(size, "id", "(id not available)"))
        try:
            self._update_tracker()
        except:
            self._logger.exception("while updating tracker")

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

        if not self.cancel_node_shutdown(cloud_size):
            arvados_node = self.arvados_nodes.find_stale_node(self.node_stale_after)
            self._logger.info("Want %i more %s nodes.  Booting a node.",
                              nodes_wanted, cloud_size.id)
            new_setup = self._node_setup.start(
                timer_actor=self._timer,
                arvados_client=self._new_arvados(),
                arvados_node=arvados_node,
                cloud_client=self._new_cloud(),
                cloud_size=self.server_calculator.find_size(cloud_size.id))
            self.booting[new_setup.actor_urn] = new_setup.proxy()
            self.sizes_booting[new_setup.actor_urn] = cloud_size

            if arvados_node is not None:
                self.arvados_nodes[arvados_node['uuid']].assignment_time = (
                    time.time())
            new_setup.tell_proxy().subscribe(self._later.node_setup_finished)

        if nodes_wanted > 1:
            self._later.start_node(cloud_size)

    def _get_actor_attrs(self, actor, *attr_names):
        return pykka.get_all([getattr(actor, name) for name in attr_names])

    def node_setup_finished(self, setup_proxy):
        # Called when a SetupActor has completed.
        cloud_node, arvados_node, error = self._get_actor_attrs(
            setup_proxy, 'cloud_node', 'arvados_node', 'error')
        setup_proxy.stop()

        if cloud_node is None:
            # If cloud_node is None then the node create wasn't successful.
            if error == dispatch.QuotaExceeded:
                # We've hit a quota limit, so adjust node_quota to stop trying to
                # boot new nodes until the node count goes down.
                self.node_quota = len(self.cloud_nodes)
                self._logger.warning("After quota exceeded error setting node quota to %s", self.node_quota)
        else:
            # Node creation succeeded.  Update cloud node list.
            cloud_node._nodemanager_recently_booted = True
            self._register_cloud_node(cloud_node)

            # Different quota policies may in force depending on the cloud
            # provider, account limits, and the specific mix of nodes sizes
            # that are already created.  If we are right at the quota limit,
            # we want to probe to see if the last quota still applies or if we
            # are allowed to create more nodes.
            #
            # For example, if the quota is actually based on core count, the
            # quota might be 20 single-core machines or 10 dual-core machines.
            # If we previously set node_quota to 10 dual core machines, but are
            # now booting single core machines (actual quota 20), we want to
            # allow the quota to expand so we don't get stuck at 10 machines
            # forever.
            if len(self.cloud_nodes) >= self.node_quota:
                self.node_quota = len(self.cloud_nodes)+1
                self._logger.warning("After successful boot setting node quota to %s", self.node_quota)

        self.node_quota = min(self.node_quota, self.max_nodes)
        del self.booting[setup_proxy.actor_ref.actor_urn]
        del self.sizes_booting[setup_proxy.actor_ref.actor_urn]

    @_check_poll_freshness
    def stop_booting_node(self, size):
        nodes_excess = self._nodes_excess(size)
        if (nodes_excess < 1) or not self.booting:
            return None
        for key, node in self.booting.iteritems():
            try:
                if node and node.cloud_size.get().id == size.id and node.stop_if_no_cloud_node().get(2):
                    del self.booting[key]
                    del self.sizes_booting[key]
                    if nodes_excess > 1:
                        self._later.stop_booting_node(size)
                    return
            except pykka.Timeout:
                pass

    @_check_poll_freshness
    def cancel_node_shutdown(self, size):
        # Go through shutdown actors and see if there are any of the appropriate size that can be cancelled
        for record in self.cloud_nodes.nodes.itervalues():
            try:
                if (record.shutdown_actor is not None and
                    record.cloud_node.size.id == size.id and
                    record.shutdown_actor.cancel_shutdown("Node size is in wishlist").get(2)):
                        return True
            except (pykka.ActorDeadError, pykka.Timeout) as e:
                pass
        return False

    def _begin_node_shutdown(self, node_actor, cancellable):
        cloud_node_obj = node_actor.cloud_node.get()
        cloud_node_id = cloud_node_obj.id
        record = self.cloud_nodes[cloud_node_id]
        if record.shutdown_actor is not None:
            return None
        shutdown = self._node_shutdown.start(
            timer_actor=self._timer, cloud_client=self._new_cloud(),
            arvados_client=self._new_arvados(),
            node_monitor=node_actor.actor_ref, cancellable=cancellable)
        record.shutdown_actor = shutdown.proxy()
        shutdown.tell_proxy().subscribe(self._later.node_finished_shutdown)

    @_check_poll_freshness
    def node_can_shutdown(self, node_actor):
        try:
            if self._nodes_excess(node_actor.cloud_node.get().size) > 0:
                self._begin_node_shutdown(node_actor, cancellable=True)
            elif self.cloud_nodes.nodes.get(node_actor.cloud_node.get().id).arvados_node is None:
                # Node is unpaired, which means it probably exceeded its booting
                # grace period without a ping, so shut it down so we can boot a new
                # node in its place.
                self._begin_node_shutdown(node_actor, cancellable=False)
            elif node_actor.in_state('down', 'fail').get():
                # Node is down and unlikely to come back.
                self._begin_node_shutdown(node_actor, cancellable=False)
        except pykka.ActorDeadError as e:
            # The monitor actor sends shutdown suggestions every time the
            # node's state is updated, and these go into the daemon actor's
            # message queue.  It's possible that the node has already been shut
            # down (which shuts down the node monitor actor).  In that case,
            # this message is stale and we'll get ActorDeadError when we try to
            # access node_actor.  Log the error.
            self._logger.debug("ActorDeadError in node_can_shutdown: %s", e)

    def node_finished_shutdown(self, shutdown_actor):
        try:
            cloud_node, success = self._get_actor_attrs(
                shutdown_actor, 'cloud_node', 'success')
        except pykka.ActorDeadError:
            return
        cloud_node_id = cloud_node.id

        try:
            shutdown_actor.stop()
        except pykka.ActorDeadError:
            pass

        try:
            record = self.cloud_nodes[cloud_node_id]
        except KeyError:
            # Cloud node was already removed from the cloud node list
            # supposedly while the destroy_node call was finishing its
            # job.
            return
        record.shutdown_actor = None

        if not success:
            return

        # Shutdown was successful, so stop the monitor actor, otherwise it
        # will keep offering the node as a candidate for shutdown.
        record.actor.stop()
        record.actor = None

        # If the node went from being booted to being shut down without ever
        # appearing in the cloud node list, it will have the
        # _nodemanager_recently_booted tag, so get rid of it so that the node
        # can be forgotten completely.
        if hasattr(record.cloud_node, "_nodemanager_recently_booted"):
            del record.cloud_node._nodemanager_recently_booted

    def shutdown(self):
        self._logger.info("Shutting down after signal.")
        self.poll_stale_after = -1  # Inhibit starting/stopping nodes

        # Shut down pollers
        self._server_wishlist_actor.stop()
        self._arvados_nodes_actor.stop()
        self._cloud_nodes_actor.stop()

        # Clear cloud node list
        self.update_cloud_nodes([])

        # Stop setup actors unless they are in the middle of setup.
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
