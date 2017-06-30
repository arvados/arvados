#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import io
import os
import sys
import tempfile
import unittest

import arvnodeman.launcher as nodeman
from . import testutil

class ArvNodemArgumentsTestCase(unittest.TestCase):
    def run_nodeman(self, args):
        return nodeman.main(args)

    def test_unsupported_arg(self):
        with self.assertRaises(SystemExit):
            self.run_nodeman(['-x=unknown'])

    def test_version_argument(self):
        err = io.BytesIO()
        out = io.BytesIO()
        with testutil.redirected_streams(stdout=out, stderr=err):
            with self.assertRaises(SystemExit):
                self.run_nodeman(['--version'])
        self.assertEqual(out.getvalue(), '')
        self.assertRegexpMatches(err.getvalue(), "[0-9]+\.[0-9]+\.[0-9]+")
