#!/usr/bin/env python
#
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0
#

from __future__ import print_function, absolute_import
import argparse
import arvados
import arvados.util
import csv
import sys

"""
Given a list of collections missing blocks (as produced by
keep-balance), delete the collections and re-run associated containers.
"""

def rerun_request(arv, container_requests_to_rerun, ct):
    requests = arvados.util.list_all(arv.container_requests().list, filters=[["container_uuid", "=", ct["uuid"]]])
    for cr in requests:
        if cr["requesting_container_uuid"]:
            rerun_request(arv, container_requests_to_rerun, arv.containers().get(uuid=cr["requesting_container_uuid"]).execute())
        else:
            container_requests_to_rerun[cr["uuid"]] = cr

def get_owner(arv, owners, uuid):
    if uuid not in owners:
        if uuid[6:11] == "tpzed":
            owners[uuid] = arv.users().get(uuid=uuid).execute()["full_name"]
        else:
            owners[uuid] = arv.groups().get(uuid=uuid).execute()["name"]
    return owners[uuid]

def main():
    parser = argparse.ArgumentParser(description='Re-run containers associated with missing blocks')
    parser.add_argument('inp')
    args = parser.parse_args()

    arv = arvados.api('v1')

    busted_collections = set()

    # Get the list of bad collection PDHs
    blocksfile = open(args.inp, "rt")
    for line in blocksfile:
        # Ignore the first item, that's the block id
        collections = line.rstrip().split(" ")[1:]
        for c in collections:
            busted_collections.add(c)

    out = csv.writer(sys.stdout)

    out.writerow(("collection uuid", "container request uuid", "record name", "modified at", "owner uuid", "owner name", "notes"))

    owners = {}
    collections_to_delete = {}
    container_requests_to_rerun = {}
    # Get containers that produced these collections
    i = 0
    for b in busted_collections:
        i += 1
        collections_to_delete = arvados.util.list_all(arv.collections().list, filters=[["portable_data_hash", "=", b]])
        for d in collections_to_delete:
            t = ""
            if d["properties"].get("type") not in ("output", "log"):
                t = "\"type\" was '%s', expected one of 'output' or 'log'" % d["properties"].get("type")
            out.writerow((d["uuid"], "", d["name"], d["modified_at"], d["owner_uuid"], get_owner(arv, owners, d["owner_uuid"]), t))

        maybe_containers_to_rerun = arvados.util.list_all(arv.containers().list, filters=[["output", "=", b]])
        for ct in maybe_containers_to_rerun:
            rerun_request(arv, container_requests_to_rerun, ct)

    i = 0
    for _, cr in container_requests_to_rerun.items():
        i += 1
        out.writerow(("", cr["uuid"], cr["name"], cr["modified_at"], cr["owner_uuid"], get_owner(arv, owners, cr["owner_uuid"]), ""))


if __name__ == "__main__":
    main()
