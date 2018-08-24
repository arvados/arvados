#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import, print_function

import unittest
import mock

import arvnodeman.jobqueue as jobqueue
from . import testutil

class ServerCalculatorTestCase(unittest.TestCase):
    def make_calculator(self, factors, **kwargs):
        return jobqueue.ServerCalculator(
            [(testutil.MockSize(n), {'cores': n}) for n in factors], **kwargs)

    def calculate(self, servcalc, *constraints):
        return servcalc.servers_for_queue(
            [{'uuid': 'zzzzz-jjjjj-{:015x}'.format(index),
              'runtime_constraints': cdict}
             for index, cdict in enumerate(constraints)])

    def test_empty_queue_needs_no_servers(self):
        servcalc = self.make_calculator([1])
        self.assertEqual(([], {}), servcalc.servers_for_queue([]))

    def test_easy_server_count(self):
        servcalc = self.make_calculator([1])
        servlist, _ = self.calculate(servcalc, {'min_nodes': 3})
        self.assertEqual(3, len(servlist))

    def test_default_5pct_ram_value_decrease(self):
        servcalc = self.make_calculator([1])
        servlist, _ = self.calculate(servcalc, {'min_ram_mb_per_node': 128})
        self.assertEqual(0, len(servlist))
        servlist, _ = self.calculate(servcalc, {'min_ram_mb_per_node': 121})
        self.assertEqual(1, len(servlist))

    def test_custom_node_mem_scaling_factor(self):
        # Simulate a custom 'node_mem_scaling' config parameter by passing
        # the value to ServerCalculator
        servcalc = self.make_calculator([1], node_mem_scaling=0.5)
        servlist, _ = self.calculate(servcalc, {'min_ram_mb_per_node': 128})
        self.assertEqual(0, len(servlist))
        servlist, _ = self.calculate(servcalc, {'min_ram_mb_per_node': 64})
        self.assertEqual(1, len(servlist))

    def test_implicit_server_count(self):
        servcalc = self.make_calculator([1])
        servlist, _ = self.calculate(servcalc, {}, {'min_nodes': 3})
        self.assertEqual(4, len(servlist))

    def test_bad_min_nodes_override(self):
        servcalc = self.make_calculator([1])
        servlist, _ = self.calculate(servcalc,
                                     {'min_nodes': -2}, {'min_nodes': 'foo'})
        self.assertEqual(2, len(servlist))

    def test_ignore_and_return_unsatisfiable_jobs(self):
        servcalc = self.make_calculator([1], max_nodes=9)
        servlist, u_jobs = self.calculate(servcalc,
                                          {'min_cores_per_node': 2},
                                          {'min_ram_mb_per_node': 256},
                                          {'min_nodes': 6},
                                          {'min_nodes': 12},
                                          {'min_scratch_mb_per_node': 300000})
        self.assertEqual(6, len(servlist))
        # Only unsatisfiable jobs are returned on u_jobs
        self.assertIn('zzzzz-jjjjj-000000000000000', u_jobs.keys())
        self.assertIn('zzzzz-jjjjj-000000000000001', u_jobs.keys())
        self.assertNotIn('zzzzz-jjjjj-000000000000002', u_jobs.keys())
        self.assertIn('zzzzz-jjjjj-000000000000003', u_jobs.keys())
        self.assertIn('zzzzz-jjjjj-000000000000004', u_jobs.keys())

    def test_ignore_too_expensive_jobs(self):
        servcalc = self.make_calculator([1, 2], max_nodes=12, max_price=6)
        servlist, _ = self.calculate(servcalc,
                                     {'min_cores_per_node': 1, 'min_nodes': 6})
        self.assertEqual(6, len(servlist))

        servlist, _ = self.calculate(servcalc,
                                     {'min_cores_per_node': 2, 'min_nodes': 6})
        self.assertEqual(0, len(servlist))

    def test_job_requesting_max_nodes_accepted(self):
        servcalc = self.make_calculator([1], max_nodes=4)
        servlist, _ = self.calculate(servcalc, {'min_nodes': 4})
        self.assertEqual(4, len(servlist))

    def test_cheapest_size(self):
        servcalc = self.make_calculator([2, 4, 1, 3])
        self.assertEqual(testutil.MockSize(1), servcalc.cheapest_size())

    def test_next_biggest(self):
        servcalc = self.make_calculator([1, 2, 4, 8])
        servlist, _ = self.calculate(servcalc,
                                     {'min_cores_per_node': 3},
                                     {'min_cores_per_node': 6})
        self.assertEqual([servcalc.cloud_sizes[2].id,
                          servcalc.cloud_sizes[3].id],
                         [s.id for s in servlist])

    def test_multiple_sizes(self):
        servcalc = self.make_calculator([1, 2])
        servlist, _ = self.calculate(servcalc,
                                     {'min_cores_per_node': 2},
                                     {'min_cores_per_node': 1},
                                     {'min_cores_per_node': 1})
        self.assertEqual([servcalc.cloud_sizes[1].id,
                          servcalc.cloud_sizes[0].id,
                          servcalc.cloud_sizes[0].id],
                         [s.id for s in servlist])

        servlist, _ = self.calculate(servcalc,
                                     {'min_cores_per_node': 1},
                                     {'min_cores_per_node': 2},
                                     {'min_cores_per_node': 1})
        self.assertEqual([servcalc.cloud_sizes[0].id,
                          servcalc.cloud_sizes[1].id,
                          servcalc.cloud_sizes[0].id],
                         [s.id for s in servlist])

        servlist, _ = self.calculate(servcalc,
                                     {'min_cores_per_node': 1},
                                     {'min_cores_per_node': 1},
                                     {'min_cores_per_node': 2})
        self.assertEqual([servcalc.cloud_sizes[0].id,
                          servcalc.cloud_sizes[0].id,
                          servcalc.cloud_sizes[1].id],
                         [s.id for s in servlist])



