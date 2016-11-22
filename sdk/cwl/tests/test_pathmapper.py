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

from cwltool.pathmapper import MapperEnt
from .mock_discovery import get_rootDesc

from arvados_cwl.pathmapper import ArvPathMapper

def upload_mock(files, api, dry_run=False, num_retries=0, project=None, fnPattern="$(file %s/%s)", name=None):
    pdh = "99999999999999999999999999999991+99"
    for c in files:
        c.fn = fnPattern % (pdh, os.path.basename(c.fn))

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

        self.assertEqual({'keep:99999999999999999999999999999991+99/hw.py': MapperEnt(resolved='keep:99999999999999999999999999999991+99/hw.py', target='/test/99999999999999999999999999999991+99/hw.py', type='File')},
                         p._pathmap)

    @mock.patch("arvados.commands.run.uploadfiles")
    def test_upload(self, upl):
        """Test pathmapper uploading files."""

        arvrunner = arvados_cwl.ArvCwlRunner(self.api)

        upl.side_effect = upload_mock

        p = ArvPathMapper(arvrunner, [{
            "class": "File",
            "location": "tests/hw.py"
        }], "", "/test/%s", "/test/%s/%s")

        self.assertEqual({'tests/hw.py': MapperEnt(resolved='keep:99999999999999999999999999999991+99/hw.py', target='/test/99999999999999999999999999999991+99/hw.py', type='File')},
                         p._pathmap)

    @mock.patch("arvados.commands.run.uploadfiles")
    def test_prev_uploaded(self, upl):
        """Test pathmapper handling previously uploaded files."""

        arvrunner = arvados_cwl.ArvCwlRunner(self.api)
        arvrunner.add_uploaded('tests/hw.py', MapperEnt(resolved='keep:99999999999999999999999999999991+99/hw.py', target='', type='File'))

        upl.side_effect = upload_mock

        p = ArvPathMapper(arvrunner, [{
            "class": "File",
            "location": "tests/hw.py"
        }], "", "/test/%s", "/test/%s/%s")

        self.assertEqual({'tests/hw.py': MapperEnt(resolved='keep:99999999999999999999999999999991+99/hw.py', target='/test/99999999999999999999999999999991+99/hw.py', type='File')},
                         p._pathmap)

    @mock.patch("arvados.commands.run.uploadfiles")
    @mock.patch("arvados.commands.run.statfile")
    def test_statfile(self, statfile, upl):
        """Test pathmapper handling ArvFile references."""
        arvrunner = arvados_cwl.ArvCwlRunner(self.api)

        # An ArvFile object returned from arvados.commands.run.statfile means the file is located on a
        # keep mount, so we can construct a direct reference directly without upload.
        def statfile_mock(prefix, fn, fnPattern="$(file %s/%s)", dirPattern="$(dir %s/%s/)"):
            st = arvados.commands.run.ArvFile("", fnPattern % ("99999999999999999999999999999991+99", "hw.py"))
            return st

        upl.side_effect = upload_mock
        statfile.side_effect = statfile_mock

        p = ArvPathMapper(arvrunner, [{
            "class": "File",
            "location": "tests/hw.py"
        }], "", "/test/%s", "/test/%s/%s")

        self.assertEqual({'tests/hw.py': MapperEnt(resolved='keep:99999999999999999999999999999991+99/hw.py', target='/test/99999999999999999999999999999991+99/hw.py', type='File')},
                         p._pathmap)
