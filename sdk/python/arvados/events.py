from ws4py.client.threadedclient import WebSocketClient
import thread
import json
import os
import time
import ssl
import re
import config
import logging

_logger = logging.getLogger('arvados.events')

class EventClient(WebSocketClient):
    def __init__(self, url, filters, on_event):
        ssl_options = None
        if re.match(r'(?i)^(true|1|yes)$',
                    config.get('ARVADOS_API_HOST_INSECURE', 'no')):
            ssl_options={'cert_reqs': ssl.CERT_NONE}
        else:
            ssl_options={'cert_reqs': ssl.CERT_REQUIRED}

        super(EventClient, self).__init__(url, ssl_options)
        self.filters = filters
        self.on_event = on_event

    def opened(self):
        self.send(json.dumps({"method": "subscribe", "filters": self.filters}))

    def received_message(self, m):
        self.on_event(json.loads(str(m)))

    def close_connection(self):
        try:
            self.sock.shutdown(socket.SHUT_RDWR)
            self.sock.close()
        except:
            pass

def subscribe(api, filters, on_event):
    ws = None
    try:
        url = "{}?api_token={}".format(api._rootDesc['websocketUrl'], config.get('ARVADOS_API_TOKEN'))
        ws = EventClient(url, filters, on_event)
        ws.connect()
        return ws
    except Exception:
        if (ws):
          ws.close_connection()
        raise
