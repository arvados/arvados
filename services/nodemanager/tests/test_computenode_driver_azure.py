#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import, print_function

import ssl
import time
import unittest

import libcloud.common.types as cloud_types
import mock

import arvnodeman.computenode.driver.azure as azure
from . import testutil

class AzureComputeNodeDriverTestCase(testutil.DriverTestMixin, unittest.TestCase):
    TEST_CLASS = azure.ComputeNodeDriver

    def new_driver(self, auth_kwargs={}, list_kwargs={}, create_kwargs={}):
        list_kwargs.setdefault("ex_resource_group", "TestResourceGroup")
        return super(AzureComputeNodeDriverTestCase, self).new_driver(auth_kwargs, list_kwargs, create_kwargs)

    def test_driver_instantiation(self):
        kwargs = {'key': 'testkey'}
        driver = self.new_driver(auth_kwargs=kwargs)
        self.assertTrue(self.driver_mock.called)
        self.assertEqual(kwargs, self.driver_mock.call_args[1])

    def test_create_image_loaded_at_initialization(self):
        get_method = self.driver_mock().get_image
        get_method.return_value = testutil.cloud_object_mock('id_b')
        driver = self.new_driver(create_kwargs={'image': 'id_b'})
        self.assertEqual(1, get_method.call_count)

    def test_create_includes_ping(self):
        arv_node = testutil.arvados_node_mock(info={'ping_secret': 'ssshh'})
        arv_node["hostname"] = None
        driver = self.new_driver()
        driver.create_node(testutil.MockSize(1), arv_node)
        create_method = self.driver_mock().create_node
        self.assertTrue(create_method.called)
        self.assertIn('ping_secret=ssshh',
                      create_method.call_args[1].get('ex_tags', {}).get('arv-ping-url', ""))

    def test_create_includes_arvados_node_size(self):
        arv_node = testutil.arvados_node_mock()
        arv_node["hostname"] = None
        size = testutil.MockSize(1)
        driver = self.new_driver()
        driver.create_node(size, arv_node)
        create_method = self.driver_mock().create_node
        self.assertTrue(create_method.called)
        self.assertIn(
            ('arvados_node_size', size.id),
            create_method.call_args[1].get('ex_tags', {'tags': 'missing'}).items()
        )

    def test_name_from_new_arvados_node(self):
        arv_node = testutil.arvados_node_mock(hostname=None)
        driver = self.new_driver()
        self.assertEqual('compute-000000000000063-zzzzz',
                         driver.arvados_create_kwargs(testutil.MockSize(1), arv_node)['name'])

    def check_node_tagged(self, cloud_node, expected_tags):
        tag_mock = self.driver_mock().ex_create_tags
        self.assertTrue(tag_mock.called)
        self.assertIs(cloud_node, tag_mock.call_args[0][0])
        self.assertEqual(expected_tags, tag_mock.call_args[0][1])

    def test_node_create_time(self):
        refsecs = int(time.time())
        reftuple = time.gmtime(refsecs)
        node = testutil.cloud_node_mock()
        node.extra = {'tags': {'booted_at': time.strftime('%Y-%m-%dT%H:%M:%S.000Z',
                                                   reftuple)}}
        self.assertEqual(refsecs, azure.ComputeNodeDriver.node_start_time(node))

    def test_node_fqdn(self):
        name = 'fqdntest.zzzzz.arvadosapi.com'
        node = testutil.cloud_node_mock()
        node.extra = {'tags': {"hostname": name}}
        self.assertEqual(name, azure.ComputeNodeDriver.node_fqdn(node))

    def test_sync_node(self):
        arv_node = testutil.arvados_node_mock(1)
        cloud_node = testutil.cloud_node_mock(2)
        driver = self.new_driver()
        driver.sync_node(cloud_node, arv_node)
        self.check_node_tagged(cloud_node,
                               {'hostname': 'compute1.zzzzz.arvadosapi.com'})

    def test_custom_data(self):
        arv_node = testutil.arvados_node_mock(hostname=None)
        driver = self.new_driver()
        self.assertEqual("""#!/bin/sh
mkdir -p    /var/tmp/arv-node-data/meta-data
echo 'https://100::/arvados/v1/nodes/zzzzz-yyyyy-000000000000063/ping?ping_secret=defaulttestsecret' > /var/tmp/arv-node-data/arv-ping-url
echo compute-000000000000063-zzzzz > /var/tmp/arv-node-data/meta-data/instance-id
echo z1.test > /var/tmp/arv-node-data/meta-data/instance-type
""",
                         driver.arvados_create_kwargs(testutil.MockSize(1), arv_node)['ex_customdata'])

    def test_list_nodes_ignores_nodes_without_tags(self):
        driver = self.new_driver(create_kwargs={"tag_arvados-class": "dynamic-compute"})
        # Mock cloud node without tags
        nodelist = [testutil.cloud_node_mock(1)]
        self.driver_mock().list_nodes.return_value = nodelist
        n = driver.list_nodes()
        self.assertEqual([], n)

    def test_create_raises_but_actually_succeeded(self):
        arv_node = testutil.arvados_node_mock(1, hostname=None)
        driver = self.new_driver(create_kwargs={"tag_arvados-class": "dynamic-compute"})
        nodelist = [testutil.cloud_node_mock(1, tags={"arvados-class": "dynamic-compute"})]
        nodelist[0].name = 'compute-000000000000001-zzzzz'
        self.driver_mock().list_nodes.return_value = nodelist
        self.driver_mock().create_node.side_effect = IOError
        n = driver.create_node(testutil.MockSize(1), arv_node)
        self.assertEqual('compute-000000000000001-zzzzz', n.name)

    def test_ex_fetch_nic_false(self):
        arv_node = testutil.arvados_node_mock(1, hostname=None)
        driver = self.new_driver(create_kwargs={"tag_arvados-class": "dynamic-compute"})
        nodelist = [testutil.cloud_node_mock(1, tags={"arvados-class": "dynamic-compute"})]
        nodelist[0].name = 'compute-000000000000001-zzzzz'
        self.driver_mock().list_nodes.return_value = nodelist
        n = driver.list_nodes()
        self.assertEqual(nodelist, n)
        self.driver_mock().list_nodes.assert_called_with(ex_fetch_nic=False, ex_fetch_power_state=False, ex_resource_group='TestResourceGroup')

    def test_create_can_find_node_after_timeout(self):
        super(AzureComputeNodeDriverTestCase,
              self).test_create_can_find_node_after_timeout(
                  create_kwargs={'tag_arvados-class': 'test'},
                  node_extra={'tags': {'arvados-class': 'test'}})

    def test_node_found_after_timeout_has_fixed_size(self):
        size = testutil.MockSize(4)
        node_props = {'hardwareProfile': {'vmSize': size.id}}
        cloud_node = testutil.cloud_node_mock(tags={'arvados-class': 'test'}, properties=node_props)
        cloud_node.size = None
        self.check_node_found_after_timeout_has_fixed_size(
            size, cloud_node, {'tag_arvados-class': 'test'})
