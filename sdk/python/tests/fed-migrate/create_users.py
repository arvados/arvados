# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import arvados
import json
import sys

j = json.load(open(sys.argv[1]))

apiA = arvados.api(host=j["arvados_api_hosts"][0], token=j["superuser_tokens"][0], insecure=True)
apiB = arvados.api(host=j["arvados_api_hosts"][1], token=j["superuser_tokens"][1], insecure=True)
apiC = arvados.api(host=j["arvados_api_hosts"][2], token=j["superuser_tokens"][2], insecure=True)

def maketoken(newtok):
    return 'v2/' + newtok["uuid"] + '/' + newtok["api_token"]

def get_user_data(case_nr, is_active=True):
    return {
        "email": "case{}@test".format(case_nr),
        "first_name": "Case{}".format(case_nr),
        "last_name": "Testuser",
        "is_active": is_active
    }

# case 1
# user only exists on cluster A
apiA.users().create(body={"user": get_user_data(case_nr=1)}).execute()

# case 2
# user exists on cluster A and has remotes on B and C
case2 = apiA.users().create(body={"user": get_user_data(case_nr=2)}).execute()
newtok = apiA.api_client_authorizations().create(body={
    "api_client_authorization": {'owner_uuid': case2["uuid"]}}).execute()
arvados.api(host=j["arvados_api_hosts"][1], token=maketoken(newtok), insecure=True).users().current().execute()
arvados.api(host=j["arvados_api_hosts"][2], token=maketoken(newtok), insecure=True).users().current().execute()

# case 3
# user only exists on cluster B
case3 = apiB.users().create(body={"user": get_user_data(case_nr=3)}).execute()

# case 4
# user only exists on cluster B and has remotes on A and C
case4 = apiB.users().create(body={"user": get_user_data(case_nr=4)}).execute()
newtok = apiB.api_client_authorizations().create(body={
    "api_client_authorization": {'owner_uuid': case4["uuid"]}}).execute()
arvados.api(host=j["arvados_api_hosts"][0], token=maketoken(newtok), insecure=True).users().current().execute()
arvados.api(host=j["arvados_api_hosts"][2], token=maketoken(newtok), insecure=True).users().current().execute()


# case 5
# user exists on both cluster A and B
case5 = apiA.users().create(body={"user": get_user_data(case_nr=5)}).execute()
case5 = apiB.users().create(body={"user": get_user_data(case_nr=5)}).execute()

# case 6
# user exists on both cluster A and B, with remotes on A, B and C
case6_A = apiA.users().create(body={"user": get_user_data(case_nr=6)}).execute()
newtokA = apiA.api_client_authorizations().create(body={
    "api_client_authorization": {'owner_uuid': case6_A["uuid"]}}).execute()
arvados.api(host=j["arvados_api_hosts"][1], token=maketoken(newtokA), insecure=True).users().current().execute()
arvados.api(host=j["arvados_api_hosts"][2], token=maketoken(newtokA), insecure=True).users().current().execute()

case6_B = apiB.users().create(body={"user": get_user_data(case_nr=6)}).execute()
newtokB = apiB.api_client_authorizations().create(body={
    "api_client_authorization": {'owner_uuid': case6_B["uuid"]}}).execute()
arvados.api(host=j["arvados_api_hosts"][0], token=maketoken(newtokB), insecure=True).users().current().execute()
arvados.api(host=j["arvados_api_hosts"][2], token=maketoken(newtokB), insecure=True).users().current().execute()

# case 7
# user exists on both cluster B and A, with remotes on A, B and C
case7_B = apiB.users().create(body={"user": get_user_data(case_nr=7)}).execute()
newtokB = apiB.api_client_authorizations().create(body={
    "api_client_authorization": {'owner_uuid': case7_B["uuid"]}}).execute()
arvados.api(host=j["arvados_api_hosts"][0], token=maketoken(newtokB), insecure=True).users().current().execute()
arvados.api(host=j["arvados_api_hosts"][2], token=maketoken(newtokB), insecure=True).users().current().execute()

case7_A = apiA.users().create(body={"user": get_user_data(case_nr=7)}).execute()
newtokA = apiA.api_client_authorizations().create(body={
    "api_client_authorization": {'owner_uuid': case7_A["uuid"]}}).execute()
arvados.api(host=j["arvados_api_hosts"][1], token=maketoken(newtokA), insecure=True).users().current().execute()
arvados.api(host=j["arvados_api_hosts"][2], token=maketoken(newtokA), insecure=True).users().current().execute()

# case 8
# user exists on both cluster B and C, with remotes on A, B and C
case8_B = apiB.users().create(body={"user": get_user_data(case_nr=8)}).execute()
newtokB = apiB.api_client_authorizations().create(body={
    "api_client_authorization": {'owner_uuid': case8_B["uuid"]}}).execute()
arvados.api(host=j["arvados_api_hosts"][0], token=maketoken(newtokB), insecure=True).users().current().execute()
arvados.api(host=j["arvados_api_hosts"][2], token=maketoken(newtokB), insecure=True).users().current().execute()

case8_C = apiC.users().create(body={"user": get_user_data(case_nr=8)}).execute()
newtokC = apiC.api_client_authorizations().create(body={
    "api_client_authorization": {'owner_uuid': case8_C["uuid"]}}).execute()
arvados.api(host=j["arvados_api_hosts"][0], token=maketoken(newtokC), insecure=True).users().current().execute()
arvados.api(host=j["arvados_api_hosts"][1], token=maketoken(newtokC), insecure=True).users().current().execute()

# case 9
# user only exists on cluster B, but is inactive
case9 = apiB.users().create(body={"user": get_user_data(case_nr=9, is_active=False)}).execute()
