#!/usr/bin/env python

from __future__ import absolute_import, print_function

import time
import unittest

import mock

import arvnodeman.daemon as nmdaemon
from . import testutil

class NodeManagerDaemonActorTestCase(testutil.ActorTestMixin,
                                     unittest.TestCase):
    def make_daemon(self, cloud_nodes=[], arvados_nodes=[], want_sizes=[]):
        for name in ['cloud_nodes', 'arvados_nodes', 'server_wishlist']:
            setattr(self, name + '_poller', mock.MagicMock(name=name + '_mock'))
        self.arv_factory = mock.MagicMock(name='arvados_mock')
        self.cloud_factory = mock.MagicMock(name='cloud_mock')
        self.cloud_factory().node_start_time.return_value = time.time()
        self.cloud_updates = mock.MagicMock(name='updates_mock')
        self.timer = testutil.MockTimer()
        self.node_factory = mock.MagicMock(name='factory_mock')
        self.node_setup = mock.MagicMock(name='setup_mock')
        self.node_shutdown = mock.MagicMock(name='shutdown_mock')
        self.daemon = nmdaemon.NodeManagerDaemonActor.start(
            self.server_wishlist_poller, self.arvados_nodes_poller,
            self.cloud_nodes_poller, self.cloud_updates, self.timer,
            self.arv_factory, self.cloud_factory,
            [54, 5, 1], 8, 600, 3600,
            self.node_setup, self.node_shutdown, self.node_factory).proxy()
        if cloud_nodes is not None:
            self.daemon.update_cloud_nodes(cloud_nodes).get(self.TIMEOUT)
        if arvados_nodes is not None:
            self.daemon.update_arvados_nodes(arvados_nodes).get(self.TIMEOUT)
        if want_sizes is not None:
            self.daemon.update_server_wishlist(want_sizes).get(self.TIMEOUT)

    def test_easy_node_creation(self):
        size = testutil.MockSize(1)
        self.make_daemon(want_sizes=[size])
        self.stop_proxy(self.daemon)
        self.assertTrue(self.node_setup.start.called)

    def test_node_pairing(self):
        cloud_node = testutil.cloud_node_mock(1)
        arv_node = testutil.arvados_node_mock(1)
        self.make_daemon([cloud_node], [arv_node])
        self.stop_proxy(self.daemon)
        self.node_factory.start().proxy().offer_arvados_pair.assert_called_with(
            arv_node)

    def test_node_pairing_after_arvados_update(self):
        cloud_node = testutil.cloud_node_mock(2)
        arv_node = testutil.arvados_node_mock(2, ip_address=None)
        self.make_daemon([cloud_node], None)
        pair_func = self.node_factory.start().proxy().offer_arvados_pair
        pair_func().get.return_value = None
        self.daemon.update_arvados_nodes([arv_node]).get(self.TIMEOUT)
        pair_func.assert_called_with(arv_node)

        pair_func().get.return_value = cloud_node.id
        pair_func.reset_mock()
        arv_node = testutil.arvados_node_mock(2)
        self.daemon.update_arvados_nodes([arv_node]).get(self.TIMEOUT)
        pair_func.assert_called_with(arv_node)

    def test_old_arvados_node_not_double_assigned(self):
        arv_node = testutil.arvados_node_mock(3, age=9000)
        size = testutil.MockSize(3)
        self.make_daemon(arvados_nodes=[arv_node])
        setup_ref = self.node_setup.start().proxy().actor_ref
        setup_ref.actor_urn = 0
        self.node_setup.start.reset_mock()
        self.daemon.update_server_wishlist([size]).get(self.TIMEOUT)
        self.daemon.max_nodes.get(self.TIMEOUT)
        setup_ref.actor_urn += 1
        self.daemon.update_server_wishlist([size, size]).get(self.TIMEOUT)
        self.stop_proxy(self.daemon)
        used_nodes = [call[1].get('arvados_node')
                      for call in self.node_setup.start.call_args_list]
        self.assertEqual(2, len(used_nodes))
        self.assertIn(arv_node, used_nodes)
        self.assertIn(None, used_nodes)

    def test_node_count_satisfied(self):
        self.make_daemon([testutil.cloud_node_mock()],
                         want_sizes=[testutil.MockSize(1)])
        self.stop_proxy(self.daemon)
        self.assertFalse(self.node_setup.called)

    def test_booting_nodes_counted(self):
        cloud_node = testutil.cloud_node_mock(1)
        arv_node = testutil.arvados_node_mock(1)
        server_wishlist = [testutil.MockSize(1)] * 2
        self.make_daemon([cloud_node], [arv_node], server_wishlist)
        self.daemon.max_nodes.get(self.TIMEOUT)
        self.assertTrue(self.node_setup.start.called)
        self.daemon.update_server_wishlist(server_wishlist).get(self.TIMEOUT)
        self.stop_proxy(self.daemon)
        self.assertEqual(1, self.node_setup.start.call_count)

    def test_no_duplication_when_booting_node_listed_fast(self):
        # Test that we don't start two ComputeNodeMonitorActors when
        # we learn about a booting node through a listing before we
        # get the "node up" message from CloudNodeSetupActor.
        cloud_node = testutil.cloud_node_mock(1)
        self.make_daemon(want_sizes=[testutil.MockSize(1)])
        self.daemon.max_nodes.get(self.TIMEOUT)
        self.assertEqual(1, self.node_setup.start.call_count)
        setup = mock.MagicMock(name='setup_node_mock')
        setup.actor_ref = self.node_setup.start().proxy().actor_ref
        setup.cloud_node.get.return_value = cloud_node
        setup.arvados_node.get.return_value = testutil.arvados_node_mock(1)
        self.daemon.update_cloud_nodes([cloud_node]).get(self.TIMEOUT)
        self.assertTrue(self.node_factory.start.called)
        self.daemon.node_up(setup).get(self.TIMEOUT)
        self.assertEqual(1, self.node_factory.start.call_count)

    def test_booting_nodes_shut_down(self):
        self.make_daemon(want_sizes=[testutil.MockSize(1)])
        self.daemon.update_server_wishlist([]).get(self.TIMEOUT)
        self.stop_proxy(self.daemon)
        self.assertTrue(
            self.node_setup.start().proxy().stop_if_no_cloud_node.called)

    def test_shutdown_declined_at_wishlist_capacity(self):
        cloud_node = testutil.cloud_node_mock(1)
        size = testutil.MockSize(1)
        self.make_daemon(cloud_nodes=[cloud_node], want_sizes=[size])
        node_actor = self.node_factory().proxy()
        self.daemon.node_can_shutdown(node_actor).get(self.TIMEOUT)
        self.stop_proxy(self.daemon)
        self.assertFalse(node_actor.shutdown.called)

    def test_shutdown_accepted_below_capacity(self):
        self.make_daemon(cloud_nodes=[testutil.cloud_node_mock()])
        node_actor = self.node_factory().proxy()
        self.daemon.node_can_shutdown(node_actor).get(self.TIMEOUT)
        self.stop_proxy(self.daemon)
        self.assertTrue(self.node_shutdown.start.called)

    def test_clean_shutdown_waits_for_node_setup_finish(self):
        self.make_daemon(want_sizes=[testutil.MockSize(1)])
        self.daemon.max_nodes.get(self.TIMEOUT)
        self.assertTrue(self.node_setup.start.called)
        new_node = self.node_setup.start().proxy()
        self.daemon.shutdown().get(self.TIMEOUT)
        self.assertTrue(new_node.stop_if_no_cloud_node.called)
        self.daemon.node_up(new_node).get(self.TIMEOUT)
        self.assertTrue(new_node.stop.called)
        self.assertTrue(
            self.daemon.actor_ref.actor_stopped.wait(self.TIMEOUT))

    def test_wishlist_ignored_after_shutdown(self):
        size = testutil.MockSize(2)
        self.make_daemon(want_sizes=[size])
        self.daemon.shutdown().get(self.TIMEOUT)
        self.daemon.update_server_wishlist([size] * 2).get(self.TIMEOUT)
        self.stop_proxy(self.daemon)
        self.assertEqual(1, self.node_setup.start.call_count)
