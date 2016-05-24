#!/usr/bin/env python

from __future__ import absolute_import, print_function

import functools
import logging
import time

import libcloud.common.types as cloud_types
import pykka

from .. import \
    arvados_node_fqdn, arvados_node_mtime, arvados_timestamp, timestamp_fresh, \
    arvados_node_missing, RetryMixin
from ...clientactor import _notify_subscribers
from ... import config
from .transitions import transitions

class ComputeNodeStateChangeBase(config.actor_class, RetryMixin):
    """Base class for actors that change a compute node's state.

    This base class takes care of retrying changes and notifying
    subscribers when the change is finished.
    """
    def __init__(self, cloud_client, arvados_client, timer_actor,
                 retry_wait, max_retry_wait):
        super(ComputeNodeStateChangeBase, self).__init__()
        RetryMixin.__init__(self, retry_wait, max_retry_wait,
                            None, cloud_client, timer_actor)
        self._later = self.actor_ref.tell_proxy()
        self._arvados = arvados_client
        self.subscribers = set()

    def _set_logger(self):
        self._logger = logging.getLogger("%s.%s" % (self.__class__.__name__, self.actor_urn[33:]))

    def on_start(self):
        self._set_logger()

    def _finished(self):
        if self.subscribers is None:
            raise Exception("Actor tried to finish twice")
        _notify_subscribers(self.actor_ref.proxy(), self.subscribers)
        self.subscribers = None
        self._logger.info("finished")

    def subscribe(self, subscriber):
        if self.subscribers is None:
            try:
                subscriber(self.actor_ref.proxy())
            except pykka.ActorDeadError:
                pass
        else:
            self.subscribers.add(subscriber)

    def _clean_arvados_node(self, arvados_node, explanation):
        return self._arvados.nodes().update(
            uuid=arvados_node['uuid'],
            body={'hostname': None,
                  'ip_address': None,
                  'slot_number': None,
                  'first_ping_at': None,
                  'last_ping_at': None,
                  'properties': {},
                  'info': {'ec2_instance_id': None,
                           'last_action': explanation}},
            ).execute()

    @staticmethod
    def _finish_on_exception(orig_func):
        @functools.wraps(orig_func)
        def finish_wrapper(self, *args, **kwargs):
            try:
                return orig_func(self, *args, **kwargs)
            except Exception as error:
                self._logger.error("Actor error %s", error)
                self._finished()
        return finish_wrapper


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
            cloud_client, arvados_client, timer_actor,
            retry_wait, max_retry_wait)
        self.cloud_size = cloud_size
        self.arvados_node = None
        self.cloud_node = None
        if arvados_node is None:
            self._later.create_arvados_node()
        else:
            self._later.prepare_arvados_node(arvados_node)

    @ComputeNodeStateChangeBase._finish_on_exception
    @RetryMixin._retry(config.ARVADOS_ERRORS)
    def create_arvados_node(self):
        self.arvados_node = self._arvados.nodes().create(body={}).execute()
        self._later.create_cloud_node()

    @ComputeNodeStateChangeBase._finish_on_exception
    @RetryMixin._retry(config.ARVADOS_ERRORS)
    def prepare_arvados_node(self, node):
        self.arvados_node = self._clean_arvados_node(
            node, "Prepared by Node Manager")
        self._later.create_cloud_node()

    @ComputeNodeStateChangeBase._finish_on_exception
    @RetryMixin._retry()
    def create_cloud_node(self):
        self._logger.info("Sending create_node request for node size %s.",
                          self.cloud_size.name)
        self.cloud_node = self._cloud.create_node(self.cloud_size,
                                                  self.arvados_node)
        if not self.cloud_node.size:
             self.cloud_node.size = self.cloud_size
        self._logger.info("Cloud node %s created.", self.cloud_node.id)
        self._later.update_arvados_node_properties()

    @ComputeNodeStateChangeBase._finish_on_exception
    @RetryMixin._retry(config.ARVADOS_ERRORS)
    def update_arvados_node_properties(self):
        """Tell Arvados some details about the cloud node.

        Currently we only include size/price from our request, which
        we already knew before create_cloud_node(), but doing it here
        gives us an opportunity to provide more detail from
        self.cloud_node, too.
        """
        self.arvados_node['properties']['cloud_node'] = {
            # Note this 'size' is the node size we asked the cloud
            # driver to create -- not necessarily equal to the size
            # reported by the cloud driver for the node that was
            # created.
            'size': self.cloud_size.id,
            'price': self.cloud_size.price,
        }
        self.arvados_node = self._arvados.nodes().update(
            uuid=self.arvados_node['uuid'],
            body={'properties': self.arvados_node['properties']},
        ).execute()
        self._logger.info("%s updated properties.", self.arvados_node['uuid'])
        self._later.post_create()

    @RetryMixin._retry()
    def post_create(self):
        self._cloud.post_create_node(self.cloud_node)
        self._logger.info("%s post-create work done.", self.cloud_node.id)
        self._finished()

    def stop_if_no_cloud_node(self):
        if self.cloud_node is not None:
            return False
        self.stop()
        return True


