#!/usr/bin/env python
# -*- coding: utf-8 -*-

import io
import os
import sys
import tempfile
import unittest

import arvados.commands.keepdocker as arv_keepdocker
import arvados_testutil as tutil


class ArvKeepdockerTestCase(unittest.TestCase):
    def run_arv_keepdocker(self, args):
        sys.argv = ['arv-keepdocker'] + args
        return arv_keepdocker.main()

    def test_unsupported_arg(self):
        with self.assertRaises(SystemExit):
            self.run_arv_keepdocker(['-x=unknown'])

    def test_version_argument(self):
        err = io.BytesIO()
        out = io.BytesIO()
        with tutil.redirected_streams(stdout=out, stderr=err):
            with self.assertRaises(SystemExit):
                self.run_arv_keepdocker(['--version'])
        self.assertEqual(out.getvalue(), '')
        self.assertRegexpMatches(err.getvalue(), "[0-9]+\.[0-9]+\.[0-9]+")
