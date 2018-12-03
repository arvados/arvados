# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import functools
import mock
import sys
import unittest
import json
import logging
import os
import threading

from arvados_cwl.task_queue import TaskQueue

def success_task():
    pass

def fail_task():
    raise Exception("Testing error handling")

class TestTaskQueue(unittest.TestCase):
    def test_tq(self):
        tq = TaskQueue(threading.Lock(), 2)
        try:
            self.assertIsNone(tq.error)

            unlock = threading.Lock()
            unlock.acquire()
            check_done = threading.Event()

            tq.add(success_task, unlock, check_done)
            tq.add(success_task, unlock, check_done)
            tq.add(success_task, unlock, check_done)
            tq.add(success_task, unlock, check_done)
        finally:
            tq.join()

        self.assertIsNone(tq.error)


    def test_tq_error(self):
        tq = TaskQueue(threading.Lock(), 2)
        try:
            self.assertIsNone(tq.error)

            unlock = threading.Lock()
            unlock.acquire()
            check_done = threading.Event()

            tq.add(success_task, unlock, check_done)
            tq.add(success_task, unlock, check_done)
            tq.add(fail_task, unlock, check_done)
            tq.add(success_task, unlock, check_done)
        finally:
            tq.join()

        self.assertIsNotNone(tq.error)
