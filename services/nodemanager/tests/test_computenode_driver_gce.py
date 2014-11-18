#!/usr/bin/env python

from __future__ import absolute_import, print_function

import time
import unittest

import mock

import arvnodeman.computenode.driver.gce as gce
from . import testutil

class GCEComputeNodeDriverTestCase(unittest.TestCase):
    def setUp(self):
        self.driver_mock = mock.MagicMock(name='driver_mock')

    def new_driver(self, auth_kwargs={}, list_kwargs={}, create_kwargs={}):
        create_kwargs.setdefault('ping_host', '100::')
        return gce.ComputeNodeDriver(
            auth_kwargs, list_kwargs, create_kwargs,
            driver_class=self.driver_mock)

    def test_driver_instantiation(self):
        kwargs = {'user_id': 'foo'}
        driver = self.new_driver(auth_kwargs=kwargs)
        self.assertTrue(self.driver_mock.called)
        self.assertEqual(kwargs, self.driver_mock.call_args[1])

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

    def test_generate_metadata_for_new_arvados_node(self):
        arv_node = testutil.arvados_node_mock(8)
        driver = self.new_driver(list_kwargs={'list': 'test'})
        self.assertEqual({'ex_metadata': {'list': 'test'}},
                         driver.arvados_create_kwargs(arv_node))

    def test_tags_set_default_hostname_from_new_arvados_node(self):
        arv_node = testutil.arvados_node_mock(hostname=None)
        cloud_node = testutil.cloud_node_mock(1)
        driver = self.new_driver()
        driver.sync_node(cloud_node, arv_node)
        tag_mock = self.driver_mock().ex_set_node_tags
        self.assertTrue(tag_mock.called)
        self.assertEqual(['hostname-dynamic.compute.zzzzz.arvadosapi.com'],
                         tag_mock.call_args[0][1])

    def test_sync_node_sets_static_hostname(self):
        arv_node = testutil.arvados_node_mock(1)
        cloud_node = testutil.cloud_node_mock(2)
        driver = self.new_driver()
        driver.sync_node(cloud_node, arv_node)
        tag_mock = self.driver_mock().ex_set_node_tags
        self.assertTrue(tag_mock.called)
        self.assertEqual(['hostname-compute1.zzzzz.arvadosapi.com'],
                         tag_mock.call_args[0][1])

    def test_node_create_time(self):
        refsecs = int(time.time())
        reftuple = time.gmtime(refsecs)
        node = testutil.cloud_node_mock()
        node.extra = {'launch_time': time.strftime('%Y-%m-%dT%H:%M:%S.000Z',
                                                   reftuple)}
        self.assertEqual(refsecs, gce.ComputeNodeDriver.node_start_time(node))

    def test_generate_metadata_for_new_arvados_node(self):
        arv_node = testutil.arvados_node_mock(8)
        driver = self.new_driver(list_kwargs={'list': 'test'})
        self.assertEqual({'ex_metadata': {'list': 'test'}},
                         driver.arvados_create_kwargs(arv_node))

    def test_deliver_ssh_key_in_metadata(self):
        test_ssh_key = 'ssh-rsa-foo'
        arv_node = testutil.arvados_node_mock(1)
        with mock.patch('__builtin__.open', mock.mock_open(read_data=test_ssh_key)) as mock_file:
            driver = self.new_driver(create_kwargs={'ssh_key': 'ssh-key-file'})
        mock_file.assert_called_once_with('ssh-key-file')
        self.assertEqual({'ex_metadata': {'sshKeys': 'root:ssh-rsa-foo'}},
                         driver.arvados_create_kwargs(arv_node))

    def test_create_driver_with_service_accounts(self):
        srv_acct_config = { 'service_accounts': '{ "email": "foo@bar", "scopes":["storage-full"]}' }
        arv_node = testutil.arvados_node_mock(1)
        driver = self.new_driver(create_kwargs=srv_acct_config)
        create_kwargs = driver.arvados_create_kwargs(arv_node)
        self.assertEqual({u'email': u'foo@bar', u'scopes': [u'storage-full']},
                         create_kwargs['ex_service_accounts'])
