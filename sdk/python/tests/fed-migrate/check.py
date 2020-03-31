import arvados
import json
import sys

j = json.load(open(sys.argv[1]))

apiA = arvados.api(host=j["arvados_api_hosts"][0], token=j["superuser_tokens"][0], insecure=True)
apiB = arvados.api(host=j["arvados_api_hosts"][1], token=j["superuser_tokens"][1], insecure=True)
apiC = arvados.api(host=j["arvados_api_hosts"][2], token=j["superuser_tokens"][2], insecure=True)

###
### Check users on API server "A" (the LoginCluster) ###
###

users = apiA.users().list(bypass_federation=True).execute()

assert len(users["items"]) == 11

by_username = {}

for i in range(1, 10):
    found = False
    for u in users["items"]:
        if u["username"] == ("case%d" % i) and u["email"] == ("case%d@test" % i):
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


###
### Check users on API server "B" (federation member) ###
###
users = apiB.users().list(bypass_federation=True).execute()
assert len(users["items"]) == 11

for i in range(2, 9):
    found = False
    for u in users["items"]:
        if (u["username"] == ("case%d" % i) and u["email"] == ("case%d@test" % i) and
            u["uuid"] == by_username[u["username"]] and u["is_active"] is True):
            found = True
    assert found, "Not found case%i" % i

found = False
for u in users["items"]:
    if (u["username"] == "case9" and u["email"] == "case9@test" and
        u["uuid"] == by_username[u["username"]] and u["is_active"] is False):
        found = True
assert found


###
### Check users on API server "C" (federation member) ###
###
users = apiC.users().list(bypass_federation=True).execute()
assert len(users["items"]) == 8

for i in (2, 4, 6, 7, 8):
    found = False
    for u in users["items"]:
        if (u["username"] == ("case%d" % i) and u["email"] == ("case%d@test" % i) and
            u["uuid"] == by_username[u["username"]] and u["is_active"] is True):
            found = True
    assert found

# cases 3, 5, 9 involve users that have never accessed cluster C so
# there's nothing to migrate.
for i in (3, 5, 9):
    found = False
    for u in users["items"]:
        if (u["username"] == ("case%d" % i) and u["email"] == ("case%d@test" % i) and
            u["uuid"] == by_username[u["username"]] and u["is_active"] is True):
            found = True
    assert not found

print("Passed checks")
