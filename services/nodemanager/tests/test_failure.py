#!/usr/bin/env python

from __future__ import absolute_import, print_function

import errno
import logging
import threading
import unittest

import mock
import pykka

from . import testutil

import arvnodeman.baseactor

class BogusActor(arvnodeman.baseactor.BaseNodeManagerActor):
    def __init__(self, e):
        super(BogusActor, self).__init__()
        self.exp = e

    def doStuff(self):
        raise self.exp

class ActorUnhandledExceptionTest(unittest.TestCase):
    def test_fatal_error(self):
        for e in (MemoryError(), threading.ThreadError(), OSError(errno.ENOMEM, "")):
            with mock.patch('os.killpg') as killpg_mock:
                act = BogusActor.start(e).tell_proxy()
                act.doStuff()
                act.actor_ref.stop(block=True)
                self.assertTrue(killpg_mock.called)

    def test_nonfatal_error(self):
        with mock.patch('os.killpg') as killpg_mock:
            act = BogusActor.start(OSError(errno.ENOENT, "")).tell_proxy()
            act.doStuff()
            act.actor_ref.stop(block=True)
            self.assertFalse(killpg_mock.called)
