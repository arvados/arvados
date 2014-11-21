#!/usr/bin/env python

from __future__ import absolute_import, print_function

import functools
import logging
import time

import libcloud.common.types as cloud_types
import pykka

from .. import arvados_node_fqdn, arvados_node_mtime, timestamp_fresh
from ...clientactor import _notify_subscribers
from ... import config

class ComputeNodeStateChangeBase(config.actor_class):
    """Base class for actors that change a compute node's state.

    This base class takes care of retrying changes and notifying
    subscribers when the change is finished.
    """
    def __init__(self, logger_name, timer_actor, retry_wait, max_retry_wait):
        super(ComputeNodeStateChangeBase, self).__init__()
        self._later = self.actor_ref.proxy()
        self._timer = timer_actor
        self._logger = logging.getLogger(logger_name)
        self.min_retry_wait = retry_wait
        self.max_retry_wait = max_retry_wait
        self.retry_wait = retry_wait
        self.subscribers = set()

    @staticmethod
    def _retry(errors):
        """Retry decorator for an actor method that makes remote requests.

        Use this function to decorator an actor method, and pass in a
        tuple of exceptions to catch.  This decorator will schedule
        retries of that method with exponential backoff if the
        original method raises any of the given errors.
        """
        def decorator(orig_func):
            @functools.wraps(orig_func)
            def wrapper(self, *args, **kwargs):
                start_time = time.time()
                try:
                    orig_func(self, *args, **kwargs)
                except errors as error:
                    self._logger.warning(
                        "Client error: %s - waiting %s seconds",
                        error, self.retry_wait)
                    self._timer.schedule(start_time + self.retry_wait,
                                         getattr(self._later,
                                                 orig_func.__name__),
                                         *args, **kwargs)
                    self.retry_wait = min(self.retry_wait * 2,
                                          self.max_retry_wait)
                else:
                    self.retry_wait = self.min_retry_wait
            return wrapper
        return decorator

    def _finished(self):
        _notify_subscribers(self._later, self.subscribers)
        self.subscribers = None

    def subscribe(self, subscriber):
        if self.subscribers is None:
            try:
                subscriber(self._later)
            except pykka.ActorDeadError:
                pass
        else:
            self.subscribers.add(subscriber)


class ComputeNodeSetupActor(ComputeNodeStateChangeBase):
    """Actor to create and set up a cloud compute node.

    This actor prepares an Arvados node record for a new compute node
    (either creating one or cleaning one passed in), then boots the
    actual compute node.  It notifies subscribers when the cloud node
    is successfully created (the last step in the process for Node
    Manager to handle).
    """
    def __init__(self, timer_actor, arvados_client, cloud_client,
                 cloud_size, arvados_node=None,
                 retry_wait=1, max_retry_wait=180):
        super(ComputeNodeSetupActor, self).__init__(
            'arvnodeman.nodeup', timer_actor, retry_wait, max_retry_wait)
        self._arvados = arvados_client
        self._cloud = cloud_client
        self.cloud_size = cloud_size
        self.arvados_node = None
        self.cloud_node = None
        if arvados_node is None:
            self._later.create_arvados_node()
        else:
            self._later.prepare_arvados_node(arvados_node)

    @ComputeNodeStateChangeBase._retry(config.ARVADOS_ERRORS)
    def create_arvados_node(self):
        self.arvados_node = self._arvados.nodes().create(body={}).execute()
        self._later.create_cloud_node()

    @ComputeNodeStateChangeBase._retry(config.ARVADOS_ERRORS)
    def prepare_arvados_node(self, node):
        self.arvados_node = self._arvados.nodes().update(
            uuid=node['uuid'],
            body={'hostname': None,
                  'ip_address': None,
                  'slot_number': None,
                  'first_ping_at': None,
                  'last_ping_at': None,
                  'info': {'ec2_instance_id': None,
                           'last_action': "Prepared by Node Manager"}}
            ).execute()
        self._later.create_cloud_node()

    @ComputeNodeStateChangeBase._retry(config.CLOUD_ERRORS)
    def create_cloud_node(self):
        self._logger.info("Creating cloud node with size %s.",
                          self.cloud_size.name)
        self.cloud_node = self._cloud.create_node(self.cloud_size,
                                                  self.arvados_node)
        self._logger.info("Cloud node %s created.", self.cloud_node.id)
        self._finished()

    def stop_if_no_cloud_node(self):
        if self.cloud_node is None:
            self.stop()


class ComputeNodeShutdownActor(ComputeNodeStateChangeBase):
    """Actor to shut down a compute node.

    This actor simply destroys a cloud node, retrying as needed.
    """
    def __init__(self, timer_actor, cloud_client, node_monitor,
                 retry_wait=1, max_retry_wait=180):
        super(ComputeNodeShutdownActor, self).__init__(
            'arvnodeman.nodedown', timer_actor, retry_wait, max_retry_wait)
        self._cloud = cloud_client
        self._monitor = node_monitor.proxy()
        self.cloud_node = self._monitor.cloud_node.get()
        self.success = None

    def on_start(self):
        self._later.shutdown_node()

    def cancel_shutdown(self):
        self.success = False
        self._finished()

    def _stop_if_window_closed(orig_func):
        @functools.wraps(orig_func)
        def wrapper(self, *args, **kwargs):
            if not self._monitor.shutdown_eligible().get():
                self._logger.info(
                    "Cloud node %s shutdown cancelled - no longer eligible.",
                    self.cloud_node.id)
                self._later.cancel_shutdown()
                return None
            else:
                return orig_func(self, *args, **kwargs)
        return wrapper

    @_stop_if_window_closed
    @ComputeNodeStateChangeBase._retry(config.CLOUD_ERRORS)
    def shutdown_node(self):
        if self._cloud.destroy_node(self.cloud_node):
            self._logger.info("Cloud node %s shut down.", self.cloud_node.id)
            self.success = True
            self._finished()
        else:
            # Force a retry.
            raise cloud_types.LibcloudError("destroy_node failed")

    # Make the decorator available to subclasses.
    _stop_if_window_closed = staticmethod(_stop_if_window_closed)


