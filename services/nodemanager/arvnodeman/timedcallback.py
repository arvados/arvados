#!/usr/bin/env python

import heapq
import time

import pykka

from .config import actor_class

class TimedCallBackActor(actor_class):
    def __init__(self, max_sleep=1):
        super(TimedCallBackActor, self).__init__()
        self._proxy = self.actor_ref.proxy()
        self.messages = []
        self.max_sleep = max_sleep

    def schedule(self, delivery_time, receiver, *args, **kwargs):
        heapq.heappush(self.messages, (delivery_time, receiver, args, kwargs))
        self._proxy.deliver()

    def deliver(self):
        if not self.messages:
            return None
        til_next = self.messages[0][0] - time.time()
        if til_next < 0:
            t, receiver, args, kwargs = heapq.heappop(self.messages)
            try:
                receiver(*args, **kwargs)
            except pykka.ActorDeadError:
                pass
        else:
            time.sleep(min(til_next, self.max_sleep))
        self._proxy.deliver()
