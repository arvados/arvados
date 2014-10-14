#!/usr/bin/env python

from __future__ import absolute_import, print_function

import unittest

import arvnodeman.nodelist as nodelist
from . import testutil

class ArvadosNodeListMonitorActorTestCase(testutil.RemotePollLoopActorTestMixin,
                                          unittest.TestCase):
    TEST_CLASS = nodelist.ArvadosNodeListMonitorActor

    def build_monitor(self, side_effect, *args, **kwargs):
        super(ArvadosNodeListMonitorActorTestCase, self).build_monitor(
            *args, **kwargs)
        self.client.nodes().list().execute.side_effect = side_effect

    def test_uuid_is_subscription_key(self):
        node = testutil.arvados_node_mock()
        self.build_monitor([{'items': [node]}])
        self.monitor.subscribe_to(node['uuid'],
                                  self.subscriber).get(self.TIMEOUT)
        self.stop_proxy(self.monitor)
        self.subscriber.assert_called_with(node)


class CloudNodeListMonitorActorTestCase(testutil.RemotePollLoopActorTestMixin,
                                        unittest.TestCase):
    TEST_CLASS = nodelist.CloudNodeListMonitorActor

    class MockNode(object):
        def __init__(self, count):
            self.id = str(count)
            self.name = 'test{}.example.com'.format(count)
            self.private_ips = ['10.0.0.{}'.format(count)]
            self.public_ips = []
            self.size = None
            self.state = 0


    def build_monitor(self, side_effect, *args, **kwargs):
        super(CloudNodeListMonitorActorTestCase, self).build_monitor(
            *args, **kwargs)
        self.client.list_nodes.side_effect = side_effect

    def test_id_is_subscription_key(self):
        node = self.MockNode(1)
        self.build_monitor([[node]])
        self.monitor.subscribe_to('1', self.subscriber).get(self.TIMEOUT)
        self.stop_proxy(self.monitor)
        self.subscriber.assert_called_with(node)


if __name__ == '__main__':
    unittest.main()

