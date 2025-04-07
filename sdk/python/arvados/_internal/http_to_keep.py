# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import datetime
import logging
import re
import time
import urllib.parse

import pycurl

import arvados
import arvados.collection
import arvados._internal

from .downloaderbase import DownloaderBase
from .pycurl import PyCurlHelper
from .to_keep_util import (Response, url_to_keep, generic_check_cached_url)

logger = logging.getLogger('arvados.http_import')

class _Downloader(DownloaderBase, PyCurlHelper):
    # Wait up to 60 seconds for connection
    # How long it can be in "low bandwidth" state before it gives up
    # Low bandwidth threshold is 32 KiB/s
    DOWNLOADER_TIMEOUT = (60, 300, 32768)

    def __init__(self, apiclient):
        DownloaderBase.__init__(self)
        PyCurlHelper.__init__(self, title_case_headers=True)
        self.curl = pycurl.Curl()
        self.curl.setopt(pycurl.NOSIGNAL, 1)
        self.curl.setopt(pycurl.OPENSOCKETFUNCTION,
                    lambda *args, **kwargs: self._socket_open(*args, **kwargs))
        self.apiclient = apiclient

    def head(self, url):
        get_headers = {'Accept': 'application/octet-stream'}
        self._headers = {}

        self.curl.setopt(pycurl.URL, url.encode('utf-8'))
        self.curl.setopt(pycurl.HTTPHEADER, [
            '{}: {}'.format(k,v) for k,v in get_headers.items()])

        self.curl.setopt(pycurl.HEADERFUNCTION, self._headerfunction)
        self.curl.setopt(pycurl.CAINFO, arvados.util.ca_certs_path())
        self.curl.setopt(pycurl.NOBODY, True)
        self.curl.setopt(pycurl.FOLLOWLOCATION, True)

        self._setcurltimeouts(self.curl, self.DOWNLOADER_TIMEOUT, True)

        try:
            self.curl.perform()
        except Exception as e:
            raise arvados.errors.HttpError(0, str(e))
        finally:
            if self._socket:
                self._socket.close()
                self._socket = None

        return Response(self.curl.getinfo(pycurl.RESPONSE_CODE), self._headers)

    def download(self, url, headers):
        self.count = 0
        self.start = time.time()
        self.checkpoint = self.start
        self._headers = {}
        self._first_chunk = True
        self.collection = None
        self.parsedurl = urllib.parse.urlparse(url)

        get_headers = {'Accept': 'application/octet-stream'}
        get_headers.update(headers)

        self.curl.setopt(pycurl.URL, url.encode('utf-8'))
        self.curl.setopt(pycurl.HTTPHEADER, [
            '{}: {}'.format(k,v) for k,v in get_headers.items()])

        self.curl.setopt(pycurl.WRITEFUNCTION, self.body_write)
        self.curl.setopt(pycurl.HEADERFUNCTION, self._headerfunction)

        self.curl.setopt(pycurl.CAINFO, arvados.util.ca_certs_path())
        self.curl.setopt(pycurl.HTTPGET, True)
        self.curl.setopt(pycurl.FOLLOWLOCATION, True)

        self._setcurltimeouts(self.curl, self.DOWNLOADER_TIMEOUT, False)

        try:
            self.curl.perform()
        except Exception as e:
            raise arvados.errors.HttpError(0, str(e))
        finally:
            if self._socket:
                self._socket.close()
                self._socket = None

        return Response(self.curl.getinfo(pycurl.RESPONSE_CODE), self._headers)

    def headers_received(self):
        self.collection = arvados.collection.Collection(api_client=self.apiclient)

        if "Content-Length" in self._headers:
            self.contentlength = int(self._headers["Content-Length"])
            logger.info("File size is %s bytes", self.contentlength)
        else:
            self.contentlength = None

        if self._headers.get("Content-Disposition"):
            grp = re.search(r'filename=("((\"|[^"])+)"|([^][()<>@,;:\"/?={} ]+))',
                            self._headers["Content-Disposition"])
            if grp.group(2):
                self.name = grp.group(2)
            else:
                self.name = grp.group(4)
        else:
            self.name = self.parsedurl.path.split("/")[-1]

        # Can't call curl.getinfo(pycurl.RESPONSE_CODE) until
        # perform() is done but we need to know the status before that
        # so we have to parse the status line ourselves.
        mt = re.match(r'^HTTP\/(\d(\.\d)?) ([1-5]\d\d) ([^\r\n\x00-\x08\x0b\x0c\x0e-\x1f\x7f]*)\r\n$', self._headers["x-status-line"])
        code = int(mt.group(3))

        if not self.name:
            logger.error("Cannot determine filename from URL or headers")
            return

        if code == 200:
            self.target = self.collection.open(self.name, "wb")

    def body_write(self, chunk):
        if self._first_chunk:
            self.headers_received()
            self._first_chunk = False

        self.count += len(chunk)

        if self.target is None:
            # "If this number is not equal to the size of the byte
            # string, this signifies an error and libcurl will abort
            # the request."
            return 0

        self.target.write(chunk)
        loopnow = time.time()
        if (loopnow - self.checkpoint) < 20:
            return

        bps = self.count / (loopnow - self.start)
        if self.contentlength is not None:
            logger.info("%2.1f%% complete, %6.2f MiB/s, %1.0f seconds left",
                        ((self.count * 100) / self.contentlength),
                        (bps / (1024.0*1024.0)),
                        ((self.contentlength-self.count) // bps))
        else:
            logger.info("%d downloaded, %6.2f MiB/s", self.count, (bps / (1024.0*1024.0)))
        self.checkpoint = loopnow


def check_cached_url(api, project_uuid, url, etags,
                     utcnow=datetime.datetime.utcnow,
                     varying_url_params="",
                     prefer_cached_downloads=False):
    return generic_check_cached_url(api, _Downloader(api),
                            project_uuid, url, etags,
                            utcnow=utcnow,
                            varying_url_params=varying_url_params,
                            prefer_cached_downloads=prefer_cached_downloads)


def http_to_keep(api, project_uuid, url,
                 utcnow=datetime.datetime.utcnow, varying_url_params="",
                 prefer_cached_downloads=False):
    """Download a file over HTTP and upload it to keep, with HTTP headers as metadata.

    Before downloading the URL, checks to see if the URL already
    exists in Keep and applies HTTP caching policy, the
    varying_url_params and prefer_cached_downloads flags in order to
    decide whether to use the version in Keep or re-download it.
    """

    return url_to_keep(api, _Downloader(api),
                       project_uuid, url,
                       utcnow,
                       varying_url_params,
                       prefer_cached_downloads)
