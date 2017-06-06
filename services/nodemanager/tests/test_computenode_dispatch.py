#!/usr/bin/env python

from __future__ import absolute_import, print_function

import time
import unittest

import arvados.errors as arverror
import httplib2
import mock
import pykka
import threading

from libcloud.common.exceptions import BaseHTTPError

import arvnodeman.computenode.dispatch as dispatch
from arvnodeman.computenode.driver import BaseComputeNodeDriver
from . import testutil

class ComputeNodeSetupActorTestCase(testutil.ActorTestMixin, unittest.TestCase):
    def make_mocks(self, arvados_effect=None):
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
            self.make_mocks(arvados_effect=[arv_node] if arv_node else None)
        self.setup_actor = dispatch.ComputeNodeSetupActor.start(
            self.timer, self.api_client, self.cloud_client,
            testutil.MockSize(1), arv_node).proxy()

    def assert_node_properties_updated(self, uuid=None,
                                       size=testutil.MockSize(1)):
        self.api_client.nodes().update.assert_any_call(
            uuid=(uuid or self.arvados_effect[-1]['uuid']),
            body={
                'properties': {
                    'cloud_node': {
                        'size': size.id,
                        'price': size.price}}})

    def test_creation_without_arvados_node(self):
        self.make_actor()
        finished = threading.Event()
        self.setup_actor.subscribe(lambda _: finished.set())
        self.assertEqual(self.arvados_effect[-1],
                         self.setup_actor.arvados_node.get(self.TIMEOUT))
        assert(finished.wait(self.TIMEOUT))
        self.assertEqual(1, self.api_client.nodes().create().execute.call_count)
        self.assertEqual(1, self.api_client.nodes().update().execute.call_count)
        self.assert_node_properties_updated()
        self.assertEqual(self.cloud_client.create_node(),
                         self.setup_actor.cloud_node.get(self.TIMEOUT))

    def test_creation_with_arvados_node(self):
        self.make_mocks(arvados_effect=[testutil.arvados_node_mock()]*2)
        self.make_actor(testutil.arvados_node_mock())
        finished = threading.Event()
        self.setup_actor.subscribe(lambda _: finished.set())
        self.assertEqual(self.arvados_effect[-1],
                         self.setup_actor.arvados_node.get(self.TIMEOUT))
        assert(finished.wait(self.TIMEOUT))
        self.assert_node_properties_updated()
        self.assertEqual(2, self.api_client.nodes().update().execute.call_count)
        self.assertEqual(self.cloud_client.create_node(),
                         self.setup_actor.cloud_node.get(self.TIMEOUT))

    def test_failed_arvados_calls_retried(self):
        self.make_mocks([
                arverror.ApiError(httplib2.Response({'status': '500'}), ""),
                testutil.arvados_node_mock(),
                ])
        self.make_actor()
        self.wait_for_assignment(self.setup_actor, 'arvados_node')

    def test_failed_cloud_calls_retried(self):
        self.make_mocks()
        self.cloud_client.create_node.side_effect = [
            Exception("test cloud creation error"),
            self.cloud_client.create_node.return_value,
            ]
        self.make_actor()
        self.wait_for_assignment(self.setup_actor, 'cloud_node')

    def test_basehttperror_retried(self):
        self.make_mocks()
        self.cloud_client.create_node.side_effect = [
            BaseHTTPError(500, "Try again"),
            self.cloud_client.create_node.return_value,
            ]
        self.make_actor()
        self.wait_for_assignment(self.setup_actor, 'cloud_node')
        self.assertEqual(1, self.cloud_client.post_create_node.call_count)

    def test_instance_exceeded_not_retried(self):
        self.make_mocks()
        self.cloud_client.create_node.side_effect = [
            BaseHTTPError(400, "InstanceLimitExceeded"),
            self.cloud_client.create_node.return_value,
            ]
        self.make_actor()
        done = self.FUTURE_CLASS()
        self.setup_actor.subscribe(done.set)
        done.get(self.TIMEOUT)
        self.assertEqual(0, self.cloud_client.post_create_node.call_count)

    def test_failed_post_create_retried(self):
        self.make_mocks()
        self.cloud_client.post_create_node.side_effect = [
            Exception("test cloud post-create error"), None]
        self.make_actor()
        done = self.FUTURE_CLASS()
        self.setup_actor.subscribe(done.set)
        done.get(self.TIMEOUT)
        self.assertEqual(2, self.cloud_client.post_create_node.call_count)

    def test_stop_when_no_cloud_node(self):
        self.make_mocks(
            arverror.ApiError(httplib2.Response({'status': '500'}), ""))
        self.make_actor()
        self.assertTrue(
            self.setup_actor.stop_if_no_cloud_node().get(self.TIMEOUT))
        self.assertTrue(
            self.setup_actor.actor_ref.actor_stopped.wait(self.TIMEOUT))

    def test_no_stop_when_cloud_node(self):
        self.make_actor()
        self.wait_for_assignment(self.setup_actor, 'cloud_node')
        self.assertFalse(
            self.setup_actor.stop_if_no_cloud_node().get(self.TIMEOUT))
        self.assertTrue(self.stop_proxy(self.setup_actor),
                        "actor was stopped by stop_if_no_cloud_node")

    def test_subscribe(self):
        self.make_mocks(
            arverror.ApiError(httplib2.Response({'status': '500'}), ""))
        self.make_actor()
        subscriber = mock.Mock(name='subscriber_mock')
        self.setup_actor.subscribe(subscriber)
        retry_resp = [testutil.arvados_node_mock()]
        self.api_client.nodes().create().execute.side_effect = retry_resp
        self.api_client.nodes().update().execute.side_effect = retry_resp
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


