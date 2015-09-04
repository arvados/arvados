#!/usr/bin/env python

from __future__ import absolute_import, print_function

import subprocess
import unittest

import mock

import arvnodeman.computenode.dispatch.slurm as slurm_dispatch
from . import testutil
from .test_computenode_dispatch import ComputeNodeShutdownActorMixin

@mock.patch('subprocess.check_output')
class SLURMComputeNodeShutdownActorTestCase(ComputeNodeShutdownActorMixin,
                                            unittest.TestCase):
    ACTOR_CLASS = slurm_dispatch.ComputeNodeShutdownActor

    def check_slurm_got_args(self, proc_mock, *args):
        self.assertTrue(proc_mock.called)
        slurm_cmd = proc_mock.call_args[0][0]
        for s in args:
            self.assertIn(s, slurm_cmd)

    def check_success_after_reset(self, proc_mock, end_state='drain\n'):
        self.make_mocks(arvados_node=testutil.arvados_node_mock(63))
        self.make_actor()
        self.check_success_flag(None, 0)
        self.check_success_flag(None, 0)
        # Order is critical here: if the mock gets called when no return value
        # or side effect is set, we may invoke a real subprocess.
        proc_mock.return_value = end_state
        proc_mock.side_effect = None
        self.check_success_flag(True, 3)
        self.check_slurm_got_args(proc_mock, 'compute63')

    def make_wait_state_test(start_state='drng\n', end_state='drain\n'):
        def test(self, proc_mock):
            proc_mock.return_value = start_state
            self.check_success_after_reset(proc_mock, end_state)
        return test

    for wait_state in ['alloc\n', 'drng\n', 'idle*\n']:
        locals()['test_wait_while_' + wait_state.strip()
                 ] = make_wait_state_test(start_state=wait_state)

    for end_state in ['down\n', 'down*\n', 'drain\n', 'fail\n']:
        locals()['test_wait_until_' + end_state.strip()
                 ] = make_wait_state_test(end_state=end_state)

    def test_retry_failed_slurm_calls(self, proc_mock):
        proc_mock.side_effect = subprocess.CalledProcessError(1, ["mock"])
        self.check_success_after_reset(proc_mock)

    def test_slurm_bypassed_when_no_arvados_node(self, proc_mock):
        # Test we correctly handle a node that failed to bootstrap.
        proc_mock.return_value = 'idle\n'
        self.make_actor()
        self.check_success_flag(True)
        self.assertFalse(proc_mock.called)

    def test_node_undrained_when_shutdown_window_closes(self, proc_mock):
        proc_mock.return_value = 'alloc\n'
        self.make_mocks(arvados_node=testutil.arvados_node_mock(job_uuid=True))
        self.make_actor()
        self.check_success_flag(False, 2)
        self.check_slurm_got_args(proc_mock, 'NodeName=compute99',
                                  'State=RESUME')

    def test_arvados_node_cleaned_after_shutdown(self, proc_mock):
        proc_mock.return_value = 'drain\n'
        super(SLURMComputeNodeShutdownActorTestCase,
              self).test_arvados_node_cleaned_after_shutdown()
