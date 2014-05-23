#!/usr/bin/env python

# TODO:
# --md5sum - display md5 of each file as read from disk

import argparse
import arvados
import base64
import errno
import fcntl
import hashlib
import json
import os
import sys
import tempfile

def parse_arguments(arguments):
    parser = argparse.ArgumentParser(
        description='Copy data from the local filesystem to Keep.')

    parser.add_argument('paths', metavar='path', type=str, nargs='*',
                        help="""
    Local file or directory. Default: read from standard input.
    """)

    parser.add_argument('--max-manifest-depth', type=int, metavar='N',
                        default=-1, help="""
    Maximum depth of directory tree to represent in the manifest
    structure. A directory structure deeper than this will be represented
    as a single stream in the manifest. If N=0, the manifest will contain
    a single stream. Default: -1 (unlimited), i.e., exactly one manifest
    stream per filesystem directory that contains files.
    """)

    group = parser.add_mutually_exclusive_group()

    group.add_argument('--as-stream', action='store_true', dest='stream',
                       help="""
    Synonym for --stream.
    """)

    group.add_argument('--stream', action='store_true',
                       help="""
    Store the file content and display the resulting manifest on
    stdout. Do not write the manifest to Keep or save a Collection object
    in Arvados.
    """)

    group.add_argument('--as-manifest', action='store_true', dest='manifest',
                       help="""
    Synonym for --manifest.
    """)

    group.add_argument('--in-manifest', action='store_true', dest='manifest',
                       help="""
    Synonym for --manifest.
    """)

    group.add_argument('--manifest', action='store_true',
                       help="""
    Store the file data and resulting manifest in Keep, save a Collection
    object in Arvados, and display the manifest locator (Collection uuid)
    on stdout. This is the default behavior.
    """)

    group.add_argument('--as-raw', action='store_true', dest='raw',
                       help="""
    Synonym for --raw.
    """)

    group.add_argument('--raw', action='store_true',
                       help="""
    Store the file content and display the data block locators on stdout,
    separated by commas, with a trailing newline. Do not store a
    manifest.
    """)

    parser.add_argument('--use-filename', type=str, default=None,
                        dest='filename', help="""
    Synonym for --filename.
    """)

    parser.add_argument('--filename', type=str, default=None,
                        help="""
    Use the given filename in the manifest, instead of the name of the
    local file. This is useful when "-" or "/dev/stdin" is given as an
    input file. It can be used only if there is exactly one path given and
    it is not a directory. Implies --manifest.
    """)

    group = parser.add_mutually_exclusive_group()
    group.add_argument('--progress', action='store_true',
                       help="""
    Display human-readable progress on stderr (bytes and, if possible,
    percentage of total data size). This is the default behavior when
    stderr is a tty.
    """)

    group.add_argument('--no-progress', action='store_true',
                       help="""
    Do not display human-readable progress on stderr, even if stderr is a
    tty.
    """)

    group.add_argument('--batch-progress', action='store_true',
                       help="""
    Display machine-readable progress on stderr (bytes and, if known,
    total data size).
    """)

    args = parser.parse_args(arguments)

    if len(args.paths) == 0:
        args.paths += ['/dev/stdin']

    if len(args.paths) != 1 or os.path.isdir(args.paths[0]):
        if args.filename:
            parser.error("""
    --filename argument cannot be used when storing a directory or
    multiple files.
    """)

    # Turn on --progress by default if stderr is a tty.
    if (not (args.batch_progress or args.no_progress)
        and os.isatty(sys.stderr.fileno())):
        args.progress = True

    if args.paths == ['-']:
        args.paths = ['/dev/stdin']
        if not args.filename:
            args.filename = '-'

    return args

class ResumeCacheConflict(Exception):
    pass


