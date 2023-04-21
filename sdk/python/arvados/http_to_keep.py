# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import division
from future import standard_library
standard_library.install_aliases()

import email.utils
import time
import datetime
import re
import arvados
import arvados.collection
import urllib.parse
import logging
import calendar
import urllib.parse
import pycurl
import dataclasses
import typing
from arvados._pycurlhelper import PyCurlHelper

logger = logging.getLogger('arvados.http_import')

def _my_formatdate(dt):
    return email.utils.formatdate(timeval=calendar.timegm(dt.timetuple()),
                                  localtime=False, usegmt=True)

def _my_parsedate(text):
    parsed = email.utils.parsedate_tz(text)
    if parsed:
        if parsed[9]:
            # Adjust to UTC
            return datetime.datetime(*parsed[:6]) + datetime.timedelta(seconds=parsed[9])
        else:
            # TZ is zero or missing, assume UTC.
            return datetime.datetime(*parsed[:6])
    else:
        return datetime.datetime(1970, 1, 1)

def _fresh_cache(url, properties, now):
    pr = properties[url]
    expires = None

    logger.debug("Checking cache freshness for %s using %s", url, pr)

    if "Cache-Control" in pr:
        if re.match(r"immutable", pr["Cache-Control"]):
            return True

        g = re.match(r"(s-maxage|max-age)=(\d+)", pr["Cache-Control"])
        if g:
            expires = _my_parsedate(pr["Date"]) + datetime.timedelta(seconds=int(g.group(2)))

    if expires is None and "Expires" in pr:
        expires = _my_parsedate(pr["Expires"])

    if expires is None:
        # Use a default cache time of 24 hours if upstream didn't set
        # any cache headers, to reduce redundant downloads.
        expires = _my_parsedate(pr["Date"]) + datetime.timedelta(hours=24)

    if not expires:
        return False

    return (now < expires)

def _remember_headers(url, properties, headers, now):
    properties.setdefault(url, {})
    for h in ("Cache-Control", "Etag", "Expires", "Date", "Content-Length"):
        if h in headers:
            properties[url][h] = headers[h]
    if "Date" not in headers:
        properties[url]["Date"] = _my_formatdate(now)

@dataclasses.dataclass
class _Response:
    status_code: int
    headers: typing.Mapping[str, str]


class _Downloader(PyCurlHelper):
    # Wait up to 60 seconds for connection
    # How long it can be in "low bandwidth" state before it gives up
    # Low bandwidth threshold is 32 KiB/s
    DOWNLOADER_TIMEOUT = (60, 300, 32768)

    def __init__(self, apiclient):
        super(_Downloader, self).__init__(title_case_headers=True)
        self.curl = pycurl.Curl()
        self.curl.setopt(pycurl.NOSIGNAL, 1)
        self.curl.setopt(pycurl.OPENSOCKETFUNCTION,
                    lambda *args, **kwargs: self._socket_open(*args, **kwargs))
        self.target = None
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

        return _Response(self.curl.getinfo(pycurl.RESPONSE_CODE), self._headers)

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

        return _Response(self.curl.getinfo(pycurl.RESPONSE_CODE), self._headers)

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

        if code == 200:
            self.target = self.collection.open(self.name, "wb")

    def body_write(self, chunk):
        if self._first_chunk:
            self.headers_received()
            self._first_chunk = False

        self.count += len(chunk)
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
            logger.info("%d downloaded, %6.2f MiB/s", count, (bps / (1024.0*1024.0)))
        self.checkpoint = loopnow


def _changed(url, clean_url, properties, now, curldownloader):
    req = curldownloader.head(url)

    if req.status_code != 200:
        # Sometimes endpoints are misconfigured and will deny HEAD but
        # allow GET so instead of failing here, we'll try GET If-None-Match
        return True

    # previous version of this code used "ETag", now we are
    # normalizing to "Etag", check for both.
    etag = properties[url].get("Etag") or properties[url].get("ETag")

    if url in properties:
        del properties[url]
    _remember_headers(clean_url, properties, req.headers, now)

    if "Etag" in req.headers and etag == req.headers["Etag"]:
        # Didn't change
        return False

    return True

def _etag_quote(etag):
    # if it already has leading and trailing quotes, do nothing
    if etag[0] == '"' and etag[-1] == '"':
        return etag
    else:
        # Add quotes.
        return '"' + etag + '"'


