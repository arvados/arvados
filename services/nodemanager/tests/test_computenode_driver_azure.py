#!/usr/bin/env python

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
        list_method = self.driver_mock().list_images
        list_method.return_value = [testutil.cloud_object_mock(c)
                                    for c in 'abc']
        driver = self.new_driver(create_kwargs={'image': 'id_b'})
        self.assertEqual(1, list_method.call_count)

    def test_create_includes_ping_url(self):
        arv_node = testutil.arvados_node_mock(info={'ping_secret': 'ssshh'})
        driver = self.new_driver()
        driver.create_node(testutil.MockSize(1), arv_node)
        create_method = self.driver_mock().create_node
        self.assertTrue(create_method.called)
        print(create_method.call_args[1])
        self.assertIn('ping_secret=ssshh',
                      create_method.call_args[1].get('ex_tags', {}).get('arv-ping-url', ""))

    def test_name_from_new_arvados_node(self):
        arv_node = testutil.arvados_node_mock(hostname=None)
        driver = self.new_driver()
        self.assertEqual('compute-000000000000063-zzzzz',
                         driver.arvados_create_kwargs(arv_node)['name'])

    def check_node_tagged(self, cloud_node, expected_tags):
        tag_mock = self.driver_mock().ex_create_tags
        self.assertTrue(tag_mock.called)
        self.assertIs(cloud_node, tag_mock.call_args[0][0])
        self.assertEqual(expected_tags, tag_mock.call_args[0][1])

    def test_sync_node(self):
        arv_node = testutil.arvados_node_mock(1)
        cloud_node = testutil.cloud_node_mock(2)
        driver = self.new_driver()
        driver.sync_node(cloud_node, arv_node)
        self.check_node_tagged(cloud_node,
                               {'hostname': 'compute1.zzzzz.arvadosapi.com'})

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

    def test_cloud_exceptions(self):
        for error in [Exception("test exception"),
                      IOError("test exception"),
                      ssl.SSLError("test exception"),
                      cloud_types.LibcloudError("test exception")]:
            self.assertTrue(azure.ComputeNodeDriver.is_cloud_exception(error),
                            "{} not flagged as cloud exception".format(error))

    def test_noncloud_exceptions(self):
        self.assertFalse(
            azure.ComputeNodeDriver.is_cloud_exception(ValueError("test error")),
            "ValueError flagged as cloud exception")
