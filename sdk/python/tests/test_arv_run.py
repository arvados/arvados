#!/usr/bin/env python
# -*- coding: utf-8 -*-

import io
import os
import sys
import tempfile
import unittest

import arvados.commands.run as arv_run
import arvados_testutil as tutil

class ArvRunTestCase(unittest.TestCase):
    def run_arv_run(self, args):
        sys.argv = ['arv-run'] + args
        return arv_run.main()

    def test_unsupported_arg(self):
        with self.assertRaises(SystemExit):
            self.run_arv_run(['-x=unknown'])

    def test_version_argument(self):
        err = io.BytesIO()
        out = io.BytesIO()
        with tutil.redirected_streams(stdout=out, stderr=err):
            with self.assertRaises(SystemExit):
                self.run_arv_run(['--version'])
        self.assertEqual(out.getvalue(), '')
        self.assertRegexpMatches(err.getvalue(), "[0-9]+\.[0-9]+\.[0-9]+")
