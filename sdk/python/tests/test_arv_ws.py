#!/usr/bin/env python

import io
import os
import sys
import tempfile
import unittest

import arvados.errors as arv_error
import arvados.commands.ws as arv_ws
import arvados_testutil as tutil

class ArvWsTestCase(unittest.TestCase):
    def run_ws(self, args):
        return arv_ws.main(args)

    def test_unsupported_arg(self):
        with self.assertRaises(SystemExit):
            self.run_ws(['-x=unknown'])

    def test_version_argument(self):
        err = io.BytesIO()
        out = io.BytesIO()
        with tutil.redirected_streams(stdout=out, stderr=err):
            with self.assertRaises(SystemExit):
                self.run_ws(['--version'])
        self.assertEqual(out.getvalue(), '')
        self.assertRegexpMatches(err.getvalue(), "[0-9]+\.[0-9]+\.[0-9]+")
