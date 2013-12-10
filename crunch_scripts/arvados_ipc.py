import os
import re
import sys
import subprocess

def pipe_setup(pipes, name):
    pipes[name,'r'], pipes[name,'w'] = os.pipe()

def pipe_closeallbut(pipes, *keepus):
    for n,m in pipes.keys():
        if (n,m) not in keepus:
            os.close(pipes.pop((n,m), None))

def named_fork(children, name):
    children[name] = os.fork()
    return children[name]

def waitpid_and_check_children(children):
    """
    Given a dict of childname->pid, wait for each child process to
    finish, and report non-zero exit status on stderr. Return True if
    all children exited 0.
    """
    all_ok = True
    for (childname, pid) in children.items():
        # all_ok must be on RHS here -- we need to call waitpid() on
        # every child, even if all_ok is already False.
        all_ok = waitpid_and_check_exit(pid, childname) and all_ok
    return all_ok

def waitpid_and_check_exit(pid, childname=''):
    """
    Wait for a child process to finish. If it exits non-zero, report
    exit status on stderr (mentioning the given childname) and return
    False. If it exits zero, return True.
    """
    _, childstatus = os.waitpid(pid, 0)
    exitvalue = childstatus >> 8
    signal = childstatus & 127
    dumpedcore = childstatus & 128
    if childstatus != 0:
        sys.stderr.write("%s child %d failed: exit %d signal %d core %s\n"
                         % (childname, pid, exitvalue, signal,
                            ('y' if dumpedcore else 'n')))
        return False
    return True

