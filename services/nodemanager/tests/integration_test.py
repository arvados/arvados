#!/usr/bin/env python
"""Integration test framework for node manager.

Runs full node manager with an API server (needs ARVADOS_API_HOST and
ARVADOS_API_TOKEN).  Stubs out the cloud driver and slurm commands to mock
specific behaviors.  Monitors the log output to verify an expected sequence of
events or behaviors for each test.

"""

import subprocess
import os
import sys
import re
import time
import logging
import stat
import tempfile
import shutil
from functools import partial
import arvados
import StringIO

logger = logging.getLogger("logger")
logger.setLevel(logging.INFO)
logger.addHandler(logging.StreamHandler(sys.stderr))

detail = logging.getLogger("detail")
detail.setLevel(logging.INFO)
if os.environ.get("ANMTEST_LOGLEVEL"):
    detail_content = sys.stderr
else:
    detail_content = StringIO.StringIO()
detail.addHandler(logging.StreamHandler(detail_content))

fake_slurm = None
compute_nodes = None
all_jobs = None

def update_script(path, val):
    with open(path+"_", "w") as f:
        f.write(val)
    os.chmod(path+"_", stat.S_IRUSR | stat.S_IWUSR | stat.S_IXUSR)
    os.rename(path+"_", path)
    detail.info("Update script %s: %s", path, val)

def set_squeue(g):
    global all_jobs
    update_script(os.path.join(fake_slurm, "squeue"), "#!/bin/sh\n" +
                  "\n".join("echo '1|100|100|%s|%s'" % (v, k) for k,v in all_jobs.items()))
    return 0


def node_paired(g):
    global compute_nodes
    compute_nodes[g.group(1)] = g.group(3)

    update_script(os.path.join(fake_slurm, "sinfo"), "#!/bin/sh\n" +
                  "\n".join("echo '%s alloc'" % (v) for k,v in compute_nodes.items()))

    for k,v in all_jobs.items():
        if v == "ReqNodeNotAvail":
            all_jobs[k] = "Running"
            break

    set_squeue(g)

    return 0

def remaining_jobs(g):
    update_script(os.path.join(fake_slurm, "sinfo"), "#!/bin/sh\n" +
                  "\n".join("echo '%s alloc'" % (v) for k,v in compute_nodes.items()))

    for k,v in all_jobs.items():
        all_jobs[k] = "Running"

    set_squeue(g)

    return 0


def node_busy(g):
    update_script(os.path.join(fake_slurm, "sinfo"), "#!/bin/sh\n" +
                  "\n".join("echo '%s idle'" % (v) for k,v in compute_nodes.items()))
    return 0

def node_shutdown(g):
    global compute_nodes
    del compute_nodes[g.group(1)]
    return 0

def jobs_req(g):
    global all_jobs
    for k,v in all_jobs.items():
        all_jobs[k] = "ReqNodeNotAvail"
    set_squeue(g)
    return 0

def noop(g):
    return 0

def fail(checks, pattern, g):
    return 1

def expect_count(count, checks, pattern, g):
    if count == 0:
        return 1
    else:
        checks[pattern] = partial(expect_count, count-1)
        return 0

