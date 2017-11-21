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

    @mock.patch("arvados.collection.CollectionReader")
    def test_collection_cache_limit(self, cr):
        cache = CollectionCache(mock.MagicMock(), mock.MagicMock(), 4)
        cr().manifest_text.return_value = 'x' * 524289
        self.assertEqual(0, cache.total)
        c1 = cache.get("99999999999999999999999999999991+99")
        self.assertIn("99999999999999999999999999999991+99", cache.collections)
        self.assertNotIn("99999999999999999999999999999992+99", cache.collections)
        self.assertEqual((524289*128)*1, cache.total)

        c2 = cache.get("99999999999999999999999999999992+99")
        self.assertIn("99999999999999999999999999999991+99", cache.collections)
        self.assertIn("99999999999999999999999999999992+99", cache.collections)
        self.assertEqual((524289*128)*2, cache.total)

        c1 = cache.get("99999999999999999999999999999991+99")
        self.assertIn("99999999999999999999999999999991+99", cache.collections)
        self.assertIn("99999999999999999999999999999992+99", cache.collections)
        self.assertEqual((524289*128)*2, cache.total)

        c3 = cache.get("99999999999999999999999999999993+99")
        self.assertIn("99999999999999999999999999999991+99", cache.collections)
        self.assertIn("99999999999999999999999999999992+99", cache.collections)
        self.assertEqual((524289*128)*3, cache.total)

        c4 = cache.get("99999999999999999999999999999994+99")
        self.assertIn("99999999999999999999999999999991+99", cache.collections)
        self.assertNotIn("99999999999999999999999999999992+99", cache.collections)
        self.assertEqual((524289*128)*3, cache.total)

        c5 = cache.get("99999999999999999999999999999995+99")
        self.assertNotIn("99999999999999999999999999999991+99", cache.collections)
        self.assertNotIn("99999999999999999999999999999992+99", cache.collections)
        self.assertEqual((524289*128)*3, cache.total)


    @mock.patch("arvados.collection.CollectionReader")
    def test_collection_cache_limit2(self, cr):
        cache = CollectionCache(mock.MagicMock(), mock.MagicMock(), 4)
        cr().manifest_text.return_value = 'x' * 524287
        self.assertEqual(0, cache.total)
        c1 = cache.get("99999999999999999999999999999991+99")
        self.assertIn("99999999999999999999999999999991+99", cache.collections)
        self.assertNotIn("99999999999999999999999999999992+99", cache.collections)
        self.assertEqual((524287*128)*1, cache.total)

        c2 = cache.get("99999999999999999999999999999992+99")
        self.assertIn("99999999999999999999999999999991+99", cache.collections)
        self.assertIn("99999999999999999999999999999992+99", cache.collections)
        self.assertEqual((524287*128)*2, cache.total)

        c1 = cache.get("99999999999999999999999999999991+99")
        self.assertIn("99999999999999999999999999999991+99", cache.collections)
        self.assertIn("99999999999999999999999999999992+99", cache.collections)
        self.assertEqual((524287*128)*2, cache.total)

        c3 = cache.get("99999999999999999999999999999993+99")
        self.assertIn("99999999999999999999999999999991+99", cache.collections)
        self.assertIn("99999999999999999999999999999992+99", cache.collections)
        self.assertEqual((524287*128)*3, cache.total)

        c4 = cache.get("99999999999999999999999999999994+99")
        self.assertIn("99999999999999999999999999999991+99", cache.collections)
        self.assertIn("99999999999999999999999999999992+99", cache.collections)
        self.assertEqual((524287*128)*4, cache.total)

        c5 = cache.get("99999999999999999999999999999995+99")
        self.assertIn("99999999999999999999999999999991+99", cache.collections)
        self.assertNotIn("99999999999999999999999999999992+99", cache.collections)
        self.assertEqual((524287*128)*4, cache.total)

        c6 = cache.get("99999999999999999999999999999996+99")
        self.assertNotIn("99999999999999999999999999999991+99", cache.collections)
        self.assertNotIn("99999999999999999999999999999992+99", cache.collections)
        self.assertEqual((524287*128)*4, cache.total)
