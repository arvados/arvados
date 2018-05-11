import requests
import email.utils
import time
import datetime
import re
import arvados
import arvados.collection
import urlparse

def my_formatdate(dt):
    return email.utils.formatdate(timeval=time.mktime(now.timetuple()), localtime=False, usegmt=True)

def my_parsedate(text):
    return datetime.datetime(*email.utils.parsedate(text)[:6])

def fresh_cache(url, properties):
    pr = properties[url]
    expires = None

    if "Cache-Control" in pr:
        if re.match(r"immutable", pr["Cache-Control"]):
            return True

        g = re.match(r"(s-maxage|max-age)=(\d+)", pr["Cache-Control"])
        if g:
            expires = my_parsedate(pr["Date"]) + datetime.timedelta(seconds=int(g.group(2)))

    if expires is None and "Expires" in pr:
        expires = my_parsedate(pr["Expires"])

    if not expires:
        return False

    return (datetime.datetime.utcnow() < expires)

def remember_headers(url, properties, headers):
    properties.setdefault(url, {})
    for h in ("Cache-Control", "ETag", "Expires", "Date"):
        if h in headers:
            properties[url][h] = headers[h]
    if "Date" not in headers:
        properties[url]["Date"] = my_formatdate(datetime.datetime.utcnow())


def changed(url, properties):
    req = requests.head(url)
    remember_headers(url, properties, req.headers)

    if req.status_code != 200:
        raise Exception("Got status %s" % req.status_code)

    pr = properties[url]
    if "ETag" in pr and "ETag" in req.headers:
        if pr["ETag"] == req.headers["ETag"]:
            return False
    return True

def http_to_keep(api, project_uuid, url):
    r = api.collections().list(filters=[["properties", "exists", url]]).execute()
    name = urlparse.urlparse(url).path.split("/")[-1]

    for item in r["items"]:
        properties = item["properties"]
        if fresh_cache(url, properties):
            # Do nothing
            return "keep:%s/%s" % (item["portable_data_hash"], name)

        if not changed(url, properties):
            # ETag didn't change, same content, just update headers
            api.collections().update(uuid=item["uuid"], body={"collection":{"properties": properties}}).execute()
            return "keep:%s/%s" % (item["portable_data_hash"], name)

    properties = {}
    req = requests.get(url, stream=True)

    if req.status_code != 200:
        raise Exception("Got status %s" % req.status_code)

    remember_headers(url, properties, req.headers)

    c = arvados.collection.Collection()

    with c.open(name, "w") as f:
        for chunk in req.iter_content(chunk_size=128):
            f.write(chunk)

    c.save_new(name="Downloaded from %s" % url, owner_uuid=project_uuid)

    api.collections().update(uuid=c.manifest_locator(), body={"collection":{"properties": properties}}).execute()

    return "keep:%s/%s" % (c.portable_data_hash(), name)
