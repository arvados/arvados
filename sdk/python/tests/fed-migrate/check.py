# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import arvados
import arvados.errors
import json
import sys

j = json.load(open(sys.argv[1]))

apiA = arvados.api(host=j["arvados_api_hosts"][0], token=j["superuser_tokens"][0], insecure=True)
tok = apiA.api_client_authorizations().current().execute()
v2_token = "v2/%s/%s" % (tok["uuid"], tok["api_token"])

apiB = arvados.api(host=j["arvados_api_hosts"][1], token=v2_token, insecure=True)
apiC = arvados.api(host=j["arvados_api_hosts"][2], token=v2_token, insecure=True)

###
### Check users on API server "A" (the LoginCluster) ###
###
by_username = {}
def check_A(users):
    assert len(users["items"]) == 11

    for i in range(1, 10):
        found = False
        for u in users["items"]:
            if u["username"] == ("case%d" % i) and u["email"] == ("case%d@test" % i) and u["first_name"] == ("Case%d" % i) and u["last_name"] == "Testuser":
                found = True
                by_username[u["username"]] = u["uuid"]
        assert found

    # Should be active
    for i in (1, 2, 3, 4, 5, 6, 7, 8):
        found = False
        for u in users["items"]:
            if u["username"] == ("case%d" % i) and u["email"] == ("case%d@test" % i) and u["is_active"] is True:
                found = True
        assert found, "Not found case%i" % i

    # case9 should not be active
    found = False
    for u in users["items"]:
        if (u["username"] == "case9" and u["email"] == "case9@test" and
            u["uuid"] == by_username[u["username"]] and u["is_active"] is False):
            found = True
    assert found

users = apiA.users().list().execute()
check_A(users)

users = apiA.users().list(bypass_federation=True).execute()
check_A(users)

###
### Check users on API server "B" (federation member) ###
###

# check for expected migrations on B
users = apiB.users().list(bypass_federation=True).execute()
assert len(users["items"]) == 11

for i in range(2, 9):
    found = False
    for u in users["items"]:
        if (u["username"] == ("case%d" % i) and u["email"] == ("case%d@test" % i) and
            u["first_name"] == ("Case%d" % i) and u["last_name"] == "Testuser" and
            u["uuid"] == by_username[u["username"]] and u["is_active"] is True):
            found = True
    assert found, "Not found case%i" % i

found = False
for u in users["items"]:
    if (u["username"] == "case9" and u["email"] == "case9@test" and
        u["first_name"] == "Case9" and u["last_name"] == "Testuser" and
        u["uuid"] == by_username[u["username"]] and u["is_active"] is False):
        found = True
assert found

# check that federated user listing works
users = apiB.users().list().execute()
check_A(users)

###
### Check users on API server "C" (federation member) ###
###

# check for expected migrations on C
users = apiC.users().list(bypass_federation=True).execute()
assert len(users["items"]) == 8

for i in (2, 4, 6, 7, 8):
    found = False
    for u in users["items"]:
        if (u["username"] == ("case%d" % i) and u["email"] == ("case%d@test" % i) and
            u["first_name"] == ("Case%d" % i) and u["last_name"] == "Testuser" and
            u["uuid"] == by_username[u["username"]] and u["is_active"] is True):
            found = True
    assert found

# cases 3, 5, 9 involve users that have never accessed cluster C so
# there's nothing to migrate.
for i in (3, 5, 9):
    found = False
    for u in users["items"]:
        if (u["username"] == ("case%d" % i) and u["email"] == ("case%d@test" % i) and
            u["first_name"] == ("Case%d" % i) and u["last_name"] == "Testuser" and
            u["uuid"] == by_username[u["username"]] and u["is_active"] is True):
            found = True
    assert not found

# check that federated user listing works
users = apiC.users().list().execute()
check_A(users)


####
# bug 16683 tests

# Check that this query returns empty, instead of returning a 500 or
# 502 error.
# Yes, we're asking for a group from the users endpoint.  This is not a
# mistake, this is something workbench does to populate the sharing
# dialog.
clusterID_B = apiB.configs().get().execute()["ClusterID"]
i = apiB.users().list(filters=[["uuid", "in", ["%s-j7d0g-fffffffffffffff" % clusterID_B]]], count="none").execute()
assert len(i["items"]) == 0

# Check that we can create a project and give a remote user access to it

tok3 = apiA.api_client_authorizations().create(body={"api_client_authorization": {"owner_uuid": by_username["case3"]}}).execute()
tok4 = apiA.api_client_authorizations().create(body={"api_client_authorization": {"owner_uuid": by_username["case4"]}}).execute()

v2_token3 = "v2/%s/%s" % (tok3["uuid"], tok3["api_token"])
v2_token4 = "v2/%s/%s" % (tok4["uuid"], tok4["api_token"])

apiB_3 = arvados.api(host=j["arvados_api_hosts"][1], token=v2_token3, insecure=True)
apiB_4 = arvados.api(host=j["arvados_api_hosts"][1], token=v2_token4, insecure=True)

assert apiB_3.users().current().execute()["uuid"] == by_username["case3"]
assert apiB_4.users().current().execute()["uuid"] == by_username["case4"]

newproject = apiB_3.groups().create(body={"group_class": "project",
                                           "name":"fed test project"},
                                    ensure_unique_name=True).execute()

try:
    # Expect to fail
    apiB_4.groups().get(uuid=newproject["uuid"]).execute()
except arvados.errors.ApiError as e:
    if e.resp['status'] == '404':
        pass
    else:
        raise

l = apiB_3.links().create(body={"link_class": "permission",
                            "name":"can_read",
                            "tail_uuid": by_username["case4"],
                            "head_uuid": newproject["uuid"]}).execute()

# Expect to succeed
apiB_4.groups().get(uuid=newproject["uuid"]).execute()

# remove permission
apiB_3.links().delete(uuid=l["uuid"]).execute()

try:
    # Expect to fail again
    apiB_4.groups().get(uuid=newproject["uuid"]).execute()
except arvados.errors.ApiError as e:
    if e.resp['status'] == '404':
        pass
    else:
        raise

print("Passed checks")
