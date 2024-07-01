# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import arvados
import arvados.util
from . import run_test_server
from .test_util import KeysetTestHelper

class ComputedPermissionTest(run_test_server.TestCaseWithServers):
    def test_computed_permission(self):
        run_test_server.authorize_with('admin')
        api_client = arvados.api('v1')
        active_user_uuid = run_test_server.fixture('users')['active']['uuid']
        resp = api_client.computed_permissions().list(
            filters=[['user_uuid', '=', active_user_uuid]],
        ).execute()
        assert len(resp['items']) > 0
        for item in resp['items']:
            assert item['user_uuid'] == active_user_uuid

    def test_keyset_list_all(self):
        run_test_server.authorize_with('admin')
        api_client = arvados.api('v1')
        seen = {}
        for item in arvados.util.keyset_list_all(api_client.computed_permissions().list, order_key='user_uuid', key_fields=('user_uuid', 'target_uuid')):
            assert (item['user_uuid'], item['target_uuid']) not in seen
            seen[(item['user_uuid'], item['target_uuid'])] = True

    def test_iter_computed_permissions(self):
        run_test_server.authorize_with('admin')
        api_client = arvados.api('v1')
        seen = {}
        for item in arvados.util.iter_computed_permissions(api_client.computed_permissions().list):
            assert item['perm_level']
            assert (item['user_uuid'], item['target_uuid']) not in seen
            seen[(item['user_uuid'], item['target_uuid'])] = True

    def test_iter_computed_permissions_defaults(self):
        ks = KeysetTestHelper([[
            {"limit": 1000, "count": "none", "order": ["user_uuid asc", "target_uuid asc"], "filters": []},
            {"items": [{"user_uuid": "u", "target_uuid": "t"}]}
        ], [
            {"limit": 1000, "count": "none", "order": ["user_uuid asc", "target_uuid asc"], "filters": [['user_uuid', '=', 'u'], ['target_uuid', '>', 't']]},
            {"items": []},
        ], [
            {"limit": 1000, "count": "none", "order": ["user_uuid asc", "target_uuid asc"], "filters": [['user_uuid', '>', 'u']]},
            {"items": []},
        ]])
        ls = list(arvados.util.iter_computed_permissions(ks.fn))
        assert ls == ks.expect[0][1]['items']

    def test_iter_computed_permissions_order_key(self):
        ks = KeysetTestHelper([[
            {"limit": 1000, "count": "none", "order": ["target_uuid desc", "user_uuid desc"], "filters": []},
            {"items": [{"user_uuid": "u", "target_uuid": "t"}]}
        ], [
            {"limit": 1000, "count": "none", "order": ["target_uuid desc", "user_uuid desc"], "filters": [['target_uuid', '=', 't'], ['user_uuid', '<', 'u']]},
            {"items": []},
        ], [
            {"limit": 1000, "count": "none", "order": ["target_uuid desc", "user_uuid desc"], "filters": [['target_uuid', '<', 't']]},
            {"items": []},
        ]])
        ls = list(arvados.util.iter_computed_permissions(ks.fn, order_key='target_uuid', ascending=False))
        assert ls == ks.expect[0][1]['items']

    def test_iter_computed_permissions_num_retries(self):
        ks = KeysetTestHelper([[
            {"limit": 1000, "count": "none", "order": ["user_uuid asc", "target_uuid asc"], "filters": []},
            {"items": []}
        ]], expect_num_retries=33)
        assert list(arvados.util.iter_computed_permissions(ks.fn, num_retries=33)) == []

    def test_iter_computed_permissions_invalid_key_fields(self):
        ks = KeysetTestHelper([])
        with self.assertRaises(arvados.errors.ArgumentError) as exc:
            _ = list(arvados.util.iter_computed_permissions(ks.fn, key_fields=['target_uuid', 'perm_level']))
        assert exc.exception.args[0] == 'key_fields can have at most one entry that is not order_key'
