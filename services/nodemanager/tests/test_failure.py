#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import, print_function

import errno
import logging
import time
import threading
import unittest

import mock
import pykka

from . import testutil

import arvnodeman.baseactor
import arvnodeman.status as status

class BogusActor(arvnodeman.baseactor.BaseNodeManagerActor):
    def __init__(self, e, killfunc=None):
        super(BogusActor, self).__init__(killfunc=killfunc)
        self.exp = e

    def doStuff(self):
        raise self.exp

    def ping(self):
        # Called by WatchdogActorTest, this delay is longer than the test timeout
        # of 1 second, which should cause the watchdog ping to fail.
        time.sleep(2)
        return True

class ActorUnhandledExceptionTest(testutil.ActorTestMixin, unittest.TestCase):
    def test_fatal_error(self):
        for e in (MemoryError(), threading.ThreadError(), OSError(errno.ENOMEM, "")):
            kill_mock = mock.Mock('os.kill')
            bgact = BogusActor.start(e, killfunc=kill_mock)
            act_thread = bgact.proxy().get_thread().get()
            act = bgact.tell_proxy()
            act.doStuff()
            act.actor_ref.stop(block=True)
            act_thread.join()
            self.assertTrue(kill_mock.called)

    def test_nonfatal_error(self):
        status.tracker.update({'actor_exceptions': 0})
        kill_mock = mock.Mock('os.kill')
        bgact = BogusActor.start(OSError(errno.ENOENT, ""), killfunc=kill_mock)
        act_thread = bgact.proxy().get_thread().get()
        act = bgact.tell_proxy()
        act.doStuff()
        act.actor_ref.stop(block=True)
        act_thread.join()
        self.assertFalse(kill_mock.called)
        self.assertEqual(1, status.tracker.get('actor_exceptions'))

class WatchdogActorTest(testutil.ActorTestMixin, unittest.TestCase):

    def test_time_timout(self):
        kill_mock = mock.Mock('os.kill')
        act = BogusActor.start(OSError(errno.ENOENT, ""))
        watch = arvnodeman.baseactor.WatchdogActor.start(1, act, killfunc=kill_mock)
        time.sleep(1)
        watch.stop(block=True)
        act.stop(block=True)
        self.assertTrue(kill_mock.called)
