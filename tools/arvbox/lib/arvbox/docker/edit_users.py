#!/opt/arvados-py/bin/python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import ruamel.yaml
import sys
import getpass
import os

def print_help():
    print("%s <path/to/config.yaml> <clusterid> add <username> <email> [pass]" % (sys.argv[0]))
    print("%s <path/to/config.yaml> <clusterid> remove <username>" % (" " * len(sys.argv[0])))
    print("%s <path/to/config.yaml> <clusterid> list" % (" " * len(sys.argv[0])))
    exit()

if len(sys.argv) < 4:
    print_help()

fn = sys.argv[1]
cl = sys.argv[2]
op = sys.argv[3]

if op == "remove" and len(sys.argv) < 5:
    print_help()
if op == "add" and len(sys.argv) < 6:
    print_help()

if op in ("add", "remove"):
    user = sys.argv[4]

if not os.path.exists(fn):
    open(fn, "w").close()

with open(fn, "r") as f:
    conf = ruamel.yaml.round_trip_load(f)

if not conf:
    conf = {}

conf["Clusters"] = conf.get("Clusters", {})
conf["Clusters"][cl] = conf["Clusters"].get(cl, {})
conf["Clusters"][cl]["Login"] = conf["Clusters"][cl].get("Login", {})
conf["Clusters"][cl]["Login"]["Test"] = conf["Clusters"][cl]["Login"].get("Test", {})
conf["Clusters"][cl]["Login"]["Test"]["Users"] = conf["Clusters"][cl]["Login"]["Test"].get("Users", {})

users_obj = conf["Clusters"][cl]["Login"]["Test"]["Users"]

if op == "add":
    email = sys.argv[5]
    if len(sys.argv) == 7:
        p = sys.argv[6]
    else:
        p = getpass.getpass("Password for %s: " % user)

    users_obj[user] = {
        "Email": email,
        "Password": p
    }
    print("Added %s" % user)
elif op == "remove":
    del users_obj[user]
    print("Removed %s" % user)
elif op == "list":
    print(ruamel.yaml.round_trip_dump(users_obj))
else:
    print("Operations are 'add', 'remove' and 'list'")

with open(fn, "w") as f:
    f.write(ruamel.yaml.round_trip_dump(conf))
