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

from arvados_cwl.fsaccess import CollectionCache

class TestFsAccess(unittest.TestCase):
    @mock.patch("arvados.collection.CollectionReader")
    def test_collection_cache(self, cr):
        cache = CollectionCache(mock.MagicMock(), mock.MagicMock(), 4)
        c1 = cache.get("99999999999999999999999999999991+99")
        c2 = cache.get("99999999999999999999999999999991+99")
        self.assertIs(c1, c2)
        self.assertEqual(1, cr.call_count)
        c3 = cache.get("99999999999999999999999999999992+99")
        self.assertEqual(2, cr.call_count)
