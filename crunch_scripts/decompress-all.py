#!/usr/bin/env python

import arvados
import re
import subprocess

arvados.job_setup.one_task_per_input_file(if_sequence=0, and_end_task=True,
                                          input_as_path=True)

task = arvados.current_task()

input_file = arvados.gettaskparam('input')

result = re.match(r"(^[a-f0-9]{32}\+\d+)(\+\S+)*(/.*)(/.*)?$", input_file)

outdir = os.path.join(task.tmpdir, "output")
os.mkdirs(outdir)
os.chdir(outdir)

if result != None:
    cr = arvados.CollectionReader(re.group(1))
    streamname = '.'
    if re.group(3) != None:
        streamname += re.group(2)
        filename = re.group(3)[1:]
    else:
        filename = re.group(2)[1:]

    os.mkdirs(streamname)
    os.chdir(streamname)
    streamreader = filter(lambda s: s.name() == streamname, cr.all_streams())[0]
    filereader = stream.files()[filename]
    rc = subprocess.call("dtrx", "-r", "-n", arvados.get_task_param_mount('input'))
    if rc == 0:
        out.write_directory_tree(outdir, max_manifest_depth=0)
        arvados.task_set_output(out.finish())
    else:
        arvados.task_set_output(streamname + filereader.as_manifest()[1:])
else:
    sys.exit(1)
