import arvados
import config
import errors

import logging
import json
import threading
import time
import os
import re
import ssl
from ws4py.client.threadedclient import WebSocketClient

_logger = logging.getLogger('arvados.events')

class EventClient(WebSocketClient):
    def __init__(self, url, filters, on_event):
        # Prefer system's CA certificates (if available)
        ssl_options = {}
        certs_path = '/etc/ssl/certs/ca-certificates.crt'
        if os.path.exists(certs_path):
            ssl_options['ca_certs'] = certs_path
        if config.flag_is_true('ARVADOS_API_HOST_INSECURE'):
            ssl_options['cert_reqs'] = ssl.CERT_NONE
        else:
            ssl_options['cert_reqs'] = ssl.CERT_REQUIRED

        # Warning: If the host part of url resolves to both IPv6 and
        # IPv4 addresses (common with "localhost"), only one of them
        # will be attempted -- and it might not be the right one. See
        # ws4py's WebSocketBaseClient.__init__.
        super(EventClient, self).__init__(url, ssl_options=ssl_options)
        self.filters = filters
        self.on_event = on_event

    def opened(self):
        self.subscribe(self.filters)

    def received_message(self, m):
        self.on_event(json.loads(str(m)))

    def close_connection(self):
        try:
            self.sock.shutdown(socket.SHUT_RDWR)
            self.sock.close()
        except:
            pass

    def subscribe(self, filters, last_log_id=None):
        m = {"method": "subscribe", "filters": filters}
        if last_log_id is not None:
            m["last_log_id"] = last_log_id
        self.send(json.dumps(m))

    def unsubscribe(self, filters):
        self.send(json.dumps({"method": "unsubscribe", "filters": filters}))

class PollClient(threading.Thread):
    def __init__(self, api, filters, on_event, poll_time):
        super(PollClient, self).__init__()
        self.api = api
        if filters:
            self.filters = [filters]
        else:
            self.filters = [[]]
        self.on_event = on_event
        self.poll_time = poll_time
        self.stop = threading.Event()

    def run(self):
        self.id = 0
        for f in self.filters:
            items = self.api.logs().list(limit=1, order="id desc", filters=f).execute()['items']
            if items:
                if items[0]['id'] > self.id:
                    self.id = items[0]['id']

        self.on_event({'status': 200})

        while not self.stop.isSet():
            max_id = self.id
            for f in self.filters:
                items = self.api.logs().list(order="id asc", filters=f+[["id", ">", str(self.id)]]).execute()['items']
                for i in items:
                    if i['id'] > max_id:
                        max_id = i['id']
                    self.on_event(i)
            self.id = max_id
            self.stop.wait(self.poll_time)

    def run_forever(self):
        # Have to poll here, otherwise KeyboardInterrupt will never get processed.
        while not self.stop.is_set():
            self.stop.wait(1)

    def close(self):
        self.stop.set()
        try:
            self.join()
        except RuntimeError:
            # "join() raises a RuntimeError if an attempt is made to join the
            # current thread as that would cause a deadlock. It is also an
            # error to join() a thread before it has been started and attempts
            # to do so raises the same exception."
            pass

    def subscribe(self, filters):
        self.on_event({'status': 200})
        self.filters.append(filters)

    def unsubscribe(self, filters):
        del self.filters[self.filters.index(filters)]


def _subscribe_websocket(api, filters, on_event):
    endpoint = api._rootDesc.get('websocketUrl', None)
    if not endpoint:
        raise errors.FeatureNotEnabledError(
            "Server does not advertise a websocket endpoint")
    uri_with_token = "{}?api_token={}".format(endpoint, api.api_token)
    client = EventClient(uri_with_token, filters, on_event)
    ok = False
    try:
        client.connect()
        ok = True
        return client
    finally:
        if not ok:
            client.close_connection()

def subscribe(api, filters, on_event, poll_fallback=15):
    '''
    api: a client object retrieved from arvados.api(). The caller should not use this client object for anything else after calling subscribe().
    filters: Initial subscription filters.
    on_event: The callback when a message is received.
    poll_fallback: If websockets are not available, fall back to polling every N seconds.  If poll_fallback=False, this will return None if websockets are not available.
    '''
    if not poll_fallback:
        return _subscribe_websocket(api, filters, on_event)

    try:
        return _subscribe_websocket(api, filters, on_event)
    except Exception as e:
        _logger.warn("Falling back to polling after websocket error: %s" % e)
    p = PollClient(api, filters, on_event, poll_fallback)
    p.start()
    return p
