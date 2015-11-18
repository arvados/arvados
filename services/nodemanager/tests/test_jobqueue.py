#!/usr/bin/env python

from __future__ import absolute_import, print_function

import unittest

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
        self.assertEqual([], servcalc.servers_for_queue([]))

    def test_easy_server_count(self):
        servcalc = self.make_calculator([1])
        servlist = self.calculate(servcalc, {'min_nodes': 3})
        self.assertEqual(3, len(servlist))

    def test_implicit_server_count(self):
        servcalc = self.make_calculator([1])
        servlist = self.calculate(servcalc, {}, {'min_nodes': 3})
        self.assertEqual(4, len(servlist))

    def test_bad_min_nodes_override(self):
        servcalc = self.make_calculator([1])
        servlist = self.calculate(servcalc,
                                  {'min_nodes': -2}, {'min_nodes': 'foo'})
        self.assertEqual(2, len(servlist))

    def test_ignore_unsatisfiable_jobs(self):
        servcalc = self.make_calculator([1], max_nodes=9)
        servlist = self.calculate(servcalc,
                                  {'min_cores_per_node': 2},
                                  {'min_ram_mb_per_node': 256},
                                  {'min_nodes': 6},
                                  {'min_nodes': 12},
                                  {'min_scratch_mb_per_node': 200})
        self.assertEqual(6, len(servlist))

    def test_ignore_too_expensive_jobs(self):
        servcalc = self.make_calculator([1, 2], max_nodes=12, max_price=6)
        servlist = self.calculate(servcalc,
                                  {'min_cores_per_node': 1, 'min_nodes': 6})
        self.assertEqual(6, len(servlist))

        servlist = self.calculate(servcalc,
                                  {'min_cores_per_node': 2, 'min_nodes': 6})
        self.assertEqual(0, len(servlist))

    def test_job_requesting_max_nodes_accepted(self):
        servcalc = self.make_calculator([1], max_nodes=4)
        servlist = self.calculate(servcalc, {'min_nodes': 4})
        self.assertEqual(4, len(servlist))

    def test_cheapest_size(self):
        servcalc = self.make_calculator([2, 4, 1, 3])
        self.assertEqual(testutil.MockSize(1), servcalc.cheapest_size())

    def test_next_biggest(self):
        servcalc = self.make_calculator([1, 2, 4, 8])
        servlist = self.calculate(servcalc,
                                  {'min_cores_per_node': 3},
                                  {'min_cores_per_node': 6})
        self.assertEqual([servcalc.cloud_sizes[2].id,
                          servcalc.cloud_sizes[3].id],
                         [s.id for s in servlist])

    def test_multiple_sizes(self):
        servcalc = self.make_calculator([1, 2])
        servlist = self.calculate(servcalc,
                                  {'min_cores_per_node': 2},
                                  {'min_cores_per_node': 1},
                                  {'min_cores_per_node': 1})
        self.assertEqual([servcalc.cloud_sizes[1].id,
                          servcalc.cloud_sizes[0].id,
                          servcalc.cloud_sizes[0].id],
                         [s.id for s in servlist])

        servlist = self.calculate(servcalc,
                                  {'min_cores_per_node': 1},
                                  {'min_cores_per_node': 2},
                                  {'min_cores_per_node': 1})
        self.assertEqual([servcalc.cloud_sizes[0].id,
                          servcalc.cloud_sizes[1].id,
                          servcalc.cloud_sizes[0].id],
                         [s.id for s in servlist])

        servlist = self.calculate(servcalc,
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
            return [testutil.MockSize(n) for n in queue]


    def build_monitor(self, side_effect, *args, **kwargs):
        super(JobQueueMonitorActorTestCase, self).build_monitor(*args, **kwargs)
        self.client.jobs().queue().execute.side_effect = side_effect

    def test_subscribers_get_server_lists(self):
        self.build_monitor([{'items': [1, 2]}], self.MockCalculator())
        self.monitor.subscribe(self.subscriber).get(self.TIMEOUT)
        self.stop_proxy(self.monitor)
        self.subscriber.assert_called_with([testutil.MockSize(1),
                                            testutil.MockSize(2)])


if __name__ == '__main__':
    unittest.main()
