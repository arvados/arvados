#!/usr/bin/env python3
#
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0
#

import argparse
import arvados
import arvados.util
import csv
import sys
import logging

lglvl = logging.INFO+1
logging.basicConfig(level=lglvl, format='%(message)s')

"""
 Given a list of collections missing blocks (as produced by
keep-balance), produce a report listing affected collections and
container requests.
"""

def rerun_request(arv, container_requests_to_rerun, ct):
    requests = arvados.util.keyset_list_all(
        arv.container_requests().list,
        filters=[["container_uuid", "=", ct["uuid"]]],
        order='uuid')
    for cr in requests:
        if cr["requesting_container_uuid"]:
            rerun_request(arv, container_requests_to_rerun, arv.containers().get(uuid=cr["requesting_container_uuid"]).execute())
        else:
            container_requests_to_rerun[cr["uuid"]] = cr

def get_owner(arv, owners, record):
    uuid = record["owner_uuid"]
    if uuid not in owners:
        if uuid[6:11] == "tpzed":
            owners[uuid] = (arv.users().get(uuid=uuid).execute()["full_name"], uuid)
        else:
            grp = arv.groups().get(uuid=uuid).execute()
            _, ou = get_owner(arv, owners, grp)
            owners[uuid] = (grp["name"], ou)
    return owners[uuid]

def main():
    parser = argparse.ArgumentParser(description='Re-run containers associated with missing blocks')
    parser.add_argument('inp')
    args = parser.parse_args()

    arv = arvados.api('v1')

    busted_collections = set()

    logging.log(lglvl, "Reading %s", args.inp)

    # Get the list of bad collection PDHs
    blocksfile = open(args.inp, "rt")
    for line in blocksfile:
        # Ignore the first item, that's the block id
        collections = line.rstrip().split(" ")[1:]
        for c in collections:
            busted_collections.add(c)

    out = csv.writer(sys.stdout)

    out.writerow(("collection uuid", "container request uuid", "record name", "modified at", "owner uuid", "owner name", "root owner uuid", "root owner name", "notes"))

    logging.log(lglvl, "Finding collections")

    owners = {}
    collections_to_delete = {}
    container_requests_to_rerun = {}
    # Get containers that produced these collections
    i = 0
    for b in busted_collections:
        if (i % 100) == 0:
            logging.log(lglvl, "%d/%d", i, len(busted_collections))
        i += 1
        collections_to_delete = arvados.util.keyset_list_all(arv.collections().list, filters=[["portable_data_hash", "=", b]], order='uuid')
        for d in collections_to_delete:
            t = ""
            if d["properties"].get("type") not in ("output", "log"):
                t = "\"type\" was '%s', expected one of 'output' or 'log'" % d["properties"].get("type")
            ou = get_owner(arv, owners, d)
            out.writerow((d["uuid"], "", d["name"], d["modified_at"], d["owner_uuid"], ou[0], ou[1], owners[ou[1]][0], t))

        maybe_containers_to_rerun = arvados.util.keyset_list_all(arv.containers().list, filters=[["output", "=", b]], order='uuid')
        for ct in maybe_containers_to_rerun:
            rerun_request(arv, container_requests_to_rerun, ct)

    logging.log(lglvl, "%d/%d", i, len(busted_collections))
    logging.log(lglvl, "Finding container requests")

    i = 0
    for _, cr in container_requests_to_rerun.items():
        if (i % 100) == 0:
            logging.log(lglvl, "%d/%d", i, len(container_requests_to_rerun))
        i += 1
        ou = get_owner(arv, owners, cr)
        out.writerow(("", cr["uuid"], cr["name"], cr["modified_at"], cr["owner_uuid"], ou[0], ou[1], owners[ou[1]][0], ""))

    logging.log(lglvl, "%d/%d", i, len(container_requests_to_rerun))

if __name__ == "__main__":
    main()
