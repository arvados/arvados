# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import subprocess
import sys
import tempfile
import unittest

from . import arvados_testutil as tutil


class ArvNormalizeTestCase(unittest.TestCase, tutil.VersionChecker):
    def run_arv_normalize(self, args=[]):
        p = subprocess.Popen([sys.executable, 'bin/arv-normalize'] + args,
                             stdout=subprocess.PIPE,
                             stderr=subprocess.PIPE)
        out, err = p.communicate()
        sys.stdout.write(out.decode())
        sys.stderr.write(err.decode())
        return p.returncode

    def test_unsupported_arg(self):
        with tutil.redirected_streams(
                stdout=tutil.StringIO, stderr=tutil.StringIO) as (out, err):
            returncode = self.run_arv_normalize(['-x=unknown'])
        self.assertNotEqual(0, returncode)

    def test_version_argument(self):
        with tutil.redirected_streams(
                stdout=tutil.StringIO, stderr=tutil.StringIO) as (out, err):
            returncode = self.run_arv_normalize(['--version'])
        self.assertVersionOutput(out, err)
        self.assertEqual(0, returncode)
