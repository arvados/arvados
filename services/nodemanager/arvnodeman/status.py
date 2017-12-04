# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import, print_function
from future import standard_library

import http.server
import json
import logging
import socketserver
import threading

from ._version import __version__

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
        elif self.path == '/_health/ping':
            code, msg = self.check_auth()

            if code != 200:
              self.send_response(code)
              self.wfile.write(msg)
            else:
              self.send_response(200)
              self.send_header('Content-type', 'application/json')
              self.end_headers()
              self.wfile.write(json.dumps({"health":"OK"}))
        else:
            self.send_response(404)

    def log_message(self, fmt, *args, **kwargs):
        _logger.info(fmt, *args, **kwargs)

    def check_auth(self):
        mgmt_token = self.server._config.get('Manage', 'ManagementToken')
        auth_header = self.headers.get('Authorization', None)

        if mgmt_token == '':
          return 404, "disabled"
        elif auth_header == None:
          return 401, "authorization required"
        elif auth_header != 'Bearer '+mgmt_token:
          return 403, "authorization error"
        return 200, ""

class Tracker(object):
    def __init__(self):
        self._mtx = threading.Lock()
        self._latest = {}
        self._version = {'Version' : __version__}

    def get_json(self):
        with self._mtx:
            return json.dumps(dict(self._latest, **self._version))

    def keys(self):
        with self._mtx:
            return self._latest.keys()

    def update(self, updates):
        with self._mtx:
            self._latest.update(updates)


tracker = Tracker()
