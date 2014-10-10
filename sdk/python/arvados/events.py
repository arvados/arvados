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

class PollClient(threading.Thread):
    def __init__(self, api, filters, on_event):
        self.api = api
        self.filters = filters
        self.on_event = on_event
        items = self.api.logs().list(limit=1, order=json.dumps(["id desc"]), filters=json.dumps(filters)).execute()['items']
        if len(items) > 0:
            self.id = items[0]["id"]
        else:
            self.id = 0
        self.loop = True

    def run_forever(self):
        while self.loop:
            time.sleep(15)
            items = self.api.logs().list(limit=1, order=json.dumps(["id asc"]), filters=json.dumps(self.filters+[["id", ">", str(self.id)]])).execute()['items']
            for i in items:
                self.id = i['id']
                self.on_event(i)

    def close_connection(self):
        self.loop = False

def subscribe(api, filters, on_event):
    ws = None
    try:
        if 'websocketUrl' in api._rootDesc:
            url = "{}?api_token={}".format(api._rootDesc['websocketUrl'], config.get('ARVADOS_API_TOKEN'))
            ws = EventClient(url, filters, on_event)
            ws.connect()
        else:
            ws = PollClient(api, filters, on_event)
        return ws
    except Exception:
        if (ws):
          ws.close_connection()
        try:
            return PollClient(api, filters, on_event)
        except:
            raise
