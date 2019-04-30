#!/usr/bin/env python
from __future__ import print_function, absolute_import
import argparse
import arvados
import arvados.util

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
            container_requests_to_rerun.append(cr)


def main():
    parser = argparse.ArgumentParser(description='Re-run containers associated with missing blocks')
    parser.add_argument('--dry-run', action='store_true', default=False)
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

    collections_to_delete = {}
    container_requests_to_rerun = []
    # Get containers that produced these collections
    for b in busted_collections:
        collections_to_delete = arvados.util.list_all(arv.collections().list, filters=[["portable_data_hash", "=", b]])
        for d in collections_to_delete:
            print("Will delete %s" % d["uuid"])
            if not args.dry_run:
                arv.collections().delete(uuid=d["uuid"]).execute()
        maybe_containers_to_rerun = arvados.util.list_all(arv.containers().list, filters=[["output", "=", b]])
        for ct in maybe_containers_to_rerun:
            rerun_request(arv, container_requests_to_rerun, ct)

    for cr in container_requests_to_rerun:
        new_cr = {}
        for f in ("command",
                  "cwd",
                  "environment",
                  "output_path",
                  "container_image",
                  "mounts",
                  "runtime_constraints",
                  "scheduling_parameters",
                  "owner_uuid"):
            new_cr[f] = cr[f]
        new_cr["name"] = cr["name"] + " rerun"
        new_cr["state"] = "Committed"
        new_cr["priority"] = 500
        print("Will re-run %s" % cr["uuid"])
        if not args.dry_run:
            new_cr = arv.container_requests().create(body=new_cr).execute()
            print("Submitted %s" % new_cr["uuid"])

if __name__ == "__main__":
    main()
