import arvados
import re
import os
import sys
import fcntl
import subprocess

gatk2_install_path = None

def install_path():
    global gatk2_install_path
    if gatk2_install_path:
        return gatk2_install_path
    gatk2_install_path = arvados.util.tarball_extract(
        tarball = arvados.current_job()['script_parameters']['gatk_tbz'],
        path = 'gatk2')
    return gatk2_install_path

def memory_limit():
    taskspernode = int(os.environ.get('CRUNCH_NODE_SLOTS', '1'))
    with open('/proc/meminfo', 'r') as f:
        ram = int(re.search(r'MemTotal:\s*(\d+)', f.read()).group(1)) / 1024
    if taskspernode > 1:
        ram = ram / taskspernode
    return max(ram-700, 500)

def cpus_on_this_node():
    with open('/proc/cpuinfo', 'r') as cpuinfo:
        return max(int(os.environ.get('SLURM_CPUS_ON_NODE', 1)),
                   len(re.findall(r'^processor\s*:\s*\d',
                                  cpuinfo.read(),
                                  re.MULTILINE)))

def cpus_per_task():
    return max(1, (cpus_on_this_node()
                   / int(os.environ.get('CRUNCH_NODE_SLOTS', 1))))

def run(**kwargs):
    kwargs.setdefault('cwd', arvados.current_task().tmpdir)
    kwargs.setdefault('stdout', sys.stderr)
    execargs = ['java',
                '-Xmx%dm' % memory_limit(),
                '-Djava.io.tmpdir=' + arvados.current_task().tmpdir,
                '-jar', os.path.join(install_path(), 'GenomeAnalysisTK.jar')]
    execargs += [str(arg) for arg in kwargs.pop('args', [])]
    sys.stderr.write("%s.run: exec %s\n" % (__name__, str(execargs)))
    return arvados.util.run_command(execargs, **kwargs)

