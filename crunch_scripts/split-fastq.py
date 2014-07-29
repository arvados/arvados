#!/usr/bin/python

import arvados
import re
import hashlib

api = arvados.api('v1')

piece = 0
manifest_text = ""

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

# Look for paired reads

inp = arvados.CollectionReader(arvados.getjobparam('input'))

with open("/home/peter/manifest") as f:
    inp = arvados.CollectionReader(f.read())

prog = re.compile("(.*?)_1.fastq(.gz)?$")

manifest_text = ""

for s in inp.all_streams():
    if s.name() == ".":
        for f in s.all_files():
            result = prog.match(f.name())
            if result != None:
                p = [{}, {}]
                p[0]["reader"] = s.files()[result.group(0)]
                p[1]["reader"] = s.files()[result.group(1) + "_2.fastq" + result.group(2)]
                m0 = p[0]["reader"].as_manifest()[1:]
                m1 = p[1]["reader"].as_manifest()[1:]
                manifest_text += "_" + str(piece) + m0
                manifest_text += "_" + str(piece) + m1
                piece += 1

# No pairs found so just put each fastq file into a separate directory
if manifest_text == "":
    for s in inp.all_streams():
        prog = re.compile("(.*?).fastq(.gz)?$")
        if s.name() == ".":
            for f in s.all_files():
                result = prog.match(f.name())
                if result != None:
                    p = [{}]
                    p[0]["reader"] = s.files()[result.group(0)]
                    m0 = p[0]["reader"].as_manifest()[1:]
                    manifest_text += "_" + str(piece) + m0
                    piece += 1

arvados.current_task().set_output(put_manifest(manifest_text, [arvados.get_job_param('input')]))
