#!/usr/bin/env python

from __future__ import absolute_import, print_function

import subprocess
import time

from . import \
    ComputeNodeSetupActor, ComputeNodeUpdateActor, ComputeNodeMonitorActor
from . import ComputeNodeShutdownActor as ShutdownActorBase

class ComputeNodeShutdownActor(ShutdownActorBase):
    SLURM_END_STATES = frozenset(['down\n', 'down*\n',
                                  'drain\n', 'drain*\n',
                                  'fail\n', 'fail*\n'])
    SLURM_DRAIN_STATES = frozenset(['drain\n', 'drng\n'])

    def on_start(self):
        arv_node = self._arvados_node()
        if arv_node is None:
            self._nodename = None
            return super(ComputeNodeShutdownActor, self).on_start()
        else:
            self._nodename = arv_node['hostname']
            self._logger.info("Draining SLURM node %s", self._nodename)
            self._later.issue_slurm_drain()

    def _set_node_state(self, state, *args):
        cmd = ['scontrol', 'update', 'NodeName=' + self._nodename,
               'State=' + state]
        cmd.extend(args)
        subprocess.check_output(cmd)

    def _get_slurm_state(self):
        return subprocess.check_output(['sinfo', '--noheader', '-o', '%t', '-n', self._nodename])

    # The following methods retry on OSError.  This is intended to mitigate bug
    # #6321 where fork() of node manager raises "OSError: [Errno 12] Cannot
    # allocate memory" resulting in the untimely death of the shutdown actor
    # and tends to result in node manager getting into a wedged state where it
    # won't allocate new nodes or shut down gracefully.  The underlying causes
    # of the excessive memory usage that result in the "Cannot allocate memory"
    # error are still being investigated.

    @ShutdownActorBase._retry((subprocess.CalledProcessError, OSError))
    def cancel_shutdown(self, reason):
        if self._nodename:
            if self._get_slurm_state() in self.SLURM_DRAIN_STATES:
                # Resume from "drng" or "drain"
                self._set_node_state('RESUME')
            else:
                # Node is in a state such as 'idle' or 'alloc' so don't
                # try to resume it because that will just raise an error.
                pass
        return super(ComputeNodeShutdownActor, self).cancel_shutdown(reason)

    @ShutdownActorBase._retry((subprocess.CalledProcessError, OSError))
    @ShutdownActorBase._stop_if_window_closed
    def issue_slurm_drain(self):
        self._set_node_state('DRAIN', 'Reason=Node Manager shutdown')
        self._logger.info("Waiting for SLURM node %s to drain", self._nodename)
        self._later.await_slurm_drain()

    @ShutdownActorBase._retry((subprocess.CalledProcessError, OSError))
    @ShutdownActorBase._stop_if_window_closed
    def await_slurm_drain(self):
        output = self._get_slurm_state()
        if output in self.SLURM_END_STATES:
            self._later.shutdown_node()
        else:
            self._timer.schedule(time.time() + 10,
                                 self._later.await_slurm_drain)
