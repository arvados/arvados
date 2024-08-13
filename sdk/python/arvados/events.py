# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
"""Follow events on an Arvados cluster

This module provides different ways to get notified about events that happen
on an Arvados cluster. You indicate which events you want updates about, and
provide a function that is called any time one of those events is received
from the server.

`subscribe` is the main entry point. It helps you construct one of the two
API-compatible client classes: `EventClient` (which uses WebSockets) or
`PollClient` (which periodically queries the logs list methods).
"""

import enum
import json
import logging
import os
import re
import ssl
import sys
import _thread
import threading
import time

import websockets.exceptions as ws_exc
import websockets.sync.client as ws_client

from . import config
from . import errors
from . import util
from .retry import RetryLoop
from ._version import __version__

from typing import (
    Any,
    Callable,
    Dict,
    Iterable,
    List,
    Optional,
    Union,
)

EventCallback = Callable[[Dict[str, Any]], object]
"""Type signature for an event handler callback"""
FilterCondition = List[Union[None, str, 'Filter']]
"""Type signature for a single filter condition"""
Filter = List[FilterCondition]
"""Type signature for an entire filter"""

_logger = logging.getLogger('arvados.events')

class WSMethod(enum.Enum):
    """Arvados WebSocket methods

    This enum represents valid values for the `method` field in messages
    sent to an Arvados WebSocket server.
    """
    SUBSCRIBE = 'subscribe'
    SUB = SUBSCRIBE
    UNSUBSCRIBE = 'unsubscribe'
    UNSUB = UNSUBSCRIBE


