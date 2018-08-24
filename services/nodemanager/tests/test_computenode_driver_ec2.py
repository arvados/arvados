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

import arvnodeman.computenode.driver.ec2 as ec2
from . import testutil

class EC2ComputeNodeDriverTestCase(testutil.DriverTestMixin, unittest.TestCase):
    TEST_CLASS = ec2.ComputeNodeDriver

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

    def test_create_image_loaded_at_initialization(self):
        list_method = self.driver_mock().list_images
        list_method.return_value = [testutil.cloud_object_mock(c)
                                    for c in 'abc']
        driver = self.new_driver(create_kwargs={'image_id': 'id_b'})
        self.assertEqual(1, list_method.call_count)

    def test_create_includes_ping_secret(self):
        arv_node = testutil.arvados_node_mock(info={'ping_secret': 'ssshh'})
        driver = self.new_driver()
        driver.create_node(testutil.MockSize(1), arv_node)
        create_method = self.driver_mock().create_node
        self.assertTrue(create_method.called)
        self.assertIn('ping_secret=ssshh',
                      create_method.call_args[1].get('ex_userdata',
                                                     'arg missing'))

    def test_create_includes_metadata(self):
        arv_node = testutil.arvados_node_mock()
        driver = self.new_driver(list_kwargs={'tag_test': 'testvalue'})
        driver.create_node(testutil.MockSize(1), arv_node)
        create_method = self.driver_mock().create_node
        self.assertTrue(create_method.called)
        self.assertIn(
            ('test', 'testvalue'),
            create_method.call_args[1].get('ex_metadata', {'arg': 'missing'}).items()
        )

    def test_create_includes_arvados_node_size(self):
        arv_node = testutil.arvados_node_mock()
        size = testutil.MockSize(1)
        driver = self.new_driver()
        driver.create_node(size, arv_node)
        create_method = self.driver_mock().create_node
        self.assertTrue(create_method.called)
        self.assertIn(
            ('arvados_node_size', size.id),
            create_method.call_args[1].get('ex_metadata', {'arg': 'missing'}).items()
        )

    def test_create_preemptible_instance(self):
        arv_node = testutil.arvados_node_mock()
        driver = self.new_driver()
        driver.create_node(testutil.MockSize(1, preemptible=True), arv_node)
        create_method = self.driver_mock().create_node
        self.assertTrue(create_method.called)
        self.assertEqual(
            True,
            create_method.call_args[1].get('ex_spot_market', 'arg missing')
        )

    def test_hostname_from_arvados_node(self):
        arv_node = testutil.arvados_node_mock(8)
        driver = self.new_driver()
        self.assertEqual('compute8.zzzzz.arvadosapi.com',
                         driver.arvados_create_kwargs(testutil.MockSize(1), arv_node)['name'])

    def test_default_hostname_from_new_arvados_node(self):
        arv_node = testutil.arvados_node_mock(hostname=None)
        driver = self.new_driver()
        self.assertEqual('dynamic.compute.zzzzz.arvadosapi.com',
                         driver.arvados_create_kwargs(testutil.MockSize(1), arv_node)['name'])

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
                               {'Name': 'compute1.zzzzz.arvadosapi.com'})

    def test_node_create_time(self):
        refsecs = int(time.time())
        reftuple = time.gmtime(refsecs)
        node = testutil.cloud_node_mock()
        node.extra = {'launch_time': time.strftime('%Y-%m-%dT%H:%M:%S.000Z',
                                                   reftuple)}
        self.assertEqual(refsecs, ec2.ComputeNodeDriver.node_start_time(node))

    def test_node_fqdn(self):
        name = 'fqdntest.zzzzz.arvadosapi.com'
        node = testutil.cloud_node_mock()
        node.name = name
        self.assertEqual(name, ec2.ComputeNodeDriver.node_fqdn(node))

    def test_create_ebs_volume(self):
        arv_node = testutil.arvados_node_mock()
        driver = self.new_driver()
        # libcloud/ec2 "disk" sizes are in GB, Arvados/SLURM "scratch" value is in MB
        size = testutil.MockSize(1)
        size.disk=5
        size.scratch=20000
        driver.create_node(size, arv_node)
        create_method = self.driver_mock().create_node
        self.assertTrue(create_method.called)
        self.assertEqual([{
            "DeviceName": "/dev/xvdt",
            "Ebs": {
                "DeleteOnTermination": True,
                "VolumeSize": 16,
                "VolumeType": "gp2"
            }}],
                         create_method.call_args[1].get('ex_blockdevicemappings'))

    def test_ebs_volume_not_needed(self):
        arv_node = testutil.arvados_node_mock()
        driver = self.new_driver()
        # libcloud/ec2 "disk" sizes are in GB, Arvados/SLURM "scratch" value is in MB
        size = testutil.MockSize(1)
        size.disk=80
        size.scratch=20000
        driver.create_node(size, arv_node)
        create_method = self.driver_mock().create_node
        self.assertTrue(create_method.called)
        self.assertIsNone(create_method.call_args[1].get('ex_blockdevicemappings'))

    def test_ebs_volume_too_big(self):
        arv_node = testutil.arvados_node_mock()
        driver = self.new_driver()
        # libcloud/ec2 "disk" sizes are in GB, Arvados/SLURM "scratch" value is in MB
        size = testutil.MockSize(1)
        size.disk=80
        size.scratch=20000000
        driver.create_node(size, arv_node)
        create_method = self.driver_mock().create_node
        self.assertTrue(create_method.called)
        self.assertEqual([{
            "DeviceName": "/dev/xvdt",
            "Ebs": {
                "DeleteOnTermination": True,
                "VolumeSize": 16384,
                "VolumeType": "gp2"
            }}],
                         create_method.call_args[1].get('ex_blockdevicemappings'))
