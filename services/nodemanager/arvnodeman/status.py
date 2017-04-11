from __future__ import absolute_import, print_function
from future import standard_library

import http.server
import json
import logging
import socketserver
import threading

_logger = logging.getLogger('status.Handler')


class Server(socketserver.ThreadingMixIn, http.server.HTTPServer, object):
    def __init__(self, config):
        port = config.getint('Manage', 'port')
        self.enabled = port >= 0
        if not self.enabled:
            _logger.warning("Management server disabled. "+
                            "Use [Manage] config section to enable.")
            return
        self._config = config
        self._tracker = tracker
        super(Server, self).__init__(
            (config.get('Manage', 'address'), port), Handler)
        self._thread = threading.Thread(target=self.serve_forever)
        self._thread.daemon = True

    def start(self):
        if self.enabled:
            self._thread.start()


class Handler(http.server.BaseHTTPRequestHandler, object):
    def do_GET(self):
        if self.path == '/status.json':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            self.wfile.write(tracker.get_json())
        else:
            self.send_response(404)

    def log_message(self, fmt, *args, **kwargs):
        _logger.info(fmt, *args, **kwargs)


class Tracker(object):
    def __init__(self):
        self._mtx = threading.Lock()
        self._latest = {}

    def get_json(self):
        with self._mtx:
            return json.dumps(self._latest)

    def keys(self):
        with self._mtx:
            return self._latest.keys()

    def update(self, updates):
        with self._mtx:
            self._latest.update(updates)


tracker = Tracker()
