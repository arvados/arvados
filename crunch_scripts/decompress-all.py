#!/usr/bin/env python

#
# decompress-all.py
#
# Decompress all compressed files in the collection using the "dtrx" tool and
# produce a new collection with the contents.  Uncompressed files
# are passed through.
#
# input:
# A collection at script_parameters["input"]
#
# output:
# A manifest of the uncompressed contents of the input collection.

import arvados
import re
import subprocess
import os
import sys
import crunchutil.robust_put as robust_put

arvados.job_setup.one_task_per_input_file(if_sequence=0, and_end_task=True,
                                          input_as_path=True)

task = arvados.current_task()

input_file = task['parameters']['input']

infile_parts = re.match(r"(^[a-f0-9]{32}\+\d+)(\+\S+)*(/.*)?(/[^/]+)$", input_file)

outdir = os.path.join(task.tmpdir, "output")
os.makedirs(outdir)
os.chdir(outdir)

if infile_parts is None:
    print >>sys.stderr, "Failed to parse input filename '%s' as a Keep file\n" % input_file
    sys.exit(1)

cr = arvados.CollectionReader(infile_parts.group(1))
streamname = infile_parts.group(3)[1:]
filename = infile_parts.group(4)[1:]

if streamname is not None:
    subprocess.call(["mkdir", "-p", streamname])
    os.chdir(streamname)
else:
    streamname = '.'

m = re.match(r'.*\.(gz|Z|bz2|tgz|tbz|zip|rar|7z|cab|deb|rpm|cpio|gem)$', arvados.get_task_param_mount('input'), re.IGNORECASE)

if m is not None:
    rc = subprocess.call(["dtrx", "-r", "-n", "-q", arvados.get_task_param_mount('input')])
    if rc == 0:
        task.set_output(robust_put.upload(outdir))
    else:
        sys.exit(rc)
else:
    streamreader = filter(lambda s: s.name() == streamname, cr.all_streams())[0]
    filereader = streamreader.files()[filename]
    task.set_output(streamname + filereader.as_manifest()[1:])
