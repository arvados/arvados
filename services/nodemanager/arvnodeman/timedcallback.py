#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import, print_function

import heapq
import time

import pykka

from .config import actor_class

class TimedCallBackActor(actor_class):
    """Send messages to other actors on a schedule.

    Other actors can call the schedule() method to schedule delivery of a
    message at a later time.  This actor runs the necessary event loop for
    delivery.
    """
    def __init__(self, max_sleep=1, timefunc=None):
        super(TimedCallBackActor, self).__init__()
        self._proxy = self.actor_ref.tell_proxy()
        self.messages = []
        self.max_sleep = max_sleep
        if timefunc is None:
            self._timefunc = time.time
        else:
            self._timefunc = timefunc

    def schedule(self, delivery_time, receiver, *args, **kwargs):
        if not self.messages:
            self._proxy.deliver()
        heapq.heappush(self.messages, (delivery_time, receiver, args, kwargs))

    def deliver(self):
        if not self.messages:
            return
        til_next = self.messages[0][0] - self._timefunc()
        if til_next <= 0:
            t, receiver, args, kwargs = heapq.heappop(self.messages)
            try:
                receiver(*args, **kwargs)
            except pykka.ActorDeadError:
                pass
        else:
            time.sleep(min(til_next, self.max_sleep))
        self._proxy.deliver()
