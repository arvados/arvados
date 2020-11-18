#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import argparse
import sys

import arvados
import arvados.util
import datetime
import ciso8601

def parse_arguments(arguments):
    arg_parser = argparse.ArgumentParser()
    arg_parser.add_argument('--days', type=int, required=True)
    args = arg_parser.parse_args(arguments)
    return args

def getowner(arv, uuid, owners):
    if uuid is None:
        return None
    if uuid[6:11] == "tpzed":
        return uuid

    if uuid not in owners:
        try:
            gp = arv.groups().get(uuid=uuid).execute()
            owners[uuid] = gp["owner_uuid"]
        except:
            owners[uuid] = None

    return getowner(arv, owners[uuid], owners)

def getusername(arv, uuid):
    u = arv.users().get(uuid=uuid).execute()
    return "%s %s <%s> (%s)" % (u["first_name"], u["last_name"], u["email"], uuid)

def getname(u):
    return "\"%s\" (%s)" % (u["name"], u["uuid"])

def main(arguments=None):
    if arguments is None:
        arguments = sys.argv[1:]

    args = parse_arguments(arguments)

    arv = arvados.api()

    since = datetime.datetime.utcnow() - datetime.timedelta(days=args.days)

    print("Activity since %s\n" % (datetime.datetime.now() - datetime.timedelta(days=args.days)).isoformat())

    events = arvados.util.keyset_list_all(arv.logs().list, filters=[["created_at", ">=", since.isoformat()]])

    users = {}
    owners = {}

    for e in events:
        owner = getowner(arv, e["object_owner_uuid"], owners)
        users.setdefault(owner, [])
        event_at = ciso8601.parse_datetime(e["event_at"]).astimezone().isoformat()
        # loguuid = e["uuid"]
        loguuid = ""

        if e["event_type"] == "create" and e["object_uuid"][6:11] == "tpzed":
            users.setdefault(e["object_uuid"], [])
            users[e["object_uuid"]].append("%s User account created" % event_at)

        elif e["event_type"] == "update" and e["object_uuid"][6:11] == "tpzed":
            pass

        elif e["event_type"] == "create" and e["object_uuid"][6:11] == "xvhdp":
            if e["properties"]["new_attributes"]["requesting_container_uuid"] is None:
                users[owner].append("%s Ran container %s %s" % (event_at, getname(e["properties"]["new_attributes"]), loguuid))

        elif e["event_type"] == "update" and e["object_uuid"][6:11] == "xvhdp":
            pass

        elif e["event_type"] == "create" and e["object_uuid"][6:11] == "j7d0g":
            users[owner].append("%s Created project %s" %  (event_at, getname(e["properties"]["new_attributes"])))

        elif e["event_type"] == "delete" and e["object_uuid"][6:11] == "j7d0g":
            users[owner].append("%s Deleted project %s" % (event_at, getname(e["properties"]["old_attributes"])))

        elif e["event_type"] == "update" and e["object_uuid"][6:11] == "j7d0g":
            users[owner].append("%s Updated project %s" % (event_at, getname(e["properties"]["new_attributes"])))

        elif e["event_type"] in ("create", "update") and e["object_uuid"][6:11] == "gj3su":
            if len(users[owner]) > 0 and users[owner][-1].endswith("activity"):
                sp = users[owner][-1].split(" ")
                users[owner][-1] = "%s to %s Account activity" % (sp[0], event_at)
            else:
                users[owner].append("%s Account activity" % (event_at))

        elif e["event_type"] == "create" and e["object_uuid"][6:11] == "o0j2j":
            if e["properties"]["new_attributes"]["link_class"] == "tag":
                users[owner].append("%s Tagged %s" % (event_at, e["properties"]["new_attributes"]["head_uuid"]))
            elif e["properties"]["new_attributes"]["link_class"] == "permission":
                users[owner].append("%s Shared %s with %s" % (event_at, e["properties"]["new_attributes"]["tail_uuid"], e["properties"]["new_attributes"]["head_uuid"]))
            else:
                users[owner].append("%s %s %s %s" % (e["event_type"], e["object_kind"], e["object_uuid"], loguuid))

        elif e["event_type"] == "delete" and e["object_uuid"][6:11] == "o0j2j":
            if e["properties"]["old_attributes"]["link_class"] == "tag":
                users[owner].append("%s Untagged %s" % (event_at, e["properties"]["old_attributes"]["head_uuid"]))
            elif e["properties"]["old_attributes"]["link_class"] == "permission":
                users[owner].append("%s Unshared %s with %s" % (event_at, e["properties"]["old_attributes"]["tail_uuid"], e["properties"]["old_attributes"]["head_uuid"]))
            else:
                users[owner].append("%s %s %s %s" % (e["event_type"], e["object_kind"], e["object_uuid"], loguuid))

        elif e["event_type"] == "create" and e["object_uuid"][6:11] == "4zz18":
            if e["properties"]["new_attributes"]["properties"].get("type") in ("log", "output", "intermediate"):
                pass
            else:
                users[owner].append("%s Created collection %s %s" % (event_at, getname(e["properties"]["new_attributes"]), loguuid))

        elif e["event_type"] == "update" and e["object_uuid"][6:11] == "4zz18":
            users[owner].append("%s Updated collection %s %s" % (event_at, getname(e["properties"]["new_attributes"]), loguuid))

        elif e["event_type"] == "delete" and e["object_uuid"][6:11] == "4zz18":
            if e["properties"]["old_attributes"]["properties"].get("type") in ("log", "output", "intermediate"):
                pass
            else:
                users[owner].append("%s Deleted collection %s %s" % (event_at, getname(e["properties"]["old_attributes"]), loguuid))

        else:
            users[owner].append("%s %s %s %s" % (e["event_type"], e["object_kind"], e["object_uuid"], loguuid))

    for k,v in users.items():
        if k is None or k.endswith("-tpzed-000000000000000"):
            continue
        print("%s:" % getusername(arv, k))
        for ev in v:
            print("  %s" % ev)
        print("")

if __name__ == "__main__":
    main()
