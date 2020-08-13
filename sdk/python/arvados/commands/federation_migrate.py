#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

#
# Migration tool for merging user accounts belonging to the same user
# but on separate clusters to use a single user account managed by a
# specific cluster.
#
# If you're working on this, see
# arvados/sdk/python/tests/fed-migrate/README for information about
# the testing infrastructure.

import arvados
import arvados.util
import arvados.errors
import csv
import sys
import argparse
import hmac
import urllib.parse
import os
import hashlib
import re
from arvados._version import __version__

EMAIL=0
USERNAME=1
UUID=2
HOMECLUSTER=3

def connect_clusters(args):
    clusters = {}
    errors = []
    loginCluster = None
    if args.tokens:
        print("Reading %s" % args.tokens)
        with open(args.tokens, "rt") as f:
            for r in csv.reader(f):
                if len(r) != 2:
                    continue
                host = r[0]
                token = r[1]
                print("Contacting %s" % (host))
                arv = arvados.api(host=host, token=token, cache=False)
                clusters[arv._rootDesc["uuidPrefix"]] = arv
    else:
        arv = arvados.api(cache=False)
        rh = arv._rootDesc["remoteHosts"]
        tok = arv.api_client_authorizations().current().execute()
        token = "v2/%s/%s" % (tok["uuid"], tok["api_token"])

        for k,v in rh.items():
            arv = arvados.api(host=v, token=token, cache=False, insecure=os.environ.get("ARVADOS_API_HOST_INSECURE"))
            clusters[k] = arv

    for _, arv in clusters.items():
        config = arv.configs().get().execute()
        if config["Login"]["LoginCluster"] != "" and loginCluster is None:
            loginCluster = config["Login"]["LoginCluster"]

    print("Checking that the federation is well connected")
    for arv in clusters.values():
        config = arv.configs().get().execute()
        if loginCluster and config["Login"]["LoginCluster"] != loginCluster and config["ClusterID"] != loginCluster:
            errors.append("Inconsistent login cluster configuration, expected '%s' on %s but was '%s'" % (loginCluster, config["ClusterID"], config["Login"]["LoginCluster"]))
            continue

        if arv._rootDesc["revision"] < "20200331":
            errors.append("Arvados API server revision on cluster '%s' is too old, must be updated to at least Arvados 2.0.2 before running migration." % config["ClusterID"])
            continue

        try:
            cur = arv.users().current().execute()
        except arvados.errors.ApiError as e:
            errors.append("checking token for %s   %s" % (arv._rootDesc["rootUrl"], e))
            continue

        if not cur["is_admin"]:
            errors.append("User %s is not admin on %s" % (cur["uuid"], arv._rootDesc["uuidPrefix"]))
            continue

        for r in clusters:
            if r != arv._rootDesc["uuidPrefix"] and r not in arv._rootDesc["remoteHosts"]:
                errors.append("%s is missing from remoteHosts of %s" % (r, arv._rootDesc["uuidPrefix"]))
        for r in arv._rootDesc["remoteHosts"]:
            if r != "*" and r not in clusters:
                print("WARNING: %s is federated with %s but %s is missing from the tokens file or the token is invalid" % (arv._rootDesc["uuidPrefix"], r, r))

    return clusters, errors, loginCluster