class ComputeNodeShutdownActorMixin(testutil.ActorTestMixin):
    def make_mocks(self, cloud_node=None, arvados_node=None,
                   shutdown_open=True, node_broken=False):
        self.timer = testutil.MockTimer()
        self.shutdowns = testutil.MockShutdownTimer()
        self.shutdowns._set_state(shutdown_open, 300)
        self.cloud_client = mock.MagicMock(name='cloud_client')
        self.cloud_client.broken.return_value = node_broken
        self.arvados_client = mock.MagicMock(name='arvados_client')
        self.updates = mock.MagicMock(name='update_mock')
        if cloud_node is None:
            cloud_node = testutil.cloud_node_mock()
        self.cloud_node = cloud_node
        self.arvados_node = arvados_node

    def make_actor(self, cancellable=True, start_time=None):
        if not hasattr(self, 'timer'):
            self.make_mocks()
        if start_time is None:
            start_time = time.time()
        monitor_actor = dispatch.ComputeNodeMonitorActor.start(
            self.cloud_node, start_time, self.shutdowns,
            testutil.cloud_node_fqdn, self.timer, self.updates, self.cloud_client,
            self.arvados_node)
        self.shutdown_actor = self.ACTOR_CLASS.start(
            self.timer, self.cloud_client, self.arvados_client, monitor_actor,
            cancellable).proxy()
        self.monitor_actor = monitor_actor.proxy()

    def check_success_flag(self, expected, allow_msg_count=1):
        # allow_msg_count is the number of internal messages that may
        # need to be handled for shutdown to finish.
        for try_num in range(1 + allow_msg_count):
            last_flag = self.shutdown_actor.success.get(self.TIMEOUT)
            if last_flag is expected:
                break
        else:
            self.fail("success flag {} is not {}".format(last_flag, expected))

    def test_cancellable_shutdown(self, *mocks):
        self.make_mocks(shutdown_open=True, arvados_node=testutil.arvados_node_mock(crunch_worker_state="busy"))
        self.cloud_client.destroy_node.return_value = True
        self.make_actor(cancellable=True)
        self.check_success_flag(False)
        self.assertFalse(self.cloud_client.destroy_node.called)

    def test_uncancellable_shutdown(self, *mocks):
        self.make_mocks(shutdown_open=True, arvados_node=testutil.arvados_node_mock(crunch_worker_state="busy"))
        self.cloud_client.destroy_node.return_value = True
        self.make_actor(cancellable=False)
        self.check_success_flag(True, 2)
        self.assertTrue(self.cloud_client.destroy_node.called)

    def test_arvados_node_cleaned_after_shutdown(self, *mocks):
        cloud_node = testutil.cloud_node_mock(62)
        arv_node = testutil.arvados_node_mock(62)
        self.make_mocks(cloud_node, arv_node)
        self.make_actor()
        self.check_success_flag(True, 3)
        update_mock = self.arvados_client.nodes().update
        self.assertTrue(update_mock.called)
        update_kwargs = update_mock.call_args_list[0][1]
        self.assertEqual(arv_node['uuid'], update_kwargs.get('uuid'))
        self.assertIn('body', update_kwargs)
        for clear_key in ['slot_number', 'hostname', 'ip_address',
                          'first_ping_at', 'last_ping_at']:
            self.assertIn(clear_key, update_kwargs['body'])
            self.assertIsNone(update_kwargs['body'][clear_key])
        self.assertTrue(update_mock().execute.called)

    def test_arvados_node_not_cleaned_after_shutdown_cancelled(self, *mocks):
        cloud_node = testutil.cloud_node_mock(61)
        arv_node = testutil.arvados_node_mock(61)
        self.make_mocks(cloud_node, arv_node, shutdown_open=False)
        self.cloud_client.destroy_node.return_value = False
        self.make_actor(cancellable=True)
        self.shutdown_actor.cancel_shutdown("test")
        self.check_success_flag(False, 2)
        self.assertFalse(self.arvados_client.nodes().update.called)


