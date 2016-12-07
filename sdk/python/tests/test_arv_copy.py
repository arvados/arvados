#!/usr/bin/env python
# -*- coding: utf-8 -*-

import io
import os
import sys
import tempfile
import unittest

import arvados.commands.arv_copy as arv_copy
import arvados_testutil as tutil

class ArvCopyTestCase(unittest.TestCase):
    def run_copy(self, args):
        sys.argv = ['arv-copy'] + args
        return arv_copy.main()

    def test_unsupported_arg(self):
        with self.assertRaises(SystemExit):
            self.run_copy(['-x=unknown'])

    def test_version_argument(self):
        err = io.BytesIO()
        out = io.BytesIO()
        with tutil.redirected_streams(stdout=out, stderr=err):
            with self.assertRaises(SystemExit):
                self.run_copy(['--version'])
        self.assertEqual(out.getvalue(), '')
        self.assertRegexpMatches(err.getvalue(), "[0-9]+\.[0-9]+\.[0-9]+")
