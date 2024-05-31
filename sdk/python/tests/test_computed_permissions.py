# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import arvados
import arvados.util
from . import run_test_server

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
        for item in arvados.util.keyset_list_all(api_client.computed_permissions().list, order_key='user_uuid'):
            import sys
            print(f"{item['user_uuid']} {item['target_uuid']}", file=sys.stderr)
            assert (item['user_uuid'], item['target_uuid']) not in seen
            seen[(item['user_uuid'], item['target_uuid'])] = True
