from __future__ import print_function
from __future__ import division

import argparse
import collections
import logging
import re
import sys

import arvados
import arvados.commands._util as arv_cmd

from arvados._version import __version__

FileInfo = collections.namedtuple('FileInfo', ['stream_name', 'name', 'size'])

def parse_args(args):
    parser = argparse.ArgumentParser(
        description='List contents of a manifest',
        parents=[arv_cmd.retry_opt])

    parser.add_argument('locator', type=str,
                        help="""Collection UUID or locator, optionally with a subdir path.""")
    parser.add_argument('-s', action='store_true',
                        help="""List file sizes, in KiB.""")
    parser.add_argument('--version', action='version',
                        version="%s %s" % (sys.argv[0], __version__),
                        help='Print version and exit.')

    return parser.parse_args(args)

def size_formatter(coll_file):
    return "{:>10}".format((coll_file.size + 1023) // 1024)

def name_formatter(coll_file):
    return "{}/{}".format(coll_file.stream_name, coll_file.name)

def main(args, stdout, stderr, api_client=None, logger=None):
    args = parse_args(args)

    if api_client is None:
        api_client = arvados.api('v1')

    if logger is None:
        logger = logging.getLogger('arvados.arv-ls')

    try:
        r = re.search(r'^(.*?)(/.*)?$', args.locator)
        collection = r.group(1)
        get_prefix = r.group(2)

        cr = arvados.CollectionReader(collection, api_client=api_client,
                                      num_retries=args.retries)
        if get_prefix:
            if get_prefix[-1] == '/':
                get_prefix = get_prefix[:-1]
            stream_name = '.' + get_prefix
            reader = cr.find(stream_name)
            if not (isinstance(reader, arvados.CollectionReader) or
                    isinstance(reader, arvados.collection.Subcollection)):
                logger.error("'{}' is not a subdirectory".format(get_prefix))
                return 1
        else:
            stream_name = '.'
            reader = cr
    except (arvados.errors.ApiError,
            arvados.errors.ArgumentError,
            arvados.errors.NotFoundError) as error:
        logger.error("error fetching collection: {}".format(error))
        return 1

    formatters = []
    if args.s:
        formatters.append(size_formatter)
    formatters.append(name_formatter)

    for f in files_in_collection(reader, stream_name):
        print(*(info_func(f) for info_func in formatters), file=stdout)

    return 0

def files_in_collection(c, stream_name='.'):
    # Sort first by file type, then alphabetically by file path.
    for i in sorted(c.keys(),
                    key=lambda k: (
                        isinstance(c[k], arvados.collection.Subcollection),
                        k.upper())):
        if isinstance(c[i], arvados.arvfile.ArvadosFile):
            yield FileInfo(stream_name=stream_name,
                           name=i,
                           size=c[i].size())
        elif isinstance(c[i], arvados.collection.Subcollection):
            for f in files_in_collection(c[i], "{}/{}".format(stream_name, i)):
                yield f
