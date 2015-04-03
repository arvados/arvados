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
import arvados.events
import threading

EX_TEMPFAIL = 75

def parse_sinfo(sinfo):
    nodes = {}
    for l in sinfo.splitlines():
        m = re.match("(\d+) (.*)", l)
        if m:
            cpus = int(m.group(1))
            for n in m.group(2).split(","):
                rn = re.match("([^[]+)\[(\d+)-(\d+)\]", n)
                if rn:
                    for c in range(int(rn.group(2)), int(rn.group(3))+1):
                        nodes["%s%i" % (rn.group(1), c)] = {"slots": cpus}
                else:
                    nodes[n] = {"slots": cpus}
    return nodes

def make_slots(nodes):
    slots = {}
    for n in nodes:
        for c in range(0, nodes[n]["slots"]):
            slots["%s[%i]" % (n, c)] = {"node": n, "slot": c, "task": None}
    return slots

def determine_resources(slurm_jobid=None, slurm_nodelist=None):
    have_slurm = (slurm_jobid is not None) and (slurm_nodelist is not None)

    if have_slurm:
        sinfo = subprocess.check_output(["sinfo", "-h", "--format=%c %N", "--nodes=" + slurm_nodelist])
    else:
        with open("/proc/cpuinfo") as cpuinfo:
            n = 0
            for p in cpuinfo:
                if p.startswith("processor"):
                    n += 1
        sinfo = "%i localhost" % n

    nodes = parse_sinfo(sinfo)
    slots = make_slots(nodes)

    return {"have_slurm": have_slurm,
            "nodes": nodes,
            "slots": slots}

def run_on_slot(resources, slot, task):
    tmpdir = "/tmp/%s-%i" % (slot, random.randint(1, 100000))

    execution_script = Template("""
if ! docker images -q --no-trunc --all | grep -qxF $docker_hash ; then
    arv-get $docker_locator/$docker_hash.tar | docker load
fi
rm -rf $tmpdir
mkdir -p $tmpdir/job_output $tmpdir/keep
if which crunchstat ; then
  CRUNCHSTAT="crunchstat -cgroup-root=/sys/fs/cgroup -cgroup-parent=docker -cgroup-cid=$tmpdir/cidfile -poll=10000"
else
  CRUNCHSTAT=""
fi

exec  \
  arv-mount --by-id $tmpdir/keep --allow-other --exec \
  $$CRUNCHSTAT \
  docker run --attach=stdout --attach=stderr -i --rm --cidfile=$tmpdir/cidfile --sig-proxy \
  --volume=$tmpdir/keep:/keep:ro --volume=$tmpdir/job_output:/tmp/job_output:rw \
  --workdir=/tmp/job_output --user=$$UID $env $docker_hash $cmd
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

    if resources["have_slurm"]:
        pass
    else:
        resources["slots"][slot]["task"] = subprocess.Popen(ex, shell=True)
        resources["slots"][slot]["task"].wait()

class TaskEvents(object):
    def __init__(self, api_config, resources, job_uuid):
        self.resources = resources
        self.ws = arvados.events.subscribe(api_from_config("v1", api_config), [["object_uuid", "=", job_uuid]], self.on_event)
        self.ws.subscribe([["object_uuid", "is_a", "arvados#jobTask"]])

    def on_event(self, ev):
        if ev.get('event_type') == "update" and ev['object_kind'] == "arvados#job":
            if ev["properties"]["new_attributes"]["state"] in ("Complete", "Failed", "Cancelled"):
                self.ws.close()
        elif ev.get('object_kind') == "arvados#jobTask":
            if ev.get('event_type') == "create":
                pass
            if ev.get('event_type') == "update":
                pass

def main(arguments=None):

    parser = argparse.ArgumentParser()

    parser.add_argument("--job-api-token", type=str)
    parser.add_argument("--job", type=str)
    parser.add_argument("--git-dir", type=str)

    print sys.argv
    args = parser.parse_args(arguments)

    api = None
    if os.environ.get("ARVADOS_API_HOST"):
        api_config = {"ARVADOS_API_HOST": os.environ["ARVADOS_API_HOST"],
                      "ARVADOS_API_TOKEN": args.job_api_token,
                      "ARVADOS_API_HOST_INSECURE": os.environ.get("ARVADOS_API_HOST_INSECURE")}
        api = api_from_config("v1", api_config)

    job_uuid = None
    if arvados.util.job_uuid_pattern.match(args.job):
        if not api:
            print "Missing ARVADOS_API_HOST"
            return 1
        try:
            job_uuid = args.job
            ex = api.jobs().lock(uuid=args.job).execute()
            if "error" in ex:
                return EX_TEMPFAIL
        except:
            return EX_TEMPFAIL

        job = api.jobs().get(uuid=args.job).execute()

        if job["state"] in ("Complete", "Failed", "Cancelled"):
            print "Job is already %s" % (job["state"])
            return EX_TEMPFAIL
    else:
        job = json.loads(args.job)

    resources = determine_resources(os.environ.get("SLURM_JOBID"),
                                    os.environ.get("SLURM_NODELIST"))

    if job_uuid:
        ts = TaskEvents(api_config, resources, job_uuid)
        ts.ws.run_forever()
    else:
        run_on_slot(resources, resources["slots"].keys()[0], job["script_parameters"])

    return 0
