#!/usr/bin/env python

from __future__ import absolute_import, print_function

import time
import unittest

import mock
import pykka

import arvnodeman.timedcallback as timedcallback
from . import testutil

@testutil.no_sleep
class TimedCallBackActorTestCase(testutil.ActorTestMixin, unittest.TestCase):
    def test_immediate_turnaround(self):
        future = self.FUTURE_CLASS()
        deliverer = timedcallback.TimedCallBackActor.start().proxy()
        deliverer.schedule(time.time() - 1, future.set, 'immediate')
        self.assertEqual('immediate', future.get(self.TIMEOUT))

    def test_delayed_turnaround(self):
        future = self.FUTURE_CLASS()
        with mock.patch('time.time', return_value=0) as mock_now:
            deliverer = timedcallback.TimedCallBackActor.start().proxy()
            deliverer.schedule(1, future.set, 'delayed')
            self.assertRaises(pykka.Timeout, future.get, .5)
            mock_now.return_value = 2
            self.assertEqual('delayed', future.get(self.TIMEOUT))

    def test_out_of_order_scheduling(self):
        future1 = self.FUTURE_CLASS()
        future2 = self.FUTURE_CLASS()
        with mock.patch('time.time', return_value=1.5) as mock_now:
            deliverer = timedcallback.TimedCallBackActor.start().proxy()
            deliverer.schedule(2, future2.set, 'second')
            deliverer.schedule(1, future1.set, 'first')
            self.assertEqual('first', future1.get(self.TIMEOUT))
            self.assertRaises(pykka.Timeout, future2.get, .1)
            mock_now.return_value = 3
            self.assertEqual('second', future2.get(self.TIMEOUT))

    def test_dead_actors_ignored(self):
        receiver = mock.Mock(name='dead_actor', spec=pykka.ActorRef)
        receiver.tell.side_effect = pykka.ActorDeadError
        deliverer = timedcallback.TimedCallBackActor.start().proxy()
        deliverer.schedule(time.time() - 1, receiver.tell, 'error')
        self.wait_for_call(receiver.tell)
        receiver.tell.assert_called_with('error')
        self.assertTrue(deliverer.actor_ref.is_alive(), "deliverer died")


if __name__ == '__main__':
    unittest.main()

