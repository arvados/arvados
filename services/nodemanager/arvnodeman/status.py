from __future__ import absolute_import, print_function
from future import standard_library

import http.server
import json
import logging
import socketserver
import threading
import time

from . import ARVADOS_TIMEFMT

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
        elif self.path == '/status-full.json':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            self.wfile.write(tracker_full.get_json())
        else:
            self.send_response(404)

    def log_message(self, fmt, *args, **kwargs):
        _logger.info(fmt, *args, **kwargs)


class Tracker(object):
    def __init__(self):
        self._mtx = threading.Lock()
        self._latest = {"error_count": 0}

    def get_json(self):
        with self._mtx:
            return json.dumps(self._latest)

    def keys(self):
        with self._mtx:
            return self._latest.keys()

    def update(self, updates):
        with self._mtx:
            self._latest['timestamp'] = time.strftime(ARVADOS_TIMEFMT, time.gmtime())
            self._latest.update(updates)
            st = "OK"
            for k,v in self._latest.iteritems():
                if k.startswith("status_") and v != "OK":
                    st = v
                    break
            self._latest["status"] = st

    def report_ok(self, name):
        self.update({
            "status_"+name: "OK",
            "exception_"+name: None
        })

    def report_error(self, name, type=None, value=None, tb=None):
        if type is not None:
            msg = "\n".join(traceback.format_exception(type, value, tb))
        else:
            msg = traceback.format_exc()
        with self._mtx:
            self._latest["error_count"] += 1
        self.update({
            "status_"+name: "WARNING",
            "exception_"+name: msg
        })


tracker = Tracker()
tracker_full = Tracker()
