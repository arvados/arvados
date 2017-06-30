#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import, print_function

import unittest

import libcloud.common.types as cloud_types
import mock

import arvnodeman.computenode.driver as driver_base
from . import testutil

class ComputeNodeDriverTestCase(unittest.TestCase):
    def setUp(self):
        self.driver_mock = mock.MagicMock(name='driver_mock')
        driver_base.BaseComputeNodeDriver.SEARCH_CACHE = {}

    def test_search_for_now_uses_public_method(self):
        image = testutil.cloud_object_mock(1)
        self.driver_mock().list_images.return_value = [image]
        driver = driver_base.BaseComputeNodeDriver({}, {}, {}, self.driver_mock)
        self.assertIs(image, driver.search_for_now('id_1', 'list_images'))
        self.assertEqual(1, self.driver_mock().list_images.call_count)

    def test_search_for_now_uses_private_method(self):
        net = testutil.cloud_object_mock(1)
        self.driver_mock().ex_list_networks.return_value = [net]
        driver = driver_base.BaseComputeNodeDriver({}, {}, {}, self.driver_mock)
        self.assertIs(net, driver.search_for_now('id_1', 'ex_list_networks'))
        self.assertEqual(1, self.driver_mock().ex_list_networks.call_count)

    def test_search_for_now_raises_ValueError_on_zero_results(self):
        self.driver_mock().list_images.return_value = []
        driver = driver_base.BaseComputeNodeDriver({}, {}, {}, self.driver_mock)
        with self.assertRaises(ValueError) as test:
            driver.search_for_now('id_1', 'list_images')

    def test_search_for_now_raises_ValueError_on_extra_results(self):
        image = testutil.cloud_object_mock(1)
        self.driver_mock().list_images.return_value = [image, image]
        driver = driver_base.BaseComputeNodeDriver({}, {}, {}, self.driver_mock)
        with self.assertRaises(ValueError) as test:
            driver.search_for_now('id_1', 'list_images')

    def test_search_for_now_does_not_cache_results(self):
        image1 = testutil.cloud_object_mock(1)
        image2 = testutil.cloud_object_mock(1)
        self.driver_mock().list_images.side_effect = [[image1], [image2]]
        driver = driver_base.BaseComputeNodeDriver({}, {}, {}, self.driver_mock)
        self.assertIsNot(driver.search_for_now('id_1', 'list_images'),
                         driver.search_for_now('id_1', 'list_images'))
        self.assertEqual(2, self.driver_mock().list_images.call_count)

    def test_search_for_returns_cached_results(self):
        image1 = testutil.cloud_object_mock(1)
        image2 = testutil.cloud_object_mock(1)
        self.driver_mock().list_images.side_effect = [[image1], [image2]]
        driver = driver_base.BaseComputeNodeDriver({}, {}, {}, self.driver_mock)
        self.assertIs(driver.search_for('id_1', 'list_images'),
                      driver.search_for('id_1', 'list_images'))
        self.assertEqual(1, self.driver_mock().list_images.call_count)