class ComputeNodeShutdownActorTestCase(ComputeNodeShutdownActorMixin,
                                       unittest.TestCase):
    ACTOR_CLASS = dispatch.ComputeNodeShutdownActor

    def test_easy_shutdown(self):
        self.make_actor(start_time=0)
        self.check_success_flag(True)
        self.assertTrue(self.cloud_client.destroy_node.called)

    def test_shutdown_cancelled_when_destroy_node_fails(self):
        self.make_mocks(node_broken=True)
        self.cloud_client.destroy_node.return_value = False
        self.make_actor(start_time=0)
        self.check_success_flag(False, 2)
        self.assertEqual(1, self.cloud_client.destroy_node.call_count)
        self.assertEqual(self.ACTOR_CLASS.DESTROY_FAILED,
                         self.shutdown_actor.cancel_reason.get(self.TIMEOUT))

    def test_late_subscribe(self):
        self.make_actor()
        subscriber = mock.Mock(name='subscriber_mock')
        self.shutdown_actor.subscribe(subscriber).get(self.TIMEOUT)
        self.stop_proxy(self.shutdown_actor)
        self.assertTrue(subscriber.called)
        self.assertEqual(self.shutdown_actor.actor_ref.actor_urn,
                         subscriber.call_args[0][0].actor_ref.actor_urn)


class ComputeNodeUpdateActorTestCase(testutil.ActorTestMixin,
                                     unittest.TestCase):
    ACTOR_CLASS = dispatch.ComputeNodeUpdateActor

    def make_actor(self):
        self.driver = mock.MagicMock(name='driver_mock')
        self.timer = mock.MagicMock(name='timer_mock')
        self.updater = self.ACTOR_CLASS.start(self.driver, self.timer).proxy()

    def test_node_sync(self, *args):
        self.make_actor()
        cloud_node = testutil.cloud_node_mock()
        arv_node = testutil.arvados_node_mock()
        self.updater.sync_node(cloud_node, arv_node).get(self.TIMEOUT)
        self.driver().sync_node.assert_called_with(cloud_node, arv_node)

    @testutil.no_sleep
    def test_node_sync_error(self, *args):
        self.make_actor()
        cloud_node = testutil.cloud_node_mock()
        arv_node = testutil.arvados_node_mock()
        self.driver().sync_node.side_effect = (IOError, Exception, True)
        self.updater.sync_node(cloud_node, arv_node).get(self.TIMEOUT)
        self.updater.sync_node(cloud_node, arv_node).get(self.TIMEOUT)
        self.updater.sync_node(cloud_node, arv_node).get(self.TIMEOUT)
        self.driver().sync_node.assert_called_with(cloud_node, arv_node)

