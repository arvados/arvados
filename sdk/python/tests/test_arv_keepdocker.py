#!/usr/bin/env python
# -*- coding: utf-8 -*-

import multiprocessing
import os
import sys
import tempfile
import unittest

import arvados.commands.keepdocker as arv_keepdocker


class ArvKeepdockerTestCase(unittest.TestCase):
    def run_arv_keepdocker(self, args):
        sys.argv = ['arv-keepdocker'] + args
        return arv_keepdocker.main()

    def run_arv_keepdocker_process(self, args):
        _, stdout_path = tempfile.mkstemp()
        _, stderr_path = tempfile.mkstemp()
        def wrap():
            def wrapper():
                sys.argv = ['arv-keepdocker'] + args
                sys.stdout = open(stdout_path, 'w')
                sys.stderr = open(stderr_path, 'w')
                arv_keepdocker.main()
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
            self.run_arv_keepdocker(['-x=unknown'])

    def test_version_argument(self):
        exitcode, out, err = self.run_arv_keepdocker_process(['--version'])
        self.assertEqual(0, exitcode)
        self.assertEqual('', out)
        self.assertNotEqual('', err)
        self.assertRegexpMatches(err, "[0-9]+\.[0-9]+\.[0-9]+")
