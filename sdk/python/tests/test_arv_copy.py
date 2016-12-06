#!/usr/bin/env python
# -*- coding: utf-8 -*-

import multiprocessing
import os
import sys
import tempfile
import unittest

import arvados.commands.arv_copy as arv_copy

class ArvCopyTestCase(unittest.TestCase):
    def run_copy(self, args):
        sys.argv = ['arv-copy'] + args
        return arv_copy.main()

    def run_copy_process(self, args):
        _, stdout_path = tempfile.mkstemp()
        _, stderr_path = tempfile.mkstemp()
        def wrap():
            def wrapper():
                sys.argv = ['arv-copy'] + args
                sys.stdout = open(stdout_path, 'w')
                sys.stderr = open(stderr_path, 'w')
                arv_copy.main()
            return wrapper
        p = multiprocessing.Process(target=wrap())
        p.start()
        p.join()
        out = open(stdout_path, 'r').read()
        err = open(stderr_path, 'r').read()
        os.unlink(stdout_path)
        os.unlink(stderr_path)
        return p.exitcode, out, err

    def test_unsupported_arg(self):
        with self.assertRaises(SystemExit):
            self.run_copy(['-x=unknown'])

    def test_version_argument(self):
        exitcode, out, err = self.run_copy_process(['--version'])
        self.assertEqual(0, exitcode)
        self.assertEqual('', out)
        self.assertNotEqual('', err)
        self.assertRegexpMatches(err, "[0-9]+\.[0-9]+\.[0-9]+")
