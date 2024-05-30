# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import arvados
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
