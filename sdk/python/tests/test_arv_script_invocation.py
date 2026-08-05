# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
"""Invoke installed scripts directly as executables in order to perform basic
sanity checks.
"""
import re
import subprocess

import pytest


ARV_SCRIPTS = (
    "arv", "arv-copy", "arv-get", "arv-keepdocker", "arv-ls", "arv-normalize",
    "arv-put", "arv-ws"
)


def run_script(args: list[str], **kwargs):
    return subprocess.run(
        args, capture_output=True, text=True, check=False, **kwargs
    )


@pytest.mark.parametrize("script", ARV_SCRIPTS)
class TestArvScriptRun:
    def test_help(self, script):
        completed_process = run_script([script, "-h"])
        assert completed_process.returncode == 0
        assert f"usage: {script}" in completed_process.stdout
        assert not completed_process.stderr

    def test_version(self, script):
        completed_process = run_script([script, "--version"])
        assert completed_process.returncode == 0
        assert re.match(
            rf"^{re.escape(script)} [0-9]+\.[0-9]+\.[0-9]+(\.dev[0-9]+)?$\n",
            completed_process.stdout
        )
        assert not completed_process.stderr

    def test_invalid_argument(self, script):
        completed_process = run_script([script, "--x-invalid-argument"])
        assert completed_process.returncode == 2
        assert not completed_process.stdout
        assert f"usage: {script}" in completed_process.stderr
