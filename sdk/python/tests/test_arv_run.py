# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import absolute_import
import os
import sys
import tempfile
import unittest
import random
import mock

import arvados.commands.run as arv_run
from . import arvados_testutil as tutil

class ArvRunTestCase(unittest.TestCase, tutil.VersionChecker):
    def run_arv_run(self, args):
        sys.argv = ['arv-run'] + args
        return arv_run.main()

    def test_unsupported_arg(self):
        with self.assertRaises(SystemExit):
            self.run_arv_run(['-x=unknown'])

    def test_version_argument(self):
        with tutil.redirected_streams(
                stdout=tutil.StringIO, stderr=tutil.StringIO) as (out, err):
            with self.assertRaises(SystemExit):
                self.run_arv_run(['--version'])
        self.assertVersionOutput(out, err)

    @mock.patch('arvados.commands.run.write_file')
    def test_uploadfiles(self, write_file_mock):
        path = os.getcwd()
        files = [arv_run.statfile('', 'tests/upf/'+s) for s in ('a', 'b', 'b/y', 'c/x', 'd')]
        random.shuffle(files)
        mockcol = mock.MagicMock()
        arv_run.uploadfiles(files, mock.MagicMock(), collection=mockcol)
        write_file_mock.assert_has_calls([mock.call(mockcol, path+"/tests/upf/", 'a'),
                                          mock.call(mockcol, path+"/tests/upf/", 'd'),
                                          mock.call(mockcol, path+"/tests/upf/", 'b/x'),
                                          mock.call(mockcol, path+"/tests/upf/", 'b/y'),
                                          mock.call(mockcol, path+"/tests/upf/", 'c/x')])
