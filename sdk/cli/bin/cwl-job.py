#!/usr/bin/env python

import arvados
import argparse
import subprocess
import re
import json
import random
from string import Template
import pipes
import sys
from arvados.api import api_from_config
import os
import pprint

EX_TEMPFAIL = 75

def determine_resources():
    have_slurm = (os.environ.get("SLURM_JOBID", False) and os.environ.get("SLURM_NODELIST", False)) != False

    if have_slurm:
        sinfo = subprocess.check_output(["sinfo", "-h", "--format=%c %N", "--nodes=" + os.environ["SLURM_NODELIST"]])
    else:
        with open("/proc/cpuinfo") as cpuinfo:
            n = 0
            for p in cpuinfo:
                if p.startswith("processor"):
                    n += 1
        sinfo = "%i localhost" % n

    nodes = {}
    for l in sinfo.splitlines():
        m = re.match("(\d+) (.*)", l)
        cpus = int(m.group(1))
        for n in m.group(2).split(","):
            rn = re.match("([^[]+)\[(\d+)-(\d+)\]", n)
            if rn:
                for c in range(int(rn.group(2)), int(rn.group(3)+1)):
                    nodes["%s%i" % (rn.group(1), c)] = {"slots": cpus}
            else:
                nodes[m.group(2)] = {"slots": cpus}

    slots = {}
    for n in nodes:
        for c in range(0, nodes[n]["slots"]):
            slots["%s[%i]" % (n, c)] = {"node": n, "slot": c, "task": None}

    return {"have_slurm": have_slurm,
            "nodes": nodes,
            "slots": slots}

def run_on_slot(have_slurm, slot, task):
    tmpdir = "/tmp/%s-%i" % (slot, random.randint(1, 100000))

    execution_script = Template("""
if ! docker images -q --no-trunc --all | grep -qxF $docker_hash ; then
    arv-get $docker_locator/$docker_locator.tar | docker load
fi
rm -rf $tmpdir
mkdir -p $tmpdir/job_output
exec stdbuf --output=0 --error=0 \
  arv-mount --by-id $tmpdir/keep --exec \
  crunchstat -cgroup-root=/sys/fs/cgroup -cgroup-parent=docker -cgroup-cid=$tmpdir/cidfile -poll=10000 \
  docker --attach=stdout --attach=stderr -i --rm --cidfile=$tmpdir/cidfile --sig-proxy \
  --volume=$tmpdir/keep:/keep:ro --volume=$tmpdir/job_output:/tmp/job_output:rw \
  --workdir=/tmp/job_output --user=$$UID $env $cmd
""")
    env = ""
    for e in task["environment"]:
        env += " --env=%s=%s" % (e, task["environment"][e])

    ex = execution_script.substitute(docker_hash=task["docker_hash"],
                                docker_locator=task["docker_locator"],
                                tmpdir=tmpdir,
                                env=env,
                                cmd=" ".join([pipes.quote(c) for c in task["command"]]))

    print ex

    if have_slurm:
        pass
    else:
        resources["slots"][slot]["task"] = subprocess.Popen(ex, shell=True)


def main():

    parser = argparse.ArgumentParser()

    parser.add_argument("--job-api-token", type=str)
    parser.add_argument("--job", type=str)
    parser.add_argument("--git-dir", type=str)

    args = parser.parse_args()

    api = api_from_config("v1",
                          {"ARVADOS_API_HOST": os.environ["ARVADOS_API_HOST"],
                           "ARVADOS_API_TOKEN": args.job_api_token,
                           "ARVADOS_API_HOST_INSECURE": os.environ.get("ARVADOS_API_HOST_INSECURE")})

    if arvados.util.job_uuid_pattern.match(args.job):
        try:
            ex = api.jobs().lock(uuid=args.job).execute()
            if "error" in ex:
                sys.exit(EX_TEMPFAIL)
        except:
            sys.exit(EX_TEMPFAIL)

        job = api.jobs().get(args.job)
    else:
        job = json.loads(args.job)

    resources = determine_resources()

    pprint.pprint(resources)

    for t in job["script_parameters"]["tasks"]:
        for s in resources["slots"]:
            if resources["slots"][s]["task"] is None:
                run_on_slot(resources, s, t)
                break

if __name__ == "__main__":
    sys.exit(main())
