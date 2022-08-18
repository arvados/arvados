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
import csv

def parse_arguments(arguments):
    arg_parser = argparse.ArgumentParser()
    arg_parser.add_argument('--start', help='Start date for the report in YYYY-MM-DD format (UTC)')
    arg_parser.add_argument('--end', help='End date for the report in YYYY-MM-DD format (UTC)')
    arg_parser.add_argument('--days', type=int, help='Number of days before now() to start the report')
    arg_parser.add_argument('--csv', action='store_true', help='Output in csv format (default: false)')
    args = arg_parser.parse_args(arguments)

    if args.days and (args.start or args.end):
        arg_parser.print_help()
        print("Error: either specify --days or both --start and --end")
        exit(1)

    if not args.days and (not args.start or not args.end):
        arg_parser.print_help()
        print("\nError: either specify --days or both --start and --end")
        exit(1)

    if (args.start and not args.end) or (args.end and not args.start):
        arg_parser.print_help()
        print("\nError: no start or end date found, either specify --days or both --start and --end")
        exit(1)

    if args.days:
        to = datetime.datetime.utcnow()
        since = to - datetime.timedelta(days=args.days)

    if args.start:
        try:
            since = datetime.datetime.strptime(args.start,"%Y-%m-%d")
        except:
            arg_parser.print_help()
            print("\nError: start date must be in YYYY-MM-DD format")
            exit(1)

    if args.end:
        try:
            to = datetime.datetime.strptime(args.end,"%Y-%m-%d")
        except:
            arg_parser.print_help()
            print("\nError: end date must be in YYYY-MM-DD format")
            exit(1)

    return args, since, to

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

def getuserinfo(arv, uuid):
    try:
        u = arv.users().get(uuid=uuid).execute()
    except:
        return "deleted user (%susers/%s)" % (arv.config()["Services"]["Workbench1"]["ExternalURL"],
                                                       uuid)
    prof = "\n".join("  %s: \"%s\"" % (k, v) for k, v in u["prefs"].get("profile", {}).items() if v)
    if prof:
        prof = "\n"+prof+"\n"
    return "%s %s <%s> (%susers/%s)%s" % (u["first_name"], u["last_name"], u["email"],
                                                       arv.config()["Services"]["Workbench1"]["ExternalURL"],
                                                       uuid, prof)
def getuserinfocsv(arv, uuid):
    try:
        u = arv.users().get(uuid=uuid).execute()
    except:
        return [uuid,"deleted","user",""]
    return [uuid, u["first_name"], u["last_name"], u["email"]]


collectionNameCache = {}
def getCollectionName(arv, uuid, pdh):
    lookupField = uuid
    filters = [["uuid","=",uuid]]
    cached = uuid in collectionNameCache
    # look up by uuid if it is available, fall back to look up by pdh
    if len(uuid) != 27:
        # Look up by pdh. Note that this can be misleading; the download could
        # have happened from a collection with the same pdh but different name.
        # We arbitrarily pick the oldest collection with the pdh to lookup the
        # name, if the uuid for the request is not known.
        lookupField = pdh
        filters = [["portable_data_hash","=",pdh]]
        cached = pdh in collectionNameCache

    if not cached:
        u = arv.collections().list(filters=filters,order="created_at",limit=1).execute().get("items")
        if len(u) < 1:
            return "(deleted)"
        collectionNameCache[lookupField] = u[0]["name"]
    return collectionNameCache[lookupField]

def getname(u):
    return "\"%s\" (%s)" % (u["name"], u["uuid"])

