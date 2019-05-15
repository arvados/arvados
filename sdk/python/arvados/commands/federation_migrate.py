#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import arvados
import arvados.util
import arvados.errors
import csv
import sys
import argparse
import hmac
import urllib.parse

def main():

    parser = argparse.ArgumentParser(description='Migrate users to federated identity, see https://doc.arvados.org/admin/merge-remote-account.html')
    parser.add_argument('--tokens', type=str, required=True)
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument('--report', type=str, help="Generate report .csv file listing users by email address and their associated Arvados accounts")
    group.add_argument('--migrate', type=str, help="Consume report .csv and migrate users to designated Arvados accounts")
    group.add_argument('--check', action="store_true", help="Check that tokens are usable and the federation is well connected")
    args = parser.parse_args()

    clusters = {}
    errors = []
    print("Reading %s" % args.tokens)
    with open(args.tokens, "rt") as f:
        for r in csv.reader(f):
            host = r[0]
            token = r[1]
            print("Contacting %s" % (host))
            arv = arvados.api(host=host, token=token, cache=False)
            try:
                cur = arv.users().current().execute()
                arv.api_client_authorizations().list(limit=1).execute()
            except arvados.errors.ApiError as e:
                errors.append("checking token for %s: %s" % (host, e))
                errors.append('    This script requires a token issued to a trusted client in order to manipulate access tokens.')
                errors.append('    See "Trusted client setting" in https://doc.arvados.org/install/install-workbench-app.html')
                errors.append('    and https://doc.arvados.org/api/tokens.html')
                continue

            if not cur["is_admin"]:
                errors.append("Not admin of %s" % host)
                continue

            clusters[arv._rootDesc["uuidPrefix"]] = arv


    print("Checking that the federation is well connected")
    for v in clusters.values():
        for r in clusters:
            if r != v._rootDesc["uuidPrefix"] and r not in v._rootDesc["remoteHosts"]:
                errors.append("%s is missing from remoteHosts of %s" % (r, v._rootDesc["uuidPrefix"]))
        for r in v._rootDesc["remoteHosts"]:
            if r != "*" and r not in clusters:
                print("WARNING: %s is federated with %s but %s is missing from the tokens file or the token is invalid" % (v._rootDesc["uuidPrefix"], r, r))

    if errors:
        for e in errors:
            print("ERROR: "+str(e))
        exit(1)

    if args.check:
        print("Tokens file passed checks")
        exit(0)

    if args.report:
        users = []
        for c, arv in clusters.items():
            print("Getting user list from %s" % c)
            ul = arvados.util.list_all(arv.users().list)
            for l in ul:
                if l["uuid"].startswith(c):
                    users.append(l)

        out = csv.writer(open(args.report, "wt"))

        out.writerow(("email", "user uuid", "primary cluster/user"))

        users = sorted(users, key=lambda u: u["email"]+"::"+u["uuid"])

        accum = []
        lastemail = None
        for u in users:
            if u["uuid"].endswith("-anonymouspublic") or u["uuid"].endswith("-000000000000000"):
                continue
            if lastemail == None:
                lastemail = u["email"]
            if u["email"] == lastemail:
                accum.append(u)
            else:
                homeuuid = None
                for a in accum:
                    if homeuuid is None:
                        homeuuid = a["uuid"]
                    if a["uuid"] != homeuuid:
                        homeuuid = ""
                for a in accum:
                    out.writerow((a["email"], a["uuid"], homeuuid[0:5]))
                lastemail = u["email"]
                accum = [u]

        homeuuid = None
        for a in accum:
            if homeuuid is None:
                homeuuid = a["uuid"]
            if a["uuid"] != homeuuid:
                homeuuid = ""
        for a in accum:
            out.writerow((a["email"], a["uuid"], homeuuid[0:5]))

        print("Wrote %s" % args.report)

    if args.migrate:
        rows = []
        by_email = {}
        with open(args.migrate, "rt") as f:
            for r in csv.reader(f):
                if r[0] == "email":
                    continue
                by_email.setdefault(r[0], [])
                by_email[r[0]].append(r)
                rows.append(r)
        for r in rows:
            email = r[0]
            old_user_uuid = r[1]
            userhome = r[2]

            if userhome == "":
                print("(%s) Skipping %s, no home cluster specified" % (email, old_user_uuid))
            if old_user_uuid.startswith(userhome):
                continue
            candidates = []
            for b in by_email[email]:
                if b[1].startswith(userhome):
                    candidates.append(b)
            if len(candidates) == 0:
                if len(userhome) == 5 and userhome not in clusters:
                    print("(%s) Cannot migrate %s, unknown home cluster %s (typo?)" % (email, old_user_uuid, userhome))
                else:
                    print("(%s) No user listed with same email to migrate %s to %s" % (email, old_user_uuid, userhome))
                continue
            if len(candidates) > 1:
                print("(%s) Multiple users listed to migrate %s to %s, use full uuid" % (email, old_user_uuid, userhome))
                continue
            new_user_uuid = candidates[0][1]

            # cluster where the migration is happening
            migratecluster = old_user_uuid[0:5]
            migratearv = clusters[migratecluster]

            # the user's new home cluster
            newhomecluster = userhome[0:5]
            homearv = clusters[newhomecluster]

            # create a token for the new user and salt it for the
            # migration cluster, then use it to access the migration
            # cluster as the new user once before merging to ensure
            # the new user is known on that cluster.
            try:
                newtok = homearv.api_client_authorizations().create(body={
                    "api_client_authorization": {'owner_uuid': new_user_uuid}}).execute()
            except arvados.errors.ApiError as e:
                print("(%s) Could not create API token for %s: %s" % (email, new_user_uuid, e))
                continue

            salted = 'v2/' + newtok["uuid"] + '/' + hmac.new(newtok["api_token"].encode(),
                                                             msg=migratecluster.encode(),
                                                             digestmod='sha1').hexdigest()
            try:
                ru = urllib.parse.urlparse(migratearv._rootDesc["rootUrl"])
                newuser = arvados.api(host=ru.netloc, token=salted).users().current().execute()
            except arvados.errors.ApiError as e:
                print("(%s) Error getting user info for %s from %s: %s" % (email, new_user_uuid, migratecluster, e))
                continue

            try:
                olduser = migratearv.users().get(uuid=old_user_uuid).execute()
            except arvados.errors.ApiError as e:
                print("(%s) Could not retrieve user %s from %s, user may have already been migrated: %s" % (email, old_user_uuid, migratecluster, e))
                continue

            if not newuser["is_active"]:
                print("(%s) Activating user %s on %s" % (email, new_user_uuid, migratecluster))
                try:
                    migratearv.users().update(uuid=new_user_uuid, body={"is_active": True}).execute()
                except arvados.errors.ApiError as e:
                    print("(%s) Could not activate user %s on %s: %s" % (email, new_user_uuid, migratecluster, e))
                    continue

            if olduser["is_admin"] and not newuser["is_admin"]:
                print("(%s) Not migrating %s because user is admin but target user %s is not admin on %s" % (email, old_user_uuid, new_user_uuid, migratecluster))
                continue

            print("(%s) Migrating %s to %s on %s" % (email, old_user_uuid, new_user_uuid, migratecluster))

            try:
                grp = migratearv.groups().create(body={
                    "owner_uuid": new_user_uuid,
                    "name": "Migrated from %s (%s)" % (email, old_user_uuid),
                    "group_class": "project"
                }, ensure_unique_name=True).execute()
                migratearv.users().merge(old_user_uuid=old_user_uuid,
                                         new_user_uuid=new_user_uuid,
                                         new_owner_uuid=grp["uuid"],
                                         redirect_to_new_user=True).execute()
            except arvados.errors.ApiError as e:
                print("(%s) Error migrating user: %s" % (email, e))

if __name__ == "__main__":
    main()
