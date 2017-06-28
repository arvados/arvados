# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import absolute_import
import os
import sys
import tempfile
import unittest

import arvados.commands.arv_copy as arv_copy
from . import arvados_testutil as tutil

class ArvCopyTestCase(unittest.TestCase, tutil.VersionChecker):
    def run_copy(self, args):
        sys.argv = ['arv-copy'] + args
        return arv_copy.main()

    def test_unsupported_arg(self):
        with self.assertRaises(SystemExit):
            self.run_copy(['-x=unknown'])

    def test_version_argument(self):
        with tutil.redirected_streams(
                stdout=tutil.StringIO, stderr=tutil.StringIO) as (out, err):
            with self.assertRaises(SystemExit):
                self.run_copy(['--version'])
        self.assertVersionOutput(out, err)
