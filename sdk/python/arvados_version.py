# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

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