def fetch_users(clusters, loginCluster):
    rows = []
    by_email = {}
    by_username = {}

    users = []
    for c, arv in clusters.items():
        print("Getting user list from %s" % c)
        ul = arvados.util.list_all(arv.users().list, bypass_federation=True)
        for l in ul:
            if l["uuid"].startswith(c):
                users.append(l)

    # Users list is sorted by email
    # Go through users and collect users with same email
    # when we see a different email (or get to the end)
    # call add_accum_rows() to generate the report rows with
    # the "home cluster" set, and also fill in the by_email table.

    users = sorted(users, key=lambda u: u["email"]+"::"+(u["username"] or "")+"::"+u["uuid"])

    accum = []
    lastemail = None

    def add_accum_rows():
        homeuuid = None
        for a in accum:
            uuids = set(a["uuid"] for a in accum)
            homeuuid = ((len(uuids) == 1) and uuids.pop()) or ""
        for a in accum:
            r = (a["email"], a["username"], a["uuid"], loginCluster or homeuuid[0:5])
            by_email.setdefault(a["email"], {})
            by_email[a["email"]][a["uuid"]] = r
            homeuuid_and_username = "%s::%s" % (r[HOMECLUSTER], a["username"])
            if homeuuid_and_username not in by_username:
                by_username[homeuuid_and_username] = a["email"]
            elif by_username[homeuuid_and_username] != a["email"]:
                print("ERROR: the username '%s' is listed for both '%s' and '%s' on cluster '%s'" % (r[USERNAME], r[EMAIL], by_username[homeuuid_and_username], r[HOMECLUSTER]))
                exit(1)
            rows.append(r)

    for u in users:
        if u["uuid"].endswith("-anonymouspublic") or u["uuid"].endswith("-000000000000000"):
            continue
        if lastemail == None:
            lastemail = u["email"]
        if u["email"] == lastemail:
            accum.append(u)
        else:
            add_accum_rows()
            lastemail = u["email"]
            accum = [u]

    add_accum_rows()

    return rows, by_email, by_username


def read_migrations(args, by_email, by_username):
    rows = []
    with open(args.migrate or args.dry_run, "rt") as f:
        for r in csv.reader(f):
            if r[EMAIL] == "email":
                continue
            by_email.setdefault(r[EMAIL], {})
            by_email[r[EMAIL]][r[UUID]] = r

            homeuuid_and_username = "%s::%s" % (r[HOMECLUSTER], r[USERNAME])
            if homeuuid_and_username not in by_username:
                by_username[homeuuid_and_username] = r[EMAIL]
            elif by_username[homeuuid_and_username] != r[EMAIL]:
                print("ERROR: the username '%s' is listed for both '%s' and '%s' on cluster '%s'" % (r[USERNAME], r[EMAIL], by_username[homeuuid_and_username], r[HOMECLUSTER]))
                exit(1)

            rows.append(r)
    return rows

def update_username(args, email, user_uuid, username, migratecluster, migratearv):
    print("(%s) Updating username of %s to '%s' on %s" % (email, user_uuid, username, migratecluster))
    if args.dry_run:
        return
    try:
        conflicts = migratearv.users().list(filters=[["username", "=", username]], bypass_federation=True).execute()
        if conflicts["items"]:
            # There's already a user with the username, move the old user out of the way
            migratearv.users().update(uuid=conflicts["items"][0]["uuid"],
                                        bypass_federation=True,
                                        body={"user": {"username": username+"migrate"}}).execute()
        migratearv.users().update(uuid=user_uuid,
                                    bypass_federation=True,
                                    body={"user": {"username": username}}).execute()
    except arvados.errors.ApiError as e:
        print("(%s) Error updating username of %s to '%s' on %s: %s" % (email, user_uuid, username, migratecluster, e))


def choose_new_user(args, by_email, email, userhome, username, old_user_uuid, clusters):
    candidates = []
    conflict = False
    for b in by_email[email].values():
        if b[2].startswith(userhome):
            candidates.append(b)
        if b[1] != username and b[3] == userhome:
            print("(%s) Cannot migrate %s, conflicting usernames %s and %s" % (email, old_user_uuid, b[1], username))
            conflict = True
            break
    if conflict:
        return None
    if len(candidates) == 0:
        if len(userhome) == 5 and userhome not in clusters:
            print("(%s) Cannot migrate %s, unknown home cluster %s (typo?)" % (email, old_user_uuid, userhome))
            return None
        print("(%s) No user listed with same email to migrate %s to %s, will create new user with username '%s'" % (email, old_user_uuid, userhome, username))
        if not args.dry_run:
            oldhomecluster = old_user_uuid[0:5]
            oldhomearv = clusters[oldhomecluster]
            newhomecluster = userhome[0:5]
            homearv = clusters[userhome]
            user = None
            try:
                olduser = oldhomearv.users().get(uuid=old_user_uuid).execute()
                conflicts = homearv.users().list(filters=[["username", "=", username]],
                                                 bypass_federation=True).execute()
                if conflicts["items"]:
                    homearv.users().update(
                        uuid=conflicts["items"][0]["uuid"],
                        bypass_federation=True,
                        body={"user": {"username": username+"migrate"}}).execute()
                user = homearv.users().create(
                    body={"user": {
                        "email": email,
                        "first_name": olduser["first_name"],
                        "last_name": olduser["last_name"],
                        "username": username,
                        "is_active": olduser["is_active"]}}).execute()
            except arvados.errors.ApiError as e:
                print("(%s) Could not create user: %s" % (email, str(e)))
                return None

            tup = (email, username, user["uuid"], userhome)
        else:
            # dry run
            tup = (email, username, "%s-tpzed-xfakexfakexfake" % (userhome[0:5]), userhome)
        by_email[email][tup[2]] = tup
        candidates.append(tup)
    if len(candidates) > 1:
        print("(%s) Multiple users listed to migrate %s to %s, use full uuid" % (email, old_user_uuid, userhome))
        return None
    return candidates[0][2]


