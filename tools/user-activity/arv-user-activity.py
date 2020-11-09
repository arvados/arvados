#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import argparse
import sys

import arvados
import arvados.util

def keyset_list_all(fn, order_key="created_at", num_retries=0, ascending=True, **kwargs):
    pagesize = 1000
    kwargs["limit"] = pagesize
    kwargs["count"] = 'none'
    kwargs["order"] = ["%s %s" % (order_key, "asc" if ascending else "desc"), "uuid asc"]
    other_filters = kwargs.get("filters", [])

    if "select" in kwargs and "uuid" not in kwargs["select"]:
        kwargs["select"].append("uuid")

    nextpage = []
    tot = 0
    expect_full_page = True
    seen_prevpage = set()
    seen_thispage = set()
    lastitem = None
    prev_page_all_same_order_key = False

    while True:
        kwargs["filters"] = nextpage+other_filters
        items = fn(**kwargs).execute(num_retries=num_retries)

        if len(items["items"]) == 0:
            if prev_page_all_same_order_key:
                nextpage = [[order_key, ">" if ascending else "<", lastitem[order_key]]]
                prev_page_all_same_order_key = False
                continue
            else:
                return

        seen_prevpage = seen_thispage
        seen_thispage = set()

        for i in items["items"]:
            # In cases where there's more than one record with the
            # same order key, the result could include records we
            # already saw in the last page.  Skip them.
            if i["uuid"] in seen_prevpage:
                continue
            seen_thispage.add(i["uuid"])
            yield i

        firstitem = items["items"][0]
        lastitem = items["items"][-1]

        if firstitem[order_key] == lastitem[order_key]:
            # Got a page where every item has the same order key.
            # Switch to using uuid for paging.
            nextpage = [[order_key, "=", lastitem[order_key]], ["uuid", ">", lastitem["uuid"]]]
            prev_page_all_same_order_key = True
        else:
            # Start from the last order key seen, but skip the last
            # known uuid to avoid retrieving the same row twice.  If
            # there are multiple rows with the same order key it is
            # still likely we'll end up retrieving duplicate rows.
            # That's handled by tracking the "seen" rows for each page
            # so they can be skipped if they show up on the next page.
            nextpage = [[order_key, ">=" if ascending else "<=", lastitem[order_key]], ["uuid", "!=", lastitem["uuid"]]]
            prev_page_all_same_order_key = False


def parse_arguments(arguments):
    arg_parser = argparse.ArgumentParser()
    arg_parser.add_argument('--timespan', type=str)
    args = arg_parser.parse_args(arguments)
    return args

def main(arguments):
    args = parse_arguments(arguments)

    arv = arvados.api()

    events = keyset_list_all(arv.logs().list, filters=[["created_at", ">=", "2020-11-05T14:51:42-05:00"]])

    users = {}

    for e in events:
        if e["event_type"] == "create" and e["object_uuid"][6:11] == "tpzed":
            users.setdefault(e["object_uuid"], [])
            users[e["object_uuid"]].append("User was created")

        if e["event_type"] == "create" and e["object_uuid"][6:11] == "xvhdp":
            users.setdefault(e["object_owner_uuid"], [])
            users[e["object_owner_uuid"]].append("Ran a container")

        if e["event_type"] == "create" and e["object_uuid"][6:11] == "j7d0g":
            users.setdefault(e["object_owner_uuid"], [])
            users[e["object_owner_uuid"]].append("Created a project")

    for k,v in users.items():
        print("%s:" % k)
        for ev in v:
            print("  %s" % ev)


main(sys.argv[1:])
