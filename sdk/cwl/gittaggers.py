# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from setuptools.command.egg_info import egg_info
import subprocess
import time
import os

SETUP_DIR = os.path.dirname(__file__) or '.'

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

class EggInfoFromGit(egg_info):
    """Tag the build with git commit timestamp.

    If a build tag has already been set (e.g., "egg_info -b", building
    from source package), leave it alone.
    """
    def git_latest_tag(self):
        gitinfo = subprocess.check_output(
            ['git', 'describe', '--abbrev=0']).strip()
        return str(gitinfo.decode('utf-8'))

    def git_timestamp_tag(self):
        gitinfo = subprocess.check_output(
            ['git', 'log', '--first-parent', '--max-count=1',
             '--format=format:%ct', choose_version_from()]).strip()
        return time.strftime('.%Y%m%d%H%M%S', time.gmtime(int(gitinfo)))

    def tags(self):
        if self.tag_build is None:
            self.tag_build = self.git_latest_tag() + self.git_timestamp_tag()
        return egg_info.tags(self)