def activate_remote_user(args, email, homearv, migratearv, old_user_uuid, new_user_uuid):
    # create a token for the new user and salt it for the
    # migration cluster, then use it to access the migration
    # cluster as the new user once before merging to ensure
    # the new user is known on that cluster.
    migratecluster = migratearv._rootDesc["uuidPrefix"]
    try:
        if not args.dry_run:
            newtok = homearv.api_client_authorizations().create(body={
                "api_client_authorization": {'owner_uuid': new_user_uuid}}).execute()
        else:
            newtok = {"uuid": "dry-run", "api_token": "12345"}
    except arvados.errors.ApiError as e:
        print("(%s) Could not create API token for %s: %s" % (email, new_user_uuid, e))
        return None

    try:
        findolduser = migratearv.users().list(filters=[["uuid", "=", old_user_uuid]], bypass_federation=True).execute()
        if len(findolduser["items"]) == 0:
            return False
        if len(findolduser["items"]) == 1:
            olduser = findolduser["items"][0]
        else:
            print("(%s) Unexpected result" % (email))
            return None
    except arvados.errors.ApiError as e:
        print("(%s) Could not retrieve user %s from %s, user may have already been migrated: %s" % (email, old_user_uuid, migratecluster, e))
        return None

    salted = 'v2/' + newtok["uuid"] + '/' + hmac.new(newtok["api_token"].encode(),
                                                     msg=migratecluster.encode(),
                                                     digestmod=hashlib.sha1).hexdigest()
    try:
        ru = urllib.parse.urlparse(migratearv._rootDesc["rootUrl"])
        if not args.dry_run:
            newuser = arvados.api(host=ru.netloc, token=salted,
                                  insecure=os.environ.get("ARVADOS_API_HOST_INSECURE")).users().current().execute()
        else:
            newuser = {"is_active": True, "username": email.split('@')[0], "is_admin": False}
    except arvados.errors.ApiError as e:
        print("(%s) Error getting user info for %s from %s: %s" % (email, new_user_uuid, migratecluster, e))
        return None

    if not newuser["is_active"] and olduser["is_active"]:
        print("(%s) Activating user %s on %s" % (email, new_user_uuid, migratecluster))
        try:
            if not args.dry_run:
                migratearv.users().update(uuid=new_user_uuid, bypass_federation=True,
                                          body={"is_active": True}).execute()
        except arvados.errors.ApiError as e:
            print("(%s) Could not activate user %s on %s: %s" % (email, new_user_uuid, migratecluster, e))
            return None

    if olduser["is_admin"] and not newuser["is_admin"]:
        print("(%s) Not migrating %s because user is admin but target user %s is not admin on %s. Please ensure the user admin status is the same on both clusters. Note that a federated admin account has admin privileges on the entire federation." % (email, old_user_uuid, new_user_uuid, migratecluster))
        return None

    return newuser

