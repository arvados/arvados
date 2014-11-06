#!/usr/bin/env python

from __future__ import absolute_import, print_function

import time
import unittest

import arvados.errors as arverror
import httplib2
import mock
import pykka

import arvnodeman.computenode as cnode
from . import testutil

class ComputeNodeSetupActorTestCase(testutil.ActorTestMixin, unittest.TestCase):
    def make_mocks(self, arvados_effect=None, cloud_effect=None):
        if arvados_effect is None:
            arvados_effect = [testutil.arvados_node_mock()]
        self.arvados_effect = arvados_effect
        self.timer = testutil.MockTimer()
        self.api_client = mock.MagicMock(name='api_client')
        self.api_client.nodes().create().execute.side_effect = arvados_effect
        self.api_client.nodes().update().execute.side_effect = arvados_effect
        self.cloud_client = mock.MagicMock(name='cloud_client')
        self.cloud_client.create_node.return_value = testutil.cloud_node_mock(1)

    def make_actor(self, arv_node=None):
        if not hasattr(self, 'timer'):
            self.make_mocks(arvados_effect=[arv_node])
        self.setup_actor = cnode.ComputeNodeSetupActor.start(
            self.timer, self.api_client, self.cloud_client,
            testutil.MockSize(1), arv_node).proxy()

    def test_creation_without_arvados_node(self):
        self.make_actor()
        self.assertEqual(self.arvados_effect[-1],
                         self.setup_actor.arvados_node.get(self.TIMEOUT))
        self.assertTrue(self.api_client.nodes().create().execute.called)
        self.assertEqual(self.cloud_client.create_node(),
                         self.setup_actor.cloud_node.get(self.TIMEOUT))

    def test_creation_with_arvados_node(self):
        self.make_actor(testutil.arvados_node_mock())
        self.assertEqual(self.arvados_effect[-1],
                         self.setup_actor.arvados_node.get(self.TIMEOUT))
        self.assertTrue(self.api_client.nodes().update().execute.called)
        self.assertEqual(self.cloud_client.create_node(),
                         self.setup_actor.cloud_node.get(self.TIMEOUT))

    def test_failed_calls_retried(self):
        self.make_mocks([
                arverror.ApiError(httplib2.Response({'status': '500'}), ""),
                testutil.arvados_node_mock(),
                ])
        self.make_actor()
        self.wait_for_assignment(self.setup_actor, 'cloud_node')

    def test_stop_when_no_cloud_node(self):
        self.make_mocks(
            arverror.ApiError(httplib2.Response({'status': '500'}), ""))
        self.make_actor()
        self.setup_actor.stop_if_no_cloud_node()
        self.assertTrue(
            self.setup_actor.actor_ref.actor_stopped.wait(self.TIMEOUT))

    def test_no_stop_when_cloud_node(self):
        self.make_actor()
        self.wait_for_assignment(self.setup_actor, 'cloud_node')
        self.setup_actor.stop_if_no_cloud_node().get(self.TIMEOUT)
        self.assertTrue(self.stop_proxy(self.setup_actor),
                        "actor was stopped by stop_if_no_cloud_node")

    def test_subscribe(self):
        self.make_mocks(
            arverror.ApiError(httplib2.Response({'status': '500'}), ""))
        self.make_actor()
        subscriber = mock.Mock(name='subscriber_mock')
        self.setup_actor.subscribe(subscriber)
        self.api_client.nodes().create().execute.side_effect = [
            testutil.arvados_node_mock()]
        self.wait_for_assignment(self.setup_actor, 'cloud_node')
        self.assertEqual(self.setup_actor.actor_ref.actor_urn,
                         subscriber.call_args[0][0].actor_ref.actor_urn)

    def test_late_subscribe(self):
        self.make_actor()
        subscriber = mock.Mock(name='subscriber_mock')
        self.wait_for_assignment(self.setup_actor, 'cloud_node')
        self.setup_actor.subscribe(subscriber).get(self.TIMEOUT)
        self.stop_proxy(self.setup_actor)
        self.assertEqual(self.setup_actor.actor_ref.actor_urn,
                         subscriber.call_args[0][0].actor_ref.actor_urn)


