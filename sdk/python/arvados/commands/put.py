#!/usr/bin/env python

# TODO:
# --md5sum - display md5 of each file as read from disk

import argparse
import arvados
import arvados.collection
import base64
import datetime
import errno
import fcntl
import hashlib
import json
import os
import pwd
import time
import signal
import socket
import sys
import tempfile
import threading
from apiclient import errors as apiclient_errors

import arvados.commands._util as arv_cmd

CAUGHT_SIGNALS = [signal.SIGINT, signal.SIGQUIT, signal.SIGTERM]
api_client = None

upload_opts = argparse.ArgumentParser(add_help=False)

upload_opts.add_argument('paths', metavar='path', type=str, nargs='*',
                         help="""
Local file or directory. Default: read from standard input.
""")

_group = upload_opts.add_mutually_exclusive_group()

_group.add_argument('--max-manifest-depth', type=int, metavar='N',
                    default=-1, help="""
Maximum depth of directory tree to represent in the manifest
structure. A directory structure deeper than this will be represented
as a single stream in the manifest. If N=0, the manifest will contain
a single stream. Default: -1 (unlimited), i.e., exactly one manifest
stream per filesystem directory that contains files.
""")

_group.add_argument('--normalize', action='store_true',
                    help="""
Normalize the manifest by re-ordering files and streams after writing
data.
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

run_opts = argparse.ArgumentParser(add_help=False)

run_opts.add_argument('--project-uuid', metavar='UUID', help="""
Store the collection in the specified project, instead of your Home
project.
""")

run_opts.add_argument('--name', help="""
Save the collection with the specified name.
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

_group = run_opts.add_mutually_exclusive_group()
_group.add_argument('--resume', action='store_true', default=True,
                    help="""
Continue interrupted uploads from cached state (default).
""")
_group.add_argument('--no-resume', action='store_false', dest='resume',
                    help="""
Do not continue interrupted uploads from cached state.
""")

arg_parser = argparse.ArgumentParser(
    description='Copy data from the local filesystem to Keep.',
    parents=[upload_opts, run_opts, arv_cmd.retry_opt])

def parse_arguments(arguments):
    args = arg_parser.parse_args(arguments)

    if len(args.paths) == 0:
        args.paths = ['-']

    args.paths = map(lambda x: "-" if x == "/dev/stdin" else x, args.paths)

    if len(args.paths) != 1 or os.path.isdir(args.paths[0]):
        if args.filename:
            arg_parser.error("""
    --filename argument cannot be used when storing a directory or
    multiple files.
    """)

    # Turn on --progress by default if stderr is a tty.
    if (not (args.batch_progress or args.no_progress)
        and os.isatty(sys.stderr.fileno())):
        args.progress = True

    if args.paths == ['-']:
        args.resume = False
        if not args.filename:
            args.filename = 'stdin'

    return args

class ResumeCacheConflict(Exception):
    pass


class FileUploadError(Exception):
    pass


class ResumeCache(object):
    CACHE_DIR = '.cache/arvados/arv-put'

    def __init__(self, file_spec):
        self.cache_file = open(file_spec, 'a+')
        self._lock_file(self.cache_file)
        self.filename = self.cache_file.name

    @classmethod
    def make_path(cls, args):
        md5 = hashlib.md5()
        md5.update(arvados.config.get('ARVADOS_API_HOST', '!nohost'))
        realpaths = sorted(os.path.realpath(path) for path in args.paths)
        md5.update('\0'.join(realpaths))
        if any(os.path.isdir(path) for path in realpaths):
            md5.update(str(max(args.max_manifest_depth, -1)))
        elif args.filename:
            md5.update(args.filename)
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
    def __init__(self, paths, resume=True, reporter=None, bytes_expected=None,
                name=None, owner_uuid=None, ensure_unique_name=False,
                num_retries=None, write_copies=None, replication=None,
                filename=None):
        self.paths = paths
        self.resume = resume
        self.reporter = reporter
        self.bytes_expected = bytes_expected
        self.bytes_written = 0
        self.name = name
        self.owner_uuid = owner_uuid
        self.ensure_unique_name = ensure_unique_name
        self.num_retries = num_retries
        self.write_copies = write_copies
        self.replication = replication
        self.filename = filename
        self._state_lock = threading.Lock()
        self._state = None # Previous run state (file list & manifest)
        self._current_files = [] # Current run file list
        self._cache_hash = None # MD5 digest based on paths & filename
        self._cache_file = None
        self._collection = None
        self._collection_lock = threading.Lock()
        self._stop_checkpointer = threading.Event()
        self._checkpointer = threading.Thread(target=self._update_task)
        self._update_task_time = 60.0 # How many seconds wait between update runs
        # Load cached data if any and if needed
        self._setup_state()

    def start(self):
        """
        Start supporting thread & file uploading
        """
        self._checkpointer.start()
        for path in self.paths:
            try:
                # Test for stdin first, in case some file named '-' exist
                if path == '-':
                    self._write_stdin(self.filename or 'stdin')
                elif os.path.isdir(path):
                    self._write_directory_tree(path)
                else: #if os.path.isfile(path):
                    self._write_file(path, self.filename or os.path.basename(path))
                # else:
                #     raise FileUploadError('Inadequate file type, cannot upload: %s' % path)
            except:
                # Stop the thread before continue complaining
                self._stop_checkpointer.set()
                self._checkpointer.join()
                raise
        # Work finished, stop updater task
        self._stop_checkpointer.set()
        self._checkpointer.join()
        # Successful upload, one last _update()
        self._update()

    def save_collection(self):
        with self._collection_lock:
            self._my_collection().save_new(
                                name=self.name, owner_uuid=self.owner_uuid, 
                                ensure_unique_name=self.ensure_unique_name,
                                num_retries=self.num_retries,
                                replication_desired=self.replication)
        if self.resume:
            # Delete cache file upon successful collection saving
            try:
                os.unlink(self._cache_file.name)
            except OSError as error:
                if error.errno != errno.ENOENT:  # That's what we wanted anyway.
                    raise
            self._cache_file.close()

    def _collection_size(self, collection):
        """
        Recursively get the total size of the collection
        """
        size = 0
        for item in collection:
            if isinstance(item, arvados.arvfile.ArvadosFile):
                size += item.size()
            elif isinstance(item, arvados.collection.Collection):
                size += self._collection_size(item)
        return size

    def _update_task(self):
        """
        Periodically call support tasks. File uploading is
        asynchronous so we poll status from the collection.
        """
        while not self._stop_checkpointer.wait(self._update_task_time):
            self._update()

    def _update(self):
        """
        Update cached manifest text and report progress.
        """
        with self._collection_lock:
            self.bytes_written = self._collection_size(self._my_collection())
            # Update cache, if resume enabled
            if self.resume:
                with self._state_lock:
                    self._state['manifest'] = self._my_collection().manifest_text()
        if self.resume:
            self._save_state()
        # Call the reporter, if any
        self.report_progress()

    def report_progress(self):
        if self.reporter is not None:
            self.reporter(self.bytes_written, self.bytes_expected)

    def _write_directory_tree(self, path, stream_name="."):
        # TODO: Check what happens when multiple directories are passes as
        # arguments.
        # If the code below is uncommented, integration test
        # test_ArvPutSignedManifest (tests.test_arv_put.ArvPutIntegrationTest)
        # fails, I suppose it is because the manifest_uuid changes because
        # of the dir addition to stream_name.

        # if stream_name == '.':
        #     stream_name = os.path.join('.', os.path.basename(path))
        for item in os.listdir(path):
            if os.path.isdir(os.path.join(path, item)):
                self._write_directory_tree(os.path.join(path, item), 
                                os.path.join(stream_name, item))
            elif os.path.isfile(os.path.join(path, item)):
                self._write_file(os.path.join(path, item), 
                                os.path.join(stream_name, item))
            else:
                raise FileUploadError('Inadequate file type, cannot upload: %s' % path)

    def _write_stdin(self, filename):
        with self._collection_lock:
            output = self._my_collection().open(filename, 'w')
        self._write(sys.stdin, output)
        output.close()

    def _write_file(self, source, filename):
        resume_offset = 0
        resume_upload = False
        if self.resume:
            # Check if file was already uploaded (at least partially)
            with self._collection_lock:
                try:
                    file_in_collection = self._my_collection().find(filename)
                except IOError:
                    # Not found
                    file_in_collection = None
            # If no previous cached data on this file, store it for an eventual
            # repeated run.
            if source not in self._state['files'].keys():
                with self._state_lock:
                    self._state['files'][source] = {
                        'mtime' : os.path.getmtime(source),
                        'size' : os.path.getsize(source)
                    }
            cached_file_data = self._state['files'][source]
            # See if this file was already uploaded at least partially
            if file_in_collection:
                if cached_file_data['mtime'] == os.path.getmtime(source) and cached_file_data['size'] == os.path.getsize(source):
                    if os.path.getsize(source) == file_in_collection.size():
                        # File already there, skip it.
                        return
                    elif os.path.getsize(source) > file_in_collection.size():
                        # File partially uploaded, resume!
                        resume_upload = True
                        resume_offset = file_in_collection.size()
                    else:
                        # Inconsistent cache, re-upload the file
                        pass
                else:
                    # Local file differs from cached data, re-upload it
                    pass
        with open(source, 'r') as source_fd:
            if self.resume and resume_upload:
                with self._collection_lock:
                    # Open for appending
                    output = self._my_collection().open(filename, 'a')
                source_fd.seek(resume_offset)
            else:
                with self._collection_lock:
                    output = self._my_collection().open(filename, 'w')
            self._write(source_fd, output)
            output.close()

    def _write(self, source_fd, output):
        while True:
            data = source_fd.read(arvados.config.KEEP_BLOCK_SIZE)
            if not data:
                break
            output.write(data)

    def _my_collection(self):
        """
        Create a new collection if none cached. Load it from cache otherwise.
        """
        if self._collection is None:
            with self._state_lock:
                manifest = self._state['manifest']
            if self.resume and manifest is not None:
                # Create collection from saved state
                self._collection = arvados.collection.Collection(
                                        manifest,
                                        num_write_copies=self.write_copies)
            else:
                # Create new collection
                self._collection = arvados.collection.Collection(
                                        num_write_copies=self.write_copies)
        return self._collection

    def _setup_state(self):
        """
        Create a new cache file or load a previously existing one.
        """
        if self.resume:
            md5 = hashlib.md5()
            md5.update(arvados.config.get('ARVADOS_API_HOST', '!nohost'))
            realpaths = sorted(os.path.realpath(path) for path in self.paths)
            md5.update('\0'.join(realpaths))
            self._cache_hash = md5.hexdigest()
            if self.filename:
                md5.update(self.filename)
            self._cache_file = open(os.path.join(
                arv_cmd.make_home_conf_dir('.cache/arvados/arv-put', 0o700, 'raise'),
                self._cache_hash), 'a+')
            self._cache_file.seek(0)
            with self._state_lock:
                try:
                    self._state = json.load(self._cache_file)
                    if not 'manifest' in self._state.keys():
                        self._state['manifest'] = ""
                    if not 'files' in self._state.keys():
                        self._state['files'] = {}
                except ValueError:
                    # File empty, set up new cache
                    self._state = {
                        'manifest' : None,
                        # Previous run file list: {path : {size, mtime}}
                        'files' : {}
                    }
            # Load how many bytes were uploaded on previous run
            with self._collection_lock:
                self.bytes_written = self._collection_size(self._my_collection())
        # No resume required
        else:
            with self._state_lock:
                self._state = {
                    'manifest' : None,
                    'files' : {} # Previous run file list: {path : {size, mtime}}
                }

    def _lock_file(self, fileobj):
        try:
            fcntl.flock(fileobj, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except IOError:
            raise ResumeCacheConflict("{} locked".format(fileobj.name))

    def _save_state(self):
        """
        Atomically save current state into cache.
        """
        try:
            with self._state_lock:
                state = self._state
            new_cache_fd, new_cache_name = tempfile.mkstemp(
                dir=os.path.dirname(self._cache_file.name))
            self._lock_file(new_cache_fd)
            new_cache = os.fdopen(new_cache_fd, 'r+')
            json.dump(state, new_cache)
            # new_cache.flush()
            # os.fsync(new_cache)
            os.rename(new_cache_name, self._cache_file.name)
        except (IOError, OSError, ResumeCacheConflict) as error:
            try:
                os.unlink(new_cache_name)
            except NameError:  # mkstemp failed.
                pass
        else:
            self._cache_file.close()
            self._cache_file = new_cache

    def collection_name(self):
        with self._collection_lock:
            name = self._my_collection().api_response()['name'] if self._my_collection().api_response() else None
        return name

    def manifest_locator(self):
        with self._collection_lock:
            locator = self._my_collection().manifest_locator()
        return locator
    
    def portable_data_hash(self):
        with self._collection_lock:
            datahash = self._my_collection().portable_data_hash()
        return datahash
    
    def manifest_text(self, stream_name=".", strip=False, normalize=False):
        with self._collection_lock:
            manifest = self._my_collection().manifest_text(stream_name, strip, normalize)
        return manifest

    def _datablocks_on_item(self, item):
        """
        Return a list of datablock locators, recursively navigating
        through subcollections
        """
        if isinstance(item, arvados.arvfile.ArvadosFile):
            locators = []
            for segment in item.segments():
                loc = segment.locator
                if loc.startswith("bufferblock"):
                    loc = item._bufferblocks[loc].calculate_locator()
                locators.append(loc)
            return locators
        elif isinstance(item, arvados.collection.Collection) or isinstance(item, arvados.collection.Subcollection):
            l = [self._datablocks_on_item(x) for x in item.values()]
            # Fast list flattener method taken from:
            # http://stackoverflow.com/questions/952914/making-a-flat-list-out-of-list-of-lists-in-python
            return [loc for sublist in l for loc in sublist]
        else:
            return None
    
    def data_locators(self):
        with self._collection_lock:
            datablocks = self._datablocks_on_item(self._my_collection())
        return datablocks


class ArvPutCollectionWriter(arvados.ResumableCollectionWriter):
    STATE_PROPS = (arvados.ResumableCollectionWriter.STATE_PROPS +
                   ['bytes_written', '_seen_inputs'])

    def __init__(self, cache=None, reporter=None, bytes_expected=None, **kwargs):
        self.bytes_written = 0
        self._seen_inputs = []
        self.cache = cache
        self.reporter = reporter
        self.bytes_expected = bytes_expected
        super(ArvPutCollectionWriter, self).__init__(**kwargs)

    @classmethod
    def from_cache(cls, cache, reporter=None, bytes_expected=None,
                   num_retries=0, replication=0):
        try:
            state = cache.load()
            state['_data_buffer'] = [base64.decodestring(state['_data_buffer'])]
            writer = cls.from_state(state, cache, reporter, bytes_expected,
                                    num_retries=num_retries,
                                    replication=replication)
        except (TypeError, ValueError,
                arvados.errors.StaleWriterStateError) as error:
            return cls(cache, reporter, bytes_expected,
                       num_retries=num_retries,
                       replication=replication)
        else:
            return writer

    def cache_state(self):
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

    def report_progress(self):
        if self.reporter is not None:
            self.reporter(self.bytes_written, self.bytes_expected)

    def flush_data(self):
        start_buffer_len = self._data_buffer_len
        start_block_count = self.bytes_written / arvados.config.KEEP_BLOCK_SIZE
        super(ArvPutCollectionWriter, self).flush_data()
        if self._data_buffer_len < start_buffer_len:  # We actually PUT data.
            self.bytes_written += (start_buffer_len - self._data_buffer_len)
            self.report_progress()
            if (self.bytes_written / arvados.config.KEEP_BLOCK_SIZE) > start_block_count:
                self.cache_state()

    def _record_new_input(self, input_type, source_name, dest_name):
        # The key needs to be a list because that's what we'll get back
        # from JSON deserialization.
        key = [input_type, source_name, dest_name]
        if key in self._seen_inputs:
            return False
        self._seen_inputs.append(key)
        return True

    def write_file(self, source, filename=None):
        if self._record_new_input('file', source, filename):
            super(ArvPutCollectionWriter, self).write_file(source, filename)

    def write_directory_tree(self,
                             path, stream_name='.', max_manifest_depth=-1):
        if self._record_new_input('directory', path, stream_name):
            super(ArvPutCollectionWriter, self).write_directory_tree(
                path, stream_name, max_manifest_depth)


def expected_bytes_for(pathlist):
    # Walk the given directory trees and stat files, adding up file sizes,
    # so we can display progress as percent
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
        return "\r{}M / {}M {:.1%} ".format(
            bytes_written >> 20, bytes_expected >> 20,
            float(bytes_written) / bytes_expected)
    else:
        return "\r{} ".format(bytes_written)

def progress_writer(progress_func, outfile=sys.stderr):
    def write_progress(bytes_written, bytes_expected):
        outfile.write(progress_func(bytes_written, bytes_expected))
    return write_progress

def exit_signal_handler(sigcode, frame):
    sys.exit(-sigcode)

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

def main(arguments=None, stdout=sys.stdout, stderr=sys.stderr):
    global api_client

    args = parse_arguments(arguments)
    status = 0
    if api_client is None:
        api_client = arvados.api('v1')

    # Determine the name to use
    if args.name:
        if args.stream or args.raw:
            print >>stderr, "Cannot use --name with --stream or --raw"
            sys.exit(1)
        collection_name = args.name
    else:
        collection_name = "Saved at {} by {}@{}".format(
            datetime.datetime.utcnow().strftime("%Y-%m-%d %H:%M:%S UTC"),
            pwd.getpwuid(os.getuid()).pw_name,
            socket.gethostname())

    if args.project_uuid and (args.stream or args.raw):
        print >>stderr, "Cannot use --project-uuid with --stream or --raw"
        sys.exit(1)

    # Determine the parent project
    try:
        project_uuid = desired_project_uuid(api_client, args.project_uuid,
                                            args.retries)
    except (apiclient_errors.Error, ValueError) as error:
        print >>stderr, error
        sys.exit(1)

    # write_copies diverges from args.replication here.
    # args.replication is how many copies we will instruct Arvados to
    # maintain (by passing it in collections().create()) after all
    # data is written -- and if None was given, we'll use None there.
    # Meanwhile, write_copies is how many copies of each data block we
    # write to Keep, which has to be a number.
    #
    # If we simply changed args.replication from None to a default
    # here, we'd end up erroneously passing the default replication
    # level (instead of None) to collections().create().
    write_copies = (args.replication or
                    api_client._rootDesc.get('defaultCollectionReplication', 2))

    if args.progress:
        reporter = progress_writer(human_progress)
    elif args.batch_progress:
        reporter = progress_writer(machine_progress)
    else:
        reporter = None

    bytes_expected = expected_bytes_for(args.paths)
    try:
        writer = ArvPutUploadJob(paths = args.paths,
                                resume = args.resume, 
                                reporter = reporter,
                                bytes_expected = bytes_expected,
                                num_retries = args.retries,
                                write_copies = write_copies,
                                replication = args.replication,
                                name = collection_name,
                                owner_uuid = project_uuid,
                                ensure_unique_name = True)
    except ResumeCacheConflict:
        print >>stderr, "\n".join([
            "arv-put: Another process is already uploading this data.",
            "         Use --no-resume if this is really what you want."])
        sys.exit(1)

    # Install our signal handler for each code in CAUGHT_SIGNALS, and save
    # the originals.
    orig_signal_handlers = {sigcode: signal.signal(sigcode, exit_signal_handler)
                            for sigcode in CAUGHT_SIGNALS}

    if args.resume and writer.bytes_written > 0:
        print >>stderr, "\n".join([
                "arv-put: Resuming previous upload from last checkpoint.",
                "         Use the --no-resume option to start over."])

    writer.report_progress()
    output = None
    writer.start()
    if args.progress:  # Print newline to split stderr from stdout for humans.
        print >>stderr

    if args.stream:
        if args.normalize:
            output = writer.manifest_text(normalize=True)
        else:
            output = writer.manifest_text()
    elif args.raw:
        output = ','.join(writer.data_locators())
    else:
        try:
            writer.save_collection()
            print >>stderr, "Collection saved as '%s'" % writer.collection_name()
            if args.portable_data_hash:
                output = writer.portable_data_hash()
            else:
                output = writer.manifest_locator()
        except apiclient_errors.Error as error:
            print >>stderr, (
                "arv-put: Error creating Collection on project: {}.".format(
                    error))
            status = 1

    # Print the locator (uuid) of the new collection.
    if output is None:
        status = status or 1
    else:
        stdout.write(output)
        if not output.endswith('\n'):
            stdout.write('\n')

    for sigcode, orig_handler in orig_signal_handlers.items():
        signal.signal(sigcode, orig_handler)

    if status != 0:
        sys.exit(status)

    return output


if __name__ == '__main__':
    main()
