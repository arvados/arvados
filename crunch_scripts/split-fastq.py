#!/usr/bin/python

import arvados
import re
import hashlib

#api = arvados.api('v1')

piece = 0
manifest_text = ""

def readline(reader, start):
    line = ""
    n = -1
    while n == -1:
        r = reader.readfrom(start, 1024)
        if r == '':
            break
        n = string.find(r, "\n")
        line += r[0:n]
        start += len(r)
    return line

def splitfastq(p):
    for i in xrange(0, len(p)):
        p[i]["start"] = 0
        p[i]["end"] = 0

    while True:
        recordsize = [0, 0]

        # read 4 lines starting at "start"
        for ln in xrange(0, 4):
            for i in xrange(0, len(p)):
                r = readline(p[i]["reader"], p[i]["start"])
                if r == '':
                    return
                recordsize[i] += len(r)

        splitnow = False
        for i in xrange(0, len(p)):
            if ((p[i]["end"] - p[i]["start"]) + recordsize[i]) >= arvados.BLOCKSIZE:
                splitnow = True

        if splitnow:
            for i in xrange(0, len(p)):
                manifest = []
                manifest.extend(str(piece))
                manifest.extend([d[LOCATOR] for d in p["reader"]._stream._data_locators])
                manifest.extend(["{}:{}:{}".format(seg[LOCATOR], seg[BLOCKSIZE], self.name().replace(' ', '\\040')) for seg in arvados.locators_and_ranges(p[i]["reader"].segments, p[i]["start"], p[i]["end"] - p[i]["start"])])
                global manifest_text
                manifest_text += manifest.join(" ") + "\n"
                p[i]["start"] = p[i]["end"]
        else:
            for i in xrange(0, len(p)):
                p[i]["end"] += recordsize[i]

def put_manifest(manifest_text, sources=[]):
    crm = arvados.CollectionReader(manifest_text)

    combined = crm.manifest_text(strip=True)

    m = hashlib.new('md5')
    m.update(combined)

    uuid = "{}+{}".format(m.hexdigest(), len(combined))

    collection = arvados.api().collections().create(
        body={
            'uuid': uuid,
            'manifest_text': crm.manifest_text(),
        }).execute()

    for s in sources:
        l = arvados.api().links().create(body={
            "link": {
                "tail_uuid": s,
                "head_uuid": uuid,
                "link_class": "provenance",
                "name": "provided"
            }}).execute()

    return uuid


# Look for pairs

#inp = arvados.CollectionReader(arvados.get_job_param('input'))

with open("/home/peter/manifest") as f:
    inp = arvados.CollectionReader(f.read())

prog = re.compile("(.*?)_1.fastq$")

for s in inp.all_streams():
    if s.name() == ".":
        for f in s.all_files():
            result = prog.match(f.name())
            print f.name()
            if result != None:
                p = [{}, {}]
                p[0]["reader"] = s.file(f)
                p[1]["reader"] = s.file(prog.group(1) + "_2.fastq")

print manifest_text

#arvados.current_task().set_output(put_manifest(manifest_text))
