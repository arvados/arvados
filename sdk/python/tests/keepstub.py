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
            'mid_write': 0,
            'mid_read': 0,
        }
        #If self.bandwidth = None, function at maximum bandwidth
        #Otherwise, self.bandwidth is the maximum number of bytes per second to
        #   operate at.
        self.bandwidth = None
        super(Server, self).__init__(*args, **kwargs)

    def setdelays(self, **kwargs):
        """In future requests, induce delays at the given checkpoints."""
        for (k, v) in kwargs.iteritems():
            self.delays.get(k) # NameError if unknown key
            self.delays[k] = v

    def setbandwidth(self, bandwidth):
        """For future requests, impose bandwidth limits."""
        self.bandwidth = bandwidth*1024.0

    def _sleep_at_least(self, seconds):
        """Sleep for given time, even if signals are received."""
        wake = time.time() + seconds
        todo = seconds
        while todo > 0:
            time.sleep(todo)
            todo = wake - time.time()

    def _do_delay(self, k):
        #if self.delays[k] > 0:
        #    print "Delaying %f seconds for %s delay" % (self.delays[k], k)
        self._sleep_at_least(self.delays[k])


class Handler(BaseHTTPServer.BaseHTTPRequestHandler, object):
    def wfile_bandwidth_write(self, data_to_write):
        BYTES_PER_WRITE = 10
        if self.server.bandwidth == None and self.server.delays['mid_write'] == 0:
            self.wfile.write(data_to_write)
        else:
            outage_happened = False
            num_bytes = len(data_to_write)
            num_sent_bytes = 0
            while num_sent_bytes < num_bytes:
                if num_sent_bytes > self.server.bandwidth and not outage_happened:
                    print "Delaying write %fs" % (self.server.delays['mid_write'])
                    self.server._do_delay('mid_write')
                    outage_happened = True
                num_write_bytes = min(BYTES_PER_WRITE, num_bytes - num_sent_bytes)
                if self.server.bandwidth == None:
                    wait = 0
                else:
                    wait = num_write_bytes / self.server.bandwidth
                #print "Bandwidth: %f KiB. Want to write total of %f bytes. Writing %f bytes now. Waiting %f seconds" % (self.server.bandwidth/1024.0, num_bytes, num_write_bytes, wait)
                self.server._sleep_at_least(wait)
                self.wfile.write(data_to_write[num_sent_bytes:num_sent_bytes+num_write_bytes])
                num_sent_bytes += num_write_bytes
        return None

    def rfile_bandwidth_read(self, bytes_to_read):
        BYTES_PER_READ = 10
        if self.server.bandwidth == None and self.server.delays['mid_read'] == 0:
            return self.rfile.read(bytes_to_read)
        else:
            data = ''
            outage_happened = False
            bytes_read = 0
            while bytes_to_read > bytes_read:
                if bytes_read > self.server.bandwidth and not outage_happened:
                    print "Delaying read %fs" % (self.server.delays['mid_read'])
                    self.server._do_delay('mid_read')
                    outage_happened = True
                next_bytes_to_read = min(BYTES_PER_READ, bytes_to_read - bytes_read)
                t0 = time.time()
                data += self.rfile.read(next_bytes_to_read)
                time_spent_getting_data = time.time() - t0
                if self.server.bandwidth == None:
                    wait = 0
                else:
                    wait = next_bytes_to_read / self.server.bandwidth - time_spent_getting_data
                #print "Bandwidth: %f KiB. Wanted total of %f bytes. Now reading %f bytes. Have read %f bytes. Waiting %f seconds" % (self.server.bandwidth/1024.0, bytes_to_read, next_bytes_to_read, len(data), wait)
                if wait > 0:
                    self.server._sleep_at_least(wait)
                bytes_read += next_bytes_to_read
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
        #print "Writing to client"
        self.wfile_bandwidth_write(self.server.store[datahash])
        self.server._do_delay('response_close')

    def do_PUT(self):
        self.server._do_delay('request_body')

        # The comments at https://bugs.python.org/issue1491 implies that Python
        # 2.7 BaseHTTPRequestHandler was patched to support 100 Continue, but
        # reading the actual code that ships in Debian it clearly is not, so we
        # need to send the response on the socket directly.
        #print "Writing continue"
        self.wfile_bandwidth_write("%s %d %s\r\n\r\n" %
                         (self.protocol_version, 100, "Continue"))
        #print "Reading input from client"
        data = self.rfile_bandwidth_read(int(self.headers.getheader('content-length')))
        datahash = hashlib.md5(data).hexdigest()
        self.server.store[datahash] = data
        self.server._do_delay('response')
        self.send_response(200)
        self.send_header('Content-type', 'text/plain')
        self.end_headers()
        self.server._do_delay('response_body')
        #print "Write output hash"
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
