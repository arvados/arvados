#!/usr/bin/env python

import multiprocessing
import os
import sys
import tempfile
import unittest

import arvnodeman.launcher as nodeman

class ArvNodemArgumentsTestCase(unittest.TestCase):
    def run_nodeman(self, args):
        return nodeman.main(args)

    def run_nodeman_process(self, args=[]):
        _, stdout_path = tempfile.mkstemp()
        _, stderr_path = tempfile.mkstemp()
        def wrap():
            def wrapper(*args, **kwargs):
                sys.stdout = open(stdout_path, 'w')
                sys.stderr = open(stderr_path, 'w')
                nodeman.main(*args, **kwargs)
            return wrapper
        p = multiprocessing.Process(target=wrap(), args=(args,))
        p.start()
        p.join()
        out = open(stdout_path, 'r').read()
        err = open(stderr_path, 'r').read()
        os.unlink(stdout_path)
        os.unlink(stderr_path)
        return p.exitcode, out, err

    def test_unsupported_arg(self):
        with self.assertRaises(SystemExit):
            self.run_nodeman(['-x=unknown'])

    def test_version_argument(self):
        exitcode, out, err = self.run_nodeman_process(['--version'])
        self.assertEqual(0, exitcode)
        self.assertEqual('', out)
        self.assertNotEqual('', err)
        self.assertRegexpMatches(err, "[0-9]+\.[0-9]+\.[0-9]+")
