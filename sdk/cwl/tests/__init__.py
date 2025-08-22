# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import os
import subprocess

from pathlib import Path

TESTS_DIR = Path(__file__).parent

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
        '-j', '3',
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
