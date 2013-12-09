import arvados
import re
import os
import sys
import fcntl
import subprocess

bwa_install_path = None

def setup():
    global bwa_install_path
    if bwa_install_path:
        return bwa_install_path
    bwa_path = arvados.util.tarball_extract(
        tarball = arvados.current_job()['script_parameters']['bwa_tbz'],
        path = 'bwa')

    # build "bwa" binary
    lockfile = open(os.path.split(bwa_path)[0] + '.bwa-make.lock',
                    'w')
    fcntl.flock(lockfile, fcntl.LOCK_EX)
    arvados.util.run_command(['make', '-j16'], cwd=bwa_path)
    lockfile.close()

    bwa_install_path = bwa_path
    return bwa_path

def bwa_binary():
    global bwa_install_path
    return os.path.join(bwa_install_path, 'bwa')

def run(command, command_args, **kwargs):
    global bwa_install_path
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

setup()
