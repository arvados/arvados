#!/usr/bin/env python

import arvados
import arvados.commands.ws as ws
import argparse
import json
import re
import os
import stat
import put
import time
import subprocess
import logging
import arvados.commands._util as arv_cmd

logger = logging.getLogger('arvados.arv-run')

arvrun_parser = argparse.ArgumentParser(parents=[arv_cmd.retry_opt])
arvrun_parser.add_argument('--dry-run', action="store_true", help="Print out the pipeline that would be submitted and exit")
arvrun_parser.add_argument('--local', action="store_true", help="Run locally using arv-run-pipeline-instance")
arvrun_parser.add_argument('--docker-image', type=str, default="arvados/jobs", help="Docker image to use, default arvados/jobs")
arvrun_parser.add_argument('--ignore-rcode', action="store_true", help="Commands that return non-zero return codes should not be considered failed.")
arvrun_parser.add_argument('--no-reuse', action="store_true", help="Do not reuse past jobs.")
arvrun_parser.add_argument('--no-wait', action="store_true", help="Do not wait and display logs after submitting command, just exit.")
arvrun_parser.add_argument('--project-uuid', type=str, help="Parent project of the pipeline")
arvrun_parser.add_argument('--git-dir', type=str, default="", help="Git repository passed to arv-crunch-job when using --local")
arvrun_parser.add_argument('--repository', type=str, default="arvados", help="repository field of component, default 'arvados'")
arvrun_parser.add_argument('--script-version', type=str, default="master", help="script_version field of component, default 'master'")
arvrun_parser.add_argument('args', nargs=argparse.REMAINDER)

class ArvFile(object):
    def __init__(self, prefix, fn):
        self.prefix = prefix
        self.fn = fn

class UploadFile(ArvFile):
    pass

# Determine if a file is in a collection, and return a tuple consisting of the
# portable data hash and the path relative to the root of the collection.
# Return None if the path isn't with an arv-mount collection or there was is error.
def is_in_collection(root, branch):
    try:
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
    except IOError, OSError:
        return (None, None)

# Determine the project to place the output of this command by searching upward
# for arv-mount psuedofile indicating the project.  If the cwd isn't within
# an arv-mount project or there is an error, return current_user.
def determine_project(root, current_user):
    try:
        if root == "/":
            return current_user
        fn = os.path.join(root, ".arvados#project")
        if os.path.exists(fn):
            with file(fn, 'r') as f:
                c = json.load(f)
            if 'writable_by' in c and current_user in c['writable_by']:
                return c["uuid"]
            else:
                return current_user
        else:
            sp = os.path.split(root)
            return determine_project(sp[0], current_user)
    except IOError, OSError:
        return current_user

# Determine if string corresponds to a file, and if that file is part of a
# arv-mounted collection or only local to the machine.  Returns one of
# ArvFile() (file already exists in a collection), UploadFile() (file needs to
# be uploaded to a collection), or simply returns prefix+fn (which yields the
# original parameter string).
def statfile(prefix, fn):
    absfn = os.path.abspath(fn)
    if os.path.exists(absfn):
        st = os.stat(absfn)
        if stat.S_ISREG(st.st_mode):
            sp = os.path.split(absfn)
            (pdh, branch) = is_in_collection(sp[0], sp[1])
            if pdh:
                return ArvFile(prefix, "$(file %s/%s)" % (pdh, branch))
            else:
                # trim leading '/' for path prefix test later
                return UploadFile(prefix, absfn[1:])
        if stat.S_ISDIR(st.st_mode):
            sp = os.path.split(absfn)
            (pdh, branch) = is_in_collection(sp[0], sp[1])
            if pdh:
                return ArvFile(prefix, "$(dir %s/%s/)" % (pdh, branch))

    return prefix+fn

