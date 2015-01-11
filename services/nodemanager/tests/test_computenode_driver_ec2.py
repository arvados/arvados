#!/usr/bin/env python

from __future__ import absolute_import, print_function

import ssl
import time
import unittest

import libcloud.common.types as cloud_types
import mock

import arvnodeman.computenode.driver.ec2 as ec2
from . import testutil

class EC2ComputeNodeDriverTestCase(unittest.TestCase):
    def setUp(self):
        self.driver_mock = mock.MagicMock(name='driver_mock')

    def new_driver(self, auth_kwargs={}, list_kwargs={}, create_kwargs={}):
        create_kwargs.setdefault('ping_host', '100::')
        return ec2.ComputeNodeDriver(
            auth_kwargs, list_kwargs, create_kwargs,
            driver_class=self.driver_mock)

    def test_driver_instantiation(self):
        kwargs = {'key': 'testkey'}
        driver = self.new_driver(auth_kwargs=kwargs)
        self.assertTrue(self.driver_mock.called)
        self.assertEqual(kwargs, self.driver_mock.call_args[1])

    def test_list_kwargs_become_filters(self):
        # We're also testing tag name translation.
        driver = self.new_driver(list_kwargs={'tag_test': 'true'})
        driver.list_nodes()
        list_method = self.driver_mock().list_nodes
        self.assertTrue(list_method.called)
        self.assertEqual({'tag:test': 'true'},
                          list_method.call_args[1].get('ex_filters'))

    def test_create_location_loaded_at_initialization(self):
        kwargs = {'location': 'testregion'}
        driver = self.new_driver(create_kwargs=kwargs)
        self.assertTrue(self.driver_mock().list_locations)

    def test_create_image_loaded_at_initialization(self):
        kwargs = {'image': 'testimage'}
        driver = self.new_driver(create_kwargs=kwargs)
        self.assertTrue(self.driver_mock().list_images)

    def test_create_includes_ping_secret(self):
        arv_node = testutil.arvados_node_mock(info={'ping_secret': 'ssshh'})
        driver = self.new_driver()
        driver.create_node(testutil.MockSize(1), arv_node)
        create_method = self.driver_mock().create_node
        self.assertTrue(create_method.called)
        self.assertIn('ping_secret=ssshh',
                      create_method.call_args[1].get('ex_userdata',
                                                     'arg missing'))

    def test_hostname_from_arvados_node(self):
        arv_node = testutil.arvados_node_mock(8)
        driver = self.new_driver()
        self.assertEqual('compute8.zzzzz.arvadosapi.com',
                         driver.arvados_create_kwargs(arv_node)['name'])

    def test_default_hostname_from_new_arvados_node(self):
        arv_node = testutil.arvados_node_mock(hostname=None)
        driver = self.new_driver()
        self.assertEqual('dynamic.compute.zzzzz.arvadosapi.com',
                         driver.arvados_create_kwargs(arv_node)['name'])

    def check_node_tagged(self, cloud_node, expected_tags):
        tag_mock = self.driver_mock().ex_create_tags
        self.assertTrue(tag_mock.called)
        self.assertIs(cloud_node, tag_mock.call_args[0][0])
        self.assertEqual(expected_tags, tag_mock.call_args[0][1])

    def test_post_create_node_tags_from_list_kwargs(self):
        expect_tags = {'key1': 'test value 1', 'key2': 'test value 2'}
        list_kwargs = {('tag_' + key): value
                       for key, value in expect_tags.iteritems()}
        list_kwargs['instance-state-name'] = 'running'
        cloud_node = testutil.cloud_node_mock()
        driver = self.new_driver(list_kwargs=list_kwargs)
        driver.post_create_node(cloud_node)
        self.check_node_tagged(cloud_node, expect_tags)

    def test_sync_node(self):
        arv_node = testutil.arvados_node_mock(1)
        cloud_node = testutil.cloud_node_mock(2)
        driver = self.new_driver()
        driver.sync_node(cloud_node, arv_node)
        self.check_node_tagged(cloud_node,
                               {'Name': 'compute1.zzzzz.arvadosapi.com'})

    def test_node_create_time(self):
        refsecs = int(time.time())
        reftuple = time.gmtime(refsecs)
        node = testutil.cloud_node_mock()
        node.extra = {'launch_time': time.strftime('%Y-%m-%dT%H:%M:%S.000Z',
                                                   reftuple)}
        self.assertEqual(refsecs, ec2.ComputeNodeDriver.node_start_time(node))

    def test_cloud_exceptions(self):
        for error in [Exception("test exception"),
                      IOError("test exception"),
                      ssl.SSLError("test exception"),
                      cloud_types.LibcloudError("test exception")]:
            self.assertTrue(ec2.ComputeNodeDriver.is_cloud_exception(error),
                            "{} not flagged as cloud exception".format(error))

    def test_noncloud_exceptions(self):
        self.assertFalse(
            ec2.ComputeNodeDriver.is_cloud_exception(ValueError("test error")),
            "ValueError flagged as cloud exception")
