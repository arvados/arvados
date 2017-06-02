#!/usr/bin/env python
import subprocess
import os
import sys
import re
import time
import logging
import stat
import tempfile
import shutil

logging.basicConfig(level=logging.INFO)

fake_slurm = None
compute_nodes = None

def update_script(path, val):
    with open(path+"_", "w") as f:
        f.write(val)
    os.chmod(path+"_", stat.S_IRUSR | stat.S_IWUSR | stat.S_IXUSR)
    os.rename(path+"_", path)


def set_squeue(actions, checks, k, g):
    update_script(os.path.join(fake_slurm, "squeue"), """#!/bin/sh
echo '1|100|100|ReqNodeNotAvail|34t0i-dz642-h42bg3hq4bdfpf9'
""")
    return 0

def set_sinfo_alloc(actions, checks, k, g):
    update_script(os.path.join(fake_slurm, "sinfo"), """#!/bin/sh
echo '%s alloc'
""" % (g.group(3)))

    update_script(os.path.join(fake_slurm, "squeue"), """#!/bin/sh
echo '1|100|100|Running|34t0i-dz642-h42bg3hq4bdfpf9'
""")

    global compute_nodes
    compute_nodes[g.group(1)] = g.group(3)
    return 0

def set_sinfo_idle(actions, checks, k, g):
    update_script(os.path.join(fake_slurm, "sinfo"), """#!/bin/sh
echo '%s idle'
""" % (compute_nodes[g.group(1)]))
    return 0

def noop(actions, checks, k, g):
    return 0

def down_fail(actions, checks, k, g):
    return 1


def run_test(actions, checks, driver_class):
    code = 0

    global fake_slurm
    fake_slurm = tempfile.mkdtemp()
    logging.info("fake_slurm is %s", fake_slurm)

    global compute_nodes
    compute_nodes = {}

    env = os.environ.copy()
    env["PATH"] = fake_slurm + ":" + env["PATH"]

    update_script(os.path.join(fake_slurm, "squeue"), "#!/bin/sh\n")
    update_script(os.path.join(fake_slurm, "sinfo"), "#!/bin/sh\n")

    with open("tests/fake.cfg.template") as f:
        with open(os.path.join(fake_slurm, "id_rsa.pub"), "w") as ssh:
            pass
        with open(os.path.join(fake_slurm, "fake.cfg"), "w") as cfg:
            cfg.write(f.read().format(host=os.environ["ARVADOS_API_HOST"],
                                      token=os.environ["ARVADOS_API_TOKEN"],
                                      driver_class=driver_class,
                                      ssh_key=os.path.join(fake_slurm, "id_rsa.pub")))

    timeout = time.time() + 300

    p = subprocess.Popen(["bin/arvados-node-manager", "--foreground", "--config", os.path.join(fake_slurm, "fake.cfg")],
                         bufsize=1, stderr=subprocess.PIPE, env=env)
    for line in p.stderr:
        sys.stdout.write(line)

        if time.time() > timeout:
            logging.error("Exceeded timeout")
            code = 1
            p.terminate()

        for k,v in actions.items():
            g = re.match(k, line)
            if g:
                logging.info("Triggered action %s", k)
                del actions[k]
                code = v(actions, checks, k, g)
                if code != 0:
                    logging.error("Action failed")
                    p.terminate()

        for k,v in checks.items():
            g = re.match(k, line)
            if g:
                logging.info("Triggered check %s", k)
                code = v(actions, checks, k, g)
                if code != 0:
                    logging.error("Check failed")
                    p.terminate()

        if not actions:
            p.terminate()

    #shutil.rmtree(fake_slurm)

    return code


def main():
    code = run_test({
        r".*Daemon started": set_squeue,
        r".*Cloud node (\S+) is now paired with Arvados node (\S+) with hostname (\S+)": set_sinfo_alloc,
        r".*ComputeNodeMonitorActor\..*\.([^[]*).*Not eligible for shut down because node state is \('busy', 'open', .*\)": set_sinfo_idle,
        r".*ComputeNodeMonitorActor\..*\.([^[]*).*Suggesting shutdown because node state is \('idle', 'open', .*\)": noop,
        r".*Shutdown success": noop,
    }, {
        r".*Suggesting shutdown because node state is \('down', .*\)": down_fail
    },
    "arvnodeman.test.fake_driver.FakeDriver")
    exit(code)

main()
