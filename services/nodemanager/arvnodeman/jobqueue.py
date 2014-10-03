#!/usr/bin/env python

import arvados.errors as arverror

from . import clientactor

class ServerCalculator(object):
    class SizeWrapper(object):
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


    def __init__(self, server_list, max_nodes=None):
        self.sizes = [self.SizeWrapper(s, **kws) for s, kws in server_list]
        self.sizes.sort(key=lambda s: s.price)
        self.max_nodes = max_nodes or float("inf")

    @staticmethod
    def coerce_int(x, fallback):
        try:
            return int(x)
        except (TypeError, ValueError):
            return fallback

    def size_for_constraints(self, constraints):
        want_value = lambda key: self.coerce_int(constraints.get(key), 0)
        wants = {'cores': want_value('min_cores_per_node'),
                 'ram': want_value('min_ram_mb_per_node'),
                 'scratch': want_value('min_scratch_mb_per_node')}
        for size in self.sizes:
            if size.meets_constraints(**wants):
                return size
        return None

    def servers_for_queue(self, queue):
        servers = []
        for job in queue:
            constraints = job['runtime_constraints']
            want_count = self.coerce_int(constraints.get('min_nodes'), 1)
            size = self.size_for_constraints(constraints)
            if (want_count < self.max_nodes) and (size is not None):
                servers.extend([size.real] * max(1, want_count))
        return servers


class JobQueueMonitorActor(clientactor.RemotePollLoopActor):
    CLIENT_ERRORS = (arverror.ApiError,)
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
                           ', '.join(s.name for s in server_list))
        return super(JobQueueMonitorActor, self)._got_response(server_list)
