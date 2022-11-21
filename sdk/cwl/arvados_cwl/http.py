# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import division
from future import standard_library
standard_library.install_aliases()

import requests
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

logger = logging.getLogger('arvados.cwl-runner')

def my_formatdate(dt):
    return email.utils.formatdate(timeval=calendar.timegm(dt.timetuple()),
                                  localtime=False, usegmt=True)

def my_parsedate(text):
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

def fresh_cache(url, properties, now):
    pr = properties[url]
    expires = None

    logger.debug("Checking cache freshness for %s using %s", url, pr)

    if "Cache-Control" in pr:
        if re.match(r"immutable", pr["Cache-Control"]):
            return True

        g = re.match(r"(s-maxage|max-age)=(\d+)", pr["Cache-Control"])
        if g:
            expires = my_parsedate(pr["Date"]) + datetime.timedelta(seconds=int(g.group(2)))

    if expires is None and "Expires" in pr:
        expires = my_parsedate(pr["Expires"])

    if expires is None:
        # Use a default cache time of 24 hours if upstream didn't set
        # any cache headers, to reduce redundant downloads.
        expires = my_parsedate(pr["Date"]) + datetime.timedelta(hours=24)

    if not expires:
        return False

    return (now < expires)

def remember_headers(url, properties, headers, now):
    properties.setdefault(url, {})
    for h in ("Cache-Control", "ETag", "Expires", "Date", "Content-Length"):
        if h in headers:
            properties[url][h] = headers[h]
    if "Date" not in headers:
        properties[url]["Date"] = my_formatdate(now)


def changed(url, clean_url, properties, now):
    req = requests.head(url, allow_redirects=True)

    if req.status_code != 200:
        # Sometimes endpoints are misconfigured and will deny HEAD but
        # allow GET so instead of failing here, we'll try GET If-None-Match
        return True

    etag = properties[url].get("ETag")

    if url in properties:
        del properties[url]
    remember_headers(clean_url, properties, req.headers, now)

    if "ETag" in req.headers and etag == req.headers["ETag"]:
        # Didn't change
        return False

    return True

def etag_quote(etag):
    # if it already has leading and trailing quotes, do nothing
    if etag[0] == '"' and etag[-1] == '"':
        return etag
    else:
        # Add quotes.
        return '"' + etag + '"'


def http_to_keep(api, project_uuid, url, utcnow=datetime.datetime.utcnow, varying_url_params="", prefer_cached_downloads=False):
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

    for item in items:
        properties = item["properties"]

        if clean_url in properties:
            cache_url = clean_url
        elif url in properties:
            cache_url = url
        else:
            return False

        if prefer_cached_downloads or fresh_cache(cache_url, properties, now):
            # HTTP caching rules say we should use the cache
            cr = arvados.collection.CollectionReader(item["portable_data_hash"], api_client=api)
            return "keep:%s/%s" % (item["portable_data_hash"], list(cr.keys())[0])

        if not changed(cache_url, clean_url, properties, now):
            # ETag didn't change, same content, just update headers
            api.collections().update(uuid=item["uuid"], body={"collection":{"properties": properties}}).execute()
            cr = arvados.collection.CollectionReader(item["portable_data_hash"], api_client=api)
            return "keep:%s/%s" % (item["portable_data_hash"], list(cr.keys())[0])

        if "ETag" in properties[cache_url] and len(properties[cache_url]["ETag"]) > 2:
            etags[properties[cache_url]["ETag"]] = item

    logger.debug("Found ETags %s", etags)

    properties = {}
    headers = {}
    if etags:
        headers['If-None-Match'] = ', '.join([etag_quote(k) for k,v in etags.items()])
    logger.debug("Sending GET request with headers %s", headers)
    req = requests.get(url, stream=True, allow_redirects=True, headers=headers)

    if req.status_code not in (200, 304):
        raise Exception("Failed to download '%s' got status %s " % (url, req.status_code))

    remember_headers(clean_url, properties, req.headers, now)

    if req.status_code == 304 and "ETag" in req.headers and req.headers["ETag"] in etags:
        item = etags[req.headers["ETag"]]
        item["properties"].update(properties)
        api.collections().update(uuid=item["uuid"], body={"collection":{"properties": item["properties"]}}).execute()
        cr = arvados.collection.CollectionReader(item["portable_data_hash"], api_client=api)
        return "keep:%s/%s" % (item["portable_data_hash"], list(cr.keys())[0])

    if "Content-Length" in properties[clean_url]:
        cl = int(properties[clean_url]["Content-Length"])
        logger.info("Downloading %s (%s bytes)", url, cl)
    else:
        cl = None
        logger.info("Downloading %s (unknown size)", url)

    c = arvados.collection.Collection()

    if req.headers.get("Content-Disposition"):
        grp = re.search(r'filename=("((\"|[^"])+)"|([^][()<>@,;:\"/?={} ]+))', req.headers["Content-Disposition"])
        if grp.group(2):
            name = grp.group(2)
        else:
            name = grp.group(4)
    else:
        name = parsed.path.split("/")[-1]

    count = 0
    start = time.time()
    checkpoint = start
    with c.open(name, "wb") as f:
        for chunk in req.iter_content(chunk_size=1024):
            count += len(chunk)
            f.write(chunk)
            loopnow = time.time()
            if (loopnow - checkpoint) > 20:
                bps = count / (loopnow - start)
                if cl is not None:
                    logger.info("%2.1f%% complete, %3.2f MiB/s, %1.0f seconds left",
                                ((count * 100) / cl),
                                (bps // (1024*1024)),
                                ((cl-count) // bps))
                else:
                    logger.info("%d downloaded, %3.2f MiB/s", count, (bps / (1024*1024)))
                checkpoint = loopnow

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

    return "keep:%s/%s" % (c.portable_data_hash(), name)
