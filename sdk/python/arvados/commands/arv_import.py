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
from arvados._internal.arvcopy import api_for_instance, copy_collection, copy_project
from arvados._internal.stubapi import StubArvadosAPI
import arvados.commands._util as arv_cmd

arvlogger = logging.getLogger('arvados')
keeplogger = logging.getLogger('arvados.keep')
logger = logging.getLogger('arvados.arv-export')
googleapi_logger = logging.getLogger('googleapiclient.http')

def argument_parser():
    import_opts = argparse.ArgumentParser(add_help=False)

    import_opts.add_argument(
        '--version', action='version', version="%s %s" % (sys.argv[0], __version__),
        help='Print version and exit.')
    import_opts.add_argument(
        '-v', '--verbose', dest='verbose', action='store_true',
        help='Verbose output.')
    import_opts.add_argument(
        '-f', '--force', dest='force', action='store_true',
        help='Export even if the object appears to already have been exported already.')
    import_opts.add_argument(
        '--project-uuid', dest='project_uuid',
        help='The UUID of the project at the destination to which the collection or project should be imported.')
    import_opts.add_argument(
        '--storage-classes',
        type=arv_cmd.UniqueSplit(),
        help='Comma separated list of storage classes to be used when saving data to the destinaton Arvados instance.')
    import_opts.add_argument(
        '--replication',
        type=arv_cmd.RangedValue(int, range(1, sys.maxsize)),
        metavar='N',
        help="""
Number of replicas per storage class for the copied collections at the destination.
If not provided (or if provided with invalid value),
use the destination's default replication-level setting (if found),
or the fallback value 2.
""")
    import_opts.add_argument(
        '--recursive', dest='recursive', action='store_true',
        help='Recursively copy any dependencies for this object, and subprojects. (default)')
    import_opts.add_argument(
        '--no-recursive', dest='recursive', action='store_false',
        help='Do not copy any dependencies or subprojects.')

    import_opts.add_argument(
        'object_uuid',
        help='The UUID of the collection or project to import.')

    return argparse.ArgumentParser(
        description='Import stuff from local filesystem.',
        parents=[import_opts, arv_cmd.retry_opt])

def main():
    args = argument_parser().parse_args()
    args.progress = None
    args.export_all_fields = False

    if args.verbose:
        arvlogger.setLevel(logging.DEBUG)
    else:
        arvlogger.setLevel(logging.INFO)
        keeplogger.setLevel(logging.WARNING)

    apiclient = api_for_instance(args.project_uuid[0:5] if args.project_uuid else '', 3)

    stubapi = StubArvadosAPI(os.path.realpath("."))

    if re.match(arvados.util.collection_uuid_pattern, args.object_uuid):
        return copy_collection(args.object_uuid, stubapi, apiclient, args)
    elif re.match(arvados.util.group_uuid_pattern, args.object_uuid):
        return copy_project(args.object_uuid, stubapi, apiclient,
                            args.project_uuid or apiclient.users().current().execute()["uuid"],
                            args)
    else:
        logger.error("Object type not supported for import")


if __name__ == "__main__":
    main()
