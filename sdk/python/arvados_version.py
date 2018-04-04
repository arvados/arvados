# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from setuptools.command.egg_info import egg_info
import subprocess
import time

class VersionInfoFromGit():
    """Return arvados version from git
    """
    def git_latest_tag(self):
        gitinfo = subprocess.check_output(
            ['git', 'describe', '--abbrev=0']).strip()
        return str(gitinfo.decode('utf-8'))

    def git_timestamp_tag(self):
        gitinfo = subprocess.check_output(
            ['git', 'log', '--first-parent', '--max-count=1',
             '--format=format:%ct', '.']).strip()
        return str(time.strftime('.%Y%m%d%H%M%S', time.gmtime(int(gitinfo))))
    
    def tags(self):
        if self.tag_build is None:
            self.tag_build = self.git_latest_tag()+self.git_timestamp_tag()
        return egg_info.tags(self)