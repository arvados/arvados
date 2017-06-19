#!/usr/bin/env python

from __future__ import absolute_import, print_function

import subprocess
import time

from . import \
    ComputeNodeSetupActor, ComputeNodeMonitorActor
from . import ComputeNodeShutdownActor as ShutdownActorBase
from . import ComputeNodeUpdateActor as UpdateActorBase
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

    @RetryMixin._retry((subprocess.CalledProcessError, OSError))
    def cancel_shutdown(self, reason, try_resume=True):
        if self._nodename:
            if try_resume and self._get_slurm_state(self._nodename) in self.SLURM_DRAIN_STATES:
                # Resume from "drng" or "drain"
                self._set_node_state(self._nodename, 'RESUME')
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
            self._set_node_state(self._nodename, 'DRAIN', 'Reason=Node Manager shutdown')
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
        elif output in ("idle\n"):
            # Not in "drng" but idle, don't shut down
            self.cancel_shutdown("slurm state is %s" % output.strip(), try_resume=False)
        else:
            # any other state.
            self._later.shutdown_node()

    def _destroy_node(self):
        if self._nodename:
            self._set_node_state(self._nodename, 'DOWN', 'Reason=Node Manager shutdown')
        super(ComputeNodeShutdownActor, self)._destroy_node()


class ComputeNodeUpdateActor(UpdateActorBase):
    def sync_node(self, cloud_node, arvados_node):
        if arvados_node.get("hostname"):
            try:
                subprocess.check_output(['scontrol', 'update', 'NodeName=' + arvados_node["hostname"], 'Weight=%i' % int(cloud_node.size.price * 1000)])
            except:
                self._logger.error("Unable to set slurm node weight.", exc_info=True)
        return super(ComputeNodeUpdateActor, self).sync_node(cloud_node, arvados_node)
