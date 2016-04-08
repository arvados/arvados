#!/usr/bin/env python

from __future__ import absolute_import, print_function

import subprocess
import time

from . import \
    ComputeNodeSetupActor, ComputeNodeUpdateActor
from . import ComputeNodeShutdownActor as ShutdownActorBase
from . import ComputeNodeMonitorActor as MonitorActorBase
from .. import RetryMixin

class SlurmMixin(object):
    SLURM_END_STATES = frozenset(['down\n', 'down*\n',
                                  'drain\n', 'drain*\n',
                                  'fail\n', 'fail*\n'])
    SLURM_DRAIN_STATES = frozenset(['drain\n', 'drng\n'])

    def _set_node_state(self, nodename, state, *args):
        cmd = ['scontrol', 'update', 'NodeName=' + nodename,
               'State=' + state]
        cmd.extend(args)
        subprocess.check_output(cmd)

    def _get_slurm_state(self, nodename):
        return subprocess.check_output(['sinfo', '--noheader', '-o', '%t', '-n', nodename])


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

    @RetryMixin._retry((subprocess.CalledProcessError,))
    def cancel_shutdown(self, reason):
        if self._nodename:
            if self._get_slurm_state(self._nodename) in self.SLURM_DRAIN_STATES:
                # Resume from "drng" or "drain"
                self._set_node_state(self._nodename, 'RESUME')
            else:
                # Node is in a state such as 'idle' or 'alloc' so don't
                # try to resume it because that will just raise an error.
                pass
        return super(ComputeNodeShutdownActor, self).cancel_shutdown(reason)

    @RetryMixin._retry((subprocess.CalledProcessError,))
    @ShutdownActorBase._stop_if_window_closed
    def issue_slurm_drain(self):
        self._set_node_state(self._nodename, 'DRAIN', 'Reason=Node Manager shutdown')
        self._logger.info("Waiting for SLURM node %s to drain", self._nodename)
        self._later.await_slurm_drain()

    @RetryMixin._retry((subprocess.CalledProcessError,))
    @ShutdownActorBase._stop_if_window_closed
    def await_slurm_drain(self):
        output = self._get_slurm_state(self._nodename)
        if output in self.SLURM_END_STATES:
            self._later.shutdown_node()
        else:
            self._timer.schedule(time.time() + 10,
                                 self._later.await_slurm_drain)


class ComputeNodeMonitorActor(SlurmMixin, MonitorActorBase):

    def shutdown_eligible(self):
        if self.arvados_node is not None:
            state = self._get_slurm_state(self.arvados_node['hostname'])
            # Automatically eligible for shutdown if it's down or failed, but
            # not drain to avoid a race condition with resume_node().
            if state in self.SLURM_END_STATES:
                if state in self.SLURM_DRAIN_STATES:
                    return "node is draining"
                else:
                    return True
        return super(ComputeNodeMonitorActor, self).shutdown_eligible()

    def resume_node(self):
        try:
            if (self.arvados_node is not None and
                self._get_slurm_state(self.arvados_node['hostname']) in self.SLURM_DRAIN_STATES):
                # Resume from "drng" or "drain"
                self._set_node_state(self.arvados_node['hostname'], 'RESUME')
        except Exception as error:
            self._logger.warn(
                "Exception reenabling node: %s", error, exc_info=error)