class EventClient(threading.Thread):
    """Follow Arvados events via WebSocket

    EventClient follows events on Arvados cluster published by the WebSocket
    server. Users can select the events they want to follow and run their own
    callback function on each.
    """
    _USER_AGENT = 'Python/{}.{}.{} arvados.events/{}'.format(
        *sys.version_info[:3],
        __version__,
    )

    def __init__(
            self,
            url: str,
            filters: Optional[Filter],
            on_event_cb: EventCallback,
            last_log_id: Optional[int]=None,
            *,
            insecure: Optional[bool]=None,
    ) -> None:
        """Initialize a WebSocket client

        Constructor arguments:

        * url: str --- The `wss` URL for an Arvados WebSocket server.

        * filters: arvados.events.Filter | None --- One event filter to
          subscribe to after connecting to the WebSocket server. If not
          specified, the client will subscribe to all events.

        * on_event_cb: arvados.events.EventCallback --- When the client
          receives an event from the WebSocket server, it calls this
          function with the event object.

        * last_log_id: int | None --- If specified, this will be used as the
          value for the `last_log_id` field in subscribe messages sent by
          the client.

        Constructor keyword arguments:

        * insecure: bool | None --- If `True`, the client will not check the
          validity of the server's TLS certificate. If not specified, uses
          the value from the user's `ARVADOS_API_HOST_INSECURE` setting.
        """
        self.url = url
        self.filters = [filters or []]
        self.on_event_cb = on_event_cb
        self.last_log_id = last_log_id
        self.is_closed = threading.Event()
        self._ssl_ctx = ssl.create_default_context(
            purpose=ssl.Purpose.SERVER_AUTH,
            cafile=util.ca_certs_path(),
        )
        if insecure is None:
            insecure = config.flag_is_true('ARVADOS_API_HOST_INSECURE')
        if insecure:
            self._ssl_ctx.check_hostname = False
            self._ssl_ctx.verify_mode = ssl.CERT_NONE
        self._subscribe_lock = threading.Lock()
        self._connect()
        super().__init__(daemon=True)
        self.start()

    def _connect(self) -> None:
        # There are no locks protecting this method. After the thread starts,
        # it should only be called from inside.
        self._client = ws_client.connect(
            self.url,
            logger=_logger,
            ssl_context=self._ssl_ctx,
            user_agent_header=self._USER_AGENT,
        )
        self._client_ok = True

    def _subscribe(self, f: Filter, last_log_id: Optional[int]) -> None:
        extra = {}
        if last_log_id is not None:
            extra['last_log_id'] = last_log_id
        return self._update_sub(WSMethod.SUBSCRIBE, f, **extra)

    def _update_sub(self, method: WSMethod, f: Filter, **extra: Any) -> None:
        msg = json.dumps({
            'method': method.value,
            'filters': f,
            **extra,
        })
        self._client.send(msg)

    def close(self, code: int=1000, reason: str='', timeout: float=0) -> None:
        """Close the WebSocket connection and stop processing events

        Arguments:

        * code: int --- The WebSocket close code sent to the server when
          disconnecting. Default 1000.

        * reason: str --- The WebSocket close reason sent to the server when
          disconnecting. Default is an empty string.

        * timeout: float --- How long to wait for the WebSocket server to
          acknowledge the disconnection, in seconds. Default 0, which means
          no timeout.
        """
        self.is_closed.set()
        self._client.close_timeout = timeout or None
        self._client.close(code, reason)

    def run_forever(self) -> None:
        """Run the WebSocket client indefinitely

        This method blocks until the `close` method is called (e.g., from
        another thread) or the client permanently loses its connection.
        """
        # Have to poll here to let KeyboardInterrupt get raised.
        while not self.is_closed.wait(1):
            pass

    def subscribe(self, f: Filter, last_log_id: Optional[int]=None) -> None:
        """Subscribe to another set of events from the server

        Arguments:

        * f: arvados.events.Filter | None --- One filter to subscribe to
          events for.

        * last_log_id: int | None --- If specified, request events starting
          from this id. If not specified, the server will only send events
          that occur after processing the subscription.
        """
        with self._subscribe_lock:
            self._subscribe(f, last_log_id)
            self.filters.append(f)

    def unsubscribe(self, f: Filter) -> None:
        """Unsubscribe from an event stream

        Arguments:

        * f: arvados.events.Filter | None --- One event filter to stop
        receiving events for.
        """
        with self._subscribe_lock:
            try:
                index = self.filters.index(f)
            except ValueError:
                raise ValueError(f"filter not subscribed: {f!r}") from None
            self._update_sub(WSMethod.UNSUBSCRIBE, f)
            del self.filters[index]

    def on_closed(self) -> None:
        """Handle disconnection from the WebSocket server

        This method is called when the client loses its connection from
        receiving events. This implementation tries to establish a new
        connection if it was not closed client-side.
        """
        if self.is_closed.is_set():
            return
        _logger.warning("Unexpected close. Reconnecting.")
        for _ in RetryLoop(num_retries=25, backoff_start=.1, max_wait=15):
            try:
                self._connect()
            except Exception as e:
                _logger.warning("Error '%s' during websocket reconnect.", e)
            else:
                _logger.warning("Reconnect successful.")
                break
        else:
            _logger.error("EventClient thread could not contact websocket server.")
            self.is_closed.set()
            _thread.interrupt_main()

    def on_event(self, m: Dict[str, Any]) -> None:
        """Handle an event from the WebSocket server

        This method is called whenever the client receives an event from the
        server. This implementation records the `id` field internally, then
        calls the callback function provided at initialization time.

        Arguments:

        * m: Dict[str, Any] --- The event object, deserialized from JSON.
        """
        try:
            self.last_log_id = m['id']
        except KeyError:
            pass
        try:
            self.on_event_cb(m)
        except Exception:
            _logger.exception("Unexpected exception from event callback.")
            _thread.interrupt_main()

    def run(self) -> None:
        """Run the client loop

        This method runs in a separate thread to receive and process events
        from the server.
        """
        self.name = f'ArvadosWebsockets-{self.ident}'
        while self._client_ok and not self.is_closed.is_set():
            try:
                with self._subscribe_lock:
                    for f in self.filters:
                        self._subscribe(f, self.last_log_id)
                for msg_s in self._client:
                    if not self.is_closed.is_set():
                        msg = json.loads(msg_s)
                        self.on_event(msg)
            except ws_exc.ConnectionClosed:
                self._client_ok = False
                self.on_closed()


