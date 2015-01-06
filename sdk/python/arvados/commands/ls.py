#!/usr/bin/env python

from __future__ import print_function

import argparse

import arvados
import arvados.commands._util as arv_cmd

def parse_args(args):
    parser = argparse.ArgumentParser(
        description='List contents of a manifest',
        parents=[arv_cmd.retry_opt])

    parser.add_argument('locator', type=str,
                        help="""Collection UUID or locator""")
    parser.add_argument('-s', action='store_true',
                        help="""List file sizes, in KiB.""")

    return parser.parse_args(args)

def size_formatter(coll_file):
    return "{:>10}".format((coll_file.size() + 1023) / 1024)

def name_formatter(coll_file):
    return "{}/{}".format(coll_file.stream_name(), coll_file.name)

def main(args, stdout, stderr, api_client=None):
    args = parse_args(args)

    if api_client is None:
        api_client = arvados.api('v1')

    try:
        cr = arvados.CollectionReader(args.locator, api_client=api_client,
                                      num_retries=args.retries)
        cr.normalize()
    except (arvados.errors.ArgumentError,
            arvados.errors.NotFoundError) as error:
        print("arv-ls: error fetching collection: {}".format(error),
              file=stderr)
        return 1

    formatters = []
    if args.s:
        formatters.append(size_formatter)
    formatters.append(name_formatter)

    for f in cr.all_files():
        print(*(info_func(f) for info_func in formatters), file=stdout)

    return 0
