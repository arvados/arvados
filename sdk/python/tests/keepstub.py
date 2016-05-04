import BaseHTTPServer
import hashlib
import os
import re
import SocketServer
import time

class Server(SocketServer.ThreadingMixIn, BaseHTTPServer.HTTPServer, object):

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
        for (k, v) in kwargs.iteritems():
            self.delays.get(k) # NameError if unknown key
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


class Handler(BaseHTTPServer.BaseHTTPRequestHandler, object):
    def wfile_bandwidth_write(self, data_to_write):
        if self.server.bandwidth == None and self.server.delays['mid_write'] == 0:
            self.wfile.write(data_to_write)
        else:
            BYTES_PER_WRITE = int(self.server.bandwidth/4.0) or 32768
            outage_happened = False
            num_bytes = len(data_to_write)
            num_sent_bytes = 0
            target_time = time.time()
            while num_sent_bytes < num_bytes:
                if num_sent_bytes > self.server.bandwidth and not outage_happened:
                    self.server._do_delay('mid_write')
                    target_time += self.delays['mid_write']
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
        if self.server.bandwidth == None and self.server.delays['mid_read'] == 0:
            return self.rfile.read(bytes_to_read)
        else:
            BYTES_PER_READ = int(self.server.bandwidth/4.0) or 32768
            data = ''
            outage_happened = False
            bytes_read = 0
            target_time = time.time()
            while bytes_to_read > bytes_read:
                if bytes_read > self.server.bandwidth and not outage_happened:
                    self.server._do_delay('mid_read')
                    target_time += self.delays['mid_read']
                    outage_happened = True
                next_bytes_to_read = min(BYTES_PER_READ,
                    bytes_to_read - bytes_read)
                data += self.rfile.read(next_bytes_to_read)
                bytes_read += next_bytes_to_read
                if self.server.bandwidth is not None:
                    target_time += next_bytes_to_read / self.server.bandwidth
                    self.server._sleep_at_least(target_time - time.time())
        return data

    def handle(self, *args, **kwargs):
        self.server._do_delay('request')
        return super(Handler, self).handle(*args, **kwargs)

    def do_GET(self):
        self.server._do_delay('response')
        r = re.search(r'[0-9a-f]{32}', self.path)
        if not r:
            return self.send_response(422)
        datahash = r.group(0)
        if datahash not in self.server.store:
            return self.send_response(404)
        self.send_response(200)
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
        self.send_header('Content-type', 'application/octet-stream')
        self.send_header('Content-length', str(len(self.server.store[datahash])))
        self.end_headers()
        self.server._do_delay('response_close')

    def do_PUT(self):
        self.server._do_delay('request_body')
        # The comments at https://bugs.python.org/issue1491 implies that Python
        # 2.7 BaseHTTPRequestHandler was patched to support 100 Continue, but
        # reading the actual code that ships in Debian it clearly is not, so we
        # need to send the response on the socket directly.
        self.wfile_bandwidth_write("%s %d %s\r\n\r\n" %
                         (self.protocol_version, 100, "Continue"))
        data = self.rfile_bandwidth_read(int(self.headers.getheader('content-length')))
        datahash = hashlib.md5(data).hexdigest()
        self.server.store[datahash] = data
        self.server._do_delay('response')
        self.send_response(200)
        self.send_header('Content-type', 'text/plain')
        self.end_headers()
        self.server._do_delay('response_body')
        self.wfile_bandwidth_write(datahash + '+' + str(len(data)))
        self.server._do_delay('response_close')

    def log_request(self, *args, **kwargs):
        if os.environ.get('ARVADOS_DEBUG', None):
            super(Handler, self).log_request(*args, **kwargs)

    def finish(self, *args, **kwargs):
        """Ignore exceptions, notably "Broken pipe" when client times out."""
        try:
            return super(Handler, self).finish(*args, **kwargs)
        except:
            pass

    def handle_one_request(self, *args, **kwargs):
        """Ignore exceptions, notably "Broken pipe" when client times out."""
        try:
            return super(Handler, self).handle_one_request(*args, **kwargs)
        except:
            pass
