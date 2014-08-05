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

manifest_list = []

chunking = arvados.getjobparam('chunking')

def nextline(reader, start):
    n = -1
    while True:
        r = reader.readfrom(start, 128)
        if r == '':
            break
        n = string.find(r, "\n")
        if n > -1:
            break
        else:
            start += 128
    return n

def splitfastq(p):
    for i in xrange(0, len(p)):
        p[i]["start"] = 0
        p[i]["end"] = 0

    count = 0
    recordsize = [0, 0]

    global piece
    finish = False
    while not finish:
        for i in xrange(0, len(p)):
            recordsize[i] = 0

        # read next 4 lines
        for i in xrange(0, len(p)):
            for ln in xrange(0, 4):
                r = nextline(p[i]["reader"], p[i]["end"]+recordsize[i])
                if r == -1:
                    finish = True
                    break
                recordsize[i] += (r+1)

        splitnow = finish
        for i in xrange(0, len(p)):
            if ((p[i]["end"] - p[i]["start"]) + recordsize[i]) >= (64*1024*1024):
                splitnow = True

        if splitnow:
            for i in xrange(0, len(p)):
                global manifest_list
                print "Finish piece ./_%s/%s (%s %s)" % (piece, p[i]["reader"].name(), p[i]["start"], p[i]["end"])
                manifest = []
                manifest.extend(["./_" + str(piece)])
                manifest.extend([d[arvados.LOCATOR] for d in p[i]["reader"]._stream._data_locators])

                print p[i]
                print arvados.locators_and_ranges(p[i]["reader"].segments, p[i]["start"], p[i]["end"] - p[i]["start"])

                manifest.extend(["{}:{}:{}".format(seg[arvados.LOCATOR]+seg[arvados.OFFSET], seg[arvados.SEGMENTSIZE], p[i]["reader"].name().replace(' ', '\\040')) for seg in arvados.locators_and_ranges(p[i]["reader"].segments, p[i]["start"], p[i]["end"] - p[i]["start"])])
                manifest_list.append(manifest)
                print "Finish piece %s" % (" ".join(manifest))
                p[i]["start"] = p[i]["end"]
            piece += 1
        else:
            for i in xrange(0, len(p)):
                p[i]["end"] += recordsize[i]
            count += 1
            if count % 10000 == 0:
                print "Record %s at %s" % (count, p[i]["end"])

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
                if chunking:
                    splitfastq(p)
                else:
                    m0 = p[0]["reader"].as_manifest()[1:]
                    m1 = p[1]["reader"].as_manifest()[1:]
                    manifest_list.append(["./_" + str(piece), m0[:-1]])
                    manifest_list.append(["./_" + str(piece), m1[:-1]])
                    piece += 1

# No pairs found so just put each fastq file into a separate directory
if len(manifest_list) == 0:
    for s in inp.all_streams():
        prog = re.compile("(.*?).fastq(.gz)?$")
        if s.name() == ".":
            for f in s.all_files():
                result = prog.match(f.name())
                if result != None:
                    p = [{}]
                    p[0]["reader"] = s.files()[result.group(0)]
                    if chunking:
                        splitfastq(p)
                    else:
                        m0 = p[0]["reader"].as_manifest()[1:]
                        manifest_list.append(["./_" + str(piece), m0])
                        piece += 1

manifest_text = "\n".join(" ".join(m) for m in manifest_list)

arvados.current_task().set_output(manifest_text)
