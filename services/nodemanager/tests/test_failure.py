#!/usr/bin/env python

from __future__ import absolute_import, print_function

import logging
import unittest

import mock
import pykka

from . import testutil

import arvnodeman.fullstopactor

class BogusActor(arvnodeman.fullstopactor.FullStopActor):
    def doStuff(self):
        raise MemoryError

class ActorUnhandledExceptionTest(unittest.TestCase):
    def test1(self):
        with mock.patch('os.killpg') as killpg_mock:
            act = BogusActor.start()
            act.tell({
                'command': 'pykka_call',
                'attr_path': ("doStuff",),
                'args': [],
                'kwargs': {}
            })
            act.stop(block=True)
            self.assertTrue(killpg_mock.called)
