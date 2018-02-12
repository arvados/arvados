#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import, print_function

import logging
import re
import subprocess

import arvados.util

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
        def __init__(self, real_size, node_mem_scaling, **kwargs):
            self.real = real_size
            for name in ['id', 'name', 'ram', 'disk', 'bandwidth', 'price',
                         'extra']:
                setattr(self, name, getattr(self.real, name))
            self.cores = kwargs.pop('cores')
            # libcloud disk sizes are in GB, Arvados/SLURM are in MB
            # multiply by 1000 instead of 1024 to err on low side
            if self.disk is None:
                self.disk = 0
            self.scratch = self.disk * 1000
            self.ram = int(self.ram * node_mem_scaling)
            for name, override in kwargs.iteritems():
                if not hasattr(self, name):
                    raise ValueError("unrecognized size field '%s'" % (name,))
                setattr(self, name, override)

            if self.price is None:
                raise ValueError("Required field 'price' is None")

        def meets_constraints(self, **kwargs):
            for name, want_value in kwargs.iteritems():
                have_value = getattr(self, name)
                if (have_value != 0) and (have_value < want_value):
                    return False
            return True


    def __init__(self, server_list, max_nodes=None, max_price=None,
                 node_mem_scaling=0.95):
        self.cloud_sizes = [self.CloudSizeWrapper(s, node_mem_scaling, **kws)
                            for s, kws in server_list]
        self.cloud_sizes.sort(key=lambda s: s.price)
        self.max_nodes = max_nodes or float('inf')
        self.max_price = max_price or float('inf')
        self.logger = logging.getLogger('arvnodeman.jobqueue')

        self.logger.info("Using cloud node sizes:")
        for s in self.cloud_sizes:
            self.logger.info(str(s.__dict__))

    @staticmethod
    def coerce_int(x, fallback):
        try:
            return int(x)
        except (TypeError, ValueError):
            return fallback

    def cloud_size_for_constraints(self, constraints):
        specified_size = constraints.get('instance_type')
        want_value = lambda key: self.coerce_int(constraints.get(key), 0)
        wants = {'cores': want_value('min_cores_per_node'),
                 'ram': want_value('min_ram_mb_per_node'),
                 'scratch': want_value('min_scratch_mb_per_node')}
        for size in self.cloud_sizes:
            if (size.meets_constraints(**wants) and
                (specified_size is None or size.id == specified_size)):
                    return size
        return None

    def servers_for_queue(self, queue):
        servers = []
        unsatisfiable_jobs = {}
        for job in queue:
            constraints = job['runtime_constraints']
            want_count = max(1, self.coerce_int(constraints.get('min_nodes'), 1))
            cloud_size = self.cloud_size_for_constraints(constraints)
            if cloud_size is None:
                unsatisfiable_jobs[job['uuid']] = (
                    "Constraints cannot be satisfied by any node type")
            elif (want_count > self.max_nodes):
                unsatisfiable_jobs[job['uuid']] = (
                    "Job's min_nodes constraint is greater than the configured "
                    "max_nodes (%d)" % self.max_nodes)
            elif (want_count*cloud_size.price <= self.max_price):
                servers.extend([cloud_size.real] * want_count)
            else:
                unsatisfiable_jobs[job['uuid']] = (
                    "Job's price (%d) is above system's max_price "
                    "limit (%d)" % (want_count*cloud_size.price, self.max_price))
        return (servers, unsatisfiable_jobs)

    def cheapest_size(self):
        return self.cloud_sizes[0]

    def find_size(self, sizeid):
        for s in self.cloud_sizes:
            if s.id == sizeid:
                return s
        return None


class JobQueueMonitorActor(clientactor.RemotePollLoopActor):
    """Actor to generate server wishlists from the job queue.

    This actor regularly polls Arvados' job queue, and uses the provided
    ServerCalculator to turn that into a list of requested node sizes.  That
    list is sent to subscribers on every poll.
    """

    CLIENT_ERRORS = ARVADOS_ERRORS

    def __init__(self, client, timer_actor, server_calc,
                 jobs_queue, slurm_queue, *args, **kwargs):
        super(JobQueueMonitorActor, self).__init__(
            client, timer_actor, *args, **kwargs)
        self.jobs_queue = jobs_queue
        self.slurm_queue = slurm_queue
        self._calculator = server_calc

    @staticmethod
    def coerce_to_mb(x):
        v, u = x[:-1], x[-1]
        if u in ("M", "m"):
            return int(v)
        elif u in ("G", "g"):
            return float(v) * 2**10
        elif u in ("T", "t"):
            return float(v) * 2**20
        elif u in ("P", "p"):
            return float(v) * 2**30
        else:
            return int(x)

    def _send_request(self):
        queuelist = []
        if self.slurm_queue:
            # cpus, memory, tempory disk space, reason, job name
            squeue_out = subprocess.check_output(["squeue", "--state=PENDING", "--noheader", "--format=%c|%m|%d|%r|%j|%f"])
            for out in squeue_out.splitlines():
                try:
                    cpu, ram, disk, reason, jobname, features = out.split("|", 5)
                except ValueError:
                    self._logger.warning("ignored malformed line in squeue output: %r", out)
                    continue
                if '-dz642-' not in jobname:
                    continue
                if not re.search(r'ReqNodeNotAvail|Resources|Priority', reason):
                    continue

                for feature in features.split(','):
                    m = re.match(r'instancetype=(.*)', feature)
                    if not m:
                        continue
                    instance_type = m.group(1)
                    # Ignore cpu/ram/scratch requirements, bring up
                    # the requested node type.
                    queuelist.append({
                        "uuid": jobname,
                        "runtime_constraints": {
                            "instance_type": instance_type,
                        }
                    })
                    break
                else:
                    # No instance type specified. Choose a node type
                    # to suit cpu/ram/scratch requirements.
                    queuelist.append({
                        "uuid": jobname,
                        "runtime_constraints": {
                            "min_cores_per_node": cpu,
                            "min_ram_mb_per_node": self.coerce_to_mb(ram),
                            "min_scratch_mb_per_node": self.coerce_to_mb(disk)
                        }
                    })

        if self.jobs_queue:
            queuelist.extend(self._client.jobs().queue().execute()['items'])

        return queuelist

    def _got_response(self, queue):
        server_list, unsatisfiable_jobs = self._calculator.servers_for_queue(queue)
        # Cancel any job/container with unsatisfiable requirements, emitting
        # a log explaining why.
        for job_uuid, reason in unsatisfiable_jobs.iteritems():
            try:
                self._client.logs().create(body={
                    'object_uuid': job_uuid,
                    'event_type': 'stderr',
                    'properties': {'text': reason},
                }).execute()
                # Cancel the job depending on its type
                if arvados.util.container_uuid_pattern.match(job_uuid):
                    subprocess.check_call(['scancel', '--name='+job_uuid])
                elif arvados.util.job_uuid_pattern.match(job_uuid):
                    self._client.jobs().cancel(uuid=job_uuid).execute()
                else:
                    raise Exception('Unknown job type')
                self._logger.debug("Cancelled unsatisfiable job '%s'", job_uuid)
            except Exception as error:
                self._logger.error("Trying to cancel job '%s': %s",
                                   job_uuid,
                                   error)
        self._logger.debug("Calculated wishlist: %s",
                           ', '.join(s.name for s in server_list) or "(empty)")
        return super(JobQueueMonitorActor, self)._got_response(server_list)
