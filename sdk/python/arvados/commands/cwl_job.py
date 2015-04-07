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
import signal
import arvados.commands.keepdocker
import tempfile
import pprint

EX_TEMPFAIL = 75
TASK_TEMPFAIL = 111
TASK_CANCELED = 112

def parse_ranges(l):
    r = []
    for n in l.split(","):
        rn = re.match("([^[]+)\[(\d+)-(\d+)\]", n)
        if rn:
            for c in range(int(rn.group(2)), int(rn.group(3))+1):
                yield "%s%i" % (rn.group(1), c)
        else:
            yield n

def parse_sinfo(sinfo):
    nodes = {}
    for l in sinfo.splitlines():
        m = re.match("(\d+) (.*)", l)
        if m:
            cpus = int(m.group(1))
            for n in parse_ranges(m.group(2)):
                nodes[n] = {"slots": cpus}
    return nodes

def make_slots(nodes):
    slots = {}
    for n in nodes:
        for c in range(0, nodes[n]["slots"]):
            slots["%s[%i]" % (n, c)] = {"node": n, "slot": c, "task": None}
    return slots

script_header = """
set -e
set -v

read pid cmd state ppid pgrp session tty_nr tpgid rest < /proc/self/stat
trap "kill -TERM -$$pgrp; exit $TASK_CANCELED" INT QUIT TERM

export ARVADOS_API_HOST=$ARVADOS_API_HOST
export ARVADOS_API_TOKEN=$ARVADOS_API_TOKEN
export ARVADOS_API_HOST_INSECURE=$ARVADOS_API_HOST_INSECURE

arv-keepdocker --download $docker_hash

rm -rf $tmpdir
mkdir -p $tmpdir

DNS="$$(ip -o address show scope global | gawk 'match($$4, /^([0-9\.:]+)\//, x){print "--dns", x[1]}')"
"""

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

class TaskMonitor(threading.Thread):
    def __init__(self, sub, api_config, task):
        super(TaskMonitor, self).__init__()
        self.sub = sub
        self.api_config = api_config
        self.task = task

    def run(self):
        self.sub.wait()
        api = api_from_config("v1", self.api_config)
        check = api.job_tasks().get(uuid=self.task["uuid"]).execute()
        if check["success"] is None:
            # Task didn't set its own success, so mark it failed.
            check = api.job_tasks().update(uuid=self.task["uuid"], body={"success": False}).execute()
        print "Task %s completed with success %s" % (self.task["uuid"], check["success"])

