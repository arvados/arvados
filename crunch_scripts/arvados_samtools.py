# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import arvados
import re
import os
import sys
import fcntl
import subprocess

samtools_path = None

def samtools_install_path():
    """
    Extract the samtools source tree, build the samtools binary, and
    return the path to the source tree.
    """
    global samtools_path
    if samtools_path:
        return samtools_path
    samtools_path = arvados.util.tarball_extract(
        tarball = arvados.current_job()['script_parameters']['samtools_tgz'],
        path = 'samtools')

    # build "samtools" binary
    lockfile = open(os.path.split(samtools_path)[0] + '.samtools-make.lock',
                    'w')
    fcntl.flock(lockfile, fcntl.LOCK_EX)
    arvados.util.run_command(['make', '-j16'], cwd=samtools_path)
    lockfile.close()

    return samtools_path

def samtools_binary():
    """
    Return the path to the samtools executable.
    """
    return os.path.join(samtools_install_path(), 'samtools')

def run(command, command_args, **kwargs):
    """
    Build and run the samtools binary.

    command is the samtools subcommand, e.g., "view" or "sort".

    command_args is a list of additional command line arguments, e.g.,
    ['-bt', 'ref_list.txt', '-o', 'aln.bam', 'aln.sam.gz']

    It is assumed that we are running in a Crunch job environment, and
    the job's "samtools_tgz" parameter is a collection containing the
    samtools source tree in a .tgz file.
    """
    execargs = [samtools_binary(),
                command]
    execargs += command_args
    sys.stderr.write("%s.run: exec %s\n" % (__name__, str(execargs)))
    arvados.util.run_command(
        execargs,
        cwd=arvados.current_task().tmpdir,
        stdin=kwargs.get('stdin', subprocess.PIPE),
        stderr=kwargs.get('stderr', sys.stderr),
        stdout=kwargs.get('stdout', sys.stderr))

def one_task_per_bam_file(if_sequence=0, and_end_task=True):
    """
    Queue one task for each bam file in this job's input collection.

    Each new task will have an "input" parameter: a manifest
    containing one .bam file and (if available) the corresponding .bai
    index file.

    Files in the input collection that are not named *.bam or *.bai
    (as well as *.bai files that do not match any .bam file present)
    are silently ignored.

    if_sequence and and_end_task arguments have the same significance
    as in arvados.job_setup.one_task_per_input_file().
    """
    if if_sequence != arvados.current_task()['sequence']:
        return
    job_input = arvados.current_job()['script_parameters']['input']
    cr = arvados.CollectionReader(job_input)
    bam = {}
    bai = {}
    for s in cr.all_streams():
        for f in s.all_files():
            if re.search(r'\.bam$', f.name()):
                bam[s.name(), f.name()] = f
            elif re.search(r'\.bai$', f.name()):
                bai[s.name(), f.name()] = f
    for ((s_name, f_name), bam_f) in bam.items():
        bai_f = bai.get((s_name, re.sub(r'bam$', 'bai', f_name)), None)
        task_input = bam_f.as_manifest()
        if bai_f:
            task_input += bai_f.as_manifest()
        new_task_attrs = {
            'job_uuid': arvados.current_job()['uuid'],
            'created_by_job_task_uuid': arvados.current_task()['uuid'],
            'sequence': if_sequence + 1,
            'parameters': {
                'input': task_input
                }
            }
        arvados.api().job_tasks().create(body=new_task_attrs).execute()
    if and_end_task:
        arvados.api().job_tasks().update(uuid=arvados.current_task()['uuid'],
                                         body={'success':True}
                                         ).execute()
        exit(0)
