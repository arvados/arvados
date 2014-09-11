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

manifest_list = []

chunking = False #arvados.getjobparam('chunking')

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

# Chunk a fastq into approximately 64 MiB chunks.  Requires that the input data
# be decompressed ahead of time, such as using decompress-all.py.  Generates a
# new manifest, but doesn't actually move any data around.  Handles paired
# reads by ensuring that each chunk of a pair gets the same number of records.
#
# This works, but in practice is so slow that potential gains in alignment
# performance are lost in the prep time, which is why it is currently disabled.
#
# A better algorithm would seek to a file position a bit less than the desired
# chunk size and then scan ahead for the next record, making sure that record
# was matched by the read pair.
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
                print >>sys.stderr, "Finish piece ./_%s/%s (%s %s)" % (piece, p[i]["reader"].name(), p[i]["start"], p[i]["end"])
                manifest = []
                manifest.extend(["./_" + str(piece)])
                manifest.extend([d[arvados.LOCATOR] for d in p[i]["reader"]._stream._data_locators])
                manifest.extend(["{}:{}:{}".format(seg[arvados.LOCATOR]+seg[arvados.OFFSET], seg[arvados.SEGMENTSIZE], p[i]["reader"].name().replace(' ', '\\040')) for seg in arvados.locators_and_ranges(p[i]["reader"].segments, p[i]["start"], p[i]["end"] - p[i]["start"])])
                manifest_list.append(manifest)
                p[i]["start"] = p[i]["end"]
            piece += 1
        else:
            for i in xrange(0, len(p)):
                p[i]["end"] += recordsize[i]
            count += 1
            if count % 10000 == 0:
                print >>sys.stderr, "Record %s at %s" % (count, p[i]["end"])

prog = re.compile(r'(.*?)(_[12])?\.fastq(\.gz)?$')

# Look for fastq files
for s in inp.all_streams():
    for f in s.all_files():
        name_pieces = prog.match(f.name())
        if name_pieces is not None:
            if s.name() != ".":
                # The downstream tool (run-command) only iterates over the top
                # level of directories so if there are fastq files in
                # directories in the input, the choice is either to forget
                # there are directories (which might lead to name conflicts) or
                # just fail.
                print >>sys.stderr, "fastq must be at the root of the collection"
                sys.exit(1)

            p = None
            if name_pieces.group(2) is not None:
                if name_pieces.group(2) == "_1":
                    p = [{}, {}]
                    p[0]["reader"] = s.files()[name_pieces.group(0)]
                    p[1]["reader"] = s.files()[name_pieces.group(1) + "_2.fastq" + (name_pieces.group(3) if name_pieces.group(3) else '')]
            else:
                p = [{}]
                p[0]["reader"] = s.files()[name_pieces.group(0)]

            if p is not None:
                if chunking:
                    splitfastq(p)
                else:
                    for i in xrange(0, len(p)):
                        m = p[i]["reader"].as_manifest().split()
                        m[0] = "./_" + str(piece)
                        manifest_list.append(m)
                    piece += 1

manifest_text = "\n".join(" ".join(m) for m in manifest_list) + "\n"

arvados.current_task().set_output(manifest_text)
