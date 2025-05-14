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
from arvados._internal.arvcopy import api_for_instance, copy_collection
from arvados._internal.stubapi import StubArvadosAPI
import arvados.commands._util as arv_cmd

arvlogger = logging.getLogger('arvados')
keeplogger = logging.getLogger('arvados.keep')
logger = logging.getLogger('arvados.arv-export')
googleapi_logger = logging.getLogger('googleapiclient.http')

def argument_parser():
    export_opts = argparse.ArgumentParser(add_help=False)

    export_opts.add_argument(
        '--version', action='version', version="%s %s" % (sys.argv[0], __version__),
        help='Print version and exit.')
    export_opts.add_argument(
        '-v', '--verbose', dest='verbose', action='store_true',
        help='Verbose output.')
    export_opts.add_argument(
        '-f', '--force', dest='force', action='store_true',
        help='Export even if the object appears to already have been exported already.')

    export_opts.add_argument(
        'object_uuid',
        help='The UUID of the collection or project to export.')

    return argparse.ArgumentParser(
        description='Export stuff to local filesystem.',
        parents=[export_opts, arv_cmd.retry_opt])

def main():
    args = argument_parser().parse_args()
    args.replication = 1
    args.progress = None
    args.storage_classes = []
    args.project_uuid = None
    args.export_all_fields = True

    apiclient = api_for_instance(args.object_uuid[0:5], 3)

    stubapi = StubArvadosAPI(os.path.realpath("."))

    if re.match(arvados.util.collection_uuid_pattern, args.object_uuid):
        return copy_collection(args.object_uuid, apiclient, stubapi, args)


if __name__ == "__main__":
    main()