class ComputeNodeShutdownActor(ComputeNodeStateChangeBase):
    """Actor to shut down a compute node.

    This actor simply destroys a cloud node, retrying as needed.
    """
    # Reasons for a shutdown to be cancelled.
    WINDOW_CLOSED = "shutdown window closed"
    NODE_BROKEN = "cloud failed to shut down broken node"

    def __init__(self, timer_actor, cloud_client, arvados_client, node_monitor,
                 cancellable=True, retry_wait=1, max_retry_wait=180):
        # If a ShutdownActor is cancellable, it will ask the
        # ComputeNodeMonitorActor if it's still eligible before taking each
        # action, and stop the shutdown process if the node is no longer
        # eligible.  Normal shutdowns based on job demand should be
        # cancellable; shutdowns based on node misbehavior should not.
        super(ComputeNodeShutdownActor, self).__init__(
            cloud_client, arvados_client, timer_actor,
            retry_wait, max_retry_wait)
        self._monitor = node_monitor.proxy()
        self.cloud_node = self._monitor.cloud_node.get()
        self.cancellable = cancellable
        self.cancel_reason = None
        self.success = None

    def _set_logger(self):
        self._logger = logging.getLogger("%s.%s.%s" % (self.__class__.__name__, self.actor_urn[33:], self.cloud_node.name))

    def on_start(self):
        super(ComputeNodeShutdownActor, self).on_start()
        self._later.shutdown_node()

    def _arvados_node(self):
        return self._monitor.arvados_node.get()

    def _finished(self, success_flag=None):
        if success_flag is not None:
            self.success = success_flag
        return super(ComputeNodeShutdownActor, self)._finished()

    def cancel_shutdown(self, reason):
        self.cancel_reason = reason
        self._logger.info("Shutdown cancelled: %s.", reason)
        self._finished(success_flag=False)

    def _cancel_on_exception(orig_func):
        @functools.wraps(orig_func)
        def finish_wrapper(self, *args, **kwargs):
            try:
                return orig_func(self, *args, **kwargs)
            except Exception as error:
                self._logger.error("Actor error %s", error)
                self._later.cancel_shutdown("Unhandled exception %s" % error)
        return finish_wrapper

    @_cancel_on_exception
    @RetryMixin._retry()
    def shutdown_node(self):
        self._logger.info("Starting shutdown")
        if not self._cloud.destroy_node(self.cloud_node):
            if self._cloud.broken(self.cloud_node):
                self._later.cancel_shutdown(self.NODE_BROKEN)
                return
            else:
                # Force a retry.
                raise cloud_types.LibcloudError("destroy_node failed")
        self._logger.info("Shutdown success")
        arv_node = self._arvados_node()
        if arv_node is None:
            self._finished(success_flag=True)
        else:
            self._later.clean_arvados_node(arv_node)

    @ComputeNodeStateChangeBase._finish_on_exception
    @RetryMixin._retry(config.ARVADOS_ERRORS)
    def clean_arvados_node(self, arvados_node):
        self._clean_arvados_node(arvados_node, "Shut down by Node Manager")
        self._finished(success_flag=True)