class ComputeNodeShutdownActorTestCase(testutil.ActorTestMixin,
                                       unittest.TestCase):
    def make_mocks(self, cloud_node=None):
        self.timer = testutil.MockTimer()
        self.cloud_client = mock.MagicMock(name='cloud_client')
        if cloud_node is None:
            cloud_node = testutil.cloud_node_mock()
        self.cloud_node = cloud_node

    def make_actor(self, arv_node=None):
        if not hasattr(self, 'timer'):
            self.make_mocks()
        self.shutdown_actor = cnode.ComputeNodeShutdownActor.start(
            self.timer, self.cloud_client, self.cloud_node).proxy()

    def test_easy_shutdown(self):
        self.make_actor()
        self.shutdown_actor.cloud_node.get(self.TIMEOUT)
        self.stop_proxy(self.shutdown_actor)
        self.assertTrue(self.cloud_client.destroy_node.called)

    def test_late_subscribe(self):
        self.make_actor()
        subscriber = mock.Mock(name='subscriber_mock')
        self.shutdown_actor.subscribe(subscriber).get(self.TIMEOUT)
        self.stop_proxy(self.shutdown_actor)
        self.assertEqual(self.shutdown_actor.actor_ref.actor_urn,
                         subscriber.call_args[0][0].actor_ref.actor_urn)


class ComputeNodeUpdateActorTestCase(testutil.ActorTestMixin,
                                     unittest.TestCase):
    def make_actor(self):
        self.driver = mock.MagicMock(name='driver_mock')
        self.updater = cnode.ComputeNodeUpdateActor.start(self.driver).proxy()

    def test_node_sync(self):
        self.make_actor()
        cloud_node = testutil.cloud_node_mock()
        arv_node = testutil.arvados_node_mock()
        self.updater.sync_node(cloud_node, arv_node).get(self.TIMEOUT)
        self.driver().sync_node.assert_called_with(cloud_node, arv_node)


@mock.patch('time.time', return_value=1)
class ShutdownTimerTestCase(unittest.TestCase):
    def test_two_length_window(self, time_mock):
        timer = cnode.ShutdownTimer(time_mock.return_value, [8, 2])
        self.assertEqual(481, timer.next_opening())
        self.assertFalse(timer.window_open())
        time_mock.return_value += 500
        self.assertEqual(1081, timer.next_opening())
        self.assertTrue(timer.window_open())
        time_mock.return_value += 200
        self.assertEqual(1081, timer.next_opening())
        self.assertFalse(timer.window_open())

    def test_three_length_window(self, time_mock):
        timer = cnode.ShutdownTimer(time_mock.return_value, [6, 3, 1])
        self.assertEqual(361, timer.next_opening())
        self.assertFalse(timer.window_open())
        time_mock.return_value += 400
        self.assertEqual(961, timer.next_opening())
        self.assertTrue(timer.window_open())
        time_mock.return_value += 200
        self.assertEqual(961, timer.next_opening())
        self.assertFalse(timer.window_open())


