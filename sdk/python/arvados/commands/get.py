#!/usr/bin/env python

import argparse
import hashlib
import os
import re
import string
import sys
import logging

import arvados
import arvados.commands._util as arv_cmd
import arvados.util as util

from arvados._version import __version__

api_client = None
logger = logging.getLogger('arvados.arv-get')

parser = argparse.ArgumentParser(
    description='Copy data from Keep to a local file or pipe.',
    parents=[arv_cmd.retry_opt])
parser.add_argument('--version', action='version',
                    version="%s %s" % (sys.argv[0], __version__),
                    help='Print version and exit.')
parser.add_argument('locator', type=str,
                    help="""
Collection locator, optionally with a file path or prefix.
""")
parser.add_argument('destination', type=str, nargs='?', default='-',
                    help="""
Local file or directory where the data is to be written. Default: stdout.
""")
group = parser.add_mutually_exclusive_group()
group.add_argument('--progress', action='store_true',
                   help="""
Display human-readable progress on stderr (bytes and, if possible,
percentage of total data size). This is the default behavior when it
is not expected to interfere with the output: specifically, stderr is
a tty _and_ either stdout is not a tty, or output is being written to
named files rather than stdout.
""")
group.add_argument('--no-progress', action='store_true',
                   help="""
Do not display human-readable progress on stderr.
""")
group.add_argument('--batch-progress', action='store_true',
                   help="""
Display machine-readable progress on stderr (bytes and, if known,
total data size).
""")
group = parser.add_mutually_exclusive_group()
group.add_argument('--hash',
                    help="""
Display the hash of each file as it is read from Keep, using the given
hash algorithm. Supported algorithms include md5, sha1, sha224,
sha256, sha384, and sha512.
""")
group.add_argument('--md5sum', action='store_const',
                    dest='hash', const='md5',
                    help="""
Display the MD5 hash of each file as it is read from Keep.
""")
parser.add_argument('-n', action='store_true',
                    help="""
Do not write any data -- just read from Keep, and report md5sums if
requested.
""")
parser.add_argument('-r', action='store_true',
                    help="""
Retrieve all files in the specified collection/prefix. This is the
default behavior if the "locator" argument ends with a forward slash.
""")
group = parser.add_mutually_exclusive_group()
group.add_argument('-f', action='store_true',
                   help="""
Overwrite existing files while writing. The default behavior is to
refuse to write *anything* if any of the output files already
exist. As a special case, -f is not needed to write to stdout.
""")
group.add_argument('--skip-existing', action='store_true',
                   help="""
Skip files that already exist. The default behavior is to refuse to
write *anything* if any files exist that would have to be
overwritten. This option causes even devices, sockets, and fifos to be
skipped.
""")
group.add_argument('--strip-manifest', action='store_true', default=False,
                   help="""
When getting a collection manifest, strip its access tokens before writing
it.
""")

def parse_arguments(arguments, stdout, stderr):
    args = parser.parse_args(arguments)

    if args.locator[-1] == os.sep:
        args.r = True
    if (args.r and
        not args.n and
        not (args.destination and
             os.path.isdir(args.destination))):
        parser.error('Destination is not a directory.')
    if not args.r and (os.path.isdir(args.destination) or
                       args.destination[-1] == os.path.sep):
        args.destination = os.path.join(args.destination,
                                        os.path.basename(args.locator))
        logger.debug("Appended source file name to destination directory: %s",
                     args.destination)

    if args.destination == '/dev/stdout':
        args.destination = "-"

    if args.destination == '-':
        # Normally you have to use -f to write to a file (or device) that
        # already exists, but "-" and "/dev/stdout" are common enough to
        # merit a special exception.
        args.f = True
    else:
        args.destination = args.destination.rstrip(os.sep)

    # Turn on --progress by default if stderr is a tty and output is
    # either going to a named file, or going (via stdout) to something
    # that isn't a tty.
    if (not (args.batch_progress or args.no_progress)
        and stderr.isatty()
        and (args.destination != '-'
             or not stdout.isatty())):
        args.progress = True
    return args

