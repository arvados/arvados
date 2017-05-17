from __future__ import division
from future import standard_library
standard_library.install_aliases()
from builtins import str
import http.server
import hashlib
import os
import re
import socket
import socketserver
import sys
import threading
import time

from . import arvados_testutil as tutil

_debug = os.environ.get('ARVADOS_DEBUG', None)


class StubKeepServers(tutil.ApiClientMock):

    def setUp(self):
        super(StubKeepServers, self).setUp()
        sock = socket.socket()
        sock.bind(('0.0.0.0', 0))
        self.port = sock.getsockname()[1]
        sock.close()
        self.server = Server(('0.0.0.0', self.port), Handler)
        self.thread = threading.Thread(target=self.server.serve_forever)
        self.thread.daemon = True # Exit thread if main proc exits
        self.thread.start()
        self.api_client = self.mock_keep_services(
            count=1,
            service_host='localhost',
            service_port=self.port,
        )

    def tearDown(self):
        self.server.shutdown()
        super(StubKeepServers, self).tearDown()


class Server(socketserver.ThreadingMixIn, http.server.HTTPServer, object):

    allow_reuse_address = 1

    def __init__(self, *args, **kwargs):
        self.store = {}
        self.delays = {
            # before reading request headers
            'request': 0,
            # before reading request body
            'request_body': 0,
            # before setting response status and headers
            'response': 0,
            # before sending response body
            'response_body': 0,
            # before returning from handler (thus setting response EOF)
            'response_close': 0,
            # after writing over 1s worth of data at self.bandwidth
            'mid_write': 0,
            # after reading over 1s worth of data at self.bandwidth
            'mid_read': 0,
        }
        self.bandwidth = None
        super(Server, self).__init__(*args, **kwargs)

    def setdelays(self, **kwargs):
        """In future requests, induce delays at the given checkpoints."""
        for (k, v) in kwargs.items():
            self.delays.get(k)  # NameError if unknown key
            self.delays[k] = v

    def setbandwidth(self, bandwidth):
        """For future requests, set the maximum bandwidth (number of bytes per
        second) to operate at. If setbandwidth is never called, function at
        maximum bandwidth possible"""
        self.bandwidth = float(bandwidth)

    def _sleep_at_least(self, seconds):
        """Sleep for given time, even if signals are received."""
        wake = time.time() + seconds
        todo = seconds
        while todo > 0:
            time.sleep(todo)
            todo = wake - time.time()

    def _do_delay(self, k):
        self._sleep_at_least(self.delays[k])


class Handler(http.server.BaseHTTPRequestHandler, object):

    protocol_version = 'HTTP/1.1'

    def wfile_bandwidth_write(self, data_to_write):
        if self.server.bandwidth is None and self.server.delays['mid_write'] == 0:
            self.wfile.write(data_to_write)
        else:
            BYTES_PER_WRITE = int(self.server.bandwidth/4) or 32768
            outage_happened = False
            num_bytes = len(data_to_write)
            num_sent_bytes = 0
            target_time = time.time()
            while num_sent_bytes < num_bytes:
                if num_sent_bytes > self.server.bandwidth and not outage_happened:
                    self.server._do_delay('mid_write')
                    target_time += self.server.delays['mid_write']
                    outage_happened = True
                num_write_bytes = min(BYTES_PER_WRITE,
                    num_bytes - num_sent_bytes)
                self.wfile.write(data_to_write[
                    num_sent_bytes:num_sent_bytes+num_write_bytes])
                num_sent_bytes += num_write_bytes
                if self.server.bandwidth is not None:
                    target_time += num_write_bytes / self.server.bandwidth
                    self.server._sleep_at_least(target_time - time.time())
        return None

    def rfile_bandwidth_read(self, bytes_to_read):
        if self.server.bandwidth is None and self.server.delays['mid_read'] == 0:
            return self.rfile.read(bytes_to_read)
        else:
            BYTES_PER_READ = int(self.server.bandwidth/4) or 32768
            data = b''
            outage_happened = False
            bytes_read = 0
            target_time = time.time()
            while bytes_to_read > bytes_read:
                if bytes_read > self.server.bandwidth and not outage_happened:
                    self.server._do_delay('mid_read')
                    target_time += self.server.delays['mid_read']
                    outage_happened = True
                next_bytes_to_read = min(BYTES_PER_READ,
                    bytes_to_read - bytes_read)
                data += self.rfile.read(next_bytes_to_read)
                bytes_read += next_bytes_to_read
                if self.server.bandwidth is not None:
                    target_time += next_bytes_to_read / self.server.bandwidth
                    self.server._sleep_at_least(target_time - time.time())
        return data

    def finish(self, *args, **kwargs):
        try:
            return super(Handler, self).finish(*args, **kwargs)
        except Exception as err:
            if _debug:
                raise

    def handle(self, *args, **kwargs):
        try:
            return super(Handler, self).handle(*args, **kwargs)
        except:
            if _debug:
                raise

    def handle_one_request(self, *args, **kwargs):
        self._sent_continue = False
        self.server._do_delay('request')
        return super(Handler, self).handle_one_request(*args, **kwargs)

    def handle_expect_100(self):
        self.server._do_delay('request_body')
        self._sent_continue = True
        return super(Handler, self).handle_expect_100()

    def do_GET(self):
        self.server._do_delay('response')
        r = re.search(r'[0-9a-f]{32}', self.path)
        if not r:
            return self.send_response(422)
        datahash = r.group(0)
        if datahash not in self.server.store:
            return self.send_response(404)
        self.send_response(200)
        self.send_header('Connection', 'close')
        self.send_header('Content-type', 'application/octet-stream')
        self.end_headers()
        self.server._do_delay('response_body')
        self.wfile_bandwidth_write(self.server.store[datahash])
        self.server._do_delay('response_close')

    def do_HEAD(self):
        self.server._do_delay('response')
        r = re.search(r'[0-9a-f]{32}', self.path)
        if not r:
            return self.send_response(422)
        datahash = r.group(0)
        if datahash not in self.server.store:
            return self.send_response(404)
        self.send_response(200)
        self.send_header('Connection', 'close')
        self.send_header('Content-type', 'application/octet-stream')
        self.send_header('Content-length', str(len(self.server.store[datahash])))
        self.end_headers()
        self.server._do_delay('response_close')
        self.close_connection = True

    def do_PUT(self):
        if not self._sent_continue and self.headers.get('expect') == '100-continue':
            # The comments at https://bugs.python.org/issue1491
            # implies that Python 2.7 BaseHTTPRequestHandler was
            # patched to support 100 Continue, but reading the actual
            # code that ships in Debian it clearly is not, so we need
            # to send the response on the socket directly.
            self.server._do_delay('request_body')
            self.wfile.write("{} {} {}\r\n\r\n".format(
                self.protocol_version, 100, "Continue").encode())
        data = self.rfile_bandwidth_read(
            int(self.headers.get('content-length')))
        datahash = hashlib.md5(data).hexdigest()
        self.server.store[datahash] = data
        resp = '{}+{}\n'.format(datahash, len(data)).encode()
        self.server._do_delay('response')
        self.send_response(200)
        self.send_header('Connection', 'close')
        self.send_header('Content-type', 'text/plain')
        self.send_header('Content-length', len(resp))
        self.end_headers()
        self.server._do_delay('response_body')
        self.wfile_bandwidth_write(resp)
        self.server._do_delay('response_close')
        self.close_connection = True

    def log_request(self, *args, **kwargs):
        if _debug:
            super(Handler, self).log_request(*args, **kwargs)