def main(arguments=None):
    args = arvrun_parser.parse_args(arguments)

    if len(args.args) == 0:
        arvrun_parser.print_help()
        return

    starting_args = args.args

    reading_into = 2

    # Parse the command arguments into 'slots'.
    # All words following '>' are output arguments and are collected into slots[0].
    # All words following '<' are input arguments and are collected into slots[1].
    # slots[2..] store the parameters of each command in the pipeline.
    #
    # e.g. arv-run foo arg1 arg2 '|' bar arg3 arg4 '<' input1 input2 input3 '>' output.txt
    # will be parsed into:
    #   [['output.txt'],
    #    ['input1', 'input2', 'input3'],
    #    ['foo', 'arg1', 'arg2'],
    #    ['bar', 'arg3', 'arg4']]
    slots = [[], [], []]
    for c in args.args:
        if c.startswith('>'):
            reading_into = 0
            if len(c) > 1:
                slots[reading_into].append(c[1:])
        elif c.startswith('<'):
            reading_into = 1
            if len(c) > 1:
                slots[reading_into].append(c[1:])
        elif c == '|':
            reading_into = len(slots)
            slots.append([])
        else:
            slots[reading_into].append(c)

    if slots[0] and len(slots[0]) > 1:
        logger.error("Can only specify a single stdout file (run-command substitutions are permitted)")
        return

    if not args.dry_run:
        api = arvados.api('v1')
        if args.project_uuid:
            project = args.project_uuid
        else:
            project = determine_project(os.getcwd(), api.users().current().execute()["uuid"])

    # Identify input files.  Look at each parameter and test to see if there is
    # a file by that name.  This uses 'patterns' to look for within
    # command line arguments, such as --foo=file.txt or -lfile.txt
    patterns = [re.compile("([^=]+=)(.*)"),
                re.compile("(-[A-Za-z])(.+)")]
    for j, command in enumerate(slots[1:]):
        for i, a in enumerate(command):
            if j > 0 and i == 0:
                # j == 0 is stdin, j > 0 is commands
                # always skip program executable (i == 0) in commands
                pass
            elif a.startswith('\\'):
                # if it starts with a \ then don't do any interpretation
                command[i] = a[1:]
            else:
                # See if it looks like a file
                command[i] = statfile('', a)

                # If a file named command[i] was found, it would now be an
                # ArvFile or UploadFile.  If command[i] is a basestring, that
                # means it doesn't correspond exactly to a file, so do some
                # pattern matching.
                if isinstance(command[i], basestring):
                    for p in patterns:
                        m = p.match(a)
                        if m:
                            command[i] = statfile(m.group(1), m.group(2))
                            break

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

        orgdir = os.getcwd()
        os.chdir(pathprefix)

        print("Upload local files: \"%s\"" % '" "'.join([c.fn for c in files]))

        if args.dry_run:
            print("$(input) is %s" % pathprefix.rstrip('/'))
            pdh = "$(input)"
        else:
            files = sorted(files, key=lambda x: x.fn)
            collection = arvados.CollectionWriter(api, num_retries=args.retries)
            stream = None
            for f in files:
                sp = os.path.split(f.fn)
                if sp[0] != stream:
                    stream = sp[0]
                    collection.start_new_stream(stream)
                collection.write_file(f.fn, sp[1])
            item = api.collections().create(body={"owner_uuid": project, "manifest_text": collection.manifest_text()}).execute()
            pdh = item["portable_data_hash"]
            print "Uploaded to %s" % item["uuid"]

        for c in files:
            c.fn = "$(file %s/%s)" % (pdh, c.fn)

        os.chdir(orgdir)

    for i in xrange(1, len(slots)):
        slots[i] = [("%s%s" % (c.prefix, c.fn)) if isinstance(c, ArvFile) else c for c in slots[i]]

    component = {
        "script": "run-command",
        "script_version": args.script_version,
        "repository": args.repository,
        "script_parameters": {
        },
        "runtime_constraints": {
            "docker_image": args.docker_image
        }
    }

    task_foreach = []
    group_parser = argparse.ArgumentParser()
    group_parser.add_argument('-b', '--batch-size', type=int)
    group_parser.add_argument('args', nargs=argparse.REMAINDER)

    for s in xrange(2, len(slots)):
        for i in xrange(0, len(slots[s])):
            if slots[s][i] == '--':
                inp = "input%i" % (s-2)
                groupargs = group_parser.parse_args(slots[2][i+1:])
                if groupargs.batch_size:
                    component["script_parameters"][inp] = {"value": {"batch":groupargs.args, "size":groupargs.batch_size}}
                    slots[s] = slots[s][0:i] + [{"foreach": inp, "command": "$(%s)" % inp}]
                else:
                    component["script_parameters"][inp] = groupargs.args
                    slots[s] = slots[s][0:i] + ["$(%s)" % inp]
                task_foreach.append(inp)
                break
            if slots[s][i] == '\--':
                slots[s][i] = '--'

    if slots[0]:
        component["script_parameters"]["task.stdout"] = slots[0][0]
    if slots[1]:
        task_foreach.append("stdin")
        component["script_parameters"]["stdin"] = slots[1]
        component["script_parameters"]["task.stdin"] = "$(stdin)"

    if task_foreach:
        component["script_parameters"]["task.foreach"] = task_foreach

    component["script_parameters"]["command"] = slots[2:]
    if args.ignore_rcode:
        component["script_parameters"]["task.ignore_rcode"] = args.ignore_rcode

    pipeline = {
        "name": "arv-run " + " | ".join([s[0] for s in slots[2:]]),
        "description": "@" + " ".join(starting_args) + "@",
        "components": {
            "command": component
        },
        "state": "RunningOnClient" if args.local else "RunningOnServer"
    }

    if args.dry_run:
        print(json.dumps(pipeline, indent=4))
    else:
        pipeline["owner_uuid"] = project
        pi = api.pipeline_instances().create(body=pipeline, ensure_unique_name=True).execute()
        print "Running pipeline %s" % pi["uuid"]

        if args.local:
            subprocess.call(["arv-run-pipeline-instance", "--instance", pi["uuid"], "--run-jobs-here"] + (["--no-reuse"] if args.no_reuse else []))
        elif not args.no_wait:
            ws.main(["--pipeline", pi["uuid"]])

        pi = api.pipeline_instances().get(uuid=pi["uuid"]).execute()
        print "Pipeline is %s" % pi["state"]
        if "output_uuid" in pi["components"]["command"]:
            print "Output is %s" % pi["components"]["command"]["output_uuid"]
        else:
            print "No output"

if __name__ == '__main__':
    main()
