import arvados
import json
import sys

j = json.load(open(sys.argv[1]))

apiA = arvados.api(host=j["arvados_api_hosts"][0], token=j["superuser_tokens"][0], insecure=True)
apiB = arvados.api(host=j["arvados_api_hosts"][1], token=j["superuser_tokens"][1], insecure=True)
apiC = arvados.api(host=j["arvados_api_hosts"][2], token=j["superuser_tokens"][2], insecure=True)

users = apiA.users().list().execute()

assert len(users["items"]) == 10

by_username = {}

for i in range(1, 9):
    found = False
    for u in users["items"]:
        if u["username"] == ("case%d" % i) and u["email"] == ("case%d@test" % i):
            found = True
            by_username[u["username"]] = u["uuid"]
    assert found

users = apiB.users().list().execute()
assert len(users["items"]) == 10

for i in range(2, 9):
    found = False
    for u in users["items"]:
        if u["username"] == ("case%d" % i) and u["email"] == ("case%d@test" % i) and u["uuid"] == by_username[u["username"]]:
            found = True
    assert found

users = apiC.users().list().execute()
assert len(users["items"]) == 10

for i in range(2, 9):
    found = False
    for u in users["items"]:
        if u["username"] == ("case%d" % i) and u["email"] == ("case%d@test" % i) and u["uuid"] == by_username[u["username"]]:
            found = True
    assert found

print("Passed checks")
