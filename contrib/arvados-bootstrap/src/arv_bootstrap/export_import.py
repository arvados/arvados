# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import argparse
import importlib.metadata
import logging
import os
import re
import sys

import arvados.commands.arv_copy as arv_copy
import arvados.commands._util as arv_cmd
import arvados.util as arv_util

from . import stubapi

logger = logging.getLogger('arvados.arv-export-import')

class ArgumentParser(argparse.ArgumentParser):
    @classmethod
    def _base_options(cls, cmdname=sys.argv[0]):
        opts = argparse.ArgumentParser(add_help=False)
        opts.add_argument(
            '--version',
            action='version',
            version=f'{cmdname} {importlib.metadata.version("arvados-bootstrap")}',
            help='Print version and exit.',
        )
        opts.add_argument(
            '--verbose', '-v',
            dest='verbose',
            action='store_true',
            help='Verbose output.',
        )
        return opts

    @classmethod
    def _common_options(cls, verb):
        opts = cls._base_options(f'arv-{verb}')
        opts.add_argument(
            '--force', '-f',
            action='store_true',
            help=f"""{verb.capitalize()} even if the object has already been {verb}ed.
""")
        opts.add_argument(
            '--recursive',
            action='store_true',
            help=f"""Recursively {verb} any dependencies for this object
and subprojects. (default)
""")
        opts.add_argument(
            '--no-recursive',
            dest='recursive',
            action='store_false',
            help=f"""Do not {verb} any dependencies or subprojects.
""")

        opts.add_argument(
            '--block-copy',
            dest='keep_block_copy',
            action='store_true',
            help=f"""Copy Keep blocks when {verb}ing collections. (default)
""")
        opts.add_argument(
            '--no-block-copy',
            dest='keep_block_copy',
            action='store_false',
            help=f"""Do not copy Keep blocks when {verb}ing collections.
Must have administrator privileges to import collections.
""")
        opts.add_argument(
            'object_uuid',
            help=f"""The UUID of the collection or project to {verb}.
""")
        return opts

    @classmethod
    def _import_options(cls):
        opts = cls._common_options('import')
        opts.add_argument(
            '--project-uuid',
            help="""The UUID of the project at the destination to which the
collection or project should be imported.
""")
        opts.add_argument(
            '--storage-classes',
            type=arv_cmd.UniqueSplit(),
            help="""Comma-separated list of storage classes to be used when
saving data to the destinaton Arvados instance.
""")
        opts.add_argument(
            '--replication',
            type=arv_cmd.RangedValue(int, range(1, sys.maxsize)),
            metavar='N',
            help="""
Number of replicas per storage class for the copied collections at the destination.
If not provided (or if provided with invalid value),
use the destination's default replication-level setting (if found),
or the fallback value 2.
""")
        return opts

    def _set_common_defaults(self):
        self.set_defaults(
            # Common defaults should use the "safer" value.
            export_all_fields=False,
            force=False,
            keep_block_copy=True,
            prefer_cached_downloads=False,
            project_uuid=None,
            progress=None,
            recursive=True,
            varying_url_params="",
        )

    @classmethod
    def export_parser(cls):
        parser = cls(
            description=f"Export Arvados objects to a local filesystem",
            parents=[cls._common_options('export'), arv_cmd.retry_opt],
        )
        parser._set_common_defaults()
        parser.set_defaults(
            export_all_fields=True,
            progress=True,
            replication=1,
            storage_classes=[],
        )
        return parser

    @classmethod
    def import_parser(cls):
        parser = cls(
            description=f"Import Arvados objects from a local filesystem",
            parents=[cls._import_options(), arv_cmd.retry_opt],
        )
        parser._set_common_defaults()
        return parser


def setup_logging(name, args):
    global logger
    arvlogger = logging.getLogger('arvados')
    logger = arvlogger.getChild(name)
    if args.verbose:
        arvlogger.setLevel(logging.DEBUG)
    else:
        arvlogger.setLevel(logging.INFO)
        arvlogger.getChild('keep').setLevel(logging.WARNING)


def transfer(src_arv, dst_arv, args, verb):
    if re.match(arv_util.collection_uuid_pattern, args.object_uuid):
        result = arv_copy.copy_collection(args.object_uuid, src_arv, dst_arv, args)
    elif re.match(arv_util.group_uuid_pattern, args.object_uuid):
        result = arv_copy.copy_project(args.object_uuid, src_arv, dst_arv, args.project_uuid, args)
    else:
        logger.error("Unsupported object type for %s: %s", verb, args.object_uuid)
        return os.EX_DATAERR
    if error := result.get('partial_error'):
        logger.error(
            "Error copying %s: %s",
            args.object_uuid,
            result if logger.isEnabledFor(logging.DEBUG) else error,
        )
        return os.EX_IOERR
    return os.EX_OK


def export_main(arglist=None):
    args = ArgumentParser.export_parser().parse_args(arglist)
    setup_logging('arv-export', args)
    src_arv = arv_copy.api_for_instance(args.object_uuid[:5], args.retries)
    dst_arv = stubapi.StubArvadosAPI.for_cwd()
    return transfer(src_arv, dst_arv, args, 'export')


def import_main(arglist=None):
    args = ArgumentParser.import_parser().parse_args(arglist)
    setup_logging('arv-import', args)
    src_arv = stubapi.StubArvadosAPI.for_cwd()
    try:
        dst_id = args.project_uuid[:5]
    except TypeError:
        dst_id = ''
    dst_arv = arv_copy.api_for_instance(dst_id, args.retries)
    if args.project_uuid is None:
        args.project_uuid = dst_arv.users().current().execute()['uuid']
    if args.replication is None:
        try:
            args.replication = int(dst_arv.config()["Collections"]["DefaultReplication"])
        except (KeyError, TypeError, ValueError):
            args.replication = 2
    return transfer(src_arv, dst_arv, args, 'import')
