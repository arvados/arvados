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
        self.monitor.subscribe(self.subscriber)
        self.wait_for_call(self.subscriber)
        self.subscriber.assert_called_with('test1')
        self.wait_for_call(self.timer.schedule)

    def test_poll_loop_continues_after_failure(self):
        self.build_monitor(self.MockClientError)
        self.monitor.subscribe(self.subscriber)
        self.wait_for_call(self.timer.schedule)
        self.assertTrue(self.monitor.actor_ref.is_alive(),
                        "poll loop died after error")
        self.assertFalse(self.subscriber.called,
                         "poll loop notified subscribers after error")

    def test_late_subscribers_get_responses(self):
        self.build_monitor(['late_test'])
        self.monitor.subscribe(lambda response: None)
        self.monitor.subscribe(self.subscriber)
        self.monitor.poll()
        self.wait_for_call(self.subscriber)
        self.subscriber.assert_called_with('late_test')

    def test_survive_dead_subscriptions(self):
        self.build_monitor(['survive1', 'survive2'])
        dead_subscriber = mock.Mock(name='dead_subscriber')
        dead_subscriber.side_effect = pykka.ActorDeadError
        self.monitor.subscribe(dead_subscriber)
        self.wait_for_call(dead_subscriber)
        self.monitor.subscribe(self.subscriber)
        self.monitor.poll()
        self.wait_for_call(self.subscriber)
        self.subscriber.assert_called_with('survive2')
        self.assertTrue(self.monitor.actor_ref.is_alive(),
                        "poll loop died from dead subscriber")

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
        self.monitor.subscribe_to(2, self.subscriber)
        self.wait_for_call(self.subscriber)
        self.subscriber.assert_called_with({'key': 2})

    def test_survive_dead_key_subscriptions(self):
        item = {'key': 3}
        self.build_monitor([[item], [item]])
        dead_subscriber = mock.Mock(name='dead_subscriber')
        dead_subscriber.side_effect = pykka.ActorDeadError
        self.monitor.subscribe_to(3, dead_subscriber)
        self.wait_for_call(dead_subscriber)
        self.monitor.subscribe_to(3, self.subscriber)
        self.monitor.poll()
        self.wait_for_call(self.subscriber)
        self.subscriber.assert_called_with(item)
        self.assertTrue(self.monitor.actor_ref.is_alive(),
                        "poll loop died from dead key subscriber")

    def test_mixed_subscriptions(self):
        item = {'key': 4}
        self.build_monitor([[item], [item]])
        key_subscriber = mock.Mock(name='key_subscriber')
        self.monitor.subscribe(self.subscriber)
        self.monitor.subscribe_to(4, key_subscriber)
        self.monitor.poll()
        self.wait_for_call(self.subscriber)
        self.subscriber.assert_called_with([item])
        key_subscriber.assert_called_with(item)

    def test_subscription_to_missing_key(self):
        self.build_monitor([[]])
        self.monitor.subscribe_to('nonesuch', self.subscriber)
        self.wait_for_call(self.subscriber)
        self.subscriber.assert_called_with(None)


if __name__ == '__main__':
    unittest.main()

