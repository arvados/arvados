# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import subprocess
import time
import os
import re

SETUP_DIR = os.path.dirname(__file__) or '.'

def git_latest_tag():
    gitinfo = subprocess.check_output(
        ['git', 'describe', '--abbrev=0']).strip()
    return str(gitinfo.decode('utf-8'))

def choose_version_from():
    sdk_ts = subprocess.check_output(
        ['git', 'log', '--first-parent', '--max-count=1',
         '--format=format:%ct', os.path.join(SETUP_DIR, "../python")]).strip()
    cwl_ts = subprocess.check_output(
        ['git', 'log', '--first-parent', '--max-count=1',
         '--format=format:%ct', SETUP_DIR]).strip()
    if int(sdk_ts) > int(cwl_ts):
        getver = os.path.join(SETUP_DIR, "../python")
    else:
        getver = SETUP_DIR
    return getver

def git_timestamp_tag():
    gitinfo = subprocess.check_output(
        ['git', 'log', '--first-parent', '--max-count=1',
         '--format=format:%ct', choose_version_from()]).strip()
    return str(time.strftime('.%Y%m%d%H%M%S', time.gmtime(int(gitinfo))))

def save_version(setup_dir, module, v):
  with open(os.path.join(setup_dir, module, "_version.py"), 'w') as fp:
      return fp.write("__version__ = '%s'\n" % v)

def read_version(setup_dir, module):
  with open(os.path.join(setup_dir, module, "_version.py"), 'r') as fp:
      return re.match("__version__ = '(.*)'$", fp.read()).groups()[0]

def get_version(setup_dir, module):
    env_version = os.environ.get("ARVADOS_BUILDING_VERSION")

    if env_version:
        save_version(setup_dir, module, env_version)
    else:
        try:
            save_version(setup_dir, module, git_latest_tag() + git_timestamp_tag())
        except (subprocess.CalledProcessError, OSError):
            pass

    return read_version(setup_dir, module)
