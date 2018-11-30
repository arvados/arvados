# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import subprocess
import time
import os
import re

def git_latest_tag():
    gittags = subprocess.check_output(['git', 'tag', '-l']).split()
    gittags.sort(key=lambda s: [int(u) for u in s.split(b'.')],reverse=True)
    return str(next(iter(gittags)).decode('utf-8'))

def git_timestamp_tag():
    gitinfo = subprocess.check_output(
        ['git', 'log', '--first-parent', '--max-count=1',
         '--format=format:%ct', '.']).strip()
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
        except subprocess.CalledProcessError:
            pass

    return read_version(setup_dir, module)
