#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import, print_function

import subprocess32 as subprocess
import time
import unittest

import mock

import arvnodeman.computenode.dispatch.slurm as slurm_dispatch
from . import testutil
from .test_computenode_dispatch import \
    ComputeNodeShutdownActorMixin, \
    ComputeNodeSetupActorTestCase, \
    ComputeNodeUpdateActorTestCase

@mock.patch('subprocess32.check_output')
class SLURMComputeNodeShutdownActorTestCase(ComputeNodeShutdownActorMixin,
                                            unittest.TestCase):
    ACTOR_CLASS = slurm_dispatch.ComputeNodeShutdownActor

    def check_slurm_got_args(self, proc_mock, *args):
        self.assertTrue(proc_mock.called)
        slurm_cmd = proc_mock.call_args[0][0]
        for s in args:
            self.assertIn(s, slurm_cmd)

    def check_success_after_reset(self, proc_mock, end_state='drain\n', timer=False):
        self.make_mocks(arvados_node=testutil.arvados_node_mock(63))
        if not timer:
            self.timer = testutil.MockTimer(False)
        self.make_actor()
        self.check_success_flag(None, 0)
        # At this point, 1st try should have happened.

        self.timer.deliver()
        self.check_success_flag(None, 0)
        # At this point, 2nd try should have happened.

        # Order is critical here: if the mock gets called when no return value
        # or side effect is set, we may invoke a real subprocess.
        proc_mock.return_value = end_state
        proc_mock.side_effect = None

        # 3rd try
        self.timer.deliver()

        self.check_success_flag(True, 3)
        self.check_slurm_got_args(proc_mock, 'NodeName=compute63')

    def make_wait_state_test(start_state='drng\n', end_state='drain\n'):
        def test(self, proc_mock):
            proc_mock.return_value = start_state
            self.check_success_after_reset(proc_mock, end_state)
        return test

    for wait_state in ['alloc\n', 'drng\n']:
        locals()['test_wait_while_' + wait_state.strip()
                 ] = make_wait_state_test(start_state=wait_state)

    for end_state in ['idle*\n', 'down\n', 'down*\n', 'drain\n', 'fail\n']:
        locals()['test_wait_until_' + end_state.strip()
                 ] = make_wait_state_test(end_state=end_state)

    def test_retry_failed_slurm_calls(self, proc_mock):
        proc_mock.side_effect = subprocess.CalledProcessError(1, ["mock"])
        self.check_success_after_reset(proc_mock)

    def test_slurm_bypassed_when_no_arvados_node(self, proc_mock):
        # Test we correctly handle a node that failed to bootstrap.
        proc_mock.return_value = 'down\n'
        self.make_actor(start_time=0)
        self.check_success_flag(True)
        self.assertFalse(proc_mock.called)

    def test_node_resumed_when_shutdown_cancelled(self, proc_mock):
        try:
            proc_mock.side_effect = iter(['', 'drng\n', 'drng\n', ''])
            self.make_mocks(arvados_node=testutil.arvados_node_mock(job_uuid=True))
            self.timer = testutil.MockTimer(False)
            self.make_actor()
            self.busywait(lambda: proc_mock.call_args is not None)
            self.shutdown_actor.cancel_shutdown("test")
            self.check_success_flag(False, 2)
            self.assertEqual(proc_mock.call_args_list[0], mock.call(['scontrol', 'update', 'NodeName=compute99', 'State=DRAIN', 'Reason=Node Manager shutdown']))
            self.assertEqual(proc_mock.call_args_list[-1], mock.call(['scontrol', 'update', 'NodeName=compute99', 'State=RESUME']))

        finally:
            self.shutdown_actor.actor_ref.stop()

    def test_cancel_shutdown_retry(self, proc_mock):
        proc_mock.side_effect = iter([OSError, 'drain\n', OSError, 'idle\n', 'idle\n'])
        self.make_mocks(arvados_node=testutil.arvados_node_mock(job_uuid=True))
        self.make_actor()
        self.check_success_flag(False, 5)

    def test_issue_slurm_drain_retry(self, proc_mock):
        proc_mock.side_effect = iter([OSError, OSError, 'drng\n', 'drain\n'])
        self.check_success_after_reset(proc_mock, timer=False)

    def test_arvados_node_cleaned_after_shutdown(self, proc_mock):
        proc_mock.return_value = 'drain\n'
        super(SLURMComputeNodeShutdownActorTestCase,
              self).test_arvados_node_cleaned_after_shutdown()

    def test_cancellable_shutdown(self, proc_mock):
        proc_mock.return_value = 'other\n'
        super(SLURMComputeNodeShutdownActorTestCase,
              self).test_cancellable_shutdown()

    def test_uncancellable_shutdown(self, proc_mock):
        proc_mock.return_value = 'other\n'
        super(SLURMComputeNodeShutdownActorTestCase,
              self).test_uncancellable_shutdown()

@mock.patch('subprocess32.check_output')
class SLURMComputeNodeUpdateActorTestCase(ComputeNodeUpdateActorTestCase):
    ACTOR_CLASS = slurm_dispatch.ComputeNodeUpdateActor

    def test_update_node_weight(self, check_output):
        self.make_actor()
        cloud_node = testutil.cloud_node_mock()
        arv_node = testutil.arvados_node_mock()
        self.updater.sync_node(cloud_node, arv_node).get(self.TIMEOUT)
        check_output.assert_called_with(['scontrol', 'update', 'NodeName=compute99', 'Weight=99000', 'Features=instancetype=z99.test'])

class SLURMComputeNodeSetupActorTestCase(ComputeNodeSetupActorTestCase):
    ACTOR_CLASS = slurm_dispatch.ComputeNodeSetupActor

    @mock.patch('subprocess32.check_output')
    def test_update_node_features(self, check_output):
        # `scontrol update` happens only if the Arvados node record
        # has a hostname. ComputeNodeSetupActorTestCase.make_mocks
        # uses mocks with scrubbed hostnames, so we override with the
        # default testutil.arvados_node_mock.
        self.make_mocks(arvados_effect=[testutil.arvados_node_mock()])
        self.make_actor()
        self.wait_for_assignment(self.setup_actor, 'cloud_node')
        check_output.assert_called_with(['scontrol', 'update', 'NodeName=compute99', 'Weight=1000', 'Features=instancetype=z1.test'])

    @mock.patch('subprocess32.check_output')
    def test_failed_arvados_calls_retried(self, check_output):
        super(SLURMComputeNodeSetupActorTestCase, self).test_failed_arvados_calls_retried()

    @mock.patch('subprocess32.check_output')
    def test_subscribe(self, check_output):
        super(SLURMComputeNodeSetupActorTestCase, self).test_subscribe()

    @mock.patch('subprocess32.check_output')
    def test_creation_with_arvados_node(self, check_output):
        super(SLURMComputeNodeSetupActorTestCase, self).test_creation_with_arvados_node()
