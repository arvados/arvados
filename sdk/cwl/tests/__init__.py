# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import os
import subprocess
import sys
import warnings

from pathlib import Path

TESTS_DIR = Path(__file__).parent

# We default to running 3 jobs in parallel, which is tested to be what works in
# October 2025 on Jenkins under run-tests.sh with crunch-dispatch-local.
# Users testing directly against a live cluster can increase this.
_jobs = os.environ.get('ARVADOS_CWLTEST_JOBS')
try:
    ARVADOS_CWLTEST_JOBS = int(_jobs, 10)
except (TypeError, ValueError):
    _jobs_ok = False
else:
    _jobs_ok = 0 < ARVADOS_CWLTEST_JOBS < sys.maxsize
if not _jobs_ok:
    ARVADOS_CWLTEST_JOBS = 3
    if _jobs is not None:
        warnings.warn(
            f"ARVADOS_CWLTEST_JOBS value {_jobs!r} is invalid;"
            f" using default {ARVADOS_CWLTEST_JOBS!r}"
        )
del _jobs, _jobs_ok

def run_cwltest(
        cwl_test,
        cwl_tool,
        badges_dir=None,
        *,
        test_args=(),
        tool_args=(),
):
    cmd = [
        'cwltest',
        '--test', str(cwl_test),
        '--tool', str(cwl_tool),
        '-j', str(ARVADOS_CWLTEST_JOBS),
    ]
    cmd.extend(test_args)
    # FIXME?: cwltest badge generation seems buggy as of 2.5.20241122133319
    # if badges_dir:
    #     cmd.append('--badgedir')
    #     cmd.append(str(badges_dir))
    cmd.extend([
        '--',
        '--compute-checksum',
        '--disable-reuse',
        '--enable-dev',
    ])
    cmd.extend(tool_args)
    return subprocess.run(cmd)
