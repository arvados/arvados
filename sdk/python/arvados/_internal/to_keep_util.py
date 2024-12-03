# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import collections
import dataclasses
import typing
import logging
import email.utils
import calendar
import datetime
import re
import urllib.parse
import arvados
import arvados.collection
import arvados._internal

logger = logging.getLogger('arvados.file_import')

CheckCacheResult = collections.namedtuple('CheckCacheResult',
                                          ['portable_data_hash', 'file_name',
                                           'uuid', 'clean_url', 'now'])

@dataclasses.dataclass
class Response:
    status_code: int
    headers: typing.Mapping[str, str]

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

def remember_headers(url, properties, headers, now):
    properties.setdefault(url, {})
    for h in ("Cache-Control", "Etag", "Expires", "Date", "Content-Length"):
        if h in headers:
            properties[url][h] = headers[h]
    if "Date" not in headers:
        properties[url]["Date"] = _my_formatdate(now)

def _changed(url, clean_url, properties, now, downloader):
    req = downloader.head(url)

    if req.status_code != 200:
        # Sometimes endpoints are misconfigured and will deny HEAD but
        # allow GET so instead of failing here, we'll try GET If-None-Match
        return True

    # previous version of this code used "ETag", now we are
    # normalizing to "Etag", check for both.
    etag = properties[url].get("Etag") or properties[url].get("ETag")

    if url in properties:
        del properties[url]
    remember_headers(clean_url, properties, req.headers, now)

    if "Etag" in req.headers and etag == req.headers["Etag"]:
        # Didn't change
        return False

    return True


def check_cached_url(api, downloader, project_uuid, url, etags,
                     utcnow=datetime.datetime.utcnow,
                     varying_url_params="",
                     prefer_cached_downloads=False):

    logger.info("Checking Keep for %s", url)

    varying_params = set(arvados._internal.parse_seq(varying_url_params))

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
            return CheckCacheResult(item["portable_data_hash"], next(iter(cr.keys())),
                                    item["uuid"], clean_url, now)

        if not _changed(cache_url, clean_url, properties, now, downloader):
            # Etag didn't change, same content, just update headers
            api.collections().update(uuid=item["uuid"], body={"collection":{"properties": properties}}).execute()
            cr = arvados.collection.CollectionReader(item["portable_data_hash"], api_client=api)
            return CheckCacheResult(item["portable_data_hash"], next(iter(cr.keys())),
                                    item["uuid"], clean_url, now)

        for etagstr in ("Etag", "ETag"):
            if etagstr in properties[cache_url] and len(properties[cache_url][etagstr]) > 2:
                etags[properties[cache_url][etagstr]] = item

    logger.debug("Found ETag values %s", etags)

    return CheckCacheResult(None, None, None, clean_url, now)
