#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import, print_function

import unittest

import libcloud.common.types as cloud_types
import mock

import arvnodeman.computenode.driver as driver_base
import arvnodeman.status as status
import arvnodeman.config as config
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


    class TestBaseComputeNodeDriver(driver_base.BaseComputeNodeDriver):
        def arvados_create_kwargs(self, size, arvados_node):
            return {'name': arvados_node}


    def test_create_node_only_cloud_errors_are_counted(self):
        status.tracker.update({'create_node_errors': 0})
        errors = [(config.CLOUD_ERRORS[0], True), (KeyError, False)]
        self.driver_mock().list_images.return_value = []
        driver = self.TestBaseComputeNodeDriver({}, {}, {}, self.driver_mock)
        error_count = 0
        for an_error, is_cloud_error in errors:
            self.driver_mock().create_node.side_effect = an_error
            with self.assertRaises(an_error):
                driver.create_node(testutil.MockSize(1), 'id_1')
            if is_cloud_error:
                error_count += 1
            self.assertEqual(error_count, status.tracker.get('create_node_errors'))

    def test_list_nodes_only_cloud_errors_are_counted(self):
        status.tracker.update({'list_nodes_errors': 0})
        errors = [(config.CLOUD_ERRORS[0], True), (KeyError, False)]
        driver = self.TestBaseComputeNodeDriver({}, {}, {}, self.driver_mock)
        error_count = 0
        for an_error, is_cloud_error in errors:
            self.driver_mock().list_nodes.side_effect = an_error
            with self.assertRaises(an_error):
                driver.list_nodes()
            if is_cloud_error:
                error_count += 1
            self.assertEqual(error_count, status.tracker.get('list_nodes_errors'))

    def test_destroy_node_only_cloud_errors_are_counted(self):
        status.tracker.update({'destroy_node_errors': 0})
        errors = [(config.CLOUD_ERRORS[0], True), (KeyError, False)]
        self.driver_mock().list_nodes.return_value = [testutil.MockSize(1)]
        driver = self.TestBaseComputeNodeDriver({}, {}, {}, self.driver_mock)
        error_count = 0
        for an_error, is_cloud_error in errors:
            self.driver_mock().destroy_node.side_effect = an_error
            with self.assertRaises(an_error):
                driver.destroy_node(testutil.MockSize(1))
            if is_cloud_error:
                error_count += 1
            self.assertEqual(error_count, status.tracker.get('destroy_node_errors'))