class PollClient(threading.Thread):
    """Follow Arvados events via polling logs

    PollClient follows events on Arvados cluster by periodically running
    logs list API calls. Users can select the events they want to follow and
    run their own callback function on each.
    """
    def __init__(
            self,
            api: 'arvados.api_resources.ArvadosAPIClient',
            filters: Optional[Filter],
            on_event: EventCallback,
            poll_time: float=15,
            last_log_id: Optional[int]=None,
    ) -> None:
        """Initialize a polling client

        Constructor arguments:

        * api: arvados.api_resources.ArvadosAPIClient --- The Arvados API
          client used to query logs. It will be used in a separate thread,
          so if it is not an instance of `arvados.api.ThreadSafeAPIClient`
          it should not be reused after the thread is started.

        * filters: arvados.events.Filter | None --- One event filter to
          subscribe to after connecting to the WebSocket server. If not
          specified, the client will subscribe to all events.

        * on_event: arvados.events.EventCallback --- When the client
          receives an event from the WebSocket server, it calls this
          function with the event object.

        * poll_time: float --- The number of seconds to wait between querying
          logs. Default 15.

        * last_log_id: int | None --- If specified, queries will include a
          filter for logs with an `id` at least this value.
        """
        super(PollClient, self).__init__()
        self.api = api
        if filters:
            self.filters = [filters]
        else:
            self.filters = [[]]
        self.on_event = on_event
        self.poll_time = poll_time
        self.daemon = True
        self.last_log_id = last_log_id
        self._closing = threading.Event()
        self._closing_lock = threading.RLock()

        if self.last_log_id != None:
            # Caller supplied the last-seen event ID from a previous
            # connection.
            self._skip_old_events = [["id", ">", str(self.last_log_id)]]
        else:
            # We need to do a reverse-order query to find the most
            # recent event ID (see "if not self._skip_old_events"
            # in run()).
            self._skip_old_events = False

    def run(self):
        """Run the client loop

        This method runs in a separate thread to poll and process events
        from the server.
        """
        self.on_event({'status': 200})

        while not self._closing.is_set():
            moreitems = False
            for f in self.filters:
                for tries_left in RetryLoop(num_retries=25, backoff_start=.1, max_wait=self.poll_time):
                    try:
                        if not self._skip_old_events:
                            # If the caller didn't provide a known
                            # recent ID, our first request will ask
                            # for the single most recent event from
                            # the last 2 hours (the time restriction
                            # avoids doing an expensive database
                            # query, and leaves a big enough margin to
                            # account for clock skew). If we do find a
                            # recent event, we remember its ID but
                            # then discard it (we are supposed to be
                            # returning new/current events, not old
                            # ones).
                            #
                            # Subsequent requests will get multiple
                            # events in chronological order, and
                            # filter on that same cutoff time, or
                            # (once we see our first matching event)
                            # the ID of the last-seen event.
                            #
                            # Note: self._skip_old_events must not be
                            # set until the threshold is decided.
                            # Otherwise, tests will be unreliable.
                            filter_by_time = [[
                                "created_at", ">=",
                                time.strftime(
                                    "%Y-%m-%dT%H:%M:%SZ",
                                    time.gmtime(time.time()-7200))]]
                            items = self.api.logs().list(
                                order="id desc",
                                limit=1,
                                filters=f+filter_by_time).execute()
                            if items["items"]:
                                self._skip_old_events = [
                                    ["id", ">", str(items["items"][0]["id"])]]
                                items = {
                                    "items": [],
                                    "items_available": 0,
                                }
                            else:
                                # No recent events. We can keep using
                                # the same timestamp threshold until
                                # we receive our first new event.
                                self._skip_old_events = filter_by_time
                        else:
                            # In this case, either we know the most
                            # recent matching ID, or we know there
                            # were no matching events in the 2-hour
                            # window before subscribing. Either way we
                            # can safely ask for events in ascending
                            # order.
                            items = self.api.logs().list(
                                order="id asc",
                                filters=f+self._skip_old_events).execute()
                        break
                    except errors.ApiError as error:
                        pass
                    else:
                        tries_left = 0
                        break
                if tries_left == 0:
                    _logger.exception("PollClient thread could not contact API server.")
                    with self._closing_lock:
                        self._closing.set()
                    _thread.interrupt_main()
                    return
                for i in items["items"]:
                    self._skip_old_events = [["id", ">", str(i["id"])]]
                    with self._closing_lock:
                        if self._closing.is_set():
                            return
                        try:
                            self.on_event(i)
                        except Exception as e:
                            _logger.exception("Unexpected exception from event callback.")
                            _thread.interrupt_main()
                if items["items_available"] > len(items["items"]):
                    moreitems = True
            if not moreitems:
                self._closing.wait(self.poll_time)

    def run_forever(self):
        """Run the polling client indefinitely

        This method blocks until the `close` method is called (e.g., from
        another thread) or the client permanently loses its connection.
        """
        # Have to poll here, otherwise KeyboardInterrupt will never get processed.
        while not self._closing.is_set():
            self._closing.wait(1)

    def close(self, code: Optional[int]=None, reason: Optional[str]=None, timeout: float=0) -> None:
        """Stop polling and processing events

        Arguments:

        * code: Optional[int] --- Ignored; this argument exists for API
          compatibility with `EventClient.close`.

        * reason: Optional[str] --- Ignored; this argument exists for API
          compatibility with `EventClient.close`.

        * timeout: float --- How long to wait for the client thread to finish
          processing events. Default 0, which means no timeout.
        """
        with self._closing_lock:
            self._closing.set()
        try:
            self.join(timeout=timeout)
        except RuntimeError:
            # "join() raises a RuntimeError if an attempt is made to join the
            # current thread as that would cause a deadlock. It is also an
            # error to join() a thread before it has been started and attempts
            # to do so raises the same exception."
            pass

    def subscribe(self, f: Filter, last_log_id: Optional[int]=None) -> None:
        """Subscribe to another set of events from the server

        Arguments:

        * f: arvados.events.Filter | None --- One filter to subscribe to.

        * last_log_id: Optional[int] --- Ignored; this argument exists for
          API compatibility with `EventClient.subscribe`.
        """
        self.on_event({'status': 200})
        self.filters.append(f)

    def unsubscribe(self, f):
        """Unsubscribe from an event stream

        Arguments:

        * f: arvados.events.Filter | None --- One event filter to stop
        receiving events for.
        """
        del self.filters[self.filters.index(f)]