class JobQueueMonitorActorTestCase(testutil.RemotePollLoopActorTestMixin,
                                   unittest.TestCase):
    TEST_CLASS = jobqueue.JobQueueMonitorActor


    class MockCalculator(object):
        @staticmethod
        def servers_for_queue(queue):
            return ([testutil.MockSize(n) for n in queue], {})


    class MockCalculatorUnsatisfiableJobs(object):
        @staticmethod
        def servers_for_queue(queue):
            return ([], {k["uuid"]: "Unsatisfiable job mock" for k in queue})


    def build_monitor(self, side_effect, *args, **kwargs):
        super(JobQueueMonitorActorTestCase, self).build_monitor(*args, **kwargs)
        self.client.jobs().queue().execute.side_effect = side_effect

    @mock.patch("subprocess32.check_call")
    @mock.patch("subprocess32.check_output")
    def test_unsatisfiable_jobs(self, mock_squeue, mock_scancel):
        job_uuid = 'zzzzz-8i9sb-zzzzzzzzzzzzzzz'
        container_uuid = 'yyyyy-dz642-yyyyyyyyyyyyyyy'
        mock_squeue.return_value = "1|1024|0|(Resources)|" + container_uuid + "||1234567890\n"

        self.build_monitor([{'items': [{'uuid': job_uuid}]}],
                           self.MockCalculatorUnsatisfiableJobs(), True, True)
        self.monitor.subscribe(self.subscriber).get(self.TIMEOUT)
        self.monitor.ping().get(self.TIMEOUT)
        self.stop_proxy(self.monitor)
        self.client.jobs().cancel.assert_called_with(uuid=job_uuid)
        mock_scancel.assert_called_with(['scancel', '--name='+container_uuid])

    @mock.patch("subprocess32.check_output")
    def test_subscribers_get_server_lists(self, mock_squeue):
        mock_squeue.return_value = ""

        self.build_monitor([{'items': [1, 2]}], self.MockCalculator(), True, True)
        self.monitor.subscribe(self.subscriber).get(self.TIMEOUT)
        self.stop_proxy(self.monitor)
        self.subscriber.assert_called_with([testutil.MockSize(1),
                                            testutil.MockSize(2)])

    @mock.patch("subprocess32.check_output")
    def test_squeue_server_list(self, mock_squeue):
        mock_squeue.return_value = """1|1024|0|(Resources)|zzzzz-dz642-zzzzzzzzzzzzzzy|(null)|1234567890
2|1024|0|(Resources)|zzzzz-dz642-zzzzzzzzzzzzzzz|(null)|1234567890
"""

        super(JobQueueMonitorActorTestCase, self).build_monitor(jobqueue.ServerCalculator(
            [(testutil.MockSize(n), {'cores': n, 'ram': n*1024, 'scratch': n}) for n in range(1, 3)]),
                                                                True, True)
        self.monitor.subscribe(self.subscriber).get(self.TIMEOUT)
        self.stop_proxy(self.monitor)
        self.subscriber.assert_called_with([testutil.MockSize(1),
                                            testutil.MockSize(2)])

    @mock.patch("subprocess32.check_output")
    def test_squeue_server_list_suffix(self, mock_squeue):
        mock_squeue.return_value = """1|1024M|0|(ReqNodeNotAvail, UnavailableNodes:compute123)|zzzzz-dz642-zzzzzzzzzzzzzzy|(null)|1234567890
1|2G|0|(ReqNodeNotAvail)|zzzzz-dz642-zzzzzzzzzzzzzzz|(null)|1234567890
"""

        super(JobQueueMonitorActorTestCase, self).build_monitor(jobqueue.ServerCalculator(
            [(testutil.MockSize(n), {'cores': n, 'ram': n*1024, 'scratch': n}) for n in range(1, 3)]),
                                                                True, True)
        self.monitor.subscribe(self.subscriber).get(self.TIMEOUT)
        self.stop_proxy(self.monitor)
        self.subscriber.assert_called_with([testutil.MockSize(1),
                                            testutil.MockSize(2)])

    @mock.patch("subprocess32.check_output")
    def test_squeue_server_list_instancetype_constraint(self, mock_squeue):
        mock_squeue.return_value = """1|1024|0|(Resources)|zzzzz-dz642-zzzzzzzzzzzzzzy|instancetype=z2.test|1234567890\n"""
        super(JobQueueMonitorActorTestCase, self).build_monitor(jobqueue.ServerCalculator(
            [(testutil.MockSize(n), {'cores': n, 'ram': n*1024, 'scratch': n}) for n in range(1, 3)]),
                                                                True, True)
        self.monitor.subscribe(self.subscriber).get(self.TIMEOUT)
        self.stop_proxy(self.monitor)
        self.subscriber.assert_called_with([testutil.MockSize(2)])

    def test_coerce_to_mb(self):
        self.assertEqual(1, jobqueue.JobQueueMonitorActor.coerce_to_mb("1"))
        self.assertEqual(512, jobqueue.JobQueueMonitorActor.coerce_to_mb("512"))
        self.assertEqual(512, jobqueue.JobQueueMonitorActor.coerce_to_mb("512M"))
        self.assertEqual(1024, jobqueue.JobQueueMonitorActor.coerce_to_mb("1024M"))
        self.assertEqual(1024, jobqueue.JobQueueMonitorActor.coerce_to_mb("1G"))
        self.assertEqual(1536, jobqueue.JobQueueMonitorActor.coerce_to_mb("1.5G"))
        self.assertEqual(2048, jobqueue.JobQueueMonitorActor.coerce_to_mb("2G"))
        self.assertEqual(1025, jobqueue.JobQueueMonitorActor.coerce_to_mb("1025M"))
        self.assertEqual(1048576, jobqueue.JobQueueMonitorActor.coerce_to_mb("1T"))
        self.assertEqual(1572864, jobqueue.JobQueueMonitorActor.coerce_to_mb("1.5T"))
        self.assertEqual(1073741824, jobqueue.JobQueueMonitorActor.coerce_to_mb("1P"))
        self.assertEqual(1610612736, jobqueue.JobQueueMonitorActor.coerce_to_mb("1.5P"))
        self.assertEqual(0, jobqueue.JobQueueMonitorActor.coerce_to_mb("0"))
        self.assertEqual(0, jobqueue.JobQueueMonitorActor.coerce_to_mb("0M"))
        self.assertEqual(0, jobqueue.JobQueueMonitorActor.coerce_to_mb("0G"))


if __name__ == '__main__':
    unittest.main()
