# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import division
from future.utils import listitems, listvalues
from builtins import str
from builtins import object
import argparse
import arvados
import arvados.collection
import base64
import copy
import datetime
import errno
import fcntl
import fnmatch
import hashlib
import json
import logging
import os
import pwd
import re
import signal
import socket
import sys
import tempfile
import threading
import time
import traceback

from apiclient import errors as apiclient_errors
from arvados._version import __version__

import arvados.commands._util as arv_cmd

api_client = None

upload_opts = argparse.ArgumentParser(add_help=False)

upload_opts.add_argument('--version', action='version',
                         version="%s %s" % (sys.argv[0], __version__),
                         help='Print version and exit.')
upload_opts.add_argument('paths', metavar='path', type=str, nargs='*',
                         help="""
Local file or directory. If path is a directory reference with a trailing
slash, then just upload the directory's contents; otherwise upload the
directory itself. Default: read from standard input.
""")

_group = upload_opts.add_mutually_exclusive_group()

_group.add_argument('--max-manifest-depth', type=int, metavar='N',
                    default=-1, help=argparse.SUPPRESS)

_group.add_argument('--normalize', action='store_true',
                    help="""
Normalize the manifest by re-ordering files and streams after writing
data.
""")

_group.add_argument('--dry-run', action='store_true', default=False,
                    help="""
Don't actually upload files, but only check if any file should be
uploaded. Exit with code=2 when files are pending for upload.
""")

_group = upload_opts.add_mutually_exclusive_group()

_group.add_argument('--as-stream', action='store_true', dest='stream',
                    help="""
Synonym for --stream.
""")

_group.add_argument('--stream', action='store_true',
                    help="""
Store the file content and display the resulting manifest on
stdout. Do not write the manifest to Keep or save a Collection object
in Arvados.
""")

_group.add_argument('--as-manifest', action='store_true', dest='manifest',
                    help="""
Synonym for --manifest.
""")

_group.add_argument('--in-manifest', action='store_true', dest='manifest',
                    help="""
Synonym for --manifest.
""")

_group.add_argument('--manifest', action='store_true',
                    help="""
Store the file data and resulting manifest in Keep, save a Collection
object in Arvados, and display the manifest locator (Collection uuid)
on stdout. This is the default behavior.
""")

_group.add_argument('--as-raw', action='store_true', dest='raw',
                    help="""
Synonym for --raw.
""")

_group.add_argument('--raw', action='store_true',
                    help="""
Store the file content and display the data block locators on stdout,
separated by commas, with a trailing newline. Do not store a
manifest.
""")

upload_opts.add_argument('--update-collection', type=str, default=None,
                         dest='update_collection', metavar="UUID", help="""
Update an existing collection identified by the given Arvados collection
UUID. All new local files will be uploaded.
""")

upload_opts.add_argument('--use-filename', type=str, default=None,
                         dest='filename', help="""
Synonym for --filename.
""")

upload_opts.add_argument('--filename', type=str, default=None,
                         help="""
Use the given filename in the manifest, instead of the name of the
local file. This is useful when "-" or "/dev/stdin" is given as an
input file. It can be used only if there is exactly one path given and
it is not a directory. Implies --manifest.
""")

upload_opts.add_argument('--portable-data-hash', action='store_true',
                         help="""
Print the portable data hash instead of the Arvados UUID for the collection
created by the upload.
""")

upload_opts.add_argument('--replication', type=int, metavar='N', default=None,
                         help="""
Set the replication level for the new collection: how many different
physical storage devices (e.g., disks) should have a copy of each data
block. Default is to use the server-provided default (if any) or 2.
""")

upload_opts.add_argument('--storage-classes', help="""
Specify comma separated list of storage classes to be used when saving data to Keep.
""")

upload_opts.add_argument('--threads', type=int, metavar='N', default=None,
                         help="""
Set the number of upload threads to be used. Take into account that
using lots of threads will increase the RAM requirements. Default is
to use 2 threads.
On high latency installations, using a greater number will improve
overall throughput.
""")

run_opts = argparse.ArgumentParser(add_help=False)

run_opts.add_argument('--project-uuid', metavar='UUID', help="""
Store the collection in the specified project, instead of your Home
project.
""")

run_opts.add_argument('--name', help="""
Save the collection with the specified name.
""")

run_opts.add_argument('--exclude', metavar='PATTERN', default=[],
                      action='append', help="""
Exclude files and directories whose names match the given glob pattern. When
using a path-like pattern like 'subdir/*.txt', all text files inside 'subdir'
directory, relative to the provided input dirs will be excluded.
When using a filename pattern like '*.txt', any text file will be excluded
no matter where is placed.
For the special case of needing to exclude only files or dirs directly below
the given input directory, you can use a pattern like './exclude_this.gif'.
You can specify multiple patterns by using this argument more than once.
""")

