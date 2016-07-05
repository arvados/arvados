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


class ArvPutCollectionCache(object):
    def __init__(self, paths):
        md5 = hashlib.md5()
        md5.update(arvados.config.get('ARVADOS_API_HOST', '!nohost'))
        realpaths = sorted(os.path.realpath(path) for path in paths)
        self.files = {}
        for path in realpaths:
            self._get_file_data(path)
        # Only hash args paths
        md5.update('\0'.join(realpaths))
        self.cache_hash = md5.hexdigest()
        
        self.cache_file = open(os.path.join(
            arv_cmd.make_home_conf_dir('.cache/arvados/arv-put', 0o700, 'raise'), 
            self.cache_hash), 'a+')
        self._lock_file(self.cache_file)
        self.filename = self.cache_file.name
        self.data = self._load()
    
    def _load(self):
        try:
            self.cache_file.seek(0)
            ret = json.load(self.cache_file)
        except ValueError:
            # File empty, set up new cache
            ret = {
                'col_locator' : None, # Collection 
                'uploaded' : {}, # Uploaded file list: {path : {size, mtime}}
            }
        return ret
    
    def _save(self):
        """
        Atomically save (create temp file & rename() it)
        """
        # TODO: Should be a good idea to avoid _save() spamming? when writing 
        # lots of small files.
        print "SAVE START"
        try:
            new_cache_fd, new_cache_name = tempfile.mkstemp(
                dir=os.path.dirname(self.filename))
            self._lock_file(new_cache_fd)
            new_cache = os.fdopen(new_cache_fd, 'r+')
            json.dump(self.data, new_cache)
            os.rename(new_cache_name, self.filename)
        except (IOError, OSError, ResumeCacheConflict) as error:
            print "SAVE ERROR: %s" % error
            try:
                os.unlink(new_cache_name)
            except NameError:  # mkstemp failed.
                pass
        else:
            print "SAVE DONE!! %s" % self.filename
            self.cache_file.close()
            self.cache_file = new_cache
    
    def file_uploaded(self, path):
        if path in self.files.keys():
            self.data['uploaded'][path] = self.files[path]
            self._save()
    
    def set_collection(self, uuid):
        self.data['col_locator'] = uuid
    
    def collection(self):
        return self.data['col_locator']
    
    def is_dirty(self, path):
        if not path in self.data['uploaded'].keys():
            # Cannot be dirty is it wasn't even uploaded
            return False
            
        if (self.files[path]['mtime'] != self.data['uploaded'][path]['mtime']) or (self.files[path]['size'] != self.data['uploaded'][path]['size']):
            return True
        else:
            return False
    
    def dirty_files(self):
        """
        Files that were previously uploaded but changed locally between 
        upload runs. These files should be re-uploaded.
        """
        dirty = []
        for f in self.data['uploaded'].keys():
            if self.is_dirty(f):
                dirty.append(f)
        return dirty
    
    def uploaded_files(self):
        """
        Files that were uploaded and have not changed locally between 
        upload runs. These files should be checked for partial uploads
        """
        uploaded = []
        for f in self.data['uploaded'].keys():
            if not self.is_dirty(f):
                uploaded.append(f)
        return uploaded
    
    def pending_files(self):
        """
        Files that should be uploaded, because of being dirty or that
        never had the chance to be uploaded yet.
        """
        pending = []
        uploaded = self.uploaded_files()
        for f in self.files.keys():
            if f not in uploaded:
                pending.append(f)
        return pending
    
    def _get_file_data(self, path):
        if os.path.isfile(path):
            self.files[path] = {'mtime': os.path.getmtime(path),
                                'size': os.path.getsize(path)}
        elif os.path.isdir(path):
            for item in os.listdir(path):
                self._get_file_data(os.path.join(path, item))

    def _lock_file(self, fileobj):
        try:
            fcntl.flock(fileobj, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except IOError:
            raise ResumeCacheConflict("{} locked".format(fileobj.name))

    def close(self):
        self.cache_file.close()

    def destroy(self):
        # try:
        #     os.unlink(self.filename)
        # except OSError as error:
        #     if error.errno != errno.ENOENT:  # That's what we wanted anyway.
        #         raise
        self.close()

class ArvPutUploader(object):
    def __init__(self, paths):
        self.cache = ArvPutCollectionCache(paths)
        if self.cache.collection() is not None:
            self.collection = ArvPutCollection(locator=self.cache.collection(), cache=self.cache)
        else:
            self.collection = ArvPutCollection(cache=self.cache)
            self.cache.set_collection(self.collection.manifest_locator())
        for p in paths:
            if os.path.isdir(p):
                self.collection.write_directory_tree(p)
            elif os.path.isfile(p):
                self.collection.write_file(p)
        self.cache.destroy()
    
    def manifest(self):
        return self.collection.manifest()
    
    def bytes_written(self):
        return self.collection.bytes_written


class ArvPutCollection(object):
    def __init__(self, locator=None, cache=None, reporter=None, 
                    bytes_expected=None, **kwargs):
        self.collection_flush_time = 60
        self.bytes_written = 0
        self._seen_inputs = []
        self.cache = cache
        self.reporter = reporter
        self.bytes_expected = bytes_expected
        
        if locator is None:
            self.collection = arvados.collection.Collection()
            self.collection.save_new()
        else:
            self.collection = arvados.collection.Collection(locator)
    
    def manifest_locator(self):
        return self.collection.manifest_locator()
            
    def write_file(self, source, filename):
        if self.cache and source in self.cache.dirty_files():
            print "DIRTY: Removing file %s from collection to be uploaded again" % source
            self.collection.remove(filename)
        
        resume_offset = 0
        resume_upload = False

        print "FIND file %s" % filename
        if self.collection.find(filename):
            print "File %s already in the collection, checking!" % source
            if os.path.getsize(source) == self.collection.find(filename).size():
                print "WARNING: file %s already uploaded, skipping!" % source
                # File already there, skip it.
                return
            elif os.path.getsize(source) > self.collection.find(filename).size():
                print "WARNING: RESUMING file %s" % source
                # File partially uploaded, resume!
                resume_upload = True
                resume_offset = self.collection.find(filename).size()
            else:
                # Source file smaller than uploaded file, what happened here?
                # TODO: Raise exception of some kind?
                pass

        with open(source, 'r') as source_fd:
            with self.collection as c:
                if resume_upload:
                    print "Resuming file, source: %s, filename: %s" % (source, filename)
                    output = c.open(filename, 'a')
                    source_fd.seek(resume_offset)
                    first_block = False
                else:
                    print "Writing file, source: %s, filename: %s" % (source, filename)
                    output = c.open(filename, 'w')
                    first_block = True
                    
                start_time = time.time()
                while True:
                    data = source_fd.read(arvados.config.KEEP_BLOCK_SIZE)
                    if not data:
                        break
                    output.write(data)
                    output.flush() # Commit block to Keep
                    self.bytes_written += len(data)
                    # Is it time to update the collection?
                    if (time.time() - start_time) > self.collection_flush_time:
                        self.collection.save()
                        start_time = time.time()
                    # Once a block is written on each file, mark it as uploaded on the cache
                    if first_block:
                        if self.cache:
                            self.cache.file_uploaded(source)
                        first_block = False
                # File write finished
                output.close()
                self.collection.save() # One last save...

    def write_directory_tree(self, path, stream_name='.', max_manifest_depth=-1):
        if os.path.isdir(path):
            for item in os.listdir(path):
                print "Checking path: '%s' - stream_name: '%s'" % (path, stream_name)
                if os.path.isdir(os.path.join(path, item)):
                    self.write_directory_tree(os.path.join(path, item), 
                                    os.path.join(stream_name, item))
                else:
                    self.write_file(os.path.join(path, item), 
                                    os.path.join(stream_name, item))

    def manifest(self):
        print "BLOCK SIZE: %d" % arvados.config.KEEP_BLOCK_SIZE
        print "MANIFEST Locator:\n%s\nMANIFEST TEXT:\n%s" % (self.manifest_locator(), self.collection.manifest_text())
        return True
    
    def report_progress(self):
        if self.reporter is not None:
            self.reporter(self.bytes_written, self.bytes_expected)


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

    resume_cache = None
    if args.resume:
        try:
            resume_cache = ResumeCache(ResumeCache.make_path(args))
            resume_cache.check_cache(api_client=api_client, num_retries=args.retries)
        except (IOError, OSError, ValueError):
            pass  # Couldn't open cache directory/file.  Continue without it.
        except ResumeCacheConflict:
            print >>stderr, "\n".join([
                "arv-put: Another process is already uploading this data.",
                "         Use --no-resume if this is really what you want."])
            sys.exit(1)

    if resume_cache is None:
        writer = ArvPutCollectionWriter(
            resume_cache, reporter, bytes_expected,
            num_retries=args.retries,
            replication=write_copies)
    else:
        writer = ArvPutCollectionWriter.from_cache(
            resume_cache, reporter, bytes_expected,
            num_retries=args.retries,
            replication=write_copies)

    # Install our signal handler for each code in CAUGHT_SIGNALS, and save
    # the originals.
    orig_signal_handlers = {sigcode: signal.signal(sigcode, exit_signal_handler)
                            for sigcode in CAUGHT_SIGNALS}

    if writer.bytes_written > 0:  # We're resuming a previous upload.
        print >>stderr, "\n".join([
                "arv-put: Resuming previous upload from last checkpoint.",
                "         Use the --no-resume option to start over."])

    writer.report_progress()
    writer.do_queued_work()  # Do work resumed from cache.
    for path in args.paths:  # Copy file data to Keep.
        if path == '-':
            writer.start_new_stream()
            writer.start_new_file(args.filename)
            r = sys.stdin.read(64*1024)
            while r:
                # Need to bypass _queued_file check in ResumableCollectionWriter.write() to get
                # CollectionWriter.write().
                super(arvados.collection.ResumableCollectionWriter, writer).write(r)
                r = sys.stdin.read(64*1024)
        elif os.path.isdir(path):
            writer.write_directory_tree(
                path, max_manifest_depth=args.max_manifest_depth)
        else:
            writer.start_new_stream()
            writer.write_file(path, args.filename or os.path.basename(path))
    writer.finish_current_stream()

    if args.progress:  # Print newline to split stderr from stdout for humans.
        print >>stderr

    output = None
    if args.stream:
        output = writer.manifest_text()
        if args.normalize:
            output = arvados.collection.CollectionReader(output).manifest_text(normalize=True)
    elif args.raw:
        output = ','.join(writer.data_locators())
    else:
        try:
            manifest_text = writer.manifest_text()
            if args.normalize:
                manifest_text = arvados.collection.CollectionReader(manifest_text).manifest_text(normalize=True)
            replication_attr = 'replication_desired'
            if api_client._schema.schemas['Collection']['properties'].get(replication_attr, None) is None:
                # API called it 'redundancy' before #3410.
                replication_attr = 'redundancy'
            # Register the resulting collection in Arvados.
            collection = api_client.collections().create(
                body={
                    'owner_uuid': project_uuid,
                    'name': collection_name,
                    'manifest_text': manifest_text,
                    replication_attr: args.replication,
                    },
                ensure_unique_name=True
                ).execute(num_retries=args.retries)

            print >>stderr, "Collection saved as '%s'" % collection['name']

            if args.portable_data_hash and 'portable_data_hash' in collection and collection['portable_data_hash']:
                output = collection['portable_data_hash']
            else:
                output = collection['uuid']

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

    if resume_cache is not None:
        resume_cache.destroy()

    return output

if __name__ == '__main__':
    main()