def run_on_slot(resources, api_config, slot, task):
    execution_script = Template(script_header + """
mkdir $tmpdir/job_output $tmpdir/keep

if which crunchstat ; then
  CRUNCHSTAT="crunchstat -cgroup-root=/sys/fs/cgroup -cgroup-parent=docker -cgroup-cid=$cidfile -poll=10000"
else
  CRUNCHSTAT=""
fi

set +e

arv-mount --by-id $tmpdir/keep --allow-other --exec \
  $$CRUNCHSTAT \
  docker run \
    $$DNS \
    --attach=stdout \
    --attach=stderr \
    -i \
    --rm \
    --cidfile=$cidfile \
    --sig-proxy \
    --volume=$tmpdir/keep:/keep:ro \
    --volume=$tmpdir/job_output:/tmp/job_output:rw \
    --workdir=/tmp/job_output \
    --user=$$UID \
    $env \
    $docker_hash \
    $cmd \
    $stdin_redirect \
    $stdout_redirect

code=$$?

echo "Docker exit code is $$code"

OUT=`arv-put --portable-data-hash --batch-progress $tmpdir/job_output`
arv_put_code=$$?

echo "arv-put exit code is $$arv_put_code"
echo "Output is $$OUT"

if test -n "$task_uuid" ; then
  if test "$$code" = "0" ; then
    if test "$$arv_put_code" = "0" ; then
      echo "Task success"
      arv job_task update --uuid $task_uuid --job-task "{\\"output\\":\\"$$OUT\\", \\"success\\": true}"
    else
      echo "Task temporary failure"
      code=$TASK_TEMPFAIL
    fi
  else
    echo "Task failed"
    arv job_task update --uuid $task_uuid --job-task "{\\"output\\":\\"$$OUT\\", \\"success\\": false}"
  fi
fi

rm -rf $tmpdir

exit $$code
""")

    pprint.pprint(task)

    tmpdir = "/tmp/%s-%i" % (slot, random.randint(1, 100000))

    params = task["parameters"]

    env = " ".join(["--env=%s=%s" % (pipes.quote(e), pipes.quote(params["environment"][e])) for e in params.get("environment", {})])

    stdin_redirect=""
    stdout_redirect=""

    if params.get("stdin"):
        stdin_redirect = "< %s/keep/%s" % (pipes.quote(tmpdir), pipes.quote(params["stdin"]))

    if params.get("stdout"):
        stdout_redirect = "> %s/job_output/%s" % (pipes.quote(tmpdir), pipes.quote(params["stdout"]))

    ex = execution_script.substitute(docker_hash=pipes.quote(params["docker_hash"]),
                                     tmpdir=pipes.quote(tmpdir),
                                     cidfile=os.path.join(tmpdir, "cidfile"),
                                     env=env,
                                     cmd=" ".join([pipes.quote(c) for c in params["command"]]),
                                     task_uuid=pipes.quote(task["uuid"]),
                                     stdin_redirect=stdin_redirect,
                                     stdout_redirect=stdout_redirect,
                                     TASK_CANCELED=TASK_CANCELED,
                                     TASK_TEMPFAIL=TASK_TEMPFAIL,
                                     ARVADOS_API_HOST=pipes.quote(api_config["ARVADOS_API_HOST"]),
                                     ARVADOS_API_TOKEN=pipes.quote(api_config["ARVADOS_API_TOKEN"]),
                                     ARVADOS_API_HOST_INSECURE=pipes.quote(api_config.get("ARVADOS_API_HOST_INSECURE", "0")))


    if resources["have_slurm"]:
        pass
    else:
        slots = resources["slots"]
        slots[slot]["task"] = task
        sub = subprocess.Popen(ex, shell=True, stdin=subprocess.PIPE, close_fds=True)
        sub.stdin.close()
        slots[slot]["task"]["__subprocess"] = sub
        TaskMonitor(sub, api_config, task).start()

class TaskEvents(object):
    def __init__(self, api_config, resources, job_uuid):
        self.resources = resources
        self.slots = resources["slots"]
        self.ws = arvados.events.subscribe(api_from_config("v1", api_config), [["object_uuid", "=", job_uuid]], self.on_event)
        self.ws.subscribe([["object_uuid", "is_a", "arvados#jobTask"]])
        self.task_queue = []
        self.api_config = api_config

    def next_task(self):
        while self.task_queue:
            assigned = False
            for slot in self.slots:
                if self.slots[slot]["task"] is None:
                    task = self.task_queue[0]
                    try:
                        assigned = True
                        run_on_slot(self.resources, self.api_config, slot, task)
                        del self.task_queue[0]
                    except:
                        api = api_from_config("v1", self.api_config)
                        check = api.job_tasks().get(uuid=task["uuid"]).execute()
                        if check["success"] is None:
                            api.job_tasks().update(uuid=task["uuid"], body={"success": False}).execute()
                    break

            if not assigned:
                break

    def new_task(self, task):
        self.task_queue.append(task)
        self.next_task()

    def finish_task(self, task):
        for slot in self.slots:
            if self.slots[slot]["task"] and self.slots[slot]["task"]["uuid"] == task["uuid"]:
                self.slots[slot]["task"] = None
        self.next_task()

    def cancel_tasks(self):
        for slot in self.slots:
            if self.slots[slot]["task"] and self.slots[slot]["task"].get("__subprocess"):
                try:
                    self.slots[slot]["task"]["__subprocess"].terminate()
                    self.slots[slot]["task"]["__subprocess"].wait()
                except OSError:
                    pass

    def on_event(self, ev):
        if ev.get('event_type') == "update" and ev['object_kind'] == "arvados#job":
            if ev["properties"]["new_attributes"]["state"] == "Cancelled":
                self.cancel_tasks()
                self.ws.close()
            elif ev["properties"]["new_attributes"]["state"] in ("Complete", "Failed"):
                self.ws.close()
        elif ev.get('object_kind') == "arvados#jobTask":
            if ev.get('event_type') == "create":
                self.new_task(ev["properties"]["new_attributes"])
            if ev.get('event_type') == "update":
                if ev["properties"]["new_attributes"].get("success") is not None:
                    self.finish_task(ev["properties"]["new_attributes"])

