#!/usr/bin/python

import arvados
import re
import hashlib
import string

api = arvados.api('v1')

piece = 0
manifest_text = ""

# Look for paired reads

inp = arvados.CollectionReader(arvados.getjobparam('reads'))

prog = re.compile(r'(.*?)_1.fastq(.gz)?$')

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
                global piece
                global manifest_text
                manifest = []
                manifest.extend("./_" + str(piece))
                manifest.extend([d[LOCATOR] for d in p["reader"]._stream._data_locators])
                manifest.extend(["{}:{}:{}".format(seg[LOCATOR], seg[BLOCKSIZE], self.name().replace(' ', '\\040')) for seg in arvados.locators_and_ranges(p[i]["reader"].segments, p[i]["start"], p[i]["end"] - p[i]["start"])])
                manifest_text += manifest.join(" ") + "\n"
                p[i]["start"] = p[i]["end"]
        else:
            for i in xrange(0, len(p)):
                p[i]["end"] += recordsize[i]


for s in inp.all_streams():
    if s.name() == ".":
        for f in s.all_files():
            result = prog.match(f.name())
            if result != None:
                p = [{}, {}]
                p[0]["reader"] = s.files()[result.group(0)]
                if result.group(2) != None:
                    p[1]["reader"] = s.files()[result.group(1) + "_2.fastq" + result.group(2)]
                else:
                    p[1]["reader"] = s.files()[result.group(1) + "_2.fastq"]
                splitfastq(p)
                #m0 = p[0]["reader"].as_manifest()[1:]
                #m1 = p[1]["reader"].as_manifest()[1:]
                #manifest_text += "./_" + str(piece) + m0
                #manifest_text += "./_" + str(piece) + m1
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
                    splitfastq(p)
                    #m0 = p[0]["reader"].as_manifest()[1:]
                    #manifest_text += "./_" + str(piece) + m0
                    piece += 1

arvados.current_task().set_output(manifest_text)
