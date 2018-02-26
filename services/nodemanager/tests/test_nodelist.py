#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import, print_function

import unittest
import mock

import arvnodeman.nodelist as nodelist
from libcloud.compute.base import NodeSize
from . import testutil

class ArvadosNodeListMonitorActorTestCase(testutil.RemotePollLoopActorTestMixin,
                                          unittest.TestCase):
    TEST_CLASS = nodelist.ArvadosNodeListMonitorActor

    def build_monitor(self, side_effect, *args, **kwargs):
        super(ArvadosNodeListMonitorActorTestCase, self).build_monitor(
            *args, **kwargs)
        self.client.nodes().list().execute.side_effect = side_effect

    @mock.patch("subprocess.check_output")
    def test_uuid_is_subscription_key(self, sinfo_mock):
        sinfo_mock.return_value = ""
        node = testutil.arvados_node_mock()
        self.build_monitor([{
            'items': [node],
            'items_available': 1,
            'offset': 0
        }, {
            'items': [],
            'items_available': 1,
            'offset': 1
        }])
        self.monitor.subscribe_to(node['uuid'],
                                  self.subscriber).get(self.TIMEOUT)
        self.stop_proxy(self.monitor)
        self.subscriber.assert_called_with(node)
        self.assertEqual("down", node["crunch_worker_state"])

    @mock.patch("subprocess.check_output")
    def test_update_from_sinfo(self, sinfo_mock):
        sinfo_mock.return_value = """compute1|idle|instancetype=a1.test
compute2|alloc|(null)
notarvados12345|idle|(null)
"""
        nodeIdle = testutil.arvados_node_mock(node_num=1)
        nodeBusy = testutil.arvados_node_mock(node_num=2)
        nodeMissing = testutil.arvados_node_mock(node_num=99)
        self.build_monitor([{
            'items': [nodeIdle, nodeBusy, nodeMissing],
            'items_available': 1,
            'offset': 0
        }, {
            'items': [],
            'items_available': 1,
            'offset': 1
        }])
        self.monitor.subscribe_to(nodeMissing['uuid'],
                                  self.subscriber).get(self.TIMEOUT)
        self.stop_proxy(self.monitor)
        self.subscriber.assert_called_with(nodeMissing)

        self.assertEqual("idle", nodeIdle["crunch_worker_state"])
        self.assertEqual("busy", nodeBusy["crunch_worker_state"])
        self.assertEqual("down", nodeMissing["crunch_worker_state"])

        self.assertEqual("instancetype=a1.test", nodeIdle["slurm_node_features"])
        self.assertEqual("", nodeBusy["slurm_node_features"])
        self.assertEqual("", nodeMissing["slurm_node_features"])


class CloudNodeListMonitorActorTestCase(testutil.RemotePollLoopActorTestMixin,
                                        unittest.TestCase):
    TEST_CLASS = nodelist.CloudNodeListMonitorActor

    class MockNode(object):
        def __init__(self, count):
            self.id = str(count)
            self.name = 'test{}.example.com'.format(count)
            self.private_ips = ['10.0.0.{}'.format(count)]
            self.public_ips = []
            self.size = testutil.MockSize(1)
            self.state = 0


    def build_monitor(self, side_effect, *args, **kwargs):
        super(CloudNodeListMonitorActorTestCase, self).build_monitor(
            *args, **kwargs)
        self.client.list_nodes.side_effect = side_effect

    def test_id_is_subscription_key(self):
        node = self.MockNode(1)
        mock_calc = mock.MagicMock()
        mock_calc.find_size.return_value = testutil.MockSize(2)
        self.build_monitor([[node]], mock_calc)
        self.monitor.subscribe_to('1', self.subscriber).get(self.TIMEOUT)
        self.stop_proxy(self.monitor)
        self.subscriber.assert_called_with(node)
        self.assertEqual(testutil.MockSize(2), node.size)

if __name__ == '__main__':
    unittest.main()
