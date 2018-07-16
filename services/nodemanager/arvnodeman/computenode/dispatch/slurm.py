#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import, print_function

import subprocess32 as subprocess
import time

from . import ComputeNodeMonitorActor
from . import ComputeNodeSetupActor as SetupActorBase
from . import ComputeNodeShutdownActor as ShutdownActorBase
from . import ComputeNodeUpdateActor as UpdateActorBase
from .. import RetryMixin

class SlurmMixin(object):
    SLURM_END_STATES = frozenset(['down\n', 'down*\n',
                                  'drain\n', 'drain*\n',
                                  'fail\n', 'fail*\n'])
    SLURM_DRAIN_STATES = frozenset(['drain\n', 'drng\n'])

    def _update_slurm_node(self, nodename, updates):
        cmd = ['scontrol', 'update', 'NodeName=' + nodename] + updates
        try:
            subprocess.check_output(cmd)
        except:
            self._logger.error(
                "SLURM update %r failed", cmd, exc_info=True)

    def _update_slurm_size_attrs(self, nodename, size):
        self._update_slurm_node(nodename, [
            'Weight=%i' % int(size.price * 1000),
            'Features=instancetype=' + size.id,
        ])

    def _get_slurm_state(self, nodename):
        return subprocess.check_output(['sinfo', '--noheader', '-o', '%t', '-n', nodename])


class ComputeNodeSetupActor(SlurmMixin, SetupActorBase):
    def create_cloud_node(self):
        hostname = self.arvados_node.get("hostname")
        if hostname:
            self._update_slurm_size_attrs(hostname, self.cloud_size)
        return super(ComputeNodeSetupActor, self).create_cloud_node()


class ComputeNodeShutdownActor(SlurmMixin, ShutdownActorBase):
    def on_start(self):
        arv_node = self._arvados_node()
        if arv_node is None:
            self._nodename = None
            return super(ComputeNodeShutdownActor, self).on_start()
        else:
            self._set_logger()
            self._nodename = arv_node['hostname']
            self._logger.info("Draining SLURM node %s", self._nodename)
            self._later.issue_slurm_drain()

    @RetryMixin._retry((subprocess.CalledProcessError, OSError))
    def cancel_shutdown(self, reason, try_resume=True):
        if self._nodename:
            if try_resume and self._get_slurm_state(self._nodename) in self.SLURM_DRAIN_STATES:
                # Resume from "drng" or "drain"
                self._update_slurm_node(self._nodename, ['State=RESUME'])
            else:
                # Node is in a state such as 'idle' or 'alloc' so don't
                # try to resume it because that will just raise an error.
                pass
        return super(ComputeNodeShutdownActor, self).cancel_shutdown(reason)

    @RetryMixin._retry((subprocess.CalledProcessError, OSError))
    def issue_slurm_drain(self):
        if self.cancel_reason is not None:
            return
        if self._nodename:
            self._update_slurm_node(self._nodename, [
                'State=DRAIN', 'Reason=Node Manager shutdown'])
            self._logger.info("Waiting for SLURM node %s to drain", self._nodename)
            self._later.await_slurm_drain()
        else:
            self._later.shutdown_node()

    @RetryMixin._retry((subprocess.CalledProcessError, OSError))
    def await_slurm_drain(self):
        if self.cancel_reason is not None:
            return
        output = self._get_slurm_state(self._nodename)
        if output in ("drng\n", "alloc\n", "drng*\n", "alloc*\n"):
            self._timer.schedule(time.time() + 10,
                                 self._later.await_slurm_drain)
        elif output in ("idle\n",):
            # Not in "drng" but idle, don't shut down
            self.cancel_shutdown("slurm state is %s" % output.strip(), try_resume=False)
        else:
            # any other state.
            self._later.shutdown_node()

    def _destroy_node(self):
        if self._nodename:
            self._update_slurm_node(self._nodename, [
                'State=DOWN', 'Reason=Node Manager shutdown'])
        super(ComputeNodeShutdownActor, self)._destroy_node()


class ComputeNodeUpdateActor(SlurmMixin, UpdateActorBase):
    def sync_node(self, cloud_node, arvados_node):
        """Keep SLURM's node properties up to date."""
        hostname = arvados_node.get("hostname")
        features = arvados_node.get("slurm_node_features", "").split(",")
        sizefeature = "instancetype=" + cloud_node.size.id
        if hostname and sizefeature not in features:
            # This probably means SLURM has restarted and lost our
            # dynamically configured node weights and features.
            self._update_slurm_size_attrs(hostname, cloud_node.size)
        return super(ComputeNodeUpdateActor, self).sync_node(
            cloud_node, arvados_node)
