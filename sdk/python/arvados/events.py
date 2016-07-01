import arvados
import config
import errors
from retry import RetryLoop

import logging
import json
import thread
import threading
import time
import os
import re
import ssl
from ws4py.client.threadedclient import WebSocketClient

_logger = logging.getLogger('arvados.events')


class _EventClient(WebSocketClient):
    def __init__(self, url, filters, on_event, last_log_id, on_closed):
        ssl_options = {'ca_certs': arvados.util.ca_certs_path()}
        if config.flag_is_true('ARVADOS_API_HOST_INSECURE'):
            ssl_options['cert_reqs'] = ssl.CERT_NONE
        else:
            ssl_options['cert_reqs'] = ssl.CERT_REQUIRED

        # Warning: If the host part of url resolves to both IPv6 and
        # IPv4 addresses (common with "localhost"), only one of them
        # will be attempted -- and it might not be the right one. See
        # ws4py's WebSocketBaseClient.__init__.
        super(_EventClient, self).__init__(url, ssl_options=ssl_options)

        self.filters = filters
        self.on_event = on_event
        self.last_log_id = last_log_id
        self._closing_lock = threading.RLock()
        self._closing = False
        self._closed = threading.Event()
        self.on_closed = on_closed

    def opened(self):
        for f in self.filters:
            self.subscribe(f, self.last_log_id)

    def closed(self, code, reason=None):
        self._closed.set()
        self.on_closed()

    def received_message(self, m):
        with self._closing_lock:
            if not self._closing:
                self.on_event(json.loads(str(m)))

    def close(self, code=1000, reason='', timeout=0):
        """Close event client and optionally wait for it to finish.

        :timeout: is the number of seconds to wait for ws4py to
        indicate that the connection has closed.
        """
        super(_EventClient, self).close(code, reason)
        with self._closing_lock:
            # make sure we don't process any more messages.
            self._closing = True
        # wait for ws4py to tell us the connection is closed.
        self._closed.wait(timeout=timeout)

    def subscribe(self, f, last_log_id=None):
        m = {"method": "subscribe", "filters": f}
        if last_log_id is not None:
            m["last_log_id"] = last_log_id
        self.send(json.dumps(m))

    def unsubscribe(self, f):
        self.send(json.dumps({"method": "unsubscribe", "filters": f}))


class EventClient(object):
    def __init__(self, url, filters, on_event_cb, last_log_id):
        self.url = url
        if filters:
            self.filters = [filters]
        else:
            self.filters = [[]]
        self.on_event_cb = on_event_cb
        self.last_log_id = last_log_id
        self.is_closed = threading.Event()
        self._setup_event_client()

    def _setup_event_client(self):
        self.ec = _EventClient(self.url, self.filters, self.on_event,
                               self.last_log_id, self.on_closed)
        self.ec.daemon = True
        try:
            self.ec.connect()
        except Exception:
            self.ec.close_connection()
            raise

    def subscribe(self, f, last_log_id=None):
        self.filters.append(f)
        self.ec.subscribe(f, last_log_id)

    def unsubscribe(self, f):
        del self.filters[self.filters.index(f)]
        self.ec.unsubscribe(f)

    def close(self, code=1000, reason='', timeout=0):
        self.is_closed.set()
        self.ec.close(code, reason, timeout)

    def on_event(self, m):
        if m.get('id') != None:
            self.last_log_id = m.get('id')
        try:
            self.on_event_cb(m)
        except Exception as e:
            _logger.exception("Unexpected exception from event callback.")
            thread.interrupt_main()

    def on_closed(self):
        if not self.is_closed.is_set():
            _logger.warn("Unexpected close. Reconnecting.")
            for tries_left in RetryLoop(num_retries=25, backoff_start=.1, max_wait=15):
                try:
                    self._setup_event_client()
                    _logger.warn("Reconnect successful.")
                    break
                except Exception as e:
                    _logger.warn("Error '%s' during websocket reconnect.", e)
            if tries_left == 0:
                _logger.exception("EventClient thread could not contact websocket server.")
                self.is_closed.set()
                thread.interrupt_main()
                return

    def run_forever(self):
        # Have to poll here to let KeyboardInterrupt get raised.
        while not self.is_closed.wait(1):
            pass


class PollClient(threading.Thread):
    def __init__(self, api, filters, on_event, poll_time, last_log_id):
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

    def run(self):
        self.id = 0
        if self.last_log_id != None:
            self.id = self.last_log_id
        else:
            for f in self.filters:
                for tries_left in RetryLoop(num_retries=25, backoff_start=.1, max_wait=self.poll_time):
                    try:
                        items = self.api.logs().list(limit=1, order="id desc", filters=f).execute()['items']
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
                    thread.interrupt_main()
                    return
                if items:
                    if items[0]['id'] > self.id:
                        self.id = items[0]['id']

        self.on_event({'status': 200})

        while not self._closing.is_set():
            max_id = self.id
            moreitems = False
            for f in self.filters:
                for tries_left in RetryLoop(num_retries=25, backoff_start=.1, max_wait=self.poll_time):
                    try:
                        items = self.api.logs().list(order="id asc", filters=f+[["id", ">", str(self.id)]]).execute()
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
                    thread.interrupt_main()
                    return
                for i in items["items"]:
                    if i['id'] > max_id:
                        max_id = i['id']
                    with self._closing_lock:
                        if self._closing.is_set():
                            return
                        try:
                            self.on_event(i)
                        except Exception as e:
                            _logger.exception("Unexpected exception from event callback.")
                            thread.interrupt_main()
                if items["items_available"] > len(items["items"]):
                    moreitems = True
            self.id = max_id
            if not moreitems:
                self._closing.wait(self.poll_time)

    def run_forever(self):
        # Have to poll here, otherwise KeyboardInterrupt will never get processed.
        while not self._closing.is_set():
            self._closing.wait(1)

    def close(self, code=None, reason=None, timeout=0):
        """Close poll client and optionally wait for it to finish.

        If an :on_event: handler is running in a different thread,
        first wait (indefinitely) for it to return.

        After closing, wait up to :timeout: seconds for the thread to
        finish the poll request in progress (if any).

        :code: and :reason: are ignored. They are present for
        interface compatibility with EventClient.
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

    def subscribe(self, f):
        self.on_event({'status': 200})
        self.filters.append(f)

    def unsubscribe(self, f):
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
        _logger.warn("Failed to connect to websockets on %s" % endpoint)
        raise
    else:
        return client


def subscribe(api, filters, on_event, poll_fallback=15, last_log_id=None):
    """
    :api:
      a client object retrieved from arvados.api(). The caller should not use this client object for anything else after calling subscribe().
    :filters:
      Initial subscription filters.
    :on_event:
      The callback when a message is received.
    :poll_fallback:
      If websockets are not available, fall back to polling every N seconds.  If poll_fallback=False, this will return None if websockets are not available.
    :last_log_id:
      Log rows that are newer than the log id
    """

    if not poll_fallback:
        return _subscribe_websocket(api, filters, on_event, last_log_id)

    try:
        if not config.flag_is_true('ARVADOS_DISABLE_WEBSOCKETS'):
            return _subscribe_websocket(api, filters, on_event, last_log_id)
        else:
            _logger.info("Using polling because ARVADOS_DISABLE_WEBSOCKETS is true")
    except Exception as e:
        _logger.warn("Falling back to polling after websocket error: %s" % e)
    p = PollClient(api, filters, on_event, poll_fallback, last_log_id)
    p.start()
    return p
