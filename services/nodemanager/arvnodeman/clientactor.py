#!/usr/bin/env python

from __future__ import absolute_import, print_function

import logging
import time

import pykka

from .config import actor_class

def _notify_subscribers(response, subscribers):
    """Send the response to all the subscriber methods.

    If any of the subscriber actors have stopped, remove them from the
    subscriber set.
    """
    dead_subscribers = set()
    for subscriber in subscribers:
        try:
            subscriber(response)
        except pykka.ActorDeadError:
            dead_subscribers.add(subscriber)
    subscribers.difference_update(dead_subscribers)

class RemotePollLoopActor(actor_class):
    """Abstract actor class to regularly poll a remote service.

    This actor sends regular requests to a remote service, and sends each
    response to subscribers.  It takes care of error handling, and retrying
    requests with exponential backoff.

    To use this actor, define the _send_request method.  If you also
    define an _item_key method, this class will support subscribing to
    a specific item by key in responses.
    """
    def __init__(self, client, timer_actor, poll_wait=60, max_poll_wait=180):
        super(RemotePollLoopActor, self).__init__()
        self._client = client
        self._timer = timer_actor
        self._logger = logging.getLogger(self.LOGGER_NAME)
        self._later = self.actor_ref.proxy()
        self._polling_started = False
        self.log_prefix = "{} (at {})".format(self.__class__.__name__, id(self))
        self.min_poll_wait = poll_wait
        self.max_poll_wait = max_poll_wait
        self.poll_wait = self.min_poll_wait
        self.all_subscribers = set()
        self.key_subscribers = {}
        if hasattr(self, '_item_key'):
            self.subscribe_to = self._subscribe_to

    def _start_polling(self):
        if not self._polling_started:
            self._polling_started = True
            self._later.poll()

    def subscribe(self, subscriber):
        self.all_subscribers.add(subscriber)
        self._logger.debug("%r subscribed to all events", subscriber)
        self._start_polling()

    # __init__ exposes this method to the proxy if the subclass defines
    # _item_key.
    def _subscribe_to(self, key, subscriber):
        self.key_subscribers.setdefault(key, set()).add(subscriber)
        self._logger.debug("%r subscribed to events for '%s'", subscriber, key)
        self._start_polling()

    def _send_request(self):
        raise NotImplementedError("subclasses must implement request method")

    def _got_response(self, response):
        self._logger.debug("%s got response with %d items",
                           self.log_prefix, len(response))
        self.poll_wait = self.min_poll_wait
        _notify_subscribers(response, self.all_subscribers)
        if hasattr(self, '_item_key'):
            items = {self._item_key(x): x for x in response}
            for key, subscribers in self.key_subscribers.iteritems():
                _notify_subscribers(items.get(key), subscribers)

    def _got_error(self, error):
        self.poll_wait = min(self.poll_wait * 2, self.max_poll_wait)
        return "{} got error: {} - waiting {} seconds".format(
            self.log_prefix, error, self.poll_wait)

    def is_common_error(self, exception):
        return False

    def poll(self, scheduled_start=None):
        self._logger.debug("%s sending poll", self.log_prefix)
        start_time = time.time()
        if scheduled_start is None:
            scheduled_start = start_time
        try:
            response = self._send_request()
        except Exception as error:
            errmsg = self._got_error(error)
            if self.is_common_error(error):
                self._logger.warning(errmsg)
            else:
                self._logger.exception(errmsg)
            next_poll = start_time + self.poll_wait
        else:
            self._got_response(response)
            next_poll = scheduled_start + self.poll_wait
        end_time = time.time()
        if next_poll < end_time:  # We've drifted too much; start fresh.
            next_poll = end_time + self.poll_wait
        self._timer.schedule(next_poll, self._later.poll, next_poll)