class ComputeNodeMonitorActorTestCase(testutil.ActorTestMixin,
                                      unittest.TestCase):
    def make_mocks(self, node_num):
        self.shutdowns = testutil.MockShutdownTimer()
        self.shutdowns._set_state(False, 300)
        self.timer = mock.MagicMock(name='timer_mock')
        self.updates = mock.MagicMock(name='update_mock')
        self.cloud_mock = testutil.cloud_node_mock(node_num)
        self.subscriber = mock.Mock(name='subscriber_mock')
        self.cloud_client = mock.MagicMock(name='cloud_client')
        self.cloud_client.broken.return_value = False

    def make_actor(self, node_num=1, arv_node=None, start_time=None):
        if not hasattr(self, 'cloud_mock'):
            self.make_mocks(node_num)
        if start_time is None:
            start_time = time.time()
        self.node_actor = dispatch.ComputeNodeMonitorActor.start(
            self.cloud_mock, start_time, self.shutdowns,
            testutil.cloud_node_fqdn, self.timer, self.updates, self.cloud_client,
            arv_node, boot_fail_after=300).proxy()
        self.node_actor.subscribe(self.subscriber).get(self.TIMEOUT)

    def node_state(self, *states):
        return self.node_actor.in_state(*states).get(self.TIMEOUT)

    def test_in_state_when_unpaired(self):
        self.make_actor()
        self.assertTrue(self.node_state('unpaired'))

    def test_in_state_when_pairing_stale(self):
        self.make_actor(arv_node=testutil.arvados_node_mock(
                job_uuid=None, age=90000))
        self.assertTrue(self.node_state('down'))

    def test_in_state_when_no_state_available(self):
        self.make_actor(arv_node=testutil.arvados_node_mock(
                crunch_worker_state=None))
        print(self.node_actor.get_state().get())
        self.assertTrue(self.node_state('idle'))

    def test_in_state_when_no_state_available_old(self):
        self.make_actor(arv_node=testutil.arvados_node_mock(
                crunch_worker_state=None, age=90000))
        print(self.node_actor.get_state().get())
        self.assertTrue(self.node_state('down'))

    def test_in_idle_state(self):
        self.make_actor(2, arv_node=testutil.arvados_node_mock(job_uuid=None))
        self.assertTrue(self.node_state('idle'))
        self.assertFalse(self.node_state('busy'))
        self.assertTrue(self.node_state('idle', 'busy'))

    def test_in_busy_state(self):
        self.make_actor(3, arv_node=testutil.arvados_node_mock(job_uuid=True))
        self.assertFalse(self.node_state('idle'))
        self.assertTrue(self.node_state('busy'))
        self.assertTrue(self.node_state('idle', 'busy'))

    def test_init_shutdown_scheduling(self):
        self.make_actor()
        self.assertTrue(self.timer.schedule.called)
        self.assertEqual(300, self.timer.schedule.call_args[0][0])

    def test_shutdown_window_close_scheduling(self):
        self.make_actor()
        self.shutdowns._set_state(False, 600)
        self.timer.schedule.reset_mock()
        self.node_actor.consider_shutdown().get(self.TIMEOUT)
        self.stop_proxy(self.node_actor)
        self.assertTrue(self.timer.schedule.called)
        self.assertEqual(600, self.timer.schedule.call_args[0][0])
        self.assertFalse(self.subscriber.called)

    def test_shutdown_subscription(self):
        self.make_actor(start_time=0)
        self.shutdowns._set_state(True, 600)
        self.node_actor.consider_shutdown().get(self.TIMEOUT)
        self.assertTrue(self.subscriber.called)
        self.assertEqual(self.node_actor.actor_ref.actor_urn,
                         self.subscriber.call_args[0][0].actor_ref.actor_urn)

    def test_no_shutdown_booting(self):
        self.make_actor()
        self.shutdowns._set_state(True, 600)
        self.assertEquals(self.node_actor.shutdown_eligible().get(self.TIMEOUT),
                          (False, "node state is ('unpaired', 'open', 'boot wait', 'idle exceeded')"))

    def test_shutdown_without_arvados_node(self):
        self.make_actor(start_time=0)
        self.shutdowns._set_state(True, 600)
        self.assertEquals((True, "node state is ('unpaired', 'open', 'boot exceeded', 'idle exceeded')"),
                          self.node_actor.shutdown_eligible().get(self.TIMEOUT))

    def test_shutdown_missing(self):
        arv_node = testutil.arvados_node_mock(10, job_uuid=None,
                                              crunch_worker_state="down",
                                              last_ping_at='1970-01-01T01:02:03.04050607Z')
        self.make_actor(10, arv_node)
        self.shutdowns._set_state(True, 600)
        self.assertEquals((True, "node state is ('down', 'open', 'boot wait', 'idle exceeded')"),
                          self.node_actor.shutdown_eligible().get(self.TIMEOUT))

    def test_shutdown_running_broken(self):
        arv_node = testutil.arvados_node_mock(12, job_uuid=None,
                                              crunch_worker_state="down")
        self.make_actor(12, arv_node)
        self.shutdowns._set_state(True, 600)
        self.cloud_client.broken.return_value = True
        self.assertEquals((True, "node state is ('down', 'open', 'boot wait', 'idle exceeded')"),
                          self.node_actor.shutdown_eligible().get(self.TIMEOUT))

    def test_shutdown_missing_broken(self):
        arv_node = testutil.arvados_node_mock(11, job_uuid=None,
                                              crunch_worker_state="down",
                                              last_ping_at='1970-01-01T01:02:03.04050607Z')
        self.make_actor(11, arv_node)
        self.shutdowns._set_state(True, 600)
        self.cloud_client.broken.return_value = True
        self.assertEquals(self.node_actor.shutdown_eligible().get(self.TIMEOUT), (True, "node state is ('down', 'open', 'boot wait', 'idle exceeded')"))

    def test_no_shutdown_when_window_closed(self):
        self.make_actor(3, testutil.arvados_node_mock(3, job_uuid=None))
        self.assertEquals((False, "node state is ('idle', 'closed', 'boot wait', 'idle exceeded')"),
                          self.node_actor.shutdown_eligible().get(self.TIMEOUT))

    def test_no_shutdown_when_node_running_job(self):
        self.make_actor(4, testutil.arvados_node_mock(4, job_uuid=True))
        self.shutdowns._set_state(True, 600)
        self.assertEquals((False, "node state is ('busy', 'open', 'boot wait', 'idle exceeded')"),
                          self.node_actor.shutdown_eligible().get(self.TIMEOUT))

    def test_shutdown_when_node_state_unknown(self):
        self.make_actor(5, testutil.arvados_node_mock(
            5, crunch_worker_state=None))
        self.shutdowns._set_state(True, 600)
        self.assertEquals((True, "node state is ('idle', 'open', 'boot wait', 'idle exceeded')"),
                          self.node_actor.shutdown_eligible().get(self.TIMEOUT))

    def test_no_shutdown_when_node_state_stale(self):
        self.make_actor(6, testutil.arvados_node_mock(6, age=90000))
        self.shutdowns._set_state(True, 600)
        self.assertEquals((False, "node state is stale"),
                          self.node_actor.shutdown_eligible().get(self.TIMEOUT))

    def test_arvados_node_match(self):
        self.make_actor(2)
        arv_node = testutil.arvados_node_mock(
            2, hostname='compute-two.zzzzz.arvadosapi.com')
        self.cloud_client.node_id.return_value = '2'
        pair_id = self.node_actor.offer_arvados_pair(arv_node).get(self.TIMEOUT)
        self.assertEqual(self.cloud_mock.id, pair_id)
        self.stop_proxy(self.node_actor)
        self.updates.sync_node.assert_called_with(self.cloud_mock, arv_node)

    def test_arvados_node_mismatch(self):
        self.make_actor(3)
        arv_node = testutil.arvados_node_mock(1)
        self.assertIsNone(
            self.node_actor.offer_arvados_pair(arv_node).get(self.TIMEOUT))

    def test_arvados_node_mismatch_first_ping_too_early(self):
        self.make_actor(4)
        arv_node = testutil.arvados_node_mock(
            4, first_ping_at='1971-03-02T14:15:16.1717282Z')
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

    def test_update_arvados_node_syncs_when_fqdn_mismatch(self):
        self.make_mocks(5)
        self.cloud_mock.extra['testname'] = 'cloudfqdn.zzzzz.arvadosapi.com'
        self.make_actor()
        arv_node = testutil.arvados_node_mock(5)
        self.node_actor.update_arvados_node(arv_node).get(self.TIMEOUT)
        self.assertEqual(1, self.updates.sync_node.call_count)

    def test_update_arvados_node_skips_sync_when_fqdn_match(self):
        self.make_mocks(6)
        arv_node = testutil.arvados_node_mock(6)
        self.cloud_mock.extra['testname'] ='{n[hostname]}.{n[domain]}'.format(
            n=arv_node)
        self.make_actor()
        self.node_actor.update_arvados_node(arv_node).get(self.TIMEOUT)
        self.assertEqual(0, self.updates.sync_node.call_count)
