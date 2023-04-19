# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import socket
import pycurl
import math

class PyCurlHelper:
    # Default Keep server connection timeout:  2 seconds
    # Default Keep server read timeout:       256 seconds
    # Default Keep server bandwidth minimum:  32768 bytes per second
    # Default Keep proxy connection timeout:  20 seconds
    # Default Keep proxy read timeout:        256 seconds
    # Default Keep proxy bandwidth minimum:   32768 bytes per second
    DEFAULT_TIMEOUT = (2, 256, 32768)
    DEFAULT_PROXY_TIMEOUT = (20, 256, 32768)

    def __init__(self, title_case_headers=False):
        self._socket = None
        self.title_case_headers = title_case_headers

    def _socket_open(self, *args, **kwargs):
        if len(args) + len(kwargs) == 2:
            return self._socket_open_pycurl_7_21_5(*args, **kwargs)
        else:
            return self._socket_open_pycurl_7_19_3(*args, **kwargs)

    def _socket_open_pycurl_7_19_3(self, family, socktype, protocol, address=None):
        return self._socket_open_pycurl_7_21_5(
            purpose=None,
            address=collections.namedtuple(
                'Address', ['family', 'socktype', 'protocol', 'addr'],
            )(family, socktype, protocol, address))

    def _socket_open_pycurl_7_21_5(self, purpose, address):
        """Because pycurl doesn't have CURLOPT_TCP_KEEPALIVE"""
        s = socket.socket(address.family, address.socktype, address.protocol)
        s.setsockopt(socket.SOL_SOCKET, socket.SO_KEEPALIVE, 1)
        # Will throw invalid protocol error on mac. This test prevents that.
        if hasattr(socket, 'TCP_KEEPIDLE'):
            s.setsockopt(socket.IPPROTO_TCP, socket.TCP_KEEPIDLE, 75)
        s.setsockopt(socket.IPPROTO_TCP, socket.TCP_KEEPINTVL, 75)
        self._socket = s
        return s

    def _setcurltimeouts(self, curl, timeouts, ignore_bandwidth=False):
        if not timeouts:
            return
        elif isinstance(timeouts, tuple):
            if len(timeouts) == 2:
                conn_t, xfer_t = timeouts
                bandwidth_bps = self.DEFAULT_TIMEOUT[2]
            else:
                conn_t, xfer_t, bandwidth_bps = timeouts
        else:
            conn_t, xfer_t = (timeouts, timeouts)
            bandwidth_bps = self.DEFAULT_TIMEOUT[2]
        curl.setopt(pycurl.CONNECTTIMEOUT_MS, int(conn_t*1000))
        if not ignore_bandwidth:
            curl.setopt(pycurl.LOW_SPEED_TIME, int(math.ceil(xfer_t)))
            curl.setopt(pycurl.LOW_SPEED_LIMIT, int(math.ceil(bandwidth_bps)))

    def _headerfunction(self, header_line):
        if isinstance(header_line, bytes):
            header_line = header_line.decode('iso-8859-1')
        if ':' in header_line:
            name, value = header_line.split(':', 1)
            if self.title_case_headers:
                name = name.strip().title()
            else:
                name = name.strip().lower()
            value = value.strip()
        elif self._headers:
            name = self._lastheadername
            value = self._headers[name] + ' ' + header_line.strip()
        elif header_line.startswith('HTTP/'):
            name = 'x-status-line'
            value = header_line
        else:
            _logger.error("Unexpected header line: %s", header_line)
            return
        self._lastheadername = name
        self._headers[name] = value
        # Returning None implies all bytes were written