class ComputeNodeUpdateActor(config.actor_class):
    """Actor to dispatch one-off cloud management requests.

    This actor receives requests for small cloud updates, and
    dispatches them to a real driver.  ComputeNodeMonitorActors use
    this to perform maintenance tasks on themselves.  Having a
    dedicated actor for this gives us the opportunity to control the
    flow of requests; e.g., by backing off when errors occur.
    """
    def __init__(self, cloud_factory, max_retry_wait=180):
        super(ComputeNodeUpdateActor, self).__init__()
        self._cloud = cloud_factory()
        self.max_retry_wait = max_retry_wait
        self.error_streak = 0
        self.next_request_time = time.time()

    def _set_logger(self):
        self._logger = logging.getLogger("%s.%s" % (self.__class__.__name__, self.actor_urn[33:]))

    def on_start(self):
        self._set_logger()

    def _throttle_errors(orig_func):
        @functools.wraps(orig_func)
        def throttle_wrapper(self, *args, **kwargs):
            throttle_time = self.next_request_time - time.time()
            if throttle_time > 0:
                time.sleep(throttle_time)
            self.next_request_time = time.time()
            try:
                result = orig_func(self, *args, **kwargs)
            except Exception as error:
                if self._cloud.is_cloud_exception(error):
                    self.error_streak += 1
                    self.next_request_time += min(2 ** self.error_streak,
                                                  self.max_retry_wait)
                self._logger.warn(
                    "Unhandled exception: %s", error, exc_info=error)
            else:
                self.error_streak = 0
                return result
        return throttle_wrapper

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
                 cloud_fqdn_func, timer_actor, update_actor, cloud_client,
                 arvados_node=None, poll_stale_after=600, node_stale_after=3600,
                 boot_fail_after=1800
    ):
        super(ComputeNodeMonitorActor, self).__init__()
        self._later = self.actor_ref.tell_proxy()
        self._last_log = None
        self._shutdowns = shutdown_timer
        self._cloud_node_fqdn = cloud_fqdn_func
        self._timer = timer_actor
        self._update = update_actor
        self._cloud = cloud_client
        self.cloud_node = cloud_node
        self.cloud_node_start_time = cloud_node_start_time
        self.poll_stale_after = poll_stale_after
        self.node_stale_after = node_stale_after
        self.boot_fail_after = boot_fail_after
        self.subscribers = set()
        self.arvados_node = None
        self._later.update_arvados_node(arvados_node)
        self.last_shutdown_opening = None
        self._later.consider_shutdown()

    def _set_logger(self):
        self._logger = logging.getLogger("%s.%s.%s" % (self.__class__.__name__, self.actor_urn[33:], self.cloud_node.name))

    def on_start(self):
        self._set_logger()
        self._timer.schedule(self.cloud_node_start_time + self.boot_fail_after, self._later.consider_shutdown)

    def subscribe(self, subscriber):
        self.subscribers.add(subscriber)

    def _debug(self, msg, *args):
        if msg == self._last_log:
            return
        self._last_log = msg
        self._logger.debug(msg, *args)

    def get_state(self):
        """Get node state, one of ['unpaired', 'busy', 'idle', 'down']."""

        # If this node is not associated with an Arvados node, return 'unpaired'.
        if self.arvados_node is None:
            return 'unpaired'

        state = self.arvados_node['crunch_worker_state']

        # If state information is not available because it is missing or the
        # record is stale, return 'down'.
        if not state or not timestamp_fresh(arvados_node_mtime(self.arvados_node),
                                            self.node_stale_after):
            state = 'down'

        # There's a window between when a node pings for the first time and the
        # value of 'slurm_state' is synchronized by crunch-dispatch.  In this
        # window, the node will still report as 'down'.  Check that
        # first_ping_at is truthy and consider the node 'idle' during the
        # initial boot grace period.
        if (state == 'down' and
            self.arvados_node['first_ping_at'] and
            timestamp_fresh(self.cloud_node_start_time,
                            self.boot_fail_after) and
            not self._cloud.broken(self.cloud_node)):
            state = 'idle'

        # "missing" means last_ping_at is stale, this should be
        # considered "down"
        if arvados_node_missing(self.arvados_node, self.node_stale_after):
            state = 'down'

        # Turns out using 'job_uuid' this way is a bad idea.  The node record
        # is assigned the job_uuid before the job is locked (which removes it
        # from the queue) which means the job will be double-counted as both in
        # the wishlist and but also keeping a node busy.  This end result is
        # excess nodes being booted.
        #if state == 'idle' and self.arvados_node['job_uuid']:
        #    state = 'busy'

        return state

    def in_state(self, *states):
        return self.get_state() in states

    def shutdown_eligible(self):
        """Determine if node is candidate for shut down.

        Returns a tuple of (boolean, string) where the first value is whether
        the node is candidate for shut down, and the second value is the
        reason for the decision.
        """

        # Collect states and then consult state transition table whether we
        # should shut down.  Possible states are:
        # crunch_worker_state = ['unpaired', 'busy', 'idle', 'down']
        # window = ["open", "closed"]
        # boot_grace = ["boot wait", "boot exceeded"]
        # idle_grace = ["not idle", "idle wait", "idle exceeded"]

        if self.arvados_node and not timestamp_fresh(arvados_node_mtime(self.arvados_node), self.node_stale_after):
            return (False, "node state is stale")

        crunch_worker_state = self.get_state()

        window = "open" if self._shutdowns.window_open() else "closed"

        if timestamp_fresh(self.cloud_node_start_time, self.boot_fail_after):
            boot_grace = "boot wait"
        else:
            boot_grace = "boot exceeded"

        # API server side not implemented yet.
        idle_grace = 'idle exceeded'

        node_state = (crunch_worker_state, window, boot_grace, idle_grace)
        t = transitions[node_state]
        if t is not None:
            # yes, shutdown eligible
            return (True, "node state is %s" % (node_state,))
        else:
            # no, return a reason
            return (False, "node state is %s" % (node_state,))

    def consider_shutdown(self):
        try:
            eligible, reason = self.shutdown_eligible()
            next_opening = self._shutdowns.next_opening()
            if eligible:
                self._debug("Suggesting shutdown because %s", reason)
                _notify_subscribers(self.actor_ref.proxy(), self.subscribers)
            else:
                self._debug("Not eligible for shut down because %s", reason)

                if self.last_shutdown_opening != next_opening:
                    self._debug("Shutdown window closed.  Next at %s.",
                                time.strftime('%Y-%m-%d %H:%M:%S', time.localtime(next_opening)))
                    self._timer.schedule(next_opening, self._later.consider_shutdown)
                    self.last_shutdown_opening = next_opening
        except Exception:
            self._logger.exception("Unexpected exception")

    def offer_arvados_pair(self, arvados_node):
        first_ping_s = arvados_node.get('first_ping_at')
        if (self.arvados_node is not None) or (not first_ping_s):
            return None
        elif ((arvados_node['info'].get('ec2_instance_id') == self._cloud.node_id(self.cloud_node)) and
              (arvados_timestamp(first_ping_s) >= self.cloud_node_start_time)):
            self._later.update_arvados_node(arvados_node)
            return self.cloud_node.id
        else:
            return None

    def update_cloud_node(self, cloud_node):
        if cloud_node is not None:
            self.cloud_node = cloud_node
            self._later.consider_shutdown()

    def update_arvados_node(self, arvados_node):
        # If the cloud node's FQDN doesn't match what's in the Arvados node
        # record, make them match.
        # This method is a little unusual in the way it just fires off the
        # request without checking the result or retrying errors.  That's
        # because this update happens every time we reload the Arvados node
        # list: if a previous sync attempt failed, we'll see that the names
        # are out of sync and just try again.  ComputeNodeUpdateActor has
        # the logic to throttle those effective retries when there's trouble.
        if arvados_node is not None:
            self.arvados_node = arvados_node
            if (self._cloud_node_fqdn(self.cloud_node) !=
                  arvados_node_fqdn(self.arvados_node)):
                self._update.sync_node(self.cloud_node, self.arvados_node)
            self._later.consider_shutdown()
