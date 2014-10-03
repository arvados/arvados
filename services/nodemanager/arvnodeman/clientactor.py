#!/usr/bin/env python

import logging
import time

import pykka

from .config import actor_class

def _notify_subscribers(response, subscribers):
    dead_subscribers = set()
    for subscriber in subscribers:
        try:
            subscriber(response)
        except pykka.ActorDeadError:
            dead_subscribers.add(subscriber)
    subscribers.difference_update(dead_subscribers)

class RemotePollLoopActor(actor_class):
    def __init__(self, client, timer_actor, poll_wait=60, max_poll_wait=180):
        super(RemotePollLoopActor, self).__init__()
        self._client = client
        self._timer = timer_actor
        self._logger = logging.getLogger(self.LOGGER_NAME)
        self._later = self.actor_ref.proxy()
        self.min_poll_wait = poll_wait
        self.max_poll_wait = max_poll_wait
        self.poll_wait = self.min_poll_wait
        self.last_poll_time = None
        self.all_subscribers = set()
        self.key_subscribers = {}
        if hasattr(self, '_item_key'):
            self.subscribe_to = self._subscribe_to

    def _start_polling(self):
        if self.last_poll_time is None:
            self.last_poll_time = time.time()
            self._later.poll()

    def subscribe(self, subscriber):
        self.all_subscribers.add(subscriber)
        self._logger.debug("%r subscribed to all events", subscriber)
        self._start_polling()

    def _subscribe_to(self, key, subscriber):
        self.key_subscribers.setdefault(key, set()).add(subscriber)
        self._logger.debug("%r subscribed to events for '%s'", subscriber, key)
        self._start_polling()

    def _send_request(self):
        raise NotImplementedError("subclasses must implement request method")

    def _got_response(self, response):
        self.poll_wait = self.min_poll_wait
        _notify_subscribers(response, self.all_subscribers)
        if hasattr(self, '_item_key'):
            items = {self._item_key(x): x for x in response}
            for key, subscribers in self.key_subscribers.iteritems():
                _notify_subscribers(items.get(key), subscribers)

    def _got_error(self, error):
        self.poll_wait = min(self.poll_wait * 2, self.max_poll_wait)
        self._logger.warning("Client error: %s - waiting %s seconds",
                             error, self.poll_wait)

    def poll(self):
        start_time = time.time()
        try:
            response = self._send_request()
        except self.CLIENT_ERRORS as error:
            self.last_poll_time = start_time
            self._got_error(error)
        else:
            self.last_poll_time += self.poll_wait
            self._got_response(response)
        self._timer.schedule(self.last_poll_time + self.poll_wait,
                             self._later.poll)
