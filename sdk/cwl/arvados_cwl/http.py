# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import requests
import email.utils
import time
import datetime
import re
import arvados
import arvados.collection
import urlparse
import logging
import calendar

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


def changed(url, properties, now):
    req = requests.head(url, allow_redirects=True)
    remember_headers(url, properties, req.headers, now)

    if req.status_code != 200:
        raise Exception("Got status %s" % req.status_code)

    pr = properties[url]
    if "ETag" in pr and "ETag" in req.headers:
        if pr["ETag"] == req.headers["ETag"]:
            return False

    return True

def http_to_keep(api, project_uuid, url, utcnow=datetime.datetime.utcnow):
    r = api.collections().list(filters=[["properties", "exists", url]]).execute()

    now = utcnow()

    for item in r["items"]:
        properties = item["properties"]
        if fresh_cache(url, properties, now):
            # Do nothing
            cr = arvados.collection.CollectionReader(item["portable_data_hash"], api_client=api)
            return "keep:%s/%s" % (item["portable_data_hash"], cr.keys()[0])

        if not changed(url, properties, now):
            # ETag didn't change, same content, just update headers
            api.collections().update(uuid=item["uuid"], body={"collection":{"properties": properties}}).execute()
            cr = arvados.collection.CollectionReader(item["portable_data_hash"], api_client=api)
            return "keep:%s/%s" % (item["portable_data_hash"], cr.keys()[0])

    properties = {}
    req = requests.get(url, stream=True, allow_redirects=True)

    if req.status_code != 200:
        raise Exception("Failed to download '%s' got status %s " % (url, req.status_code))

    remember_headers(url, properties, req.headers, now)

    if "Content-Length" in properties[url]:
        cl = int(properties[url]["Content-Length"])
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
        name = urlparse.urlparse(url).path.split("/")[-1]

    count = 0
    start = time.time()
    checkpoint = start
    with c.open(name, "w") as f:
        for chunk in req.iter_content(chunk_size=1024):
            count += len(chunk)
            f.write(chunk)
            loopnow = time.time()
            if (loopnow - checkpoint) > 20:
                bps = (float(count)/float(loopnow - start))
                if cl is not None:
                    logger.info("%2.1f%% complete, %3.2f MiB/s, %1.0f seconds left",
                                float(count * 100) / float(cl),
                                bps/(1024*1024),
                                (cl-count)/bps)
                else:
                    logger.info("%d downloaded, %3.2f MiB/s", count, bps/(1024*1024))
                checkpoint = loopnow

    c.save_new(name="Downloaded from %s" % url, owner_uuid=project_uuid, ensure_unique_name=True)

    api.collections().update(uuid=c.manifest_locator(), body={"collection":{"properties": properties}}).execute()

    return "keep:%s/%s" % (c.portable_data_hash(), name)
