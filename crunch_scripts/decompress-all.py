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

arvados.job_setup.one_task_per_input_file(if_sequence=0, and_end_task=True,
                                          input_as_path=True)

task = arvados.current_task()

input_file = task['parameters']['input']

result = re.match(r"(^[a-f0-9]{32}\+\d+)(\+\S+)*(/.*)(/[^/]+)$", input_file)

outdir = os.path.join(task.tmpdir, "output")
os.makedirs(outdir)
os.chdir(outdir)

if result != None:
    cr = arvados.CollectionReader(result.group(1))
    streamname = result.group(3)[1:]
    filename = result.group(4)[1:]

    subprocess.call(["mkdir", "-p", streamname])
    os.chdir(streamname)
    streamreader = filter(lambda s: s.name() == streamname, cr.all_streams())[0]
    filereader = streamreader.files()[filename]
    rc = subprocess.call(["dtrx", "-r", "-n", "-q", arvados.get_task_param_mount('input')])
    if rc == 0:
        out = arvados.CollectionWriter()
        out.write_directory_tree(outdir, max_manifest_depth=0)
        task.set_output(out.finish())
    else:
        task.set_output(streamname + filereader.as_manifest()[1:])
else:
    sys.exit(1)
