#!/usr/bin/env python

from __future__ import absolute_import, print_function

import functools
import itertools
import logging
import time

import pykka

from ..clientactor import _notify_subscribers
from .. import config

def arvados_node_fqdn(arvados_node, default_hostname='dynamic.compute'):
    hostname = arvados_node.get('hostname') or default_hostname
    return '{}.{}'.format(hostname, arvados_node['domain'])

def arvados_node_mtime(node):
    return time.mktime(time.strptime(node['modified_at'] + 'UTC',
                                     '%Y-%m-%dT%H:%M:%SZ%Z')) - time.timezone

def timestamp_fresh(timestamp, fresh_time):
    return (time.time() - timestamp) < fresh_time

def _retry(errors):
    """Retry decorator for an actor method that makes remote requests.

    Use this function to decorator an actor method, and pass in a tuple of
    exceptions to catch.  This decorator will schedule retries of that method
    with exponential backoff if the original method raises any of the given
    errors.
    """
    def decorator(orig_func):
        @functools.wraps(orig_func)
        def wrapper(self, *args, **kwargs):
            try:
                orig_func(self, *args, **kwargs)
            except errors as error:
                self._logger.warning(
                    "Client error: %s - waiting %s seconds",
                    error, self.retry_wait)
                self._timer.schedule(self.retry_wait,
                                     getattr(self._later, orig_func.__name__),
                                     *args, **kwargs)
                self.retry_wait = min(self.retry_wait * 2,
                                      self.max_retry_wait)
            else:
                self.retry_wait = self.min_retry_wait
        return wrapper
    return decorator

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


ComputeNodeDriverClass = BaseComputeNodeDriver

class ComputeNodeSetupActor(config.actor_class):
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
        super(ComputeNodeSetupActor, self).__init__()
        self._timer = timer_actor
        self._arvados = arvados_client
        self._cloud = cloud_client
        self._later = self.actor_ref.proxy()
        self._logger = logging.getLogger('arvnodeman.nodeup')
        self.cloud_size = cloud_size
        self.subscribers = set()
        self.min_retry_wait = retry_wait
        self.max_retry_wait = max_retry_wait
        self.retry_wait = retry_wait
        self.arvados_node = None
        self.cloud_node = None
        if arvados_node is None:
            self._later.create_arvados_node()
        else:
            self._later.prepare_arvados_node(arvados_node)

    @_retry(config.ARVADOS_ERRORS)
    def create_arvados_node(self):
        self.arvados_node = self._arvados.nodes().create(body={}).execute()
        self._later.create_cloud_node()

    @_retry(config.ARVADOS_ERRORS)
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

    @_retry(config.CLOUD_ERRORS)
    def create_cloud_node(self):
        self._logger.info("Creating cloud node with size %s.",
                          self.cloud_size.name)
        self.cloud_node = self._cloud.create_node(self.cloud_size,
                                                  self.arvados_node)
        self._logger.info("Cloud node %s created.", self.cloud_node.id)
        _notify_subscribers(self._later, self.subscribers)
        self.subscribers = None

    def stop_if_no_cloud_node(self):
        if self.cloud_node is None:
            self.stop()

    def subscribe(self, subscriber):
        if self.subscribers is None:
            try:
                subscriber(self._later)
            except pykka.ActorDeadError:
                pass
        else:
            self.subscribers.add(subscriber)


class ComputeNodeShutdownActor(config.actor_class):
    """Actor to shut down a compute node.

    This actor simply destroys a cloud node, retrying as needed.
    """
    def __init__(self, timer_actor, cloud_client, cloud_node,
                 retry_wait=1, max_retry_wait=180):
        super(ComputeNodeShutdownActor, self).__init__()
        self._timer = timer_actor
        self._cloud = cloud_client
        self._later = self.actor_ref.proxy()
        self._logger = logging.getLogger('arvnodeman.nodedown')
        self.cloud_node = cloud_node
        self.min_retry_wait = retry_wait
        self.max_retry_wait = max_retry_wait
        self.retry_wait = retry_wait
        self._later.shutdown_node()

    @_retry(config.CLOUD_ERRORS)
    def shutdown_node(self):
        self._cloud.destroy_node(self.cloud_node)
        self._logger.info("Cloud node %s shut down.", self.cloud_node.id)


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


class ShutdownTimer(object):
    """Keep track of a cloud node's shutdown windows.

    Instantiate this class with a timestamp of when a cloud node started,
    and a list of durations (in minutes) of when the node must not and may
    be shut down, alternating.  The class will tell you when a shutdown
    window is open, and when the next open window will start.
    """
    def __init__(self, start_time, shutdown_windows):
        # The implementation is easiest if we have an even number of windows,
        # because then windows always alternate between open and closed.
        # Rig that up: calculate the first shutdown window based on what's
        # passed in.  Then, if we were given an odd number of windows, merge
        # that first window into the last one, since they both# represent
        # closed state.
        first_window = shutdown_windows[0]
        shutdown_windows = list(shutdown_windows[1:])
        self._next_opening = start_time + (60 * first_window)
        if len(shutdown_windows) % 2:
            shutdown_windows.append(first_window)
        else:
            shutdown_windows[-1] += first_window
        self.shutdown_windows = itertools.cycle([60 * n
                                                 for n in shutdown_windows])
        self._open_start = self._next_opening
        self._open_for = next(self.shutdown_windows)

    def _advance_opening(self):
        while self._next_opening < time.time():
            self._open_start = self._next_opening
            self._next_opening += self._open_for + next(self.shutdown_windows)
            self._open_for = next(self.shutdown_windows)

    def next_opening(self):
        self._advance_opening()
        return self._next_opening

    def window_open(self):
        self._advance_opening()
        return 0 < (time.time() - self._open_start) < self._open_for


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

    def _shutdown_eligible(self):
        if self.arvados_node is None:
            return timestamp_fresh(self.cloud_node_start_time,
                                   self.node_stale_after)
        else:
            return (timestamp_fresh(arvados_node_mtime(self.arvados_node),
                                    self.poll_stale_after) and
                    (self.arvados_node['info'].get('slurm_state') == 'idle'))

    def consider_shutdown(self):
        next_opening = self._shutdowns.next_opening()
        if self._shutdowns.window_open():
            if self._shutdown_eligible():
                self._debug("Node %s suggesting shutdown.", self.cloud_node.id)
                _notify_subscribers(self._later, self.subscribers)
            else:
                self._debug("Node %s shutdown window open but node busy.",
                            self.cloud_node.id)
        else:
            self._debug("Node %s shutdown window closed.  Next at %s.",
                        self.cloud_node.id, time.ctime(next_opening))
        if self.last_shutdown_opening != next_opening:
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