_group = run_opts.add_mutually_exclusive_group()
_group.add_argument('--progress', action='store_true',
                    help="""
Display human-readable progress on stderr (bytes and, if possible,
percentage of total data size). This is the default behavior when
stderr is a tty.
""")

_group.add_argument('--no-progress', action='store_true',
                    help="""
Do not display human-readable progress on stderr, even if stderr is a
tty.
""")

_group.add_argument('--batch-progress', action='store_true',
                    help="""
Display machine-readable progress on stderr (bytes and, if known,
total data size).
""")

run_opts.add_argument('--silent', action='store_true',
                      help="""
Do not print any debug messages to console. (Any error messages will
still be displayed.)
""")

_group = run_opts.add_mutually_exclusive_group()
_group.add_argument('--resume', action='store_true', default=True,
                    help="""
Continue interrupted uploads from cached state (default).
""")
_group.add_argument('--no-resume', action='store_false', dest='resume',
                    help="""
Do not continue interrupted uploads from cached state.
""")

_group = run_opts.add_mutually_exclusive_group()
_group.add_argument('--follow-links', action='store_true', default=True,
                    dest='follow_links', help="""
Follow file and directory symlinks (default).
""")
_group.add_argument('--no-follow-links', action='store_false', dest='follow_links',
                    help="""
Do not follow file and directory symlinks.
""")

_group = run_opts.add_mutually_exclusive_group()
_group.add_argument('--cache', action='store_true', dest='use_cache', default=True,
                    help="""
Save upload state in a cache file for resuming (default).
""")
_group.add_argument('--no-cache', action='store_false', dest='use_cache',
                    help="""
Do not save upload state in a cache file for resuming.
""")

arg_parser = argparse.ArgumentParser(
    description='Copy data from the local filesystem to Keep.',
    parents=[upload_opts, run_opts, arv_cmd.retry_opt])

def parse_arguments(arguments):
    args = arg_parser.parse_args(arguments)

    if len(args.paths) == 0:
        args.paths = ['-']

    args.paths = ["-" if x == "/dev/stdin" else x for x in args.paths]

    if len(args.paths) != 1 or os.path.isdir(args.paths[0]):
        if args.filename:
            arg_parser.error("""
    --filename argument cannot be used when storing a directory or
    multiple files.
    """)

    # Turn on --progress by default if stderr is a tty.
    if (not (args.batch_progress or args.no_progress or args.silent)
        and os.isatty(sys.stderr.fileno())):
        args.progress = True

    # Turn off --resume (default) if --no-cache is used.
    if not args.use_cache:
        args.resume = False

    if args.paths == ['-']:
        if args.update_collection:
            arg_parser.error("""
    --update-collection cannot be used when reading from stdin.
    """)
        args.resume = False
        args.use_cache = False
        if not args.filename:
            args.filename = 'stdin'

    # Remove possible duplicated patterns
    if len(args.exclude) > 0:
        args.exclude = list(set(args.exclude))

    return args


class PathDoesNotExistError(Exception):
    pass


class CollectionUpdateError(Exception):
    pass


class ResumeCacheConflict(Exception):
    pass


class ArvPutArgumentConflict(Exception):
    pass


class ArvPutUploadIsPending(Exception):
    pass


class ArvPutUploadNotPending(Exception):
    pass


class FileUploadList(list):
    def __init__(self, dry_run=False):
        list.__init__(self)
        self.dry_run = dry_run

    def append(self, other):
        if self.dry_run:
            raise ArvPutUploadIsPending()
        super(FileUploadList, self).append(other)


# Appends the X-Request-Id to the log message when log level is ERROR or DEBUG
class ArvPutLogFormatter(logging.Formatter):
    std_fmtr = logging.Formatter(arvados.log_format, arvados.log_date_format)
    err_fmtr = None
    request_id_informed = False

    def __init__(self, request_id):
        self.err_fmtr = logging.Formatter(
            arvados.log_format+' (X-Request-Id: {})'.format(request_id),
            arvados.log_date_format)

    def format(self, record):
        if (not self.request_id_informed) and (record.levelno in (logging.DEBUG, logging.ERROR)):
            self.request_id_informed = True
            return self.err_fmtr.format(record)
        return self.std_fmtr.format(record)


