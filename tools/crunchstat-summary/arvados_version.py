# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import subprocess
import time
import os
import re
import sys

SETUP_DIR = os.path.dirname(os.path.abspath(__file__))
VERSION_PATHS = {
        SETUP_DIR,
        os.path.abspath(os.path.join(SETUP_DIR, "../../sdk/python")),
        os.path.abspath(os.path.join(SETUP_DIR, "../../build/version-at-commit.sh"))
        }

def choose_version_from():
    ts = {}
    for path in VERSION_PATHS:
        ts[subprocess.check_output(
            ['git', 'log', '--first-parent', '--max-count=1',
             '--format=format:%ct', path]).strip()] = path

    sorted_ts = sorted(ts.items())
    getver = sorted_ts[-1][1]
    print("Using "+getver+" for version number calculation of "+SETUP_DIR, file=sys.stderr)
    return getver

def git_version_at_commit():
    curdir = choose_version_from()
    myhash = subprocess.check_output(['git', 'log', '-n1', '--first-parent',
                                       '--format=%H', curdir]).strip()
    myversion = subprocess.check_output([SETUP_DIR+'/../../build/version-at-commit.sh', myhash]).strip().decode()
    return myversion

def save_version(setup_dir, module, v):
    v = v.replace("~dev", ".dev").replace("~rc", "rc")
    with open(os.path.join(setup_dir, module, "_version.py"), 'wt') as fp:
        return fp.write("__version__ = '%s'\n" % v)

def read_version(setup_dir, module):
    with open(os.path.join(setup_dir, module, "_version.py"), 'rt') as fp:
        return re.match("__version__ = '(.*)'$", fp.read()).groups()[0]

def get_version(setup_dir, module):
    env_version = os.environ.get("ARVADOS_BUILDING_VERSION")

    if env_version:
        save_version(setup_dir, module, env_version)
    else:
        try:
            save_version(setup_dir, module, git_version_at_commit())
        except (subprocess.CalledProcessError, OSError) as err:
            print("ERROR: {0}".format(err), file=sys.stderr)
            pass

    return read_version(setup_dir, module)
