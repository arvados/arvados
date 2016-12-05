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

from arvados_cwl.fsaccess import CollectionFetcher

class TestUrljoin(unittest.TestCase):
    def test_urljoin(self):
        """Test path joining for keep references."""

        cf = CollectionFetcher({}, None)

        self.assertEquals("keep:99999999999999999999999999999991+99/hw.py",
                          cf.urljoin("keep:99999999999999999999999999999991+99", "hw.py"))

        self.assertEquals("keep:99999999999999999999999999999991+99/hw.py",
                          cf.urljoin("keep:99999999999999999999999999999991+99/", "hw.py"))

        self.assertEquals("keep:99999999999999999999999999999991+99/hw.py#main",
                          cf.urljoin("keep:99999999999999999999999999999991+99", "hw.py#main"))

        self.assertEquals("keep:99999999999999999999999999999991+99/hw.py#main",
                          cf.urljoin("keep:99999999999999999999999999999991+99/hw.py", "#main"))

        self.assertEquals("keep:99999999999999999999999999999991+99/dir/hw.py#main",
                          cf.urljoin("keep:99999999999999999999999999999991+99/dir/hw.py", "#main"))

        self.assertEquals("keep:99999999999999999999999999999991+99/dir/wh.py",
                          cf.urljoin("keep:99999999999999999999999999999991+99/dir/hw.py", "wh.py"))

        self.assertEquals("keep:99999999999999999999999999999991+99/wh.py",
                          cf.urljoin("keep:99999999999999999999999999999991+99/dir/hw.py", "/wh.py"))

        self.assertEquals("keep:99999999999999999999999999999991+99/wh.py#main",
                          cf.urljoin("keep:99999999999999999999999999999991+99/dir/hw.py", "/wh.py#main"))

        self.assertEquals("keep:99999999999999999999999999999991+99/wh.py",
                          cf.urljoin("keep:99999999999999999999999999999991+99/hw.py#main", "wh.py"))

        self.assertEquals("keep:99999999999999999999999999999992+99",
                          cf.urljoin("keep:99999999999999999999999999999991+99", "keep:99999999999999999999999999999992+99"))

        self.assertEquals("keep:99999999999999999999999999999991+99/dir/wh.py",
                          cf.urljoin("keep:99999999999999999999999999999991+99/dir/", "wh.py"))

    def test_resolver(self):
        pass
