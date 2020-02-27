# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import subprocess
import time
import os
import re

SETUP_DIR = os.path.dirname(os.path.abspath(__file__))

def choose_version_from():
    sdk_ts = subprocess.check_output(
        ['git', 'log', '--first-parent', '--max-count=1',
         '--format=format:%ct', os.path.join(SETUP_DIR, "../../sdk/python")]).strip()
    cwl_ts = subprocess.check_output(
        ['git', 'log', '--first-parent', '--max-count=1',
         '--format=format:%ct', SETUP_DIR]).strip()
    if int(sdk_ts) > int(cwl_ts):
        getver = os.path.join(SETUP_DIR, "../../sdk/python")
    else:
        getver = SETUP_DIR
    return getver

def git_version_at_commit():
    curdir = choose_version_from()
    myhash = subprocess.check_output(['git', 'log', '-n1', '--first-parent',
                                       '--format=%H', curdir]).strip()
    myversion = subprocess.check_output([curdir+'/../../build/version-at-commit.sh', myhash]).strip().decode()
    return myversion

def save_version(setup_dir, module, v):
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
        except (subprocess.CalledProcessError, OSError):
            pass

    return read_version(setup_dir, module)