class JobMonitor(threading.Thread):
    def __init__(self, sub, api_config, job):
        super(JobMonitor, self).__init__()
        self.sub = sub
        self.api_config = api_config
        self.job = job

    def run(self):
        self.sub.wait()
        api = api_from_config("v1", self.api_config)
        api.jobs().update(uuid=self.job["uuid"], body={"state":"Complete" if self.sub.returncode == 0 else "Failed"}).execute()


def run_executive(resources, job, api_config):
    execution_script = Template(script_header + """
cd $tmpdir
git init
git config --local credential.$githttp/.helper '!tok(){ echo password=$ARVADOS_API_TOKEN; };tok'
git config --local credential.$githttp/.username none
git fetch --quiet $githttp/$gitrepo
git checkout --quiet $script_version

docker run \
    $$DNS \
    --env=JOB_UUID=$job_uuid \
    --env=ARVADOS_API_HOST=$ARVADOS_API_HOST \
    --env=ARVADOS_API_TOKEN=$ARVADOS_API_TOKEN \
    --env=ARVADOS_API_HOST_INSECURE=$ARVADOS_API_HOST_INSECURE \
    --volume=$tmpdir:/tmp/git:ro \
    --privileged \
    --user=$$UID \
    --rm \
    --workdir=/tmp/git \
    $docker_hash /tmp/git/$script
""")

    tmpdir = "/tmp/%s-%i" % (job["uuid"], random.randint(1, 100000))

    api = api_from_config("v1", api_config)
    docker_hash = arvados.commands.keepdocker.image_hash_in_collection(arvados.CollectionReader(job["docker_image_locator"], api_client=api))

    repo = job["repository"]
    if not repo.endswith(".git"):
        repo += "/.git"

    ex = execution_script.substitute(docker_hash=pipes.quote(docker_hash),
                                     docker_locator=pipes.quote(job["docker_image_locator"]),
                                     tmpdir=tmpdir,
                                     ARVADOS_API_HOST=pipes.quote(api_config["ARVADOS_API_HOST"]),
                                     ARVADOS_API_TOKEN=pipes.quote(api_config["ARVADOS_API_TOKEN"]),
                                     ARVADOS_API_HOST_INSECURE=pipes.quote(api_config.get("ARVADOS_API_HOST_INSECURE", "0")),
                                     TASK_CANCELED=TASK_CANCELED,
                                     githttp=pipes.quote(api._rootDesc.get("gitHttpBase")),
                                     gitrepo=pipes.quote(repo),
                                     script=pipes.quote(job["script"]),
                                     script_version=pipes.quote(job["script_version"]),
                                     job_uuid=job["uuid"])

    print ex

    if resources["have_slurm"]:
        pass
    else:
        sub = subprocess.Popen(ex, shell=True, stdin=subprocess.PIPE, close_fds=True)
        sub.stdin.close()
        JobMonitor(sub, api_config, job).start()
        return sub


class SigHandler(object):
    def __init__(self, sub, ts):
        self.sub = sub
        self.ts = ts

    def send_signal(self, signum, fram):
        try:
            self.sub.terminate()
        except OSError:
            pass
        self.ts.cancel_tasks()
        self.ts.ws.close()

def main(arguments=None):

    parser = argparse.ArgumentParser()

    parser.add_argument("--job-api-token", type=str)
    parser.add_argument("--job", type=str)
    parser.add_argument("--git-dir", type=str)

    args = parser.parse_args(arguments)

    api = None
    if os.environ.get("ARVADOS_API_HOST"):
        api_config = {"ARVADOS_API_HOST": os.environ["ARVADOS_API_HOST"],
                      "ARVADOS_API_TOKEN": args.job_api_token,
                      "ARVADOS_API_HOST_INSECURE": os.environ.get("ARVADOS_API_HOST_INSECURE", "0")}
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
        sub = run_executive(resources, job, api_config)

        # Set up signal handling
        sig = SigHandler(sub, ts)

        # Forward terminate signals to the subprocesses.
        signal.signal(signal.SIGINT, sig.send_signal)
        signal.signal(signal.SIGTERM, sig.send_signal)
        signal.signal(signal.SIGQUIT, sig.send_signal)

        ts.ws.run_forever()
    else:
        run_on_slot(resources, api_config, resources["slots"].keys()[0], {"parameters": job["script_parameters"]})

    return 0
