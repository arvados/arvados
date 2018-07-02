# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import functools
import mock
import sys
import unittest
import json
import logging
import os
import datetime

import arvados
import arvados.keep
import arvados.collection
import arvados_cwl

from cwltool.pathmapper import MapperEnt
from .mock_discovery import get_rootDesc

from arvados_cwl.pathmapper import ArvPathMapper

def upload_mock(files, api, dry_run=False, num_retries=0, project=None, fnPattern="$(file %s/%s)", name=None, collection=None, packed=None):
    pdh = "99999999999999999999999999999991+99"
    for c in files:
        c.keepref = "%s/%s" % (pdh, os.path.basename(c.fn))
        c.fn = fnPattern % (pdh, os.path.basename(c.fn))

class MockDateTime(datetime.datetime):
    @classmethod
    def now(cls):
        return datetime.datetime(2018, 1, 1, 0, 0, 0, 0)

datetime.datetime = MockDateTime

class TestPathmap(unittest.TestCase):
    def setUp(self):
        self.api = mock.MagicMock()
        self.api._rootDesc = get_rootDesc()

    def test_keepref(self):
        """Test direct keep references."""

        arvrunner = arvados_cwl.ArvCwlRunner(self.api)

        p = ArvPathMapper(arvrunner, [{
            "class": "File",
            "location": "keep:99999999999999999999999999999991+99/hw.py"
        }], "", "/test/%s", "/test/%s/%s")

        self.assertEqual({'keep:99999999999999999999999999999991+99/hw.py': MapperEnt(resolved='keep:99999999999999999999999999999991+99/hw.py', target='/test/99999999999999999999999999999991+99/hw.py', type='File', staged=True)},
                         p._pathmap)

    @mock.patch("arvados.commands.run.uploadfiles")
    @mock.patch("arvados.commands.run.statfile")
    def test_upload(self, statfile, upl):
        """Test pathmapper uploading files."""

        arvrunner = arvados_cwl.ArvCwlRunner(self.api)

        def statfile_mock(prefix, fn, fnPattern="$(file %s/%s)", dirPattern="$(dir %s/%s/)", raiseOSError=False):
            st = arvados.commands.run.UploadFile("", "tests/hw.py")
            return st

        upl.side_effect = upload_mock
        statfile.side_effect = statfile_mock

        p = ArvPathMapper(arvrunner, [{
            "class": "File",
            "location": "file:tests/hw.py"
        }], "", "/test/%s", "/test/%s/%s")

        self.assertEqual({'file:tests/hw.py': MapperEnt(resolved='keep:99999999999999999999999999999991+99/hw.py', target='/test/99999999999999999999999999999991+99/hw.py', type='File', staged=True)},
                         p._pathmap)

    @mock.patch("arvados.commands.run.uploadfiles")
    @mock.patch("arvados.commands.run.statfile")
    def test_statfile(self, statfile, upl):
        """Test pathmapper handling ArvFile references."""
        arvrunner = arvados_cwl.ArvCwlRunner(self.api)

        # An ArvFile object returned from arvados.commands.run.statfile means the file is located on a
        # keep mount, so we can construct a direct reference directly without upload.
        def statfile_mock(prefix, fn, fnPattern="$(file %s/%s)", dirPattern="$(dir %s/%s/)", raiseOSError=False):
            st = arvados.commands.run.ArvFile("", fnPattern % ("99999999999999999999999999999991+99", "hw.py"))
            return st

        upl.side_effect = upload_mock
        statfile.side_effect = statfile_mock

        p = ArvPathMapper(arvrunner, [{
            "class": "File",
            "location": "file:tests/hw.py"
        }], "", "/test/%s", "/test/%s/%s")

        self.assertEqual({'file:tests/hw.py': MapperEnt(resolved='keep:99999999999999999999999999999991+99/hw.py', target='/test/99999999999999999999999999999991+99/hw.py', type='File', staged=True)},
                         p._pathmap)

    @mock.patch("os.stat")
    def test_missing_file(self, stat):
        """Test pathmapper handling missing references."""
        arvrunner = arvados_cwl.ArvCwlRunner(self.api)

        stat.side_effect = OSError(2, "No such file or directory")

        with self.assertRaises(OSError):
            p = ArvPathMapper(arvrunner, [{
                "class": "File",
                "location": "file:tests/hw.py"
            }], "", "/test/%s", "/test/%s/%s")

    def test_get_intermediate_collection_info(self):
        self.api.containers().current().execute.return_value = {"uuid" : "zzzzz-8i9sb-zzzzzzzzzzzzzzz"}
        arvrunner = arvados_cwl.ArvCwlRunner(self.api)
        arvrunner.intermediate_output_ttl = 60

        path_mapper = ArvPathMapper(arvrunner, [{
            "class": "File",
            "location": "keep:99999999999999999999999999999991+99/hw.py"
        }], "", "/test/%s", "/test/%s/%s")

        info = path_mapper._get_intermediate_collection_info()

        self.assertEqual(info["name"], "Intermediate collection")
        self.assertEqual(info["trash_at"], datetime.datetime(2018, 1, 1, 0, 1))
        self.assertEqual(info["properties"], {"type" : "Intermediate", "container" : "zzzzz-8i9sb-zzzzzzzzzzzzzzz"})