def migrate_user(args, migratearv, email, new_user_uuid, old_user_uuid):
    if args.dry_run:
        return
    try:
        new_owner_uuid = new_user_uuid
        if args.data_into_subproject:
            grp = migratearv.groups().create(body={
                "owner_uuid": new_user_uuid,
                "name": "Migrated from %s (%s)" % (email, old_user_uuid),
                "group_class": "project"
            }, ensure_unique_name=True).execute()
            new_owner_uuid = grp["uuid"]
        migratearv.users().merge(old_user_uuid=old_user_uuid,
                                    new_user_uuid=new_user_uuid,
                                    new_owner_uuid=new_owner_uuid,
                                    redirect_to_new_user=True).execute()
    except arvados.errors.ApiError as e:
        name_collision = re.search(r'Key \(owner_uuid, name\)=\((.*?), (.*?)\) already exists\.\n.*UPDATE "(.*?)"', e._get_reason())
        if name_collision:
            target_owner, rsc_name, rsc_type = name_collision.groups()
            print("(%s) Cannot migrate to %s because both origin and target users have a %s named '%s'. Please rename the conflicting items or use --data-into-subproject to migrate all users' data into a special subproject." % (email, target_owner, rsc_type[:-1], rsc_name))
        else:
            print("(%s) Skipping user migration because of error: %s" % (email, e))


def main():
    parser = argparse.ArgumentParser(description='Migrate users to federated identity, see https://doc.arvados.org/admin/merge-remote-account.html')
    parser.add_argument(
        '--version', action='version', version="%s %s" % (sys.argv[0], __version__),
        help='Print version and exit.')
    parser.add_argument('--tokens', type=str, metavar='FILE', required=False, help="Read tokens from FILE. Not needed when using LoginCluster.")
    parser.add_argument('--data-into-subproject', action="store_true", help="Migrate user's data into a separate subproject. This can be used to avoid name collisions from within an account.")
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument('--report', type=str, metavar='FILE', help="Generate report .csv file listing users by email address and their associated Arvados accounts.")
    group.add_argument('--migrate', type=str, metavar='FILE', help="Consume report .csv and migrate users to designated Arvados accounts.")
    group.add_argument('--dry-run', type=str, metavar='FILE', help="Consume report .csv and report how user would be migrated to designated Arvados accounts.")
    group.add_argument('--check', action="store_true", help="Check that tokens are usable and the federation is well connected.")
    args = parser.parse_args()

    clusters, errors, loginCluster = connect_clusters(args)

    if errors:
        for e in errors:
            print("ERROR: "+str(e))
        exit(1)

    if args.check:
        print("Tokens file passed checks")
        exit(0)

    rows, by_email, by_username = fetch_users(clusters, loginCluster)

    if args.report:
        out = csv.writer(open(args.report, "wt"))
        out.writerow(("email", "username", "user uuid", "home cluster"))
        for r in rows:
            out.writerow(r)
        print("Wrote %s" % args.report)
        return

    if args.migrate or args.dry_run:
        if args.dry_run:
            print("Performing dry run")

        rows = read_migrations(args, by_email, by_username)

        for r in rows:
            email = r[EMAIL]
            username = r[USERNAME]
            old_user_uuid = r[UUID]
            userhome = r[HOMECLUSTER]

            if userhome == "":
                print("(%s) Skipping %s, no home cluster specified" % (email, old_user_uuid))
            if old_user_uuid.startswith(userhome):
                migratecluster = old_user_uuid[0:5]
                migratearv = clusters[migratecluster]
                if migratearv.users().get(uuid=old_user_uuid).execute()["username"] != username:
                    update_username(args, email, old_user_uuid, username, migratecluster, migratearv)
                continue

            new_user_uuid = choose_new_user(args, by_email, email, userhome, username, old_user_uuid, clusters)
            if new_user_uuid is None:
                continue

            remote_users = {}
            got_error = False
            for migratecluster in clusters:
                # cluster where the migration is happening
                migratearv = clusters[migratecluster]

                # the user's new home cluster
                newhomecluster = userhome[0:5]
                homearv = clusters[newhomecluster]

                newuser = activate_remote_user(args, email, homearv, migratearv, old_user_uuid, new_user_uuid)
                if newuser is None:
                    got_error = True
                remote_users[migratecluster] = newuser

            if not got_error:
                for migratecluster in clusters:
                    migratearv = clusters[migratecluster]
                    newuser = remote_users[migratecluster]
                    if newuser is False:
                        continue

                    print("(%s) Migrating %s to %s on %s" % (email, old_user_uuid, new_user_uuid, migratecluster))

                    migrate_user(args, migratearv, email, new_user_uuid, old_user_uuid)

                    if newuser['username'] != username:
                        update_username(args, email, new_user_uuid, username, migratecluster, migratearv)

if __name__ == "__main__":
    main()
