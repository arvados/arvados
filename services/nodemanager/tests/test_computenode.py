#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import, print_function

import time
import unittest

import arvados.errors as arverror
import mock

import arvnodeman.computenode as cnode
from . import testutil

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
