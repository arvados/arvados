#!/usr/bin/env python

from __future__ import absolute_import, print_function

import time
import unittest

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

    def test_tags_created_from_arvados_node(self):
        arv_node = testutil.arvados_node_mock(8)
        cloud_node = testutil.cloud_node_mock(8)
        driver = self.new_driver(list_kwargs={'tag:list': 'test'})
        self.assertEqual({'ex_metadata': {'list': 'test'},
                          'name': 'compute8.zzzzz.arvadosapi.com'},
                         driver.arvados_create_kwargs(arv_node))

    def test_tags_set_default_hostname_from_new_arvados_node(self):
        arv_node = testutil.arvados_node_mock(hostname=None)
        driver = self.new_driver()
        actual = driver.arvados_create_kwargs(arv_node)
        self.assertEqual('dynamic.compute.zzzzz.arvadosapi.com',
                         actual['name'])

    def test_sync_node(self):
        arv_node = testutil.arvados_node_mock(1)
        cloud_node = testutil.cloud_node_mock(2)
        driver = self.new_driver()
        driver.sync_node(cloud_node, arv_node)
        tag_mock = self.driver_mock().ex_create_tags
        self.assertTrue(tag_mock.called)
        self.assertEqual('compute1.zzzzz.arvadosapi.com',
                         tag_mock.call_args[0][1].get('Name', 'no name'))

    def test_node_create_time(self):
        refsecs = int(time.time())
        reftuple = time.gmtime(refsecs)
        node = testutil.cloud_node_mock()
        node.extra = {'launch_time': time.strftime('%Y-%m-%dT%H:%M:%S.000Z',
                                                   reftuple)}
        self.assertEqual(refsecs, ec2.ComputeNodeDriver.node_start_time(node))
