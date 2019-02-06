# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from future import standard_library
standard_library.install_aliases()
from builtins import range
from builtins import object

import queue
import threading
import logging

logger = logging.getLogger('arvados.cwl-runner')

class TaskQueue(object):
    def __init__(self, lock, thread_count):
        self.thread_count = thread_count
        self.task_queue = queue.Queue(maxsize=self.thread_count)
        self.task_queue_threads = []
        self.lock = lock
        self.in_flight = 0
        self.error = None

        for r in range(0, self.thread_count):
            t = threading.Thread(target=self.task_queue_func)
            self.task_queue_threads.append(t)
            t.start()

    def task_queue_func(self):
        while True:
            task = self.task_queue.get()
            if task is None:
                return
            try:
                task()
            except Exception as e:
                logger.exception("Unhandled exception running task")
                self.error = e

            with self.lock:
                self.in_flight -= 1

    def add(self, task, unlock, check_done):
        if self.thread_count > 1:
            with self.lock:
                self.in_flight += 1
        else:
            task()
            return

        while True:
            try:
                unlock.release()
                if check_done.is_set():
                    return
                self.task_queue.put(task, block=True, timeout=3)
                return
            except queue.Full:
                pass
            finally:
                unlock.acquire()


    def drain(self):
        try:
            # Drain queue
            while not self.task_queue.empty():
                self.task_queue.get(True, .1)
        except queue.Empty:
            pass

    def join(self):
        for t in self.task_queue_threads:
            self.task_queue.put(None)
        for t in self.task_queue_threads:
            t.join()
