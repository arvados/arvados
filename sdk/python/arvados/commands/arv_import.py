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
import functools

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

class DeferExecution:
    def __init__(self, fn):
        self._fn = fn

    def execute(self):
        return self._fn()

def defer_execution(f):
    @functools.wraps(f)
    def wrapper(*args, **kwds):
        return DeferExecution(functools.partial(f, *args, **kwds))
    return wrapper

class StubKeepClient:
    def __init__(self, basedir):
        self._basedir = basedir

    def get(self, locator):
        blockdir = os.path.join(self._basedir, locator[0:3])
        filepath = os.path.join(blockdir, locator)
        with open(filepath, "rb") as fr:
            return fr.read()

class StubArvadosResources:
    def __init__(self, basedir):
        self._basedir = basedir

    @defer_execution
    def get(self, *, uuid=""):
        with open(os.path.join(self._basedir, uuid), "rt") as fr:
            return json.load(fr)

    @defer_execution
    def list(self, *, filters=None):
        pass

class StubArvadosAPI:
    def __init__(self, basedir):
        self._basedir = basedir
        self.keep = StubKeepClient(os.path.join(self._basedir, "keep"))

    def collections(self):
        return StubArvadosResources(os.path.join(self._basedir, "arvados/v1/collections"))

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
