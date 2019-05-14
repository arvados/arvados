#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import arvados
import arvados.util
import csv
import sys
import argparse
import hmac

def main():

    parser = argparse.ArgumentParser(description='Migrate users to federated identity, see https://doc.arvados.org/admin/merge-remote-account.html')
    parser.add_argument('--tokens', type=str, required=True)
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument('--report', type=str, help="Generate report .csv file listing users by email address and their associated Arvados accounts")
    group.add_argument('--migrate', type=str, help="Consume report .csv and migrate users to designated Arvados accounts")
    group.add_argument('--check', action="store_true", help="Check that tokens are usable and the federation is well connected")
    args = parser.parse_args()

    clusters = {}

    print("Reading %s" % args.tokens)
    with open(args.tokens, "rt") as f:
        for r in csv.reader(f):
            host = r[0]
            token = r[1]
            print("Contacting %s" % (host))
            arv = arvados.api(host=host, token=token)
            clusters[arv._rootDesc["uuidPrefix"]] = arv
            cur = arv.users().current().execute()
            if not cur["is_admin"]:
                raise Exception("Not admin of %s" % host)

    print("Checking that the federation is well connected")
    fail = False
    for v in clusters.values():
        for r in clusters:
            if r != v._rootDesc["uuidPrefix"] and r not in v._rootDesc["remoteHosts"]:
                print("%s is missing from remoteHosts on %s" % (r, v._rootDesc["uuidPrefix"]))
                fail = True

    if fail:
        exit(1)

    if args.check:
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
            if r[2] == "":
                print("(%s) Skipping %s, no home cluster specified" % (r[0], r[1]))
            if r[1].startswith(r[2]):
                continue
            candidates = []
            for b in by_email[r[0]]:
                if b[1].startswith(r[2]):
                    candidates.append(b)
            if len(candidates) == 0:
                print("(%s) No user listed to migrate %s to %s" % (r[0], r[1], r[2]))
                continue
            if len(candidates) > 1:
                print("(%s) Multiple users listed to migrate %s to %s, use full uuid" % (r[0], r[1], r[2]))
                continue
            new_user_uuid = candidates[0][1]
            print("(%s) Migrating %s to %s" % (r[0], r[1], new_user_uuid))
            oldcluster = r[1][0:5]
            newhomecluster = r[2][0:5]
            homearv = clusters[newhomecluster]
            # create a token
            newtok = homearv.api_client_authorizations().create(body={"api_client_authorization": {'owner_uuid': new_user_uuid}}).execute()
            salted = 'v2/' + newtok["uuid"] + '/' + hmac.new(newtok["api_token"].encode(), msg=oldcluster.encode(), digestmod='sha1').hexdigest()
            arvados.api(host=arv._rootDesc["rootUrl"][8:-1], token=salted).users().current().execute()

            # now migrate from local user to remote user.
            arv = clusters[oldcluster]

            grp = arv.groups().create(body={
                "owner_uuid": new_user_uuid,
                "name": "Migrated from %s (%s)" % (r[0], r[1]),
                "group_class": "project"
            }, ensure_unique_name=True).execute()
            arv.users().merge(old_user_uuid=r[1],
                              new_user_uuid=new_user_uuid,
                              new_owner_uuid=grp["uuid"],
                              redirect_to_new_user=True).execute()

if __name__ == "__main__":
    main()