def main(arguments=None, stdout=sys.stdout, stderr=sys.stderr):
    global api_client

    if stdout is sys.stdout and hasattr(stdout, 'buffer'):
        # in Python 3, write to stdout as binary
        stdout = stdout.buffer

    args = parse_arguments(arguments, stdout, stderr)
    if api_client is None:
        api_client = arvados.api('v1')

    r = re.search(r'^(.*?)(/.*)?$', args.locator)
    col_loc = r.group(1)
    get_prefix = r.group(2)
    if args.r and not get_prefix:
        get_prefix = os.sep
    try:
        reader = arvados.CollectionReader(col_loc, num_retries=args.retries)
    except Exception as error:
        logger.error("failed to read collection: {}".format(error))
        return 1

    # User asked to download the collection's manifest
    if not get_prefix:
        if not args.n:
            open_flags = os.O_CREAT | os.O_WRONLY
            if not args.f:
                open_flags |= os.O_EXCL
            try:
                if args.destination == "-":
                    stdout.write(reader.manifest_text(strip=args.strip_manifest).encode())
                else:
                    out_fd = os.open(args.destination, open_flags)
                    with os.fdopen(out_fd, 'wb') as out_file:
                        out_file.write(reader.manifest_text(strip=args.strip_manifest).encode())
            except (IOError, OSError) as error:
                logger.error("can't write to '{}': {}".format(args.destination, error))
                return 1
            except (arvados.errors.ApiError, arvados.errors.KeepReadError) as error:
                logger.error("failed to download '{}': {}".format(col_loc, error))
                return 1
        return 0

    # Scan the collection. Make an array of (stream, file, local
    # destination filename) tuples, and add up total size to extract.
    todo = []
    todo_bytes = 0
    try:
        if get_prefix == os.sep:
            item = reader
        else:
            item = reader.find('.' + get_prefix)

        if isinstance(item, arvados.collection.Subcollection) or isinstance(item, arvados.collection.CollectionReader):
            # If the user asked for a file and we got a subcollection, error out.
            if get_prefix[-1] != os.sep:
                logger.error("requested file '{}' is in fact a subcollection. Append a trailing '/' to download it.".format('.' + get_prefix))
                return 1
            # If the user asked stdout as a destination, error out.
            elif args.destination == '-':
                logger.error("cannot use 'stdout' as destination when downloading multiple files.")
                return 1
            # User asked for a subcollection, and that's what was found. Add up total size
            # to download.
            for s, f in files_in_collection(item):
                dest_path = os.path.join(
                    args.destination,
                    os.path.join(s.stream_name(), f.name)[len(get_prefix)+1:])
                if (not (args.n or args.f or args.skip_existing) and
                    os.path.exists(dest_path)):
                    logger.error('Local file %s already exists.' % (dest_path,))
                    return 1
                todo += [(s, f, dest_path)]
                todo_bytes += f.size()
        elif isinstance(item, arvados.arvfile.ArvadosFile):
            todo += [(item.parent, item, args.destination)]
            todo_bytes += item.size()
        else:
            logger.error("'{}' not found.".format('.' + get_prefix))
            return 1
    except (IOError, arvados.errors.NotFoundError) as e:
        logger.error(e)
        return 1

    out_bytes = 0
    for s, f, outfilename in todo:
        outfile = None
        digestor = None
        if not args.n:
            if outfilename == "-":
                outfile = stdout
            else:
                if args.skip_existing and os.path.exists(outfilename):
                    logger.debug('Local file %s exists. Skipping.', outfilename)
                    continue
                elif not args.f and (os.path.isfile(outfilename) or
                                   os.path.isdir(outfilename)):
                    # Good thing we looked again: apparently this file wasn't
                    # here yet when we checked earlier.
                    logger.error('Local file %s already exists.' % (outfilename,))
                    return 1
                if args.r:
                    arvados.util.mkdir_dash_p(os.path.dirname(outfilename))
                try:
                    outfile = open(outfilename, 'wb')
                except Exception as error:
                    logger.error('Open(%s) failed: %s' % (outfilename, error))
                    return 1
        if args.hash:
            digestor = hashlib.new(args.hash)
        try:
            with s.open(f.name, 'rb') as file_reader:
                for data in file_reader.readall():
                    if outfile:
                        outfile.write(data)
                    if digestor:
                        digestor.update(data)
                    out_bytes += len(data)
                    if args.progress:
                        stderr.write('\r%d MiB / %d MiB %.1f%%' %
                                     (out_bytes >> 20,
                                      todo_bytes >> 20,
                                      (100
                                       if todo_bytes==0
                                       else 100.0*out_bytes/todo_bytes)))
                    elif args.batch_progress:
                        stderr.write('%s %d read %d total\n' %
                                     (sys.argv[0], os.getpid(),
                                      out_bytes, todo_bytes))
            if digestor:
                stderr.write("%s  %s/%s\n"
                             % (digestor.hexdigest(), s.stream_name(), f.name))
        except KeyboardInterrupt:
            if outfile and (outfile.fileno() > 2) and not outfile.closed:
                os.unlink(outfile.name)
            break
        finally:
            if outfile != None and outfile != stdout:
                outfile.close()

    if args.progress:
        stderr.write('\n')
    return 0

def files_in_collection(c):
    # Sort first by file type, then alphabetically by file path.
    for i in sorted(list(c.keys()),
                    key=lambda k: (
                        isinstance(c[k], arvados.collection.Subcollection),
                        k.upper())):
        if isinstance(c[i], arvados.arvfile.ArvadosFile):
            yield (c, c[i])
        elif isinstance(c[i], arvados.collection.Subcollection):
            for s, f in files_in_collection(c[i]):
                yield (s, f)
