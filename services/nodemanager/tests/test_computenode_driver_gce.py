#!/usr/bin/env python

from __future__ import absolute_import, print_function

import json
import time
import unittest

import mock

import arvnodeman.computenode.driver.gce as gce
from . import testutil

class GCEComputeNodeDriverTestCase(testutil.DriverTestMixin, unittest.TestCase):
    TEST_CLASS = gce.ComputeNodeDriver

    def test_driver_instantiation(self):
        kwargs = {'user_id': 'foo'}
        driver = self.new_driver(auth_kwargs=kwargs)
        self.assertTrue(self.driver_mock.called)
        self.assertEqual(kwargs, self.driver_mock.call_args[1])

    def test_create_image_loaded_at_initialization_by_name(self):
        image_mocks = [testutil.cloud_object_mock(c) for c in 'abc']
        list_method = self.driver_mock().list_images
        list_method.return_value = image_mocks
        driver = self.new_driver(create_kwargs={'image': 'B'})
        self.assertEqual(1, list_method.call_count)

    def test_create_includes_ping_secret(self):
        arv_node = testutil.arvados_node_mock(info={'ping_secret': 'ssshh'})
        driver = self.new_driver()
        driver.create_node(testutil.MockSize(1), arv_node)
        metadata = self.driver_mock().create_node.call_args[1]['ex_metadata']
        self.assertIn('ping_secret=ssshh', metadata.get('arv-ping-url'))

    def test_create_sets_default_hostname(self):
        driver = self.new_driver()
        driver.create_node(testutil.MockSize(1),
                           testutil.arvados_node_mock(254, hostname=None))
        create_kwargs = self.driver_mock().create_node.call_args[1]
        self.assertEqual('compute-0000000000000fe-zzzzz',
                         create_kwargs.get('name'))
        self.assertEqual('dynamic.compute.zzzzz.arvadosapi.com',
                         create_kwargs.get('ex_metadata', {}).get('hostname'))

    def test_create_tags_from_list_tags(self):
        driver = self.new_driver(list_kwargs={'tags': 'testA, testB'})
        driver.create_node(testutil.MockSize(1), testutil.arvados_node_mock())
        self.assertEqual(['testA', 'testB'],
                         self.driver_mock().create_node.call_args[1]['ex_tags'])

    def test_list_nodes_requires_tags_match(self):
        # A node matches if our list tags are a subset of the node's tags.
        # Test behavior with no tags, no match, partial matches, different
        # order, and strict supersets.
        cloud_mocks = [
            testutil.cloud_node_mock(node_num, tags=tag_set)
            for node_num, tag_set in enumerate(
                [[], ['bad'], ['good'], ['great'], ['great', 'ok'],
                 ['great', 'good'], ['good', 'fantastic', 'great']])]
        cloud_mocks.append(testutil.cloud_node_mock())
        self.driver_mock().list_nodes.return_value = cloud_mocks
        driver = self.new_driver(list_kwargs={'tags': 'good, great'})
        self.assertItemsEqual(['5', '6'], [n.id for n in driver.list_nodes()])

    def build_gce_metadata(self, metadata_dict):
        # Convert a plain metadata dictionary to the GCE data structure.
        return {
            'kind': 'compute#metadata',
            'fingerprint': 'testprint',
            'items': [{'key': key, 'value': metadata_dict[key]}
                      for key in metadata_dict],
            }

    def check_sync_node_updates_hostname_tag(self, plain_metadata):
        start_metadata = self.build_gce_metadata(plain_metadata)
        arv_node = testutil.arvados_node_mock(1)
        cloud_node = testutil.cloud_node_mock(
            2, metadata=start_metadata.copy(),
            zone=testutil.cloud_object_mock('testzone'))
        driver = self.new_driver()
        driver.sync_node(cloud_node, arv_node)
        args, kwargs = self.driver_mock().connection.async_request.call_args
        self.assertEqual('/zones/TESTZONE/instances/2/setMetadata', args[0])
        for key in ['kind', 'fingerprint']:
            self.assertEqual(start_metadata[key], kwargs['data'][key])
        plain_metadata['hostname'] = 'compute1.zzzzz.arvadosapi.com'
        self.assertEqual(
            plain_metadata,
            {item['key']: item['value'] for item in kwargs['data']['items']})

    def test_sync_node_updates_hostname_tag(self):
        self.check_sync_node_updates_hostname_tag(
            {'testkey': 'testvalue', 'hostname': 'startvalue'})

    def test_sync_node_adds_hostname_tag(self):
        self.check_sync_node_updates_hostname_tag({'testkey': 'testval'})

    def test_sync_node_raises_exception_on_failure(self):
        arv_node = testutil.arvados_node_mock(8)
        cloud_node = testutil.cloud_node_mock(
            9, metadata={}, zone=testutil.cloud_object_mock('failzone'))
        mock_response = self.driver_mock().connection.async_request()
        mock_response.success.return_value = False
        mock_response.error = 'sync error test'
        driver = self.new_driver()
        with self.assertRaises(Exception) as err_check:
            driver.sync_node(cloud_node, arv_node)
        self.assertIs(err_check.exception.__class__, Exception)
        self.assertIn('sync error test', str(err_check.exception))

    def test_node_create_time_zero_for_unknown_nodes(self):
        node = testutil.cloud_node_mock()
        self.assertEqual(0, gce.ComputeNodeDriver.node_start_time(node))

    def test_node_create_time_for_known_node(self):
        node = testutil.cloud_node_mock(metadata=self.build_gce_metadata(
                {'booted_at': '1970-01-01T00:01:05Z'}))
        self.assertEqual(65, gce.ComputeNodeDriver.node_start_time(node))

    def test_node_create_time_recorded_when_node_boots(self):
        start_time = time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime())
        arv_node = testutil.arvados_node_mock()
        driver = self.new_driver()
        driver.create_node(testutil.MockSize(1), arv_node)
        metadata = self.driver_mock().create_node.call_args[1]['ex_metadata']
        self.assertLessEqual(start_time, metadata.get('booted_at'))

    def test_deliver_ssh_key_in_metadata(self):
        test_ssh_key = 'ssh-rsa-foo'
        arv_node = testutil.arvados_node_mock(1)
        with mock.patch('__builtin__.open',
                        mock.mock_open(read_data=test_ssh_key)) as mock_file:
            driver = self.new_driver(create_kwargs={'ssh_key': 'ssh-key-file'})
        mock_file.assert_called_once_with('ssh-key-file')
        driver.create_node(testutil.MockSize(1), arv_node)
        metadata = self.driver_mock().create_node.call_args[1]['ex_metadata']
        self.assertEqual('root:ssh-rsa-foo', metadata.get('sshKeys'))

    def test_create_driver_with_service_accounts(self):
        service_accounts = {'email': 'foo@bar', 'scopes': ['storage-full']}
        srv_acct_config = {'service_accounts': json.dumps(service_accounts)}
        arv_node = testutil.arvados_node_mock(1)
        driver = self.new_driver(create_kwargs=srv_acct_config)
        driver.create_node(testutil.MockSize(1), arv_node)
        self.assertEqual(
            service_accounts,
            self.driver_mock().create_node.call_args[1]['ex_service_accounts'])