class ResumeCache(object):
    CACHE_DIR = '.cache/arvados/arv-put'

    def __init__(self, file_spec):
        self.cache_file = open(file_spec, 'a+')
        self._lock_file(self.cache_file)
        self.filename = self.cache_file.name

    @classmethod
    def make_path(cls, args):
        md5 = hashlib.md5()
        md5.update(arvados.config.get('ARVADOS_API_HOST', '!nohost').encode())
        realpaths = sorted(os.path.realpath(path) for path in args.paths)
        md5.update(b'\0'.join([p.encode() for p in realpaths]))
        if any(os.path.isdir(path) for path in realpaths):
            md5.update(b'-1')
        elif args.filename:
            md5.update(args.filename.encode())
        return os.path.join(
            arv_cmd.make_home_conf_dir(cls.CACHE_DIR, 0o700, 'raise'),
            md5.hexdigest())

    def _lock_file(self, fileobj):
        try:
            fcntl.flock(fileobj, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except IOError:
            raise ResumeCacheConflict("{} locked".format(fileobj.name))

    def load(self):
        self.cache_file.seek(0)
        return json.load(self.cache_file)

    def check_cache(self, api_client=None, num_retries=0):
        try:
            state = self.load()
            locator = None
            try:
                if "_finished_streams" in state and len(state["_finished_streams"]) > 0:
                    locator = state["_finished_streams"][0][1][0]
                elif "_current_stream_locators" in state and len(state["_current_stream_locators"]) > 0:
                    locator = state["_current_stream_locators"][0]
                if locator is not None:
                    kc = arvados.keep.KeepClient(api_client=api_client)
                    kc.head(locator, num_retries=num_retries)
            except Exception as e:
                self.restart()
        except (ValueError):
            pass

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

    def restart(self):
        self.destroy()
        self.__init__(self.filename)


class ArvPutUploadJob(object):
    CACHE_DIR = '.cache/arvados/arv-put'
    EMPTY_STATE = {
        'manifest' : None, # Last saved manifest checkpoint
        'files' : {} # Previous run file list: {path : {size, mtime}}
    }

    def __init__(self, paths, resume=True, use_cache=True, reporter=None,
                 name=None, owner_uuid=None, api_client=None,
                 ensure_unique_name=False, num_retries=None,
                 put_threads=None, replication_desired=None, filename=None,
                 update_time=60.0, update_collection=None, storage_classes=None,
                 logger=logging.getLogger('arvados.arv_put'), dry_run=False,
                 follow_links=True, exclude_paths=[], exclude_names=None):
        self.paths = paths
        self.resume = resume
        self.use_cache = use_cache
        self.update = False
        self.reporter = reporter
        # This will set to 0 before start counting, if no special files are going
        # to be read.
        self.bytes_expected = None
        self.bytes_written = 0
        self.bytes_skipped = 0
        self.name = name
        self.owner_uuid = owner_uuid
        self.ensure_unique_name = ensure_unique_name
        self.num_retries = num_retries
        self.replication_desired = replication_desired
        self.put_threads = put_threads
        self.filename = filename
        self.storage_classes = storage_classes
        self._api_client = api_client
        self._state_lock = threading.Lock()
        self._state = None # Previous run state (file list & manifest)
        self._current_files = [] # Current run file list
        self._cache_file = None
        self._collection_lock = threading.Lock()
        self._remote_collection = None # Collection being updated (if asked)
        self._local_collection = None # Collection from previous run manifest
        self._file_paths = set() # Files to be updated in remote collection
        self._stop_checkpointer = threading.Event()
        self._checkpointer = threading.Thread(target=self._update_task)
        self._checkpointer.daemon = True
        self._update_task_time = update_time  # How many seconds wait between update runs
        self._files_to_upload = FileUploadList(dry_run=dry_run)
        self._upload_started = False
        self.logger = logger
        self.dry_run = dry_run
        self._checkpoint_before_quit = True
        self.follow_links = follow_links
        self.exclude_paths = exclude_paths
        self.exclude_names = exclude_names

        if not self.use_cache and self.resume:
            raise ArvPutArgumentConflict('resume cannot be True when use_cache is False')

        # Check for obvious dry-run responses
        if self.dry_run and (not self.use_cache or not self.resume):
            raise ArvPutUploadIsPending()

        # Load cached data if any and if needed
        self._setup_state(update_collection)

        # Build the upload file list, excluding requested files and counting the
        # bytes expected to be uploaded.
        self._build_upload_list()

    def _build_upload_list(self):
        """
        Scan the requested paths to count file sizes, excluding files & dirs if requested
        and building the upload file list.
        """
        # If there aren't special files to be read, reset total bytes count to zero
        # to start counting.
        if not any([p for p in self.paths
                    if not (os.path.isfile(p) or os.path.isdir(p))]):
            self.bytes_expected = 0

        for path in self.paths:
            # Test for stdin first, in case some file named '-' exist
            if path == '-':
                if self.dry_run:
                    raise ArvPutUploadIsPending()
                self._write_stdin(self.filename or 'stdin')
            elif not os.path.exists(path):
                 raise PathDoesNotExistError("file or directory '{}' does not exist.".format(path))
            elif os.path.isdir(path):
                # Use absolute paths on cache index so CWD doesn't interfere
                # with the caching logic.
                orig_path = path
                path = os.path.abspath(path)
                if orig_path[-1:] == os.sep:
                    # When passing a directory reference with a trailing slash,
                    # its contents should be uploaded directly to the
                    # collection's root.
                    prefixdir = path
                else:
                    # When passing a directory reference with no trailing slash,
                    # upload the directory to the collection's root.
                    prefixdir = os.path.dirname(path)
                prefixdir += os.sep
                for root, dirs, files in os.walk(path,
                                                 followlinks=self.follow_links):
                    root_relpath = os.path.relpath(root, path)
                    if root_relpath == '.':
                        root_relpath = ''
                    # Exclude files/dirs by full path matching pattern
                    if self.exclude_paths:
                        dirs[:] = [d for d in dirs
                                   if not any(pathname_match(
                                           os.path.join(root_relpath, d), pat)
                                              for pat in self.exclude_paths)]
                        files = [f for f in files
                                 if not any(pathname_match(
                                         os.path.join(root_relpath, f), pat)
                                            for pat in self.exclude_paths)]
                    # Exclude files/dirs by name matching pattern
                    if self.exclude_names is not None:
                        dirs[:] = [d for d in dirs
                                   if not self.exclude_names.match(d)]
                        files = [f for f in files
                                 if not self.exclude_names.match(f)]
                    # Make os.walk()'s dir traversing order deterministic
                    dirs.sort()
                    files.sort()
                    for f in files:
                        filepath = os.path.join(root, f)
                        # Add its size to the total bytes count (if applicable)
                        if self.follow_links or (not os.path.islink(filepath)):
                            if self.bytes_expected is not None:
                                self.bytes_expected += os.path.getsize(filepath)
                        self._check_file(filepath,
                                         os.path.join(root[len(prefixdir):], f))
            else:
                filepath = os.path.abspath(path)
                # Add its size to the total bytes count (if applicable)
                if self.follow_links or (not os.path.islink(filepath)):
                    if self.bytes_expected is not None:
                        self.bytes_expected += os.path.getsize(filepath)
                self._check_file(filepath,
                                 self.filename or os.path.basename(path))
        # If dry-mode is on, and got up to this point, then we should notify that
        # there aren't any file to upload.
        if self.dry_run:
            raise ArvPutUploadNotPending()
        # Remove local_collection's files that don't exist locally anymore, so the
        # bytes_written count is correct.
        for f in self.collection_file_paths(self._local_collection,
                                            path_prefix=""):
            if f != 'stdin' and f != self.filename and not f in self._file_paths:
                self._local_collection.remove(f)

    def start(self, save_collection):
        """
        Start supporting thread & file uploading
        """
        self._checkpointer.start()
        try:
            # Update bytes_written from current local collection and
            # report initial progress.
            self._update()
            # Actual file upload
            self._upload_started = True # Used by the update thread to start checkpointing
            self._upload_files()
        except (SystemExit, Exception) as e:
            self._checkpoint_before_quit = False
            # Log stack trace only when Ctrl-C isn't pressed (SIGINT)
            # Note: We're expecting SystemExit instead of
            # KeyboardInterrupt because we have a custom signal
            # handler in place that raises SystemExit with the catched
            # signal's code.
            if isinstance(e, PathDoesNotExistError):
                # We aren't interested in the traceback for this case
                pass
            elif not isinstance(e, SystemExit) or e.code != -2:
                self.logger.warning("Abnormal termination:\n{}".format(
                    traceback.format_exc()))
            raise
        finally:
            if not self.dry_run:
                # Stop the thread before doing anything else
                self._stop_checkpointer.set()
                self._checkpointer.join()
                if self._checkpoint_before_quit:
                    # Commit all pending blocks & one last _update()
                    self._local_collection.manifest_text()
                    self._update(final=True)
                    if save_collection:
                        self.save_collection()
            if self.use_cache:
                self._cache_file.close()

    def save_collection(self):
        if self.update:
            # Check if files should be updated on the remote collection.
            for fp in self._file_paths:
                remote_file = self._remote_collection.find(fp)
                if not remote_file:
                    # File don't exist on remote collection, copy it.
                    self._remote_collection.copy(fp, fp, self._local_collection)
                elif remote_file != self._local_collection.find(fp):
                    # A different file exist on remote collection, overwrite it.
                    self._remote_collection.copy(fp, fp, self._local_collection, overwrite=True)
                else:
                    # The file already exist on remote collection, skip it.
                    pass
            self._remote_collection.save(storage_classes=self.storage_classes,
                                         num_retries=self.num_retries)
        else:
            if self.storage_classes is None:
                self.storage_classes = ['default']
            self._local_collection.save_new(
                name=self.name, owner_uuid=self.owner_uuid,
                storage_classes=self.storage_classes,
                ensure_unique_name=self.ensure_unique_name,
                num_retries=self.num_retries)

    def destroy_cache(self):
        if self.use_cache:
            try:
                os.unlink(self._cache_filename)
            except OSError as error:
                # That's what we wanted anyway.
                if error.errno != errno.ENOENT:
                    raise
            self._cache_file.close()

    def _collection_size(self, collection):
        """
        Recursively get the total size of the collection
        """
        size = 0
        for item in listvalues(collection):
            if isinstance(item, arvados.collection.Collection) or isinstance(item, arvados.collection.Subcollection):
                size += self._collection_size(item)
            else:
                size += item.size()
        return size

    def _update_task(self):
        """
        Periodically called support task. File uploading is
        asynchronous so we poll status from the collection.
        """
        while not self._stop_checkpointer.wait(1 if not self._upload_started else self._update_task_time):
            self._update()

    def _update(self, final=False):
        """
        Update cached manifest text and report progress.
        """
        if self._upload_started:
            with self._collection_lock:
                self.bytes_written = self._collection_size(self._local_collection)
                if self.use_cache:
                    if final:
                        manifest = self._local_collection.manifest_text()
                    else:
                        # Get the manifest text without comitting pending blocks
                        manifest = self._local_collection.manifest_text(strip=False,
                                                                        normalize=False,
                                                                        only_committed=True)
                    # Update cache
                    with self._state_lock:
                        self._state['manifest'] = manifest
            if self.use_cache:
                try:
                    self._save_state()
                except Exception as e:
                    self.logger.error("Unexpected error trying to save cache file: {}".format(e))
        else:
            self.bytes_written = self.bytes_skipped
        # Call the reporter, if any
        self.report_progress()

    def report_progress(self):
        if self.reporter is not None:
            self.reporter(self.bytes_written, self.bytes_expected)

    def _write_stdin(self, filename):
        output = self._local_collection.open(filename, 'wb')
        self._write(sys.stdin, output)
        output.close()

    def _check_file(self, source, filename):
        """
        Check if this file needs to be uploaded
        """
        # Ignore symlinks when requested
        if (not self.follow_links) and os.path.islink(source):
            return
        resume_offset = 0
        should_upload = False
        new_file_in_cache = False
        # Record file path for updating the remote collection before exiting
        self._file_paths.add(filename)

        with self._state_lock:
            # If no previous cached data on this file, store it for an eventual
            # repeated run.
            if source not in self._state['files']:
                self._state['files'][source] = {
                    'mtime': os.path.getmtime(source),
                    'size' : os.path.getsize(source)
                }
                new_file_in_cache = True
            cached_file_data = self._state['files'][source]

        # Check if file was already uploaded (at least partially)
        file_in_local_collection = self._local_collection.find(filename)

        # If not resuming, upload the full file.
        if not self.resume:
            should_upload = True
        # New file detected from last run, upload it.
        elif new_file_in_cache:
            should_upload = True
        # Local file didn't change from last run.
        elif cached_file_data['mtime'] == os.path.getmtime(source) and cached_file_data['size'] == os.path.getsize(source):
            if not file_in_local_collection:
                # File not uploaded yet, upload it completely
                should_upload = True
            elif file_in_local_collection.permission_expired():
                # Permission token expired, re-upload file. This will change whenever
                # we have a API for refreshing tokens.
                self.logger.warning("Uploaded file '{}' access token expired, will re-upload it from scratch".format(filename))
                should_upload = True
                self._local_collection.remove(filename)
            elif cached_file_data['size'] == file_in_local_collection.size():
                # File already there, skip it.
                self.bytes_skipped += cached_file_data['size']
            elif cached_file_data['size'] > file_in_local_collection.size():
                # File partially uploaded, resume!
                resume_offset = file_in_local_collection.size()
                self.bytes_skipped += resume_offset
                should_upload = True
            else:
                # Inconsistent cache, re-upload the file
                should_upload = True
                self._local_collection.remove(filename)
                self.logger.warning("Uploaded version of file '{}' is bigger than local version, will re-upload it from scratch.".format(source))
        # Local file differs from cached data, re-upload it.
        else:
            if file_in_local_collection:
                self._local_collection.remove(filename)
            should_upload = True

        if should_upload:
            try:
                self._files_to_upload.append((source, resume_offset, filename))
            except ArvPutUploadIsPending:
                # This could happen when running on dry-mode, close cache file to
                # avoid locking issues.
                self._cache_file.close()
                raise

    def _upload_files(self):
        for source, resume_offset, filename in self._files_to_upload:
            with open(source, 'rb') as source_fd:
                with self._state_lock:
                    self._state['files'][source]['mtime'] = os.path.getmtime(source)
                    self._state['files'][source]['size'] = os.path.getsize(source)
                if resume_offset > 0:
                    # Start upload where we left off
                    output = self._local_collection.open(filename, 'ab')
                    source_fd.seek(resume_offset)
                else:
                    # Start from scratch
                    output = self._local_collection.open(filename, 'wb')
                self._write(source_fd, output)
                output.close(flush=False)

    def _write(self, source_fd, output):
        while True:
            data = source_fd.read(arvados.config.KEEP_BLOCK_SIZE)
            if not data:
                break
            output.write(data)

    def _my_collection(self):
        return self._remote_collection if self.update else self._local_collection

    def _setup_state(self, update_collection):
        """
        Create a new cache file or load a previously existing one.
        """
        # Load an already existing collection for update
        if update_collection and re.match(arvados.util.collection_uuid_pattern,
                                          update_collection):
            try:
                self._remote_collection = arvados.collection.Collection(
                    update_collection, api_client=self._api_client)
            except arvados.errors.ApiError as error:
                raise CollectionUpdateError("Cannot read collection {} ({})".format(update_collection, error))
            else:
                self.update = True
        elif update_collection:
            # Collection locator provided, but unknown format
            raise CollectionUpdateError("Collection locator unknown: '{}'".format(update_collection))

        if self.use_cache:
            # Set up cache file name from input paths.
            md5 = hashlib.md5()
            md5.update(arvados.config.get('ARVADOS_API_HOST', '!nohost').encode())
            realpaths = sorted(os.path.realpath(path) for path in self.paths)
            md5.update(b'\0'.join([p.encode() for p in realpaths]))
            if self.filename:
                md5.update(self.filename.encode())
            cache_filename = md5.hexdigest()
            cache_filepath = os.path.join(
                arv_cmd.make_home_conf_dir(self.CACHE_DIR, 0o700, 'raise'),
                cache_filename)
            if self.resume and os.path.exists(cache_filepath):
                self.logger.info("Resuming upload from cache file {}".format(cache_filepath))
                self._cache_file = open(cache_filepath, 'a+')
            else:
                # --no-resume means start with a empty cache file.
                self.logger.info("Creating new cache file at {}".format(cache_filepath))
                self._cache_file = open(cache_filepath, 'w+')
            self._cache_filename = self._cache_file.name
            self._lock_file(self._cache_file)
            self._cache_file.seek(0)

        with self._state_lock:
            if self.use_cache:
                try:
                    self._state = json.load(self._cache_file)
                    if not set(['manifest', 'files']).issubset(set(self._state.keys())):
                        # Cache at least partially incomplete, set up new cache
                        self._state = copy.deepcopy(self.EMPTY_STATE)
                except ValueError:
                    # Cache file empty, set up new cache
                    self._state = copy.deepcopy(self.EMPTY_STATE)
            else:
                self.logger.info("No cache usage requested for this run.")
                # No cache file, set empty state
                self._state = copy.deepcopy(self.EMPTY_STATE)
            # Load the previous manifest so we can check if files were modified remotely.
            self._local_collection = arvados.collection.Collection(
                self._state['manifest'],
                replication_desired=self.replication_desired,
                put_threads=self.put_threads,
                api_client=self._api_client)

    def collection_file_paths(self, col, path_prefix='.'):
        """Return a list of file paths by recursively go through the entire collection `col`"""
        file_paths = []
        for name, item in listitems(col):
            if isinstance(item, arvados.arvfile.ArvadosFile):
                file_paths.append(os.path.join(path_prefix, name))
            elif isinstance(item, arvados.collection.Subcollection):
                new_prefix = os.path.join(path_prefix, name)
                file_paths += self.collection_file_paths(item, path_prefix=new_prefix)
        return file_paths

    def _lock_file(self, fileobj):
        try:
            fcntl.flock(fileobj, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except IOError:
            raise ResumeCacheConflict("{} locked".format(fileobj.name))

    def _save_state(self):
        """
        Atomically save current state into cache.
        """
        with self._state_lock:
            # We're not using copy.deepcopy() here because it's a lot slower
            # than json.dumps(), and we're already needing JSON format to be
            # saved on disk.
            state = json.dumps(self._state)
        try:
            new_cache = tempfile.NamedTemporaryFile(
                mode='w+',
                dir=os.path.dirname(self._cache_filename), delete=False)
            self._lock_file(new_cache)
            new_cache.write(state)
            new_cache.flush()
            os.fsync(new_cache)
            os.rename(new_cache.name, self._cache_filename)
        except (IOError, OSError, ResumeCacheConflict) as error:
            self.logger.error("There was a problem while saving the cache file: {}".format(error))
            try:
                os.unlink(new_cache_name)
            except NameError:  # mkstemp failed.
                pass
        else:
            self._cache_file.close()
            self._cache_file = new_cache

    def collection_name(self):
        return self._my_collection().api_response()['name'] if self._my_collection().api_response() else None

    def manifest_locator(self):
        return self._my_collection().manifest_locator()

    def portable_data_hash(self):
        pdh = self._my_collection().portable_data_hash()
        m = self._my_collection().stripped_manifest().encode()
        local_pdh = '{}+{}'.format(hashlib.md5(m).hexdigest(), len(m))
        if pdh != local_pdh:
            self.logger.warning("\n".join([
                "arv-put: API server provided PDH differs from local manifest.",
                "         This should not happen; showing API server version."]))
        return pdh

    def manifest_text(self, stream_name=".", strip=False, normalize=False):
        return self._my_collection().manifest_text(stream_name, strip, normalize)

    def _datablocks_on_item(self, item):
        """
        Return a list of datablock locators, recursively navigating
        through subcollections
        """
        if isinstance(item, arvados.arvfile.ArvadosFile):
            if item.size() == 0:
                # Empty file locator
                return ["d41d8cd98f00b204e9800998ecf8427e+0"]
            else:
                locators = []
                for segment in item.segments():
                    loc = segment.locator
                    locators.append(loc)
                return locators
        elif isinstance(item, arvados.collection.Collection):
            l = [self._datablocks_on_item(x) for x in listvalues(item)]
            # Fast list flattener method taken from:
            # http://stackoverflow.com/questions/952914/making-a-flat-list-out-of-list-of-lists-in-python
            return [loc for sublist in l for loc in sublist]
        else:
            return None

    def data_locators(self):
        with self._collection_lock:
            # Make sure all datablocks are flushed before getting the locators
            self._my_collection().manifest_text()
            datablocks = self._datablocks_on_item(self._my_collection())
        return datablocks

_machine_format = "{} {}: {{}} written {{}} total\n".format(sys.argv[0],
                                                            os.getpid())

# Simulate glob.glob() matching behavior without the need to scan the filesystem
# Note: fnmatch() doesn't work correctly when used with pathnames. For example the
# pattern 'tests/*.py' will match 'tests/run_test.py' and also 'tests/subdir/run_test.py',
# so instead we're using it on every path component.
def pathname_match(pathname, pattern):
    name = pathname.split(os.sep)
    # Fix patterns like 'some/subdir/' or 'some//subdir'
    pat = [x for x in pattern.split(os.sep) if x != '' and x != '.']
    if len(name) != len(pat):
        return False
    for i in range(len(name)):
        if not fnmatch.fnmatch(name[i], pat[i]):
            return False
    return True

def machine_progress(bytes_written, bytes_expected):
    return _machine_format.format(
        bytes_written, -1 if (bytes_expected is None) else bytes_expected)

def human_progress(bytes_written, bytes_expected):
    if bytes_expected:
        return "\r{}M / {}M {:.1%} ".format(
            bytes_written >> 20, bytes_expected >> 20,
            float(bytes_written) / bytes_expected)
    else:
        return "\r{} ".format(bytes_written)

def progress_writer(progress_func, outfile=sys.stderr):
    def write_progress(bytes_written, bytes_expected):
        outfile.write(progress_func(bytes_written, bytes_expected))
    return write_progress

def desired_project_uuid(api_client, project_uuid, num_retries):
    if not project_uuid:
        query = api_client.users().current()
    elif arvados.util.user_uuid_pattern.match(project_uuid):
        query = api_client.users().get(uuid=project_uuid)
    elif arvados.util.group_uuid_pattern.match(project_uuid):
        query = api_client.groups().get(uuid=project_uuid)
    else:
        raise ValueError("Not a valid project UUID: {}".format(project_uuid))
    return query.execute(num_retries=num_retries)['uuid']

def main(arguments=None, stdout=sys.stdout, stderr=sys.stderr,
         install_sig_handlers=True):
    global api_client

    args = parse_arguments(arguments)
    logger = logging.getLogger('arvados.arv_put')
    if args.silent:
        logger.setLevel(logging.WARNING)
    else:
        logger.setLevel(logging.INFO)
    status = 0

    request_id = arvados.util.new_request_id()

    formatter = ArvPutLogFormatter(request_id)
    logging.getLogger('arvados').handlers[0].setFormatter(formatter)

    if api_client is None:
        api_client = arvados.api('v1', request_id=request_id)

    if install_sig_handlers:
        arv_cmd.install_signal_handlers()

    # Determine the name to use
    if args.name:
        if args.stream or args.raw:
            logger.error("Cannot use --name with --stream or --raw")
            sys.exit(1)
        elif args.update_collection:
            logger.error("Cannot use --name with --update-collection")
            sys.exit(1)
        collection_name = args.name
    else:
        collection_name = "Saved at {} by {}@{}".format(
            datetime.datetime.utcnow().strftime("%Y-%m-%d %H:%M:%S UTC"),
            pwd.getpwuid(os.getuid()).pw_name,
            socket.gethostname())

    if args.project_uuid and (args.stream or args.raw):
        logger.error("Cannot use --project-uuid with --stream or --raw")
        sys.exit(1)

    # Determine the parent project
    try:
        project_uuid = desired_project_uuid(api_client, args.project_uuid,
                                            args.retries)
    except (apiclient_errors.Error, ValueError) as error:
        logger.error(error)
        sys.exit(1)

    if args.progress:
        reporter = progress_writer(human_progress)
    elif args.batch_progress:
        reporter = progress_writer(machine_progress)
    else:
        reporter = None

    #  Split storage-classes argument
    storage_classes = None
    if args.storage_classes:
        storage_classes = args.storage_classes.strip().split(',')
        if len(storage_classes) > 1:
            logger.error("Multiple storage classes are not supported currently.")
            sys.exit(1)


    # Setup exclude regex from all the --exclude arguments provided
    name_patterns = []
    exclude_paths = []
    exclude_names = None
    if len(args.exclude) > 0:
        # We're supporting 2 kinds of exclusion patterns:
        # 1)   --exclude '*.jpg'    (file/dir name patterns, will only match
        #                            the name, wherever the file is on the tree)
        # 2.1) --exclude 'foo/bar'  (file/dir path patterns, will match the
        #                            entire path, and should be relative to
        #                            any input dir argument)
        # 2.2) --exclude './*.jpg'  (Special case for excluding files/dirs
        #                            placed directly underneath the input dir)
        for p in args.exclude:
            # Only relative paths patterns allowed
            if p.startswith(os.sep):
                logger.error("Cannot use absolute paths with --exclude")
                sys.exit(1)
            if os.path.dirname(p):
                # We don't support of path patterns with '..'
                p_parts = p.split(os.sep)
                if '..' in p_parts:
                    logger.error(
                        "Cannot use path patterns that include or '..'")
                    sys.exit(1)
                # Path search pattern
                exclude_paths.append(p)
            else:
                # Name-only search pattern
                name_patterns.append(p)
        # For name only matching, we can combine all patterns into a single
        # regexp, for better performance.
        exclude_names = re.compile('|'.join(
            [fnmatch.translate(p) for p in name_patterns]
        )) if len(name_patterns) > 0 else None
        # Show the user the patterns to be used, just in case they weren't
        # specified inside quotes and got changed by the shell expansion.
        logger.info("Exclude patterns: {}".format(args.exclude))

    # If this is used by a human, and there's at least one directory to be
    # uploaded, the expected bytes calculation can take a moment.
    if args.progress and any([os.path.isdir(f) for f in args.paths]):
        logger.info("Calculating upload size, this could take some time...")
    try:
        writer = ArvPutUploadJob(paths = args.paths,
                                 resume = args.resume,
                                 use_cache = args.use_cache,
                                 filename = args.filename,
                                 reporter = reporter,
                                 api_client = api_client,
                                 num_retries = args.retries,
                                 replication_desired = args.replication,
                                 put_threads = args.threads,
                                 name = collection_name,
                                 owner_uuid = project_uuid,
                                 ensure_unique_name = True,
                                 update_collection = args.update_collection,
                                 storage_classes=storage_classes,
                                 logger=logger,
                                 dry_run=args.dry_run,
                                 follow_links=args.follow_links,
                                 exclude_paths=exclude_paths,
                                 exclude_names=exclude_names)
    except ResumeCacheConflict:
        logger.error("\n".join([
            "arv-put: Another process is already uploading this data.",
            "         Use --no-cache if this is really what you want."]))
        sys.exit(1)
    except CollectionUpdateError as error:
        logger.error("\n".join([
            "arv-put: %s" % str(error)]))
        sys.exit(1)
    except ArvPutUploadIsPending:
        # Dry run check successful, return proper exit code.
        sys.exit(2)
    except ArvPutUploadNotPending:
        # No files pending for upload
        sys.exit(0)
    except PathDoesNotExistError as error:
        logger.error("\n".join([
            "arv-put: %s" % str(error)]))
        sys.exit(1)

    if not args.dry_run and not args.update_collection and args.resume and writer.bytes_written > 0:
        logger.warning("\n".join([
            "arv-put: Resuming previous upload from last checkpoint.",
            "         Use the --no-resume option to start over."]))

    if not args.dry_run:
        writer.report_progress()
    output = None
    try:
        writer.start(save_collection=not(args.stream or args.raw))
    except arvados.errors.ApiError as error:
        logger.error("\n".join([
            "arv-put: %s" % str(error)]))
        sys.exit(1)

    if args.progress:  # Print newline to split stderr from stdout for humans.
        logger.info("\n")

    if args.stream:
        if args.normalize:
            output = writer.manifest_text(normalize=True)
        else:
            output = writer.manifest_text()
    elif args.raw:
        output = ','.join(writer.data_locators())
    else:
        try:
            if args.update_collection:
                logger.info("Collection updated: '{}'".format(writer.collection_name()))
            else:
                logger.info("Collection saved as '{}'".format(writer.collection_name()))
            if args.portable_data_hash:
                output = writer.portable_data_hash()
            else:
                output = writer.manifest_locator()
        except apiclient_errors.Error as error:
            logger.error(
                "arv-put: Error creating Collection on project: {}.".format(
                    error))
            status = 1

    # Print the locator (uuid) of the new collection.
    if output is None:
        status = status or 1
    elif not args.silent:
        stdout.write(output)
        if not output.endswith('\n'):
            stdout.write('\n')

    if install_sig_handlers:
        arv_cmd.restore_signal_handlers()

    if status != 0:
        sys.exit(status)

    # Success!
    return output


if __name__ == '__main__':
    main()
