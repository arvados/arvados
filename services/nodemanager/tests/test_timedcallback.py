#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

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
        receiver = mock.Mock()
        deliverer = timedcallback.TimedCallBackActor.start().proxy()
        deliverer.schedule(time.time() - 1, receiver,
                           'immediate').get(self.TIMEOUT)
        self.stop_proxy(deliverer)
        receiver.assert_called_with('immediate')

    def test_delayed_turnaround(self):
        receiver = mock.Mock()
        mock_now = mock.Mock()
        mock_now.return_value = 0
        deliverer = timedcallback.TimedCallBackActor.start(timefunc=mock_now).proxy()
        deliverer.schedule(1, receiver, 'delayed')
        deliverer.schedule(3, receiver, 'failure').get(self.TIMEOUT)
        self.assertFalse(receiver.called)
        mock_now.return_value = 2
        deliverer.schedule(3, receiver, 'failure').get(self.TIMEOUT)
        self.stop_proxy(deliverer)
        receiver.assert_called_with('delayed')

    def test_out_of_order_scheduling(self):
        receiver = mock.Mock()
        mock_now = mock.Mock()
        mock_now.return_value = 1.5
        deliverer = timedcallback.TimedCallBackActor.start(timefunc=mock_now).proxy()
        deliverer.schedule(2, receiver, 'second')
        deliverer.schedule(1, receiver, 'first')
        deliverer.schedule(3, receiver, 'failure').get(self.TIMEOUT)
        receiver.assert_called_with('first')
        mock_now.return_value = 2.5
        deliverer.schedule(3, receiver, 'failure').get(self.TIMEOUT)
        self.stop_proxy(deliverer)
        receiver.assert_called_with('second')

    def test_dead_actors_ignored(self):
        receiver = mock.Mock(name='dead_actor', spec=pykka.ActorRef)
        receiver.tell.side_effect = pykka.ActorDeadError
        deliverer = timedcallback.TimedCallBackActor.start().proxy()
        deliverer.schedule(time.time() - 1, receiver.tell,
                           'error').get(self.TIMEOUT)
        self.assertTrue(self.stop_proxy(deliverer), "deliverer died")
        receiver.tell.assert_called_with('error')


if __name__ == '__main__':
    unittest.main()
