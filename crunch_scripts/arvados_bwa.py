# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import arvados
import re
import os
import sys
import fcntl
import subprocess

bwa_install_path = None

def install_path():
    """
    Extract the bwa source tree, build the bwa binary, and return the
    path to the source tree.
    """
    global bwa_install_path
    if bwa_install_path:
        return bwa_install_path

    bwa_install_path = arvados.util.tarball_extract(
        tarball = arvados.current_job()['script_parameters']['bwa_tbz'],
        path = 'bwa')

    # build "bwa" binary
    lockfile = open(os.path.split(bwa_install_path)[0] + '.bwa-make.lock',
                    'w')
    fcntl.flock(lockfile, fcntl.LOCK_EX)
    arvados.util.run_command(['make', '-j16'], cwd=bwa_install_path)
    lockfile.close()

    return bwa_install_path

def bwa_binary():
    """
    Return the path to the bwa executable.
    """
    return os.path.join(install_path(), 'bwa')

def run(command, command_args, **kwargs):
    """
    Build and run the bwa binary.

    command is the bwa module, e.g., "index" or "aln".

    command_args is a list of additional command line arguments, e.g.,
    ['-a', 'bwtsw', 'ref.fasta']

    It is assumed that we are running in a Crunch job environment, and
    the job's "bwa_tbz" parameter is a collection containing the bwa
    source tree in a .tbz file.
    """
    execargs = [bwa_binary(),
                command]
    execargs += command_args
    sys.stderr.write("%s.run: exec %s\n" % (__name__, str(execargs)))
    arvados.util.run_command(
        execargs,
        cwd=arvados.current_task().tmpdir,
        stderr=sys.stderr,
        stdin=kwargs.get('stdin', subprocess.PIPE),
        stdout=kwargs.get('stdout', sys.stderr))

def one_task_per_pair_input_file(if_sequence=0, and_end_task=True):
    """
    Queue one task for each pair of fastq files in this job's input
    collection.

    Each new task will have two parameters, named "input_1" and
    "input_2", each being a manifest containing a single fastq file.

    A matching pair of files in the input collection is assumed to
    have names "x_1.y" and "x_2.y".

    Files in the input collection that are not part of a matched pair
    are silently ignored.

    if_sequence and and_end_task arguments have the same significance
    as in arvados.job_setup.one_task_per_input_file().
    """
    if if_sequence != arvados.current_task()['sequence']:
        return
    job_input = arvados.current_job()['script_parameters']['input']
    cr = arvados.CollectionReader(job_input)
    all_files = []
    for s in cr.all_streams():
        all_files += list(s.all_files())
    for s in cr.all_streams():
        for left_file in s.all_files():
            left_name = left_file.name()
            right_file = None
            right_name = re.sub(r'(.*_)1\.', '\g<1>2.', left_name)
            if right_name == left_name:
                continue
            for f2 in s.all_files():
                if right_name == f2.name():
                    right_file = f2
            if right_file != None:
                new_task_attrs = {
                    'job_uuid': arvados.current_job()['uuid'],
                    'created_by_job_task_uuid': arvados.current_task()['uuid'],
                    'sequence': if_sequence + 1,
                    'parameters': {
                        'input_1':left_file.as_manifest(),
                        'input_2':right_file.as_manifest()
                        }
                    }
                arvados.api().job_tasks().create(body=new_task_attrs).execute()
    if and_end_task:
        arvados.api().job_tasks().update(uuid=arvados.current_task()['uuid'],
                                   body={'success':True}
                                   ).execute()
        exit(0)
