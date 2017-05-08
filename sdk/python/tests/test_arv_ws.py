from __future__ import absolute_import
import os
import sys
import tempfile
import unittest

import arvados.errors as arv_error
import arvados.commands.ws as arv_ws
from . import arvados_testutil as tutil

class ArvWsTestCase(unittest.TestCase, tutil.VersionChecker):
    def run_ws(self, args):
        return arv_ws.main(args)

    def test_unsupported_arg(self):
        with self.assertRaises(SystemExit):
            self.run_ws(['-x=unknown'])

    def test_version_argument(self):
        with tutil.redirected_streams(
                stdout=tutil.StringIO, stderr=tutil.StringIO) as (out, err):
            with self.assertRaises(SystemExit):
                self.run_ws(['--version'])
        self.assertVersionOutput(out, err)
