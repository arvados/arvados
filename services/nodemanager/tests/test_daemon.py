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
            self.daemon.update_cloud_nodes(cloud_nodes)
        if arvados_nodes is not None:
            self.daemon.update_arvados_nodes(arvados_nodes)
        if want_sizes is not None:
            self.daemon.update_server_wishlist(want_sizes)

    def test_easy_node_creation(self):
        size = testutil.MockSize(1)
        self.make_daemon(want_sizes=[size])
        self.wait_for_call(self.node_setup.start)

    def test_node_pairing(self):
        cloud_node = testutil.cloud_node_mock(1)
        arv_node = testutil.arvados_node_mock(1)
        self.make_daemon([cloud_node], [arv_node])
        self.wait_for_call(self.node_factory.start)
        pair_func = self.node_factory.start().proxy().offer_arvados_pair
        self.wait_for_call(pair_func)
        pair_func.assert_called_with(arv_node)

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
        self.make_daemon(arvados_nodes=[arv_node], want_sizes=[size, size])
        node_starter = self.node_setup.start
        deadline = time.time() + self.TIMEOUT
        while (time.time() < deadline) and (node_starter.call_count < 2):
            time.sleep(.1)
        self.assertEqual(2, node_starter.call_count)
        used_nodes = [call[1].get('arvados_node')
                      for call in node_starter.call_args_list]
        self.assertIn(arv_node, used_nodes)
        self.assertIn(None, used_nodes)

    def test_node_count_satisfied(self):
        self.make_daemon([testutil.cloud_node_mock()])
        self.daemon.update_server_wishlist(
            [testutil.MockSize(1)]).get(self.TIMEOUT)
        self.assertFalse(self.node_setup.called)

    def test_booting_nodes_counted(self):
        cloud_node = testutil.cloud_node_mock(1)
        arv_node = testutil.arvados_node_mock(1)
        server_wishlist = [testutil.MockSize(1)] * 2
        self.make_daemon([cloud_node], [arv_node], server_wishlist)
        self.wait_for_call(self.node_setup.start)
        self.node_setup.reset_mock()
        self.daemon.update_server_wishlist(server_wishlist).get(self.TIMEOUT)
        self.assertFalse(self.node_setup.called)

    def test_no_duplication_when_booting_node_listed_fast(self):
        # Test that we don't start two ComputeNodeMonitorActors when
        # we learn about a booting node through a listing before we
        # get the "node up" message from CloudNodeSetupActor.
        cloud_node = testutil.cloud_node_mock(1)
        self.make_daemon(want_sizes=[testutil.MockSize(1)])
        self.wait_for_call(self.node_setup.start)
        setup = mock.MagicMock(name='setup_node_mock')
        setup.actor_ref = self.node_setup.start().proxy().actor_ref
        setup.cloud_node.get.return_value = cloud_node
        setup.arvados_node.get.return_value = testutil.arvados_node_mock(1)
        self.daemon.update_cloud_nodes([cloud_node])
        self.wait_for_call(self.node_factory.start)
        self.node_factory.reset_mock()
        self.daemon.node_up(setup).get(self.TIMEOUT)
        self.assertFalse(self.node_factory.start.called)

    def test_booting_nodes_shut_down(self):
        self.make_daemon(want_sizes=[testutil.MockSize(1)])
        self.wait_for_call(self.node_setup.start)
        self.daemon.update_server_wishlist([])
        self.wait_for_call(
            self.node_setup.start().proxy().stop_if_no_cloud_node)

    def test_shutdown_declined_at_wishlist_capacity(self):
        cloud_node = testutil.cloud_node_mock(1)
        size = testutil.MockSize(1)
        self.make_daemon(cloud_nodes=[cloud_node], want_sizes=[size])
        node_actor = self.node_factory().proxy()
        self.daemon.node_can_shutdown(node_actor).get(self.TIMEOUT)
        self.assertFalse(node_actor.shutdown.called)

    def test_shutdown_accepted_below_capacity(self):
        self.make_daemon(cloud_nodes=[testutil.cloud_node_mock()])
        node_actor = self.node_factory().proxy()
        self.daemon.node_can_shutdown(node_actor)
        self.wait_for_call(self.node_shutdown.start)

    def test_clean_shutdown_waits_for_node_setup_finish(self):
        self.make_daemon(want_sizes=[testutil.MockSize(1)])
        self.wait_for_call(self.node_setup.start)
        new_node = self.node_setup.start().proxy()
        self.daemon.shutdown()
        self.wait_for_call(new_node.stop_if_no_cloud_node)
        self.daemon.node_up(new_node)
        self.wait_for_call(new_node.stop)
        self.assertTrue(
            self.daemon.actor_ref.actor_stopped.wait(self.TIMEOUT))

    def test_wishlist_ignored_after_shutdown(self):
        size = testutil.MockSize(2)
        self.make_daemon(want_sizes=[size])
        node_starter = self.node_setup.start
        self.wait_for_call(node_starter)
        node_starter.reset_mock()
        self.daemon.shutdown()
        self.daemon.update_server_wishlist([size] * 2).get(self.TIMEOUT)
        # Send another message and wait for a response, to make sure all
        # internal messages generated by the wishlist update are processed.
        self.daemon.update_server_wishlist([size] * 2).get(self.TIMEOUT)
        self.assertFalse(node_starter.called)