def http_to_keep(api, project_uuid, url,
                 utcnow=datetime.datetime.utcnow, varying_url_params="",
                 prefer_cached_downloads=False):
    """Download a file over HTTP and upload it to keep, with HTTP headers as metadata.

    Before downloading the URL, checks to see if the URL already
    exists in Keep and applies HTTP caching policy, the
    varying_url_params and prefer_cached_downloads flags in order to
    decide whether to use the version in Keep or re-download it.
    """

    logger.info("Checking Keep for %s", url)

    varying_params = [s.strip() for s in varying_url_params.split(",")]

    parsed = urllib.parse.urlparse(url)
    query = [q for q in urllib.parse.parse_qsl(parsed.query)
             if q[0] not in varying_params]

    clean_url = urllib.parse.urlunparse((parsed.scheme, parsed.netloc, parsed.path, parsed.params,
                                         urllib.parse.urlencode(query, safe="/"),  parsed.fragment))

    r1 = api.collections().list(filters=[["properties", "exists", url]]).execute()

    if clean_url == url:
        items = r1["items"]
    else:
        r2 = api.collections().list(filters=[["properties", "exists", clean_url]]).execute()
        items = r1["items"] + r2["items"]

    now = utcnow()

    etags = {}

    curldownloader = _Downloader(api)

    for item in items:
        properties = item["properties"]

        if clean_url in properties:
            cache_url = clean_url
        elif url in properties:
            cache_url = url
        else:
            raise Exception("Shouldn't happen, got an API result for %s that doesn't have the URL in properties" % item["uuid"])

        if prefer_cached_downloads or _fresh_cache(cache_url, properties, now):
            # HTTP caching rules say we should use the cache
            cr = arvados.collection.CollectionReader(item["portable_data_hash"], api_client=api)
            return (item["portable_data_hash"], next(iter(cr.keys())) )

        if not _changed(cache_url, clean_url, properties, now, curldownloader):
            # Etag didn't change, same content, just update headers
            api.collections().update(uuid=item["uuid"], body={"collection":{"properties": properties}}).execute()
            cr = arvados.collection.CollectionReader(item["portable_data_hash"], api_client=api)
            return (item["portable_data_hash"], next(iter(cr.keys())))

        for etagstr in ("Etag", "ETag"):
            if etagstr in properties[cache_url] and len(properties[cache_url][etagstr]) > 2:
                etags[properties[cache_url][etagstr]] = item

    logger.debug("Found ETag values %s", etags)

    properties = {}
    headers = {}
    if etags:
        headers['If-None-Match'] = ', '.join([_etag_quote(k) for k,v in etags.items()])
    logger.debug("Sending GET request with headers %s", headers)

    logger.info("Beginning download of %s", url)

    req = curldownloader.download(url, headers)

    c = curldownloader.collection

    if req.status_code not in (200, 304):
        raise Exception("Failed to download '%s' got status %s " % (url, req.status_code))

    if curldownloader.target is not None:
        curldownloader.target.close()

    _remember_headers(clean_url, properties, req.headers, now)

    if req.status_code == 304 and "Etag" in req.headers and req.headers["Etag"] in etags:
        item = etags[req.headers["Etag"]]
        item["properties"].update(properties)
        api.collections().update(uuid=item["uuid"], body={"collection":{"properties": item["properties"]}}).execute()
        cr = arvados.collection.CollectionReader(item["portable_data_hash"], api_client=api)
        return (item["portable_data_hash"], list(cr.keys())[0])

    logger.info("Download complete")

    collectionname = "Downloaded from %s" % urllib.parse.quote(clean_url, safe='')

    # max length - space to add a timestamp used by ensure_unique_name
    max_name_len = 254 - 28

    if len(collectionname) > max_name_len:
        over = len(collectionname) - max_name_len
        split = int(max_name_len/2)
        collectionname = collectionname[0:split] + "…" + collectionname[split+over:]

    c.save_new(name=collectionname, owner_uuid=project_uuid, ensure_unique_name=True)

    api.collections().update(uuid=c.manifest_locator(), body={"collection":{"properties": properties}}).execute()

    return (c.portable_data_hash(), curldownloader.name)