class ResumeCache(object):
    CACHE_DIR = os.path.expanduser('~/.cache/arvados/arv-put')

    def __init__(self, file_spec):
        try:
            self.cache_file = open(file_spec, 'a+')
        except TypeError:
            file_spec = self.make_path(file_spec)
            self.cache_file = open(file_spec, 'a+')
        self._lock_file(self.cache_file)
        self.filename = self.cache_file.name

    @classmethod
    def make_path(cls, args):
        md5 = hashlib.md5()
        md5.update(arvados.config.get('ARVADOS_API_HOST', '!nohost'))
        realpaths = sorted(os.path.realpath(path) for path in args.paths)
        md5.update(''.join(realpaths))
        if any(os.path.isdir(path) for path in realpaths):
            md5.update(str(max(args.max_manifest_depth, -1)))
        elif args.filename:
            md5.update(args.filename)
        return os.path.join(cls.CACHE_DIR, md5.hexdigest())

    def _lock_file(self, fileobj):
        try:
            fcntl.flock(fileobj, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except IOError:
            raise ResumeCacheConflict("{} locked".format(fileobj.name))

    def load(self):
        self.cache_file.seek(0)
        return json.load(self.cache_file)

    def save(self, data):
        try:
            new_cache_fd, new_cache_name = tempfile.mkstemp(
                dir=os.path.dirname(self.filename))
            self._lock_file(new_cache_fd)
            new_cache = os.fdopen(new_cache_fd, 'r+')
            json.dump(data, new_cache)
            os.rename(new_cache_name, self.filename)
        except (IOError, OSError, ResumeCacheConflict) as error:
            try:
                os.unlink(new_cache_name)
            except NameError:  # mkstemp failed.
                pass
        else:
            self.cache_file.close()
            self.cache_file = new_cache

    def close(self):
        self.cache_file.close()

    def destroy(self):
        try:
            os.unlink(self.filename)
        except OSError as error:
            if error.errno != errno.ENOENT:  # That's what we wanted anyway.
                raise
        self.close()


class ResumeCacheCollectionWriter(arvados.ResumableCollectionWriter):
    def __init__(self, cache=None):
        self.cache = cache
        super(ResumeCacheCollectionWriter, self).__init__()

    @classmethod
    def from_cache(cls, cache):
        try:
            state = cache.load()
            state['_data_buffer'] = [base64.decodestring(state['_data_buffer'])]
            writer = cls.from_state(state)
        except (TypeError, ValueError,
                arvados.errors.StaleWriterStateError) as error:
            return cls(cache)
        else:
            writer.cache = cache
            return writer

    def checkpoint_state(self):
        if self.cache is None:
            return
        state = self.dump_state()
        # Transform attributes for serialization.
        for attr, value in state.items():
            if attr == '_data_buffer':
                state[attr] = base64.encodestring(''.join(value))
            elif hasattr(value, 'popleft'):
                state[attr] = list(value)
        self.cache.save(state)


class CollectionWriterWithProgress(arvados.CollectionWriter):
    def flush_data(self, *args, **kwargs):
        if not getattr(self, 'display_type', None):
            return
        if not hasattr(self, 'bytes_flushed'):
            self.bytes_flushed = 0
        self.bytes_flushed += self._data_buffer_len
        super(CollectionWriterWithProgress, self).flush_data(*args, **kwargs)
        self.bytes_flushed -= self._data_buffer_len
        if self.display_type == 'machine':
            sys.stderr.write('%s %d: %d written %d total\n' %
                             (sys.argv[0],
                              os.getpid(),
                              self.bytes_flushed,
                              getattr(self, 'bytes_expected', -1)))
        elif getattr(self, 'bytes_expected', 0) > 0:
            pct = 100.0 * self.bytes_flushed / self.bytes_expected
            sys.stderr.write('\r%dM / %dM %.1f%% ' %
                             (self.bytes_flushed >> 20,
                              self.bytes_expected >> 20, pct))
        else:
            sys.stderr.write('\r%d ' % self.bytes_flushed)

    def manifest_text(self, *args, **kwargs):
        manifest_text = (super(CollectionWriterWithProgress, self)
                         .manifest_text(*args, **kwargs))
        if getattr(self, 'display_type', None):
            if self.display_type == 'human':
                sys.stderr.write('\n')
            self.display_type = None
        return manifest_text


def expected_bytes_for(pathlist):
    bytesum = 0
    for path in pathlist:
        if os.path.isdir(path):
            for filename in arvados.util.listdir_recursive(path):
                bytesum += os.path.getsize(os.path.join(path, filename))
        elif not os.path.isfile(path):
            return None
        else:
            bytesum += os.path.getsize(path)
    return bytesum

_machine_format = "{} {}: {{}} written {{}} total\n".format(sys.argv[0],
                                                            os.getpid())
def machine_progress(bytes_written, bytes_expected):
    return _machine_format.format(
        bytes_written, -1 if (bytes_expected is None) else bytes_expected)

def human_progress(bytes_written, bytes_expected):
    if bytes_expected:
        return "\r{}M / {}M {:.1f}% ".format(
            bytes_written >> 20, bytes_expected >> 20,
            bytes_written / bytes_expected)
    else:
        return "\r{} ".format(bytes_written)

def main(arguments=None):
    args = parse_arguments(arguments)

    if args.progress:
        writer = CollectionWriterWithProgress()
        writer.display_type = 'human'
    elif args.batch_progress:
        writer = CollectionWriterWithProgress()
        writer.display_type = 'machine'
    else:
        writer = arvados.CollectionWriter()

    # Walk the given directory trees and stat files, adding up file sizes,
    # so we can display progress as percent
    writer.bytes_expected = expected_bytes_for(args.paths)
    if writer.bytes_expected is None:
        del writer.bytes_expected

    # Copy file data to Keep.
    for path in args.paths:
        if os.path.isdir(path):
            writer.write_directory_tree(
                path, max_manifest_depth=args.max_manifest_depth)
        else:
            writer.start_new_stream()
            writer.write_file(path, args.filename or os.path.basename(path))

    if args.stream:
        print writer.manifest_text(),
    elif args.raw:
        writer.finish_current_stream()
        print ','.join(writer.data_locators())
    else:
        # Register the resulting collection in Arvados.
        arvados.api().collections().create(
            body={
                'uuid': writer.finish(),
                'manifest_text': writer.manifest_text(),
                },
            ).execute()

        # Print the locator (uuid) of the new collection.
        print writer.finish()

if __name__ == '__main__':
    main()