def run_test(name, actions, checks, driver_class, jobs):
    code = 0

    # Delete any stale node records
    api = arvados.api('v1')
    for n in api.nodes().list().execute()['items']:
        api.nodes().delete(uuid=n["uuid"]).execute()

    logger.info("Start %s", name)

    global fake_slurm
    fake_slurm = tempfile.mkdtemp()
    detail.info("fake_slurm is %s", fake_slurm)

    global compute_nodes
    compute_nodes = {}

    global all_jobs
    all_jobs = jobs

    env = os.environ.copy()
    env["PATH"] = fake_slurm + ":" + env["PATH"]

    # Reset fake squeue/sinfo to empty
    update_script(os.path.join(fake_slurm, "squeue"), "#!/bin/sh\n")
    update_script(os.path.join(fake_slurm, "sinfo"), "#!/bin/sh\n")

    # Write configuration file for test
    with open("tests/fake.cfg.template") as f:
        open(os.path.join(fake_slurm, "id_rsa.pub"), "w").close()
        with open(os.path.join(fake_slurm, "fake.cfg"), "w") as cfg:
            cfg.write(f.read().format(host=os.environ["ARVADOS_API_HOST"],
                                      token=os.environ["ARVADOS_API_TOKEN"],
                                      driver_class=driver_class,
                                      ssh_key=os.path.join(fake_slurm, "id_rsa.pub")))

    # Tests must complete in less than 3 minutes.
    timeout = time.time() + 180
    terminated = False

    # Now start node manager
    p = subprocess.Popen(["bin/arvados-node-manager", "--foreground", "--config", os.path.join(fake_slurm, "fake.cfg")],
                         bufsize=0, stderr=subprocess.PIPE, env=env)

    # Test main loop:
    # - Read line
    # - Apply negative checks (thinks that are not supposed to happen)
    # - Check timeout
    # - Check if the next action should trigger
    # - If all actions are exhausted, terminate with test success
    # - If it hits timeout with actions remaining, terminate with test failed
    try:
        # naive line iteration over pipes gets buffered, which isn't what we want,
        # see https://bugs.python.org/issue3907
        for line in iter(p.stderr.readline, ""):
            detail_content.write(line)

            for k,v in checks.items():
                g = re.match(k, line)
                if g:
                    detail.info("Matched check %s", k)
                    code += v(checks, k, g)
                    if code != 0:
                        detail.error("Check failed")
                        if not terminated:
                            p.terminate()
                            terminated = True

            if terminated:
                continue

            if time.time() > timeout:
                detail.error("Exceeded timeout with actions remaining: %s", actions)
                code += 1
                if not terminated:
                    p.terminate()
                    terminated = True

            k, v = actions[0]
            g = re.match(k, line)
            if g:
                detail.info("Matched action %s", k)
                actions.pop(0)
                code += v(g)
                if code != 0:
                    detail.error("Action failed")
                    p.terminate()
                    terminated = True

            if not actions:
                p.terminate()
                terminated = True
    except KeyboardInterrupt:
        p.kill()

    if actions:
        detail.error("Ended with remaining actions: %s", actions)
        code = 1

    shutil.rmtree(fake_slurm)

    if code == 0:
        logger.info("%s passed", name)
    else:
        if isinstance(detail_content, StringIO.StringIO):
            sys.stderr.write(detail_content.getvalue())
        logger.info("%s failed", name)

    return code


