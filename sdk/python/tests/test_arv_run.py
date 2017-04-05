from __future__ import absolute_import
import os
import sys
import tempfile
import unittest

import arvados.commands.run as arv_run
from . import arvados_testutil as tutil

class ArvRunTestCase(unittest.TestCase, tutil.VersionChecker):
    def run_arv_run(self, args):
        sys.argv = ['arv-run'] + args
        return arv_run.main()

    def test_unsupported_arg(self):
        with self.assertRaises(SystemExit):
            self.run_arv_run(['-x=unknown'])

    def test_version_argument(self):
        with tutil.redirected_streams(
                stdout=tutil.StringIO, stderr=tutil.StringIO) as (out, err):
            with self.assertRaises(SystemExit):
                self.run_arv_run(['--version'])
        self.assertVersionOutput(out, err)