def _subscribe_websocket(api, filters, on_event, last_log_id=None):
    endpoint = api._rootDesc.get('websocketUrl', None)
    if not endpoint:
        raise errors.FeatureNotEnabledError(
            "Server does not advertise a websocket endpoint")
    uri_with_token = "{}?api_token={}".format(endpoint, api.api_token)
    try:
        client = EventClient(uri_with_token, filters, on_event, last_log_id)
    except Exception:
        _logger.warning("Failed to connect to websockets on %s" % endpoint)
        raise
    else:
        return client

def subscribe(
        api: 'arvados.api_resources.ArvadosAPIClient',
        filters: Optional[Filter],
        on_event: EventCallback,
        poll_fallback: float=15,
        last_log_id: Optional[int]=None,
) -> Union[EventClient, PollClient]:
    """Start a thread to monitor events

    This method tries to construct an `EventClient` to process Arvados
    events via WebSockets. If that fails, or the
    `ARVADOS_DISABLE_WEBSOCKETS` flag is set in user configuration, it falls
    back to constructing a `PollClient` to process the events via API
    polling.

    Arguments:

    * api: arvados.api_resources.ArvadosAPIClient --- The Arvados API
      client used to query logs. It may be used in a separate thread,
      so if it is not an instance of `arvados.api.ThreadSafeAPIClient`
      it should not be reused after this method returns.

    * filters: arvados.events.Filter | None --- One event filter to
      subscribe to after initializing the client. If not specified, the
      client will subscribe to all events.

    * on_event: arvados.events.EventCallback --- When the client receives an
      event, it calls this function with the event object.

    * poll_time: float --- The number of seconds to wait between querying
      logs. If 0, this function will refuse to construct a `PollClient`.
      Default 15.

    * last_log_id: int | None --- If specified, start processing events with
      at least this `id` value.
    """
    if not poll_fallback:
        return _subscribe_websocket(api, filters, on_event, last_log_id)

    try:
        if not config.flag_is_true('ARVADOS_DISABLE_WEBSOCKETS'):
            return _subscribe_websocket(api, filters, on_event, last_log_id)
        else:
            _logger.info("Using polling because ARVADOS_DISABLE_WEBSOCKETS is true")
    except Exception as e:
        _logger.warning("Falling back to polling after websocket error: %s" % e)
    p = PollClient(api, filters, on_event, poll_fallback, last_log_id)
    p.start()
    return p
