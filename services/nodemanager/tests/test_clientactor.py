#!/usr/bin/env python

from __future__ import absolute_import, print_function

import unittest

import mock
import pykka

import arvnodeman.clientactor as clientactor
from . import testutil

class RemotePollLoopActorTestCase(testutil.RemotePollLoopActorTestMixin,
                                  unittest.TestCase):
    class MockClientError(Exception):
        pass

    class TestActor(clientactor.RemotePollLoopActor):
        LOGGER_NAME = 'arvnodeman.testpoll'

        def _send_request(self):
            return self._client()
    TestActor.CLIENT_ERRORS = (MockClientError,)
    TEST_CLASS = TestActor


    def build_monitor(self, side_effect, *args, **kwargs):
        super(RemotePollLoopActorTestCase, self).build_monitor(*args, **kwargs)
        self.client.side_effect = side_effect

    def test_poll_loop_starts_after_subscription(self):
        self.build_monitor(['test1'])
        self.monitor.subscribe(self.subscriber).get(self.TIMEOUT)
        self.stop_proxy(self.monitor)
        self.subscriber.assert_called_with('test1')
        self.assertTrue(self.timer.schedule.called)

    def test_poll_loop_continues_after_failure(self):
        self.build_monitor(self.MockClientError)
        self.monitor.subscribe(self.subscriber).get(self.TIMEOUT)
        self.assertTrue(self.stop_proxy(self.monitor),
                        "poll loop died after error")
        self.assertTrue(self.timer.schedule.called,
                        "poll loop did not reschedule after error")
        self.assertFalse(self.subscriber.called,
                         "poll loop notified subscribers after error")

    def test_late_subscribers_get_responses(self):
        self.build_monitor(['pre_late_test', 'late_test'])
        mock_subscriber = mock.Mock(name='mock_subscriber')
        self.monitor.subscribe(mock_subscriber).get(self.TIMEOUT)
        self.monitor.subscribe(self.subscriber)
        self.monitor.poll().get(self.TIMEOUT)
        self.stop_proxy(self.monitor)
        self.subscriber.assert_called_with('late_test')

    def test_survive_dead_subscriptions(self):
        self.build_monitor(['survive1', 'survive2'])
        dead_subscriber = mock.Mock(name='dead_subscriber')
        dead_subscriber.side_effect = pykka.ActorDeadError
        self.monitor.subscribe(dead_subscriber)
        self.monitor.subscribe(self.subscriber)
        self.monitor.poll().get(self.TIMEOUT)
        self.assertTrue(self.stop_proxy(self.monitor),
                        "poll loop died from dead subscriber")
        self.subscriber.assert_called_with('survive2')

    def check_poll_timers(self, *test_times):
        schedule_mock = self.timer.schedule
        last_expect = None
        with mock.patch('time.time') as time_mock:
            for fake_time, expect_next in test_times:
                time_mock.return_value = fake_time
                self.monitor.poll(last_expect).get(self.TIMEOUT)
                self.assertTrue(schedule_mock.called)
                self.assertEqual(expect_next, schedule_mock.call_args[0][0])
                schedule_mock.reset_mock()
                last_expect = expect_next

    def test_poll_timing_on_consecutive_successes_with_drift(self):
        self.build_monitor(['1', '2'], poll_wait=3, max_poll_wait=14)
        self.check_poll_timers((0, 3), (4, 6))

    def test_poll_backoff_on_failures(self):
        self.build_monitor(self.MockClientError, poll_wait=3, max_poll_wait=14)
        self.check_poll_timers((0, 6), (6, 18), (18, 32))

    def test_poll_timing_after_error_recovery(self):
        self.build_monitor(['a', self.MockClientError(), 'b'],
                           poll_wait=3, max_poll_wait=14)
        self.check_poll_timers((0, 3), (4, 10), (10, 13))

    def test_no_subscriptions_by_key_without_support(self):
        self.build_monitor([])
        with self.assertRaises(AttributeError):
            self.monitor.subscribe_to('key')


class RemotePollLoopActorWithKeysTestCase(testutil.RemotePollLoopActorTestMixin,
                                          unittest.TestCase):
    class TestActor(RemotePollLoopActorTestCase.TestActor):
        def _item_key(self, item):
            return item['key']
    TEST_CLASS = TestActor


    def build_monitor(self, side_effect, *args, **kwargs):
        super(RemotePollLoopActorWithKeysTestCase, self).build_monitor(
            *args, **kwargs)
        self.client.side_effect = side_effect

    def test_key_subscription(self):
        self.build_monitor([[{'key': 1}, {'key': 2}]])
        self.monitor.subscribe_to(2, self.subscriber).get(self.TIMEOUT)
        self.stop_proxy(self.monitor)
        self.subscriber.assert_called_with({'key': 2})

    def test_survive_dead_key_subscriptions(self):
        item = {'key': 3}
        self.build_monitor([[item], [item]])
        dead_subscriber = mock.Mock(name='dead_subscriber')
        dead_subscriber.side_effect = pykka.ActorDeadError
        self.monitor.subscribe_to(3, dead_subscriber)
        self.monitor.subscribe_to(3, self.subscriber)
        self.monitor.poll().get(self.TIMEOUT)
        self.assertTrue(self.stop_proxy(self.monitor),
                        "poll loop died from dead key subscriber")
        self.subscriber.assert_called_with(item)

    def test_mixed_subscriptions(self):
        item = {'key': 4}
        self.build_monitor([[item], [item]])
        key_subscriber = mock.Mock(name='key_subscriber')
        self.monitor.subscribe(self.subscriber)
        self.monitor.subscribe_to(4, key_subscriber)
        self.monitor.poll().get(self.TIMEOUT)
        self.stop_proxy(self.monitor)
        self.subscriber.assert_called_with([item])
        key_subscriber.assert_called_with(item)

    def test_subscription_to_missing_key(self):
        self.build_monitor([[]])
        self.monitor.subscribe_to('nonesuch', self.subscriber).get(self.TIMEOUT)
        self.stop_proxy(self.monitor)
        self.subscriber.assert_called_with(None)


if __name__ == '__main__':
    unittest.main()
