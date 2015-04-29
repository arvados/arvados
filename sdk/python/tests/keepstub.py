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
        }
        super(Server, self).__init__(*args, **kwargs)

    def setdelays(self, **kwargs):
        """In future requests, induce delays at the given checkpoints."""
        for (k, v) in kwargs.iteritems():
            self.delays.get(k) # NameError if unknown key
            self.delays[k] = v

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
        self.wfile.write(self.server.store[datahash])
        self.server._do_delay('response_close')

    def do_PUT(self):
        self.server._do_delay('request_body')
        data = self.rfile.read(int(self.headers.getheader('content-length')))
        datahash = hashlib.md5(data).hexdigest()
        self.server.store[datahash] = data
        self.server._do_delay('response')
        self.send_response(200)
        self.send_header('Content-type', 'text/plain')
        self.end_headers()
        self.server._do_delay('response_body')
        self.wfile.write(datahash + '+' + str(len(data)))
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