class ComputeNodeMonitorActorTestCase(testutil.ActorTestMixin,
                                      unittest.TestCase):
    class MockShutdownTimer(object):
        def _set_state(self, is_open, next_opening):
            self.window_open = lambda: is_open
            self.next_opening = lambda: next_opening


    def make_mocks(self, node_num):
        self.shutdowns = self.MockShutdownTimer()
        self.shutdowns._set_state(False, 300)
        self.timer = mock.MagicMock(name='timer_mock')
        self.updates = mock.MagicMock(name='update_mock')
        self.cloud_mock = testutil.cloud_node_mock(node_num)
        self.subscriber = mock.Mock(name='subscriber_mock')

    def make_actor(self, node_num=1, arv_node=None, start_time=None):
        if not hasattr(self, 'cloud_mock'):
            self.make_mocks(node_num)
        if start_time is None:
            start_time = time.time()
        self.node_actor = cnode.ComputeNodeMonitorActor.start(
            self.cloud_mock, start_time, self.shutdowns, self.timer,
            self.updates, arv_node).proxy()
        self.node_actor.subscribe(self.subscriber).get(self.TIMEOUT)

    def node_state(self, *states):
        return self.node_actor.in_state(*states).get(self.TIMEOUT)

    def test_in_state_when_unpaired(self):
        self.make_actor()
        self.assertIsNone(self.node_state('idle', 'alloc'))

    def test_in_state_when_pairing_stale(self):
        self.make_actor(arv_node=testutil.arvados_node_mock(
                job_uuid=None, age=90000))
        self.assertIsNone(self.node_state('idle', 'alloc'))

    def test_in_state_when_no_state_available(self):
        self.make_actor(arv_node=testutil.arvados_node_mock(info={}))
        self.assertIsNone(self.node_state('idle', 'alloc'))

    def test_in_idle_state(self):
        self.make_actor(2, arv_node=testutil.arvados_node_mock(job_uuid=None))
        self.assertTrue(self.node_state('idle'))
        self.assertFalse(self.node_state('alloc'))
        self.assertTrue(self.node_state('idle', 'alloc'))

    def test_in_alloc_state(self):
        self.make_actor(3, arv_node=testutil.arvados_node_mock(job_uuid=True))
        self.assertFalse(self.node_state('idle'))
        self.assertTrue(self.node_state('alloc'))
        self.assertTrue(self.node_state('idle', 'alloc'))

    def test_init_shutdown_scheduling(self):
        self.make_actor()
        self.assertTrue(self.timer.schedule.called)
        self.assertEqual(300, self.timer.schedule.call_args[0][0])

    def test_shutdown_subscription(self):
        self.make_actor()
        self.shutdowns._set_state(True, 600)
        self.node_actor.consider_shutdown().get(self.TIMEOUT)
        self.assertTrue(self.subscriber.called)
        self.assertEqual(self.node_actor.actor_ref.actor_urn,
                         self.subscriber.call_args[0][0].actor_ref.actor_urn)

    def test_shutdown_without_arvados_node(self):
        self.make_actor()
        self.shutdowns._set_state(True, 600)
        self.node_actor.consider_shutdown().get(self.TIMEOUT)
        self.assertTrue(self.subscriber.called)

    def test_no_shutdown_without_arvados_node_and_old_cloud_node(self):
        self.make_actor(start_time=0)
        self.shutdowns._set_state(True, 600)
        self.node_actor.consider_shutdown().get(self.TIMEOUT)
        self.assertFalse(self.subscriber.called)

    def check_shutdown_rescheduled(self, window_open, next_window,
                                   schedule_time=None):
        self.shutdowns._set_state(window_open, next_window)
        self.timer.schedule.reset_mock()
        self.node_actor.consider_shutdown().get(self.TIMEOUT)
        self.stop_proxy(self.node_actor)
        self.assertTrue(self.timer.schedule.called)
        if schedule_time is not None:
            self.assertEqual(schedule_time, self.timer.schedule.call_args[0][0])
        self.assertFalse(self.subscriber.called)

    def test_shutdown_window_close_scheduling(self):
        self.make_actor()
        self.check_shutdown_rescheduled(False, 600, 600)

    def test_no_shutdown_when_node_running_job(self):
        self.make_actor(4, testutil.arvados_node_mock(4, job_uuid=True))
        self.check_shutdown_rescheduled(True, 600)

    def test_no_shutdown_when_node_state_unknown(self):
        self.make_actor(5, testutil.arvados_node_mock(5, info={}))
        self.check_shutdown_rescheduled(True, 600)

    def test_no_shutdown_when_node_state_stale(self):
        self.make_actor(6, testutil.arvados_node_mock(6, age=90000))
        self.check_shutdown_rescheduled(True, 600)

    def test_arvados_node_match(self):
        self.make_actor(2)
        arv_node = testutil.arvados_node_mock(
            2, hostname='compute-two.zzzzz.arvadosapi.com')
        pair_id = self.node_actor.offer_arvados_pair(arv_node).get(self.TIMEOUT)
        self.assertEqual(self.cloud_mock.id, pair_id)
        self.stop_proxy(self.node_actor)
        self.updates.sync_node.assert_called_with(self.cloud_mock, arv_node)

    def test_arvados_node_mismatch(self):
        self.make_actor(3)
        arv_node = testutil.arvados_node_mock(1)
        self.assertIsNone(
            self.node_actor.offer_arvados_pair(arv_node).get(self.TIMEOUT))

    def test_update_cloud_node(self):
        self.make_actor(1)
        self.make_mocks(2)
        self.cloud_mock.id = '1'
        self.node_actor.update_cloud_node(self.cloud_mock)
        current_cloud = self.node_actor.cloud_node.get(self.TIMEOUT)
        self.assertEqual([testutil.ip_address_mock(2)],
                         current_cloud.private_ips)

    def test_missing_cloud_node_update(self):
        self.make_actor(1)
        self.node_actor.update_cloud_node(None)
        current_cloud = self.node_actor.cloud_node.get(self.TIMEOUT)
        self.assertEqual([testutil.ip_address_mock(1)],
                         current_cloud.private_ips)

    def test_update_arvados_node(self):
        self.make_actor(3)
        job_uuid = 'zzzzz-jjjjj-updatejobnode00'
        new_arvados = testutil.arvados_node_mock(3, job_uuid)
        self.node_actor.update_arvados_node(new_arvados)
        current_arvados = self.node_actor.arvados_node.get(self.TIMEOUT)
        self.assertEqual(job_uuid, current_arvados['job_uuid'])

    def test_missing_arvados_node_update(self):
        self.make_actor(4, testutil.arvados_node_mock(4))
        self.node_actor.update_arvados_node(None)
        current_arvados = self.node_actor.arvados_node.get(self.TIMEOUT)
        self.assertEqual(testutil.ip_address_mock(4),
                         current_arvados['ip_address'])
