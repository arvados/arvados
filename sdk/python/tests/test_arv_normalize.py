#!/usr/bin/env python
# -*- coding: utf-8 -*-

import subprocess
import sys
import tempfile
import unittest


class ArvNormalizeTestCase(unittest.TestCase):
    def run_arv_normalize(self, args=[]):
        p = subprocess.Popen([sys.executable, 'bin/arv-normalize'] + args,
                             stdout=subprocess.PIPE,
                             stderr=subprocess.PIPE)
        (stdout, stderr) = p.communicate()
        return p.returncode, stdout, stderr

    def test_unsupported_arg(self):
        returncode, out, err = self.run_arv_normalize(['-x=unknown'])
        self.assertNotEqual(0, returncode)

    def test_version_argument(self):
        returncode, out, err = self.run_arv_normalize(['--version'])
        self.assertEqual(0, returncode)
        self.assertEqual('', out)
        self.assertNotEqual('', err)
        self.assertRegexpMatches(err, "[0-9]+\.[0-9]+\.[0-9]+")