def main(arguments=None):
    if arguments is None:
        arguments = sys.argv[1:]

    args, since, to = parse_arguments(arguments)

    arv = arvados.api()

    prefix = ''
    suffix = "\n"
    if args.csv:
        prefix = '# '
        suffix = ''
    print("%sUser activity on %s between %s and %s%s" % (prefix, arv.config()["ClusterID"],
                                                       since.isoformat(sep=" ", timespec="minutes"),
                                                       to.isoformat(sep=" ", timespec="minutes"), suffix))

    events = arvados.util.keyset_list_all(arv.logs().list, filters=[["created_at", ">=", since.isoformat()],["created_at", "<", to.isoformat()]])

    users = {}
    owners = {}

    for e in events:
        owner = getowner(arv, e["object_owner_uuid"], owners)
        users.setdefault(owner, [])
        event_at = ciso8601.parse_datetime(e["event_at"]).astimezone().isoformat(sep=" ", timespec="minutes")
        loguuid = e["uuid"]

        if e["event_type"] == "create" and e["object_uuid"][6:11] == "tpzed":
            users.setdefault(e["object_uuid"], [])
            users[e["object_uuid"]].append([loguuid, event_at, "User account created"])

        elif e["event_type"] == "update" and e["object_uuid"][6:11] == "tpzed":
            pass

        elif e["event_type"] == "create" and e["object_uuid"][6:11] == "xvhdp":
            if e["properties"]["new_attributes"]["requesting_container_uuid"] is None:
                users[owner].append([loguuid, event_at, "Ran container %s" % (getname(e["properties"]["new_attributes"]))])

        elif e["event_type"] == "update" and e["object_uuid"][6:11] == "xvhdp":
            pass

        elif e["event_type"] == "create" and e["object_uuid"][6:11] == "j7d0g":
            users[owner].append([loguuid, event_at,"Created project %s" % (getname(e["properties"]["new_attributes"]))])

        elif e["event_type"] == "delete" and e["object_uuid"][6:11] == "j7d0g":
            users[owner].append([loguuid, event_at,"Deleted project %s" % (getname(e["properties"]["old_attributes"]))])

        elif e["event_type"] == "update" and e["object_uuid"][6:11] == "j7d0g":
            users[owner].append([loguuid, event_at,"Updated project %s" % (getname(e["properties"]["new_attributes"]))])

        elif e["event_type"] in ("create", "update") and e["object_uuid"][6:11] == "gj3su":
            # Don't log token activity, it is too noisy (bug #19179)
            pass

        # We want to report when a user goes through the login
        # process, but controller doesn't do that yet, so revisit
        # this when #19388 is done.

        elif e["event_type"] == "create" and e["object_uuid"][6:11] == "o0j2j":
            if e["properties"]["new_attributes"]["link_class"] == "tag":
                users[owner].append([event_at,"Tagged %s" % (e["properties"]["new_attributes"]["head_uuid"])])
            elif e["properties"]["new_attributes"]["link_class"] == "permission":
                users[owner].append([loguuid, event_at,"Shared %s with %s" % (e["properties"]["new_attributes"]["tail_uuid"], e["properties"]["new_attributes"]["head_uuid"])])
            else:
                users[owner].append([loguuid, event_at,"%s %s %s" % (e["event_type"], e["object_kind"], e["object_uuid"])])

        elif e["event_type"] == "delete" and e["object_uuid"][6:11] == "o0j2j":
            if e["properties"]["old_attributes"]["link_class"] == "tag":
                users[owner].append([loguuid, event_at,"Untagged %s" % (e["properties"]["old_attributes"]["head_uuid"])])
            elif e["properties"]["old_attributes"]["link_class"] == "permission":
                users[owner].append([loguuid, event_at,"Unshared %s with %s" % (e["properties"]["old_attributes"]["tail_uuid"], e["properties"]["old_attributes"]["head_uuid"])])
            else:
                users[owner].append([loguuid, event_at,"%s %s %s" % (e["event_type"], e["object_kind"], e["object_uuid"])])

        elif e["event_type"] == "create" and e["object_uuid"][6:11] == "4zz18":
            if e["properties"]["new_attributes"]["properties"].get("type") in ("log", "output", "intermediate"):
                pass
            else:
                users[owner].append([loguuid, event_at,"Created collection %s" % (getname(e["properties"]["new_attributes"]))])

        elif e["event_type"] == "update" and e["object_uuid"][6:11] == "4zz18":
            users[owner].append([loguuid, event_at,"Updated collection %s" % (getname(e["properties"]["new_attributes"]))])

        elif e["event_type"] == "delete" and e["object_uuid"][6:11] == "4zz18":
            if e["properties"]["old_attributes"]["properties"].get("type") in ("log", "output", "intermediate"):
                pass
            else:
                users[owner].append([loguuid, event_at, "Deleted collection %s" % (getname(e["properties"]["old_attributes"]))])

        elif e["event_type"] == "file_download":
                users.setdefault(e["object_uuid"], [])
                users[e["object_uuid"]].append([loguuid, event_at, "Downloaded file \"%s\" from \"%s\" (%s) (%s)" % (
                                                                                       e["properties"].get("collection_file_path") or e["properties"].get("reqPath"),
                                                                                       getCollectionName(arv, e["properties"].get("collection_uuid"), e["properties"].get("portable_data_hash")),
                                                                                       e["properties"].get("collection_uuid"),
                                                                                       e["properties"].get("portable_data_hash"))])


        elif e["event_type"] == "file_upload":
                users.setdefault(e["object_uuid"], [])
                users[e["object_uuid"]].append([loguuid, event_at, "Uploaded file \"%s\" to \"%s\" (%s)" % (
                                                                                    e["properties"].get("collection_file_path") or e["properties"].get("reqPath"),
                                                                                    getCollectionName(arv, e["properties"].get("collection_uuid"), e["properties"].get("portable_data_hash")),
                                                                                    e["properties"].get("collection_uuid"))])

        else:
            users[owner].append([loguuid, event_at, "%s %s %s" % (e["event_type"], e["object_kind"], e["object_uuid"])])

    if args.csv:
        csvwriter = csv.writer(sys.stdout, dialect='unix')

    for k,v in users.items():
        # Skip system user
        if k is None or k.endswith("-tpzed-000000000000000"):
            continue

        # Skip users with no activity to report
        if not v:
            continue

        if not args.csv:
          print(getuserinfo(arv, k))
          for ev in v:
              # Remove the log entry uuid, this report is intended for human consumption
              ev.pop(0)
              print("  %s" % ' '.join(ev))
          print("")
        else:
          user = getuserinfocsv(arv, k)
          for ev in v:
            ev = user + ev
            csvwriter.writerow(ev)

if __name__ == "__main__":
    main()
