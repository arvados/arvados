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
import json

from arvados._version import __version__
from arvados.logging import log_handler

arvlogger = logging.getLogger('arvados')
keeplogger = logging.getLogger('arvados.keep')
logger = logging.getLogger('arvados.arv-export')
googleapi_logger = logging.getLogger('googleapiclient.http')

def argument_parser():
    export_opts = argparse.ArgumentParser()

    export_opts.add_argument(
        '--version', action='version', version="%s %s" % (sys.argv[0], __version__),
        help='Print version and exit.')
    export_opts.add_argument(
        '-v', '--verbose', dest='verbose', action='store_true',
        help='Verbose output.')

    export_opts.add_argument(
        'object_uuid',
        help='The UUID of the collection or project to export.')

    return export_opts

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

def export_collection(apiclient, collection_uuid, collectionsdir, keepdir):
    c = apiclient.collections().get(uuid=collection_uuid).execute()

    manifest = c['manifest_text']

    dst_manifest = io.StringIO()
    dst_locators = set()

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

            locator = loc.md5sum
            blockdir = os.path.join(keepdir, locator[0:3])
            os.makedirs(blockdir, exist_ok=True)
            filepath = os.path.join(blockdir, locator)
            writefile = True
            try:
                st = os.stat(filepath)
                if st.st_size == loc.size:
                    # There's a file with the same name and size in
                    # the same place, assume it is already written.
                    writefile = False
            except OSError:
                pass
            if writefile:
                block = apiclient.keep.get(word)
                with open(filepath, "wb") as fw:
                    fw.write(block)
            dst_manifest.write(' ')
            dst_manifest.write(loc.stripped())
        dst_manifest.write("\n")

    c["manifest_text"] = dst_manifest.getvalue()
    with open(os.path.join(collectionsdir, c["uuid"]), "wt") as fw:
        json.dump(c, fw, indent=2)


def main():
    args = argument_parser().parse_args()

    apiclient = make_api_client(args)

    os.makedirs("keep", exist_ok=True)
    keepdir = os.path.realpath("keep")

    os.makedirs("arvados/v1/collections", exist_ok=True)
    collectionsdir = os.path.realpath("arvados/v1/collections")

    if re.match(arvados.util.collection_uuid_pattern, args.object_uuid):
        return export_collection(apiclient, args.object_uuid, collectionsdir, keepdir)



if __name__ == "__main__":
    main()
