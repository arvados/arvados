# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import arvados
import arvados.errors
import time
import json

while True:
    try:
        api = arvados.api()
        break
    except arvados.errors.ApiError:
        time.sleep(2)

existing = api.users().list(filters=[["email", "=", "test@example.com"],
                                     ["is_active", "=", True]], limit=1).execute()
if existing["items"]:
    u = existing["items"][0]
else:
    u = api.users().create(body={
        'first_name': 'Test',
        'last_name': 'User',
        'email': 'test@example.com',
        'is_admin': False
    }).execute()
    api.users().activate(uuid=u["uuid"]).execute()

tok = api.api_client_authorizations().create(body={
    "api_client_authorization": {
        "owner_uuid": u["uuid"]
    }
}).execute()

with open("cwl.output.json", "w") as f:
    json.dump({
        "test_user_uuid": u["uuid"],
        "test_user_token": "v2/%s/%s" % (tok["uuid"], tok["api_token"])
    }, f)
