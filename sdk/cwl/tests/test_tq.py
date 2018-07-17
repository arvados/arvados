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

        self.assertIsNone(tq.error)

        tq.add(success_task)
        tq.add(success_task)
        tq.add(success_task)
        tq.add(success_task)

        tq.join()

        self.assertIsNone(tq.error)


    def test_tq_error(self):
        tq = TaskQueue(threading.Lock(), 2)

        self.assertIsNone(tq.error)

        tq.add(success_task)
        tq.add(success_task)
        tq.add(fail_task)
        tq.add(success_task)

        tq.join()

        self.assertIsNotNone(tq.error)