class ComputeNodeUpdateActor(config.actor_class):
    """Actor to dispatch one-off cloud management requests.

    This actor receives requests for small cloud updates, and
    dispatches them to a real driver.  ComputeNodeMonitorActors use
    this to perform maintenance tasks on themselves.  Having a
    dedicated actor for this gives us the opportunity to control the
    flow of requests; e.g., by backing off when errors occur.

    This actor is most like a "traditional" Pykka actor: there's no
    subscribing, but instead methods return real driver results.  If
    you're interested in those results, you should get them from the
    Future that the proxy method returns.  Be prepared to handle exceptions
    from the cloud driver when you do.
    """
    def __init__(self, cloud_factory, max_retry_wait=180):
        super(ComputeNodeUpdateActor, self).__init__()
        self._cloud = cloud_factory()
        self.max_retry_wait = max_retry_wait
        self.error_streak = 0
        self.next_request_time = time.time()

    def _throttle_errors(orig_func):
        @functools.wraps(orig_func)
        def wrapper(self, *args, **kwargs):
            throttle_time = self.next_request_time - time.time()
            if throttle_time > 0:
                time.sleep(throttle_time)
            self.next_request_time = time.time()
            try:
                result = orig_func(self, *args, **kwargs)
            except config.CLOUD_ERRORS:
                self.error_streak += 1
                self.next_request_time += min(2 ** self.error_streak,
                                              self.max_retry_wait)
                raise
            else:
                self.error_streak = 0
                return result
        return wrapper

    @_throttle_errors
    def sync_node(self, cloud_node, arvados_node):
        return self._cloud.sync_node(cloud_node, arvados_node)


class ComputeNodeMonitorActor(config.actor_class):
    """Actor to manage a running compute node.

    This actor gets updates about a compute node's cloud and Arvados records.
    It uses this information to notify subscribers when the node is eligible
    for shutdown.
    """
    def __init__(self, cloud_node, cloud_node_start_time, shutdown_timer,
                 timer_actor, update_actor, arvados_node=None,
                 poll_stale_after=600, node_stale_after=3600):
        super(ComputeNodeMonitorActor, self).__init__()
        self._later = self.actor_ref.proxy()
        self._logger = logging.getLogger('arvnodeman.computenode')
        self._last_log = None
        self._shutdowns = shutdown_timer
        self._timer = timer_actor
        self._update = update_actor
        self.cloud_node = cloud_node
        self.cloud_node_start_time = cloud_node_start_time
        self.poll_stale_after = poll_stale_after
        self.node_stale_after = node_stale_after
        self.subscribers = set()
        self.arvados_node = None
        self._later.update_arvados_node(arvados_node)
        self.last_shutdown_opening = None
        self._later.consider_shutdown()

    def subscribe(self, subscriber):
        self.subscribers.add(subscriber)

    def _debug(self, msg, *args):
        if msg == self._last_log:
            return
        self._last_log = msg
        self._logger.debug(msg, *args)

    def in_state(self, *states):
        # Return a boolean to say whether or not our Arvados node record is in
        # one of the given states.  If state information is not
        # available--because this node has no Arvados record, the record is
        # stale, or the record has no state information--return None.
        if (self.arvados_node is None) or not timestamp_fresh(
              arvados_node_mtime(self.arvados_node), self.node_stale_after):
            return None
        state = self.arvados_node['info'].get('slurm_state')
        if not state:
            return None
        result = state in states
        if state == 'idle':
            result = result and not self.arvados_node['job_uuid']
        return result

    def shutdown_eligible(self):
        if not self._shutdowns.window_open():
            return False
        elif self.arvados_node is None:
            # If this is a new, unpaired node, it's eligible for
            # shutdown--we figure there was an error during bootstrap.
            return timestamp_fresh(self.cloud_node_start_time,
                                   self.node_stale_after)
        else:
            return self.in_state('idle')

    def consider_shutdown(self):
        next_opening = self._shutdowns.next_opening()
        if self.shutdown_eligible():
            self._debug("Node %s suggesting shutdown.", self.cloud_node.id)
            _notify_subscribers(self._later, self.subscribers)
        elif self._shutdowns.window_open():
            self._debug("Node %s shutdown window open but node busy.",
                        self.cloud_node.id)
        elif self.last_shutdown_opening != next_opening:
            self._debug("Node %s shutdown window closed.  Next at %s.",
                        self.cloud_node.id, time.ctime(next_opening))
            self._timer.schedule(next_opening, self._later.consider_shutdown)
            self.last_shutdown_opening = next_opening

    def offer_arvados_pair(self, arvados_node):
        if self.arvados_node is not None:
            return None
        elif arvados_node['ip_address'] in self.cloud_node.private_ips:
            self._later.update_arvados_node(arvados_node)
            return self.cloud_node.id
        else:
            return None

    def update_cloud_node(self, cloud_node):
        if cloud_node is not None:
            self.cloud_node = cloud_node
            self._later.consider_shutdown()

    def update_arvados_node(self, arvados_node):
        if arvados_node is not None:
            self.arvados_node = arvados_node
            new_hostname = arvados_node_fqdn(self.arvados_node)
            if new_hostname != self.cloud_node.name:
                self._update.sync_node(self.cloud_node, self.arvados_node)
            self._later.consider_shutdown()
