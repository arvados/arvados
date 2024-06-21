#!/usr/bin/env python3
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0
"""pypkg_info.py - Introspect installed Python packages

This tool can read metadata about any Python package installed in the current
environment and report it out in various formats. We use this mainly to pass
information through when building distribution packages.
"""

import argparse
import enum
import importlib.metadata
import os
import sys

from pathlib import PurePath

class RawFormat:
    def format_metadata(self, key, value):
        return value

    def format_path(self, path):
        return str(path)


class FPMFormat(RawFormat):
    PYTHON_METADATA_MAP = {
        'summary': 'description',
    }

    def format_metadata(self, key, value):
        key = key.lower()
        key = self.PYTHON_METADATA_MAP.get(key, key)
        return f'--{key}={value}'


class Formats(enum.Enum):
    RAW = RawFormat
    FPM = FPMFormat

    @classmethod
    def from_arg(cls, arg):
        try:
            return cls[arg.upper()]
        except KeyError:
            raise ValueError(f"unknown format {arg!r}") from None


def report_binfiles(args):
    bin_names = [
        PurePath('bin', path.name)
        for pkg_name in args.package_names
        for path in importlib.metadata.distribution(pkg_name).files
        if path.parts[-3:-1] == ('..', 'bin')
    ]
    fmt = args.format.value().format_path
    return (fmt(path) for path in bin_names)

def report_metadata(args):
    dist = importlib.metadata.distribution(args.package_name)
    fmt = args.format.value().format_metadata
    for key in args.metadata_key:
        yield fmt(key, dist.metadata.get(key, ''))

def unescape_str(arg):
    arg = arg.replace('\'', '\\\'')
    return eval(f"'''{arg}'''", {})

def parse_arguments(arglist=None):
    parser = argparse.ArgumentParser()
    parser.set_defaults(action=None)
    format_names = ', '.join(fmt.name.lower() for fmt in Formats)
    parser.add_argument(
        '--format', '-f',
        choices=list(Formats),
        default=Formats.RAW,
        type=Formats.from_arg,
        help=f"Output format. Choices are: {format_names}",
    )
    parser.add_argument(
        '--delimiter', '-d',
        default='\n',
        type=unescape_str,
        help="Line ending. Python backslash escapes are supported. Default newline.",
    )
    subparsers = parser.add_subparsers()

    binfiles = subparsers.add_parser('binfiles')
    binfiles.set_defaults(action=report_binfiles)
    binfiles.add_argument(
        'package_names',
        nargs=argparse.ONE_OR_MORE,
    )

    metadata = subparsers.add_parser('metadata')
    metadata.set_defaults(action=report_metadata)
    metadata.add_argument(
        'package_name',
    )
    metadata.add_argument(
        'metadata_key',
        nargs=argparse.ONE_OR_MORE,
    )

    args = parser.parse_args()
    if args.action is None:
        parser.error("subcommand is required")
    return args

def main(arglist=None):
    args = parse_arguments(arglist)
    try:
        for line in args.action(args):
            print(line, end=args.delimiter)
    except importlib.metadata.PackageNotFoundError as error:
        print(f"error: package not found: {error.args[0]}", file=sys.stderr)
        return os.EX_NOTFOUND
    else:
        return os.EX_OK

if __name__ == '__main__':
    exit(main())
