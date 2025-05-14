#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import arvados
import arvados.util
import argparse
import sys
import logging
import re
import io
import os

from arvados._internal.stubapi import StubArvadosAPI

from arvados._version import __version__
from arvados.logging import log_handler

arvlogger = logging.getLogger('arvados')
keeplogger = logging.getLogger('arvados.keep')
logger = logging.getLogger('arvados.arv-import')
googleapi_logger = logging.getLogger('googleapiclient.http')

def argument_parser():
    import_opts = argparse.ArgumentParser()

    import_opts.add_argument(
        '--version', action='version', version="%s %s" % (sys.argv[0], __version__),
        help='Print version and exit.')
    import_opts.add_argument(
        '-v', '--verbose', dest='verbose', action='store_true',
        help='Verbose output.')
    import_opts.add_argument(
        '--project-uuid', dest='project_uuid',
        help='The UUID of the project at the destination to which the collection or project should be imported.')

    import_opts.add_argument(
        'object_uuid',
        help='The UUID of the collection or project to import.')

    return import_opts

def make_api_client(args):

    if args.verbose:
        arvlogger.setLevel(logging.DEBUG)
    else:
        arvlogger.setLevel(logging.INFO)
        keeplogger.setLevel(logging.WARNING)

    googleapi_logger.setLevel(logging.WARN)
    googleapi_logger.addHandler(log_handler)

    apiclient = arvados.api('v1')

    # Once we've successfully contacted the cluster, we probably
    # don't want to see logging about retries (unless the user asked
    # for verbose output).
    if not args.verbose:
        googleapi_logger.setLevel(logging.ERROR)

    return apiclient


def import_collection(stubapi, remoteapi, collection_uuid,
                      owner_uuid):

    c = stubapi.collections().get(uuid=collection_uuid).execute()

    manifest = c['manifest_text']

    dst_manifest = io.StringIO()
    dst_locators = {}

    for line in manifest.splitlines():
        words = line.split()
        dst_manifest.write(words[0])
        for word in words[1:]:
            try:
                loc = arvados.KeepLocator(word)
            except ValueError:
                # If 'word' can't be parsed as a locator,
                # presume it's a filename.
                dst_manifest.write(' ')
                dst_manifest.write(word)
                continue

            blockhash = loc.md5sum
            if blockhash not in dst_locators:
                block = stubapi.keep.get(blockhash)
                dst_locators[blockhash] = remoteapi.keep.put(block)

            dst_manifest.write(' ')
            dst_manifest.write(dst_locators[blockhash])

        dst_manifest.write("\n")

    newcollection = {
        "description": c["description"],
        "manifest_text": dst_manifest.getvalue(),
        "name": c["name"],
        "portable_data_hash": c["portable_data_hash"],
        "properties": c["properties"],
        "owner_uuid": owner_uuid
    }

    result = remoteapi.collections().create(body={"collection": newcollection}).execute()
    logger.info("%s (%s) imported as %s", c["uuid"], c["portable_data_hash"], result["uuid"])

def main():
    args = argument_parser().parse_args()

    dest_project = args.project_uuid or apiclient.users().current().execute()["uuid"]

    apiclient = make_api_client(args)

    stubapi = StubArvadosAPI(os.path.realpath("."))

    if re.match(arvados.util.collection_uuid_pattern, args.object_uuid):
        return import_collection(stubapi, apiclient, args.object_uuid, dest_project)

if __name__ == "__main__":
    main()
