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

import arvados
import arvados.keep
import arvados.collection
import arvados_cwl
import arvados_cwl.executor

from cwltool.pathmapper import MapperEnt
from .mock_discovery import get_rootDesc

from arvados_cwl.pathmapper import ArvPathMapper

def upload_mock(files, api, dry_run=False, num_retries=0, project=None, fnPattern="$(file %s/%s)", name=None, collection=None, packed=None):
    pdh = "99999999999999999999999999999991+99"
    for c in files:
        c.keepref = "%s/%s" % (pdh, os.path.basename(c.fn))
        c.fn = fnPattern % (pdh, os.path.basename(c.fn))

class TestPathmap(unittest.TestCase):
    def setUp(self):
        self.api = mock.MagicMock()
        self.api._rootDesc = get_rootDesc()

    def test_keepref(self):
        """Test direct keep references."""

        arvrunner = arvados_cwl.executor.ArvCwlExecutor(self.api)

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

        arvrunner = arvados_cwl.executor.ArvCwlExecutor(self.api)

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
        arvrunner = arvados_cwl.executor.ArvCwlExecutor(self.api)

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
        arvrunner = arvados_cwl.executor.ArvCwlExecutor(self.api)

        stat.side_effect = OSError(2, "No such file or directory")

        with self.assertRaises(OSError):
            p = ArvPathMapper(arvrunner, [{
                "class": "File",
                "location": "file:tests/hw.py"
            }], "", "/test/%s", "/test/%s/%s")

    def test_needs_new_collection(self):
        arvrunner = arvados_cwl.executor.ArvCwlExecutor(self.api)

        # Plain file.  Don't need a new collection.
        a = {
            "class": "File",
            "location": "keep:99999999999999999999999999999991+99/hw.py",
            "basename": "hw.py"
        }
        p = ArvPathMapper(arvrunner, [], "", "%s", "%s/%s")
        p._pathmap["keep:99999999999999999999999999999991+99/hw.py"] = True
        self.assertFalse(p.needs_new_collection(a))

        # A file that isn't in the pathmap (for some reason).  Need a new collection.
        p = ArvPathMapper(arvrunner, [], "", "%s", "%s/%s")
        self.assertTrue(p.needs_new_collection(a))

        # A file with a secondary file in the same collection.  Don't need
        # a new collection.
        a = {
            "class": "File",
            "location": "keep:99999999999999999999999999999991+99/hw.py",
            "basename": "hw.py",
            "secondaryFiles": [{
                "class": "File",
                "location": "keep:99999999999999999999999999999991+99/hw.pyc",
                "basename": "hw.pyc"
            }]
        }
        p = ArvPathMapper(arvrunner, [], "", "%s", "%s/%s")
        p._pathmap["keep:99999999999999999999999999999991+99/hw.py"] = True
        p._pathmap["keep:99999999999999999999999999999991+99/hw.pyc"] = True
        self.assertFalse(p.needs_new_collection(a))

        # Secondary file is in a different collection from the
        # a new collectionprimary.  Need a new collection.
        a = {
            "class": "File",
            "location": "keep:99999999999999999999999999999991+99/hw.py",
            "basename": "hw.py",
            "secondaryFiles": [{
                "class": "File",
                "location": "keep:99999999999999999999999999999992+99/hw.pyc",
                "basename": "hw.pyc"
            }]
        }
        p = ArvPathMapper(arvrunner, [], "", "%s", "%s/%s")
        p._pathmap["keep:99999999999999999999999999999991+99/hw.py"] = True
        p._pathmap["keep:99999999999999999999999999999992+99/hw.pyc"] = True
        self.assertTrue(p.needs_new_collection(a))

        # Secondary file should be staged to a different name than
        # path in location.  Need a new collection.
        a = {
            "class": "File",
            "location": "keep:99999999999999999999999999999991+99/hw.py",
            "basename": "hw.py",
            "secondaryFiles": [{
                "class": "File",
                "location": "keep:99999999999999999999999999999991+99/hw.pyc",
                "basename": "hw.other"
            }]
        }
        p = ArvPathMapper(arvrunner, [], "", "%s", "%s/%s")
        p._pathmap["keep:99999999999999999999999999999991+99/hw.py"] = True
        p._pathmap["keep:99999999999999999999999999999991+99/hw.pyc"] = True
        self.assertTrue(p.needs_new_collection(a))

        # Secondary file is a directory.  Do not need a new collection.
        a = {
            "class": "File",
            "location": "keep:99999999999999999999999999999991+99/hw.py",
            "basename": "hw.py",
            "secondaryFiles": [{
                "class": "Directory",
                "location": "keep:99999999999999999999999999999991+99/hw",
                "basename": "hw",
                "listing": [{
                    "class": "File",
                    "location": "keep:99999999999999999999999999999991+99/hw/h2",
                    "basename": "h2"
                }]
            }]
        }
        p = ArvPathMapper(arvrunner, [], "", "%s", "%s/%s")
        p._pathmap["keep:99999999999999999999999999999991+99/hw.py"] = True
        p._pathmap["keep:99999999999999999999999999999991+99/hw"] = True
        p._pathmap["keep:99999999999999999999999999999991+99/hw/h2"] = True
        self.assertFalse(p.needs_new_collection(a))

        # Secondary file is a renamed directory.  Need a new collection.
        a = {
            "class": "File",
            "location": "keep:99999999999999999999999999999991+99/hw.py",
            "basename": "hw.py",
            "secondaryFiles": [{
                "class": "Directory",
                "location": "keep:99999999999999999999999999999991+99/hw",
                "basename": "wh",
                "listing": [{
                    "class": "File",
                    "location": "keep:99999999999999999999999999999991+99/hw/h2",
                    "basename": "h2"
                }]
            }]
        }
        p = ArvPathMapper(arvrunner, [], "", "%s", "%s/%s")
        p._pathmap["keep:99999999999999999999999999999991+99/hw.py"] = True
        p._pathmap["keep:99999999999999999999999999999991+99/hw"] = True
        p._pathmap["keep:99999999999999999999999999999991+99/hw/h2"] = True
        self.assertTrue(p.needs_new_collection(a))

        # Secondary file is a file literal.  Need a new collection.
        a = {
            "class": "File",
            "location": "keep:99999999999999999999999999999991+99/hw.py",
            "basename": "hw.py",
            "secondaryFiles": [{
                "class": "File",
                "location": "_:123",
                "basename": "hw.pyc",
                "contents": "123"
            }]
        }
        p = ArvPathMapper(arvrunner, [], "", "%s", "%s/%s")
        p._pathmap["keep:99999999999999999999999999999991+99/hw.py"] = True
        p._pathmap["_:123"] = True
        self.assertTrue(p.needs_new_collection(a))
