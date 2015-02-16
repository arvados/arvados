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
                for i in xrange(0, len(p)):
                    m = p[i]["reader"].as_manifest().split()
                    m[0] = "./_" + str(piece)
                    manifest_list.append(m)
                piece += 1

manifest_text = "\n".join(" ".join(m) for m in manifest_list) + "\n"

arvados.current_task().set_output(manifest_text)
