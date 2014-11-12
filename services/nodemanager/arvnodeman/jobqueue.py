#!/usr/bin/env python

from __future__ import absolute_import, print_function

import logging

from . import clientactor
from .config import ARVADOS_ERRORS

class ServerCalculator(object):
    """Generate cloud server wishlists from an Arvados job queue.

    Instantiate this class with a list of cloud node sizes you're willing to
    use, plus keyword overrides from the configuration.  Then you can pass
    job queues to servers_for_queue.  It will return a list of node sizes
    that would best satisfy the jobs, choosing the cheapest size that
    satisfies each job, and ignoring jobs that can't be satisfied.
    """

    class CloudSizeWrapper(object):
        def __init__(self, real_size, **kwargs):
            self.real = real_size
            for name in ['id', 'name', 'ram', 'disk', 'bandwidth', 'price',
                         'extra']:
                setattr(self, name, getattr(self.real, name))
            self.cores = kwargs.pop('cores')
            self.scratch = self.disk
            for name, override in kwargs.iteritems():
                if not hasattr(self, name):
                    raise ValueError("unrecognized size field '%s'" % (name,))
                setattr(self, name, override)

        def meets_constraints(self, **kwargs):
            for name, want_value in kwargs.iteritems():
                have_value = getattr(self, name)
                if (have_value != 0) and (have_value < want_value):
                    return False
            return True


    def __init__(self, server_list, min_nodes=0, max_nodes=None):
        self.cloud_sizes = [self.CloudSizeWrapper(s, **kws)
                            for s, kws in server_list]
        self.cloud_sizes.sort(key=lambda s: s.price)
        self.min_nodes = min_nodes
        self.max_nodes = max_nodes or float('inf')
        self.logger = logging.getLogger('arvnodeman.jobqueue')
        self.logged_jobs = set()

    @staticmethod
    def coerce_int(x, fallback):
        try:
            return int(x)
        except (TypeError, ValueError):
            return fallback

    def cloud_size_for_constraints(self, constraints):
        want_value = lambda key: self.coerce_int(constraints.get(key), 0)
        wants = {'cores': want_value('min_cores_per_node'),
                 'ram': want_value('min_ram_mb_per_node'),
                 'scratch': want_value('min_scratch_mb_per_node')}
        for size in self.cloud_sizes:
            if size.meets_constraints(**wants):
                return size
        return None

    def servers_for_queue(self, queue):
        servers = []
        seen_jobs = set()
        for job in queue:
            seen_jobs.add(job['uuid'])
            constraints = job['runtime_constraints']
            want_count = self.coerce_int(constraints.get('min_nodes'), 1)
            cloud_size = self.cloud_size_for_constraints(constraints)
            if cloud_size is None:
                if job['uuid'] not in self.logged_jobs:
                    self.logged_jobs.add(job['uuid'])
                    self.logger.debug("job %s not satisfiable", job['uuid'])
            elif (want_count <= self.max_nodes):
                servers.extend([cloud_size.real] * max(1, want_count))
        self.logged_jobs.intersection_update(seen_jobs)

        # Make sure the server queue has at least enough entries to
        # satisfy min_nodes.
        node_shortfall = self.min_nodes - len(servers)
        if node_shortfall > 0:
            basic_node = self.cloud_size_for_constraints({})
            servers.extend([basic_node.real] * node_shortfall)
        return servers


class JobQueueMonitorActor(clientactor.RemotePollLoopActor):
    """Actor to generate server wishlists from the job queue.

    This actor regularly polls Arvados' job queue, and uses the provided
    ServerCalculator to turn that into a list of requested node sizes.  That
    list is sent to subscribers on every poll.
    """

    CLIENT_ERRORS = ARVADOS_ERRORS
    LOGGER_NAME = 'arvnodeman.jobqueue'

    def __init__(self, client, timer_actor, server_calc, *args, **kwargs):
        super(JobQueueMonitorActor, self).__init__(
            client, timer_actor, *args, **kwargs)
        self._calculator = server_calc

    def _send_request(self):
        return self._client.jobs().queue().execute()['items']

    def _got_response(self, queue):
        server_list = self._calculator.servers_for_queue(queue)
        self._logger.debug("Sending server wishlist: %s",
                           ', '.join(s.name for s in server_list) or "(empty)")
        return super(JobQueueMonitorActor, self)._got_response(server_list)
