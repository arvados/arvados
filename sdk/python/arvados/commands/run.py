#!/usr/bin/env python

import arvados
import argparse
import json
import re
import os
import stat
import put
import arvados.events
import time

arvrun_parser = argparse.ArgumentParser()
arvrun_parser.add_argument('--dry-run', action="store_true")
arvrun_parser.add_argument('--docker-image', type=str, default="arvados/jobs")
arvrun_parser.add_argument('command')
arvrun_parser.add_argument('args', nargs=argparse.REMAINDER)

needupload_files = []

class ArvFile(object):
    def __init__(self, prefix, fn):
        self.prefix = prefix
        self.fn = fn

def statfile(prefix, fn, pattern):
    absfn = os.path.abspath(fn)
    if os.path.exists(absfn):
        fn = os.path.abspath(fn)
        st = os.stat(fn)
        if stat.S_ISREG(st.st_mode):
            mount = os.path.dirname(fn)+"/.arvados#collection"
            if os.path.exists(mount):
                with file(mount, 'r') as f:
                    c = json.load(f)
                return prefix+"$(file "+c["portable_data_hash"]+"/" + os.path.basename(fn) + ")"
            else:
                needupload_files.append(fn)
            return ArvFile(prefix, fn[1:])
    return prefix+fn

def main(arguments=None):
    args = arvrun_parser.parse_args(arguments)

    patterns = [re.compile("(--[^=]+=)(.*)"),
                re.compile("(-[^=]+=)(.*)"),
                re.compile("(-.)(.+)")]

    commandargs = []

    for a in args.args:
        if a[0] == '-':
            matched = False
            for p in patterns:
                m = p.match(a)
                if m:
                    commandargs.append(statfile(m.group(1), m.group(2), p))
                    matched = True
                    break
            if not matched:
                commandargs.append(a)
        else:
            commandargs.append(statfile('', a, None))

    n = True
    pathprefix = "/"
    files = [c for c in commandargs if isinstance(c, ArvFile)]
    if len(files) > 0:
        while n:
            pathstep = None
            for c in files:
                if pathstep is None:
                    sp = c.fn.split('/')
                    if len(sp) < 2:
                        n = False
                        break
                    pathstep = sp[0] + "/"
                else:
                    if not c.fn.startswith(pathstep):
                        n = False
                        break
            if n:
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

    commandargs = [("%s$(file %s/%s)" % (c.prefix, pdh, c.fn)) if isinstance(c, ArvFile) else c for c in commandargs]

    cut = None
    i = -1
    stdio = [None, None]
    for j in xrange(0, len(commandargs)):
        c = commandargs[j]
        if c == '<':
            stdio[0] = []
            i = 0
            cut = j if cut is None else cut
        elif c == '>':
            stdio[1] = []
            i = 1
            cut = j if cut is None else cut
        elif i > -1:
            stdio[i].append(c)

    if cut is not None:
        commandargs = commandargs[:cut]

    component = {
        "script": "run-command",
        "script_version": "bf243e064a7a2ee4e69a87dc3ba46e949a545150",
        "repository": "arvados",
        "script_parameters": {
            "command": [args.command]+commandargs
        },
        "runtime_constraints": {
            "docker_image": args.docker_image
        }
    }

    if stdio[0]:
        component["script_parameters"]["task.stdin"] = stdio[0][0]
    if stdio[1]:
        component["script_parameters"]["task.stdout"] = stdio[1][0]

    pipeline = {
        "name": "",
        "components": {
            args.command: component
        },
        "state":"RunningOnServer"
    }

    if args.dry_run:
        print(json.dumps(pipeline, indent=4))
    else:
        api = arvados.api('v1')
        pi = api.pipeline_instances().create(body=pipeline).execute()
        ws = None
        def report(x):
            if "event_type" in x:
                print "\n"
                print x
                if x["event_type"] == "stderr":
                    print x["properties"]["text"]
                elif x["event_type"] == "update" and x["properties"]["new_attributes"]["state"] in ["Complete", "Failed"]:
                    ws.close_connection()

        ws =  arvados.events.subscribe(api, [["object_uuid", "=", pi["uuid"]], ["event_type", "in", ["stderr", "update"]]], report)
        ws.run_forever()

if __name__ == '__main__':
    main()
