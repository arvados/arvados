#!/usr/bin/env python

import arvados
import argparse
import json
import re
import os
import stat
import put
import time
#import arvados.command.ws as ws
import subprocess
import logging

logger = logging.getLogger('arvados.arv-run')

arvrun_parser = argparse.ArgumentParser()
arvrun_parser.add_argument('--dry-run', action="store_true")
arvrun_parser.add_argument('--local', action="store_true")
arvrun_parser.add_argument('--docker-image', type=str, default="arvados/jobs")
arvrun_parser.add_argument('--git-dir', type=str, default="")
arvrun_parser.add_argument('args', nargs=argparse.REMAINDER)

class UploadFile(object):
    def __init__(self, prefix, fn):
        self.prefix = prefix
        self.fn = fn

def is_in_collection(root, branch):
    if root == "/":
        return (None, None)
    fn = os.path.join(root, ".arvados#collection")
    if os.path.exists(fn):
        with file(fn, 'r') as f:
            c = json.load(f)
        return (c["portable_data_hash"], branch)
    else:
        sp = os.path.split(root)
        return is_in_collection(sp[0], os.path.join(sp[1], branch))

def statfile(prefix, fn):
    absfn = os.path.abspath(fn)
    if os.path.exists(absfn):
        st = os.stat(absfn)
        if stat.S_ISREG(st.st_mode):
            sp = os.path.split(absfn)
            (pdh, branch) = is_in_collection(sp[0], sp[1])
            if pdh:
                return "%s$(file %s/%s)" % (prefix, pdh, branch)
            else:
                # trim leading '/' for path prefix test later
                return UploadFile(prefix, absfn[1:])
    return prefix+fn

def main(arguments=None):
    args = arvrun_parser.parse_args(arguments)

    reading_into = 2

    slots = [[], [], []]
    for c in args.args:
        if c == '>':
            reading_into = 0
        elif c == '<':
            reading_into = 1
        elif c == '|':
            reading_into = len(slots)
            slots.append([])
        else:
            slots[reading_into].append(c)

    if slots[0] and len(slots[0]) > 1:
        logger.error("Can only specify a single stdout file (run-command substitutions are permitted)")
        return

    patterns = [re.compile("(--[^=]+=)(.*)"),
                re.compile("(-[^=]+=)(.*)"),
                re.compile("(-.)(.+)")]

    for command in slots[1:]:
        for i in xrange(0, len(command)):
            a = command[i]
            if a[0] == '-':
                # parameter starts with '-' so it might be a command line
                # parameter with a file name, do some pattern matching
                matched = False
                for p in patterns:
                    m = p.match(a)
                    if m:
                        command[i] = statfile(m.group(1), m.group(2))
                        break
            else:
                # parameter might be a file, so test it
                command[i] = statfile('', a)

    n = True
    pathprefix = "/"
    files = [c for command in slots[1:] for c in command if isinstance(c, UploadFile)]
    if len(files) > 0:
        # Find the smallest path prefix that includes all the files that need to be uploaded.
        # This starts at the root and iteratively removes common parent directory prefixes
        # until all file pathes no longer have a common parent.
        while n:
            pathstep = None
            for c in files:
                if pathstep is None:
                    sp = c.fn.split('/')
                    if len(sp) < 2:
                        # no parent directories left
                        n = False
                        break
                    # path step takes next directory
                    pathstep = sp[0] + "/"
                else:
                    # check if pathstep is common prefix for all files
                    if not c.fn.startswith(pathstep):
                        n = False
                        break
            if n:
                # pathstep is common parent directory for all files, so remove the prefix
                # from each path
                pathprefix += pathstep
                for c in files:
                    c.fn = c.fn[len(pathstep):]

        os.chdir(pathprefix)

        if args.dry_run:
            print("cd %s" % pathprefix)
            print("arv-put \"%s\"" % '" "'.join([c.fn for c in files]))
            pdh = "$(input)"
        else:
            pdh = put.main(["--portable-data-hash"]+[c.fn for c in files])

        for i in xrange(1, len(slots)):
            slots[i] = [("%s$(file %s/%s)" % (c.prefix, pdh, c.fn)) if isinstance(c, UploadFile) else c for c in slots[i]]

    component = {
        "script": "run-command",
        "script_version": "3609-arv-run",
        "repository": "arvados",
        "script_parameters": {
            "command": slots[2:]
        },
        "runtime_constraints": {
            "docker_image": args.docker_image
        }
    }

    task_foreach = []

    if slots[0]:
        component["script_parameters"]["task.stdout"] = slots[0][0]
    if slots[1]:
        task_foreach.append("stdin")
        component["script_parameters"]["stdin"] = slots[1]
        component["script_parameters"]["task.stdin"] = "$(stdin)"\

    if task_foreach:
        component["script_parameters"]["task.foreach"] = task_foreach

    pipeline = {
        "name": "",
        "components": {
            "command": component
        },
        "state":"RunningOnServer"
    }

    if args.dry_run:
        print(json.dumps(pipeline, indent=4))
    elif args.local:
        subprocess.call(["arv-crunch-job", "--job", json.dumps(component), "--git-dir", args.git_dir])
    else:
        api = arvados.api('v1')
        pi = api.pipeline_instances().create(body=pipeline).execute()
        #ws.main(["--pipeline", pi["uuid"]])

if __name__ == '__main__':
    main()
