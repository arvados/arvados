# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import
from six import assertRegex
import arvados_fuse.command
import json
import multiprocessing
import os
from . import run_test_server
import tempfile
import unittest

from .integration_test import workerPool

try:
    from shlex import quote
except:
    from pipes import quote

def try_exec(mnt, cmd):
    try:
        os.environ['KEEP_LOCAL_STORE'] = tempfile.mkdtemp()
        arvados_fuse.command.Mount(
            arvados_fuse.command.ArgumentParser().parse_args([
                '--read-write',
                '--mount-tmp=zzz',
                '--unmount-timeout=0.1',
                mnt,
                '--exec'] + cmd)).run()
    except SystemExit:
        pass
    else:
        raise AssertionError('should have exited')


class ExecMode(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        run_test_server.run()
        run_test_server.run_keep(blob_signing=True, num_servers=2)
        run_test_server.authorize_with('active')

    @classmethod
    def tearDownClass(cls):
        run_test_server.stop_keep(num_servers=2)

    def setUp(self):
        self.mnt = tempfile.mkdtemp()
        _, self.okfile = tempfile.mkstemp()

    def tearDown(self):
        os.rmdir(self.mnt)
        os.unlink(self.okfile)

    def test_exec(self):
        workerPool().apply(try_exec, (self.mnt, [
            'sh', '-c',
            'echo -n foo >{}; cp {} {}'.format(
                quote(os.path.join(self.mnt, 'zzz', 'foo.txt')),
                quote(os.path.join(self.mnt, 'zzz', '.arvados#collection')),
                quote(os.path.join(self.okfile)))]))
        with open(self.okfile) as f:
            assertRegex(
                self,
                json.load(f)['manifest_text'],
                r' 0:3:foo.txt\n')
