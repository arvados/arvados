#!/usr/bin/env python

from __future__ import absolute_import, print_function

import errno
import logging
import threading
import unittest

import mock
import pykka

from . import testutil

import arvnodeman.fullstopactor

class BogusActor(arvnodeman.fullstopactor.FullStopActor):
    def __init__(self, e):
        super(BogusActor, self).__init__()
        self.exp = e

    def doStuff(self):
        raise self.exp

class ActorUnhandledExceptionTest(unittest.TestCase):
    def test1(self):
        for e in (MemoryError(), threading.ThreadError(), OSError(errno.ENOMEM, "")):
            with mock.patch('os.killpg') as killpg_mock:
                act = BogusActor.start(e)
                act.tell({
                    'command': 'pykka_call',
                    'attr_path': ("doStuff",),
                    'args': [],
                    'kwargs': {}
                })
                act.stop(block=True)
                self.assertTrue(killpg_mock.called)

        with mock.patch('os.killpg') as killpg_mock:
            act = BogusActor.start(OSError(errno.ENOENT, ""))
            act.tell({
                'command': 'pykka_call',
                'attr_path': ("doStuff",),
                'args': [],
                'kwargs': {}
            })
            act.stop(block=True)
            self.assertFalse(killpg_mock.called)