def main():
    # Test lifecycle.

    tests = {
        "test_single_node": (
            [
                (r".*Daemon started", set_squeue),
                (r".*Cloud node (\S+) is now paired with Arvados node (\S+) with hostname (\S+)", node_paired),
                (r".*ComputeNodeMonitorActor\..*\.([^[]*).*Not eligible for shut down because node state is \('busy', 'open', .*\)", node_busy),
                (r".*ComputeNodeMonitorActor\..*\.([^[]*).*Suggesting shutdown because node state is \('idle', 'open', .*\)", noop),
                (r".*ComputeNodeShutdownActor\..*\.([^[]*).*Shutdown success", node_shutdown),
            ], {
                r".*Suggesting shutdown because node state is \('down', .*\)": fail,
                r".*Cloud node (\S+) is now paired with Arvados node (\S+) with hostname (\S+)": partial(expect_count, 1),
                r".*Setting node quota.*": fail,
            },
            "arvnodeman.test.fake_driver.FakeDriver",
            {"34t0i-dz642-h42bg3hq4bdfpf9": "ReqNodeNotAvail"}),
        "test_multiple_nodes": (
            [
                (r".*Daemon started", set_squeue),
                (r".*Cloud node (\S+) is now paired with Arvados node (\S+) with hostname (\S+)", node_paired),
                (r".*Cloud node (\S+) is now paired with Arvados node (\S+) with hostname (\S+)", node_paired),
                (r".*Cloud node (\S+) is now paired with Arvados node (\S+) with hostname (\S+)", node_paired),
                (r".*Cloud node (\S+) is now paired with Arvados node (\S+) with hostname (\S+)", node_paired),
                (r".*ComputeNodeMonitorActor\..*\.([^[]*).*Not eligible for shut down because node state is \('busy', 'open', .*\)", node_busy),
                (r".*ComputeNodeMonitorActor\..*\.([^[]*).*Suggesting shutdown because node state is \('idle', 'open', .*\)", noop),
                (r".*ComputeNodeShutdownActor\..*\.([^[]*).*Shutdown success", node_shutdown),
                (r".*ComputeNodeShutdownActor\..*\.([^[]*).*Shutdown success", node_shutdown),
                (r".*ComputeNodeShutdownActor\..*\.([^[]*).*Shutdown success", node_shutdown),
                (r".*ComputeNodeShutdownActor\..*\.([^[]*).*Shutdown success", node_shutdown),
            ], {
                r".*Suggesting shutdown because node state is \('down', .*\)": fail,
                r".*Cloud node (\S+) is now paired with Arvados node (\S+) with hostname (\S+)": partial(expect_count, 4),
                r".*Setting node quota.*": fail,
            },
            "arvnodeman.test.fake_driver.FakeDriver",
            {"34t0i-dz642-h42bg3hq4bdfpf1": "ReqNodeNotAvail",
             "34t0i-dz642-h42bg3hq4bdfpf2": "ReqNodeNotAvail",
             "34t0i-dz642-h42bg3hq4bdfpf3": "ReqNodeNotAvail",
             "34t0i-dz642-h42bg3hq4bdfpf4": "ReqNodeNotAvail"
         }),
        "test_hit_quota": (
            [
                (r".*Daemon started", set_squeue),
                (r".*Cloud node (\S+) is now paired with Arvados node (\S+) with hostname (\S+)", node_paired),
                (r".*Cloud node (\S+) is now paired with Arvados node (\S+) with hostname (\S+)", node_paired),
                (r".*ComputeNodeMonitorActor\..*\.([^[]*).*Not eligible for shut down because node state is \('busy', 'open', .*\)", node_busy),
                (r".*ComputeNodeMonitorActor\..*\.([^[]*).*Suggesting shutdown because node state is \('idle', 'open', .*\)", remaining_jobs),
                (r".*ComputeNodeMonitorActor\..*\.([^[]*).*Not eligible for shut down because node state is \('busy', 'open', .*\)", node_busy),
                (r".*ComputeNodeShutdownActor\..*\.([^[]*).*Shutdown success", node_shutdown),
                (r".*ComputeNodeShutdownActor\..*\.([^[]*).*Shutdown success", node_shutdown)
            ], {
                r".*Suggesting shutdown because node state is \('down', .*\)": fail,
                r".*Cloud node (\S+) is now paired with Arvados node (\S+) with hostname (\S+)": partial(expect_count, 2),
                r".*Sending create_node request.*": partial(expect_count, 5)
            },
            "arvnodeman.test.fake_driver.QuotaDriver",
            {"34t0i-dz642-h42bg3hq4bdfpf1": "ReqNodeNotAvail",
             "34t0i-dz642-h42bg3hq4bdfpf2": "ReqNodeNotAvail",
             "34t0i-dz642-h42bg3hq4bdfpf3": "ReqNodeNotAvail",
             "34t0i-dz642-h42bg3hq4bdfpf4": "ReqNodeNotAvail"
         }),
        "test_probe_quota": (
            [
                (r".*Daemon started", set_squeue),
                (r".*Cloud node (\S+) is now paired with Arvados node (\S+) with hostname (\S+)", node_paired),
                (r".*Cloud node (\S+) is now paired with Arvados node (\S+) with hostname (\S+)", node_paired),
                (r".*ComputeNodeMonitorActor\..*\.([^[]*).*Not eligible for shut down because node state is \('busy', 'open', .*\)", node_busy),
                (r".*ComputeNodeMonitorActor\..*\.([^[]*).*Suggesting shutdown because node state is \('idle', 'open', .*\)", remaining_jobs),
                (r".*ComputeNodeMonitorActor\..*\.([^[]*).*Not eligible for shut down because node state is \('busy', 'open', .*\)", node_busy),
                (r".*ComputeNodeShutdownActor\..*\.([^[]*).*Shutdown success", node_shutdown),
                (r".*ComputeNodeShutdownActor\..*\.([^[]*).*Shutdown success", node_shutdown),
                (r".*sending request", jobs_req),
                (r".*Cloud node (\S+) is now paired with Arvados node (\S+) with hostname (\S+)", node_paired),
                (r".*Cloud node (\S+) is now paired with Arvados node (\S+) with hostname (\S+)", node_paired),
                (r".*Cloud node (\S+) is now paired with Arvados node (\S+) with hostname (\S+)", node_paired),
                (r".*Cloud node (\S+) is now paired with Arvados node (\S+) with hostname (\S+)", node_paired),
                (r".*ComputeNodeMonitorActor\..*\.([^[]*).*Not eligible for shut down because node state is \('busy', 'open', .*\)", node_busy),
                (r".*ComputeNodeMonitorActor\..*\.([^[]*).*Suggesting shutdown because node state is \('idle', 'open', .*\)", noop),
                (r".*ComputeNodeShutdownActor\..*\.([^[]*).*Shutdown success", node_shutdown),
                (r".*ComputeNodeShutdownActor\..*\.([^[]*).*Shutdown success", node_shutdown),
                (r".*ComputeNodeShutdownActor\..*\.([^[]*).*Shutdown success", node_shutdown),
                (r".*ComputeNodeShutdownActor\..*\.([^[]*).*Shutdown success", node_shutdown),
            ], {
                r".*Suggesting shutdown because node state is \('down', .*\)": fail,
                r".*Cloud node (\S+) is now paired with Arvados node (\S+) with hostname (\S+)": partial(expect_count, 6),
                r".*Sending create_node request.*": partial(expect_count, 9)
            },
            "arvnodeman.test.fake_driver.QuotaDriver",
            {"34t0i-dz642-h42bg3hq4bdfpf1": "ReqNodeNotAvail",
             "34t0i-dz642-h42bg3hq4bdfpf2": "ReqNodeNotAvail",
             "34t0i-dz642-h42bg3hq4bdfpf3": "ReqNodeNotAvail",
             "34t0i-dz642-h42bg3hq4bdfpf4": "ReqNodeNotAvail"
         }),
        "test_no_hang_failing_node_create": (
            [
                (r".*Daemon started", set_squeue),
                (r".*Client error: nope", noop),
                (r".*Client error: nope", noop),
                (r".*Client error: nope", noop),
                (r".*Client error: nope", noop),
            ],
            {},
            "arvnodeman.test.fake_driver.FailingDriver",
            {"34t0i-dz642-h42bg3hq4bdfpf1": "ReqNodeNotAvail",
             "34t0i-dz642-h42bg3hq4bdfpf2": "ReqNodeNotAvail",
             "34t0i-dz642-h42bg3hq4bdfpf3": "ReqNodeNotAvail",
             "34t0i-dz642-h42bg3hq4bdfpf4": "ReqNodeNotAvail"
         }),
        "test_retry_create": (
            [
                (r".*Daemon started", set_squeue),
                (r".*Rate limit exceeded - scheduling retry in 12 seconds", noop),
                (r".*Cloud node (\S+) is now paired with Arvados node (\S+) with hostname (\S+)", noop),
            ],
            {},
            "arvnodeman.test.fake_driver.RetryDriver",
            {"34t0i-dz642-h42bg3hq4bdfpf1": "ReqNodeNotAvail"
         })
    }

    code = 0
    if len(sys.argv) > 1:
        code = run_test(sys.argv[1], *tests[sys.argv[1]])
    else:
        for t in sorted(tests.keys()):
            code += run_test(t, *tests[t])

    if code == 0:
        logger.info("Tests passed")
    else:
        logger.info("Tests failed")

    exit(code)

if __name__ == '__main__':
    main()
