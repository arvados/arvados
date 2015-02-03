import functools
import logging
import os
import re
import errno
import time

from collections import deque
from stat import *

from .arvfile import ArvadosFileBase, split, ArvadosFile, ArvadosFileWriter, ArvadosFileReader, BlockManager, _synchronized, _must_be_writable, SYNC_READONLY, SYNC_EXPLICIT, SYNC_LIVE, NoopLock
from keep import *
from .stream import StreamReader, normalize_stream, locator_block_size
from .ranges import Range, LocatorAndRange
import config
import errors
import util
import events

_logger = logging.getLogger('arvados.collection')

class CollectionBase(object):
    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        pass

    def _my_keep(self):
        if self._keep_client is None:
            self._keep_client = KeepClient(api_client=self._api_client,
                                           num_retries=self.num_retries)
        return self._keep_client

    def stripped_manifest(self):
        """
        Return the manifest for the current collection with all
        non-portable hints (i.e., permission signatures and other
        hints other than size hints) removed from the locators.
        """
        raw = self.manifest_text()
        clean = []
        for line in raw.split("\n"):
            fields = line.split()
            if fields:
                clean_fields = fields[:1] + [
                    (re.sub(r'\+[^\d][^\+]*', '', x)
                     if re.match(util.keep_locator_pattern, x)
                     else x)
                    for x in fields[1:]]
                clean += [' '.join(clean_fields), "\n"]
        return ''.join(clean)


class CollectionReader(CollectionBase):
    def __init__(self, manifest_locator_or_text, api_client=None,
                 keep_client=None, num_retries=0):
        """Instantiate a CollectionReader.

        This class parses Collection manifests to provide a simple interface
        to read its underlying files.

        Arguments:
        * manifest_locator_or_text: One of a Collection UUID, portable data
          hash, or full manifest text.
        * api_client: The API client to use to look up Collections.  If not
          provided, CollectionReader will build one from available Arvados
          configuration.
        * keep_client: The KeepClient to use to download Collection data.
          If not provided, CollectionReader will build one from available
          Arvados configuration.
        * num_retries: The default number of times to retry failed
          service requests.  Default 0.  You may change this value
          after instantiation, but note those changes may not
          propagate to related objects like the Keep client.
        """
        self._api_client = api_client
        self._keep_client = keep_client
        self.num_retries = num_retries
        if re.match(util.keep_locator_pattern, manifest_locator_or_text):
            self._manifest_locator = manifest_locator_or_text
            self._manifest_text = None
        elif re.match(util.collection_uuid_pattern, manifest_locator_or_text):
            self._manifest_locator = manifest_locator_or_text
            self._manifest_text = None
        elif re.match(util.manifest_pattern, manifest_locator_or_text):
            self._manifest_text = manifest_locator_or_text
            self._manifest_locator = None
        else:
            raise errors.ArgumentError(
                "Argument to CollectionReader must be a manifest or a collection UUID")
        self._api_response = None
        self._streams = None

    def _populate_from_api_server(self):
        # As in KeepClient itself, we must wait until the last
        # possible moment to instantiate an API client, in order to
        # avoid tripping up clients that don't have access to an API
        # server.  If we do build one, make sure our Keep client uses
        # it.  If instantiation fails, we'll fall back to the except
        # clause, just like any other Collection lookup
        # failure. Return an exception, or None if successful.
        try:
            if self._api_client is None:
                self._api_client = arvados.api('v1')
                self._keep_client = None  # Make a new one with the new api.
            self._api_response = self._api_client.collections().get(
                uuid=self._manifest_locator).execute(
                num_retries=self.num_retries)
            self._manifest_text = self._api_response['manifest_text']
            return None
        except Exception as e:
            return e

    def _populate_from_keep(self):
        # Retrieve a manifest directly from Keep. This has a chance of
        # working if [a] the locator includes a permission signature
        # or [b] the Keep services are operating in world-readable
        # mode. Return an exception, or None if successful.
        try:
            self._manifest_text = self._my_keep().get(
                self._manifest_locator, num_retries=self.num_retries)
        except Exception as e:
            return e

    def _populate(self):
        error_via_api = None
        error_via_keep = None
        should_try_keep = ((self._manifest_text is None) and
                           util.keep_locator_pattern.match(
                self._manifest_locator))
        if ((self._manifest_text is None) and
            util.signed_locator_pattern.match(self._manifest_locator)):
            error_via_keep = self._populate_from_keep()
        if self._manifest_text is None:
            error_via_api = self._populate_from_api_server()
            if error_via_api is not None and not should_try_keep:
                raise error_via_api
        if ((self._manifest_text is None) and
            not error_via_keep and
            should_try_keep):
            # Looks like a keep locator, and we didn't already try keep above
            error_via_keep = self._populate_from_keep()
        if self._manifest_text is None:
            # Nothing worked!
            raise arvados.errors.NotFoundError(
                ("Failed to retrieve collection '{}' " +
                 "from either API server ({}) or Keep ({})."
                 ).format(
                    self._manifest_locator,
                    error_via_api,
                    error_via_keep))
        self._streams = [sline.split()
                         for sline in self._manifest_text.split("\n")
                         if sline]

    def _populate_first(orig_func):
        # Decorator for methods that read actual Collection data.
        @functools.wraps(orig_func)
        def wrapper(self, *args, **kwargs):
            if self._streams is None:
                self._populate()
            return orig_func(self, *args, **kwargs)
        return wrapper

    @_populate_first
    def api_response(self):
        """api_response() -> dict or None

        Returns information about this Collection fetched from the API server.
        If the Collection exists in Keep but not the API server, currently
        returns None.  Future versions may provide a synthetic response.
        """
        return self._api_response

    @_populate_first
    def normalize(self):
        # Rearrange streams
        streams = {}
        for s in self.all_streams():
            for f in s.all_files():
                streamname, filename = split(s.name() + "/" + f.name())
                if streamname not in streams:
                    streams[streamname] = {}
                if filename not in streams[streamname]:
                    streams[streamname][filename] = []
                for r in f.segments:
                    streams[streamname][filename].extend(s.locators_and_ranges(r.locator, r.range_size))

        self._streams = [normalize_stream(s, streams[s])
                         for s in sorted(streams)]

        # Regenerate the manifest text based on the normalized streams
        self._manifest_text = ''.join(
            [StreamReader(stream, keep=self._my_keep()).manifest_text()
             for stream in self._streams])

    @_populate_first
    def open(self, streampath, filename=None):
        """open(streampath[, filename]) -> file-like object

        Pass in the path of a file to read from the Collection, either as a
        single string or as two separate stream name and file name arguments.
        This method returns a file-like object to read that file.
        """
        if filename is None:
            streampath, filename = split(streampath)
        keep_client = self._my_keep()
        for stream_s in self._streams:
            stream = StreamReader(stream_s, keep_client,
                                  num_retries=self.num_retries)
            if stream.name() == streampath:
                break
        else:
            raise ValueError("stream '{}' not found in Collection".
                             format(streampath))
        try:
            return stream.files()[filename]
        except KeyError:
            raise ValueError("file '{}' not found in Collection stream '{}'".
                             format(filename, streampath))

    @_populate_first
    def all_streams(self):
        return [StreamReader(s, self._my_keep(), num_retries=self.num_retries)
                for s in self._streams]

    def all_files(self):
        for s in self.all_streams():
            for f in s.all_files():
                yield f

    @_populate_first
    def manifest_text(self, strip=False, normalize=False):
        if normalize:
            cr = CollectionReader(self.manifest_text())
            cr.normalize()
            return cr.manifest_text(strip=strip, normalize=False)
        elif strip:
            return self.stripped_manifest()
        else:
            return self._manifest_text


class _WriterFile(ArvadosFileBase):
    def __init__(self, coll_writer, name):
        super(_WriterFile, self).__init__(name, 'wb')
        self.dest = coll_writer

    def close(self):
        super(_WriterFile, self).close()
        self.dest.finish_current_file()

    @ArvadosFileBase._before_close
    def write(self, data):
        self.dest.write(data)

    @ArvadosFileBase._before_close
    def writelines(self, seq):
        for data in seq:
            self.write(data)

    @ArvadosFileBase._before_close
    def flush(self):
        self.dest.flush_data()


class CollectionWriter(CollectionBase):
    def __init__(self, api_client=None, num_retries=0):
        """Instantiate a CollectionWriter.

        CollectionWriter lets you build a new Arvados Collection from scratch.
        Write files to it.  The CollectionWriter will upload data to Keep as
        appropriate, and provide you with the Collection manifest text when
        you're finished.

        Arguments:
        * api_client: The API client to use to look up Collections.  If not
          provided, CollectionReader will build one from available Arvados
          configuration.
        * num_retries: The default number of times to retry failed
          service requests.  Default 0.  You may change this value
          after instantiation, but note those changes may not
          propagate to related objects like the Keep client.
        """
        self._api_client = api_client
        self.num_retries = num_retries
        self._keep_client = None
        self._data_buffer = []
        self._data_buffer_len = 0
        self._current_stream_files = []
        self._current_stream_length = 0
        self._current_stream_locators = []
        self._current_stream_name = '.'
        self._current_file_name = None
        self._current_file_pos = 0
        self._finished_streams = []
        self._close_file = None
        self._queued_file = None
        self._queued_dirents = deque()
        self._queued_trees = deque()
        self._last_open = None

    def __exit__(self, exc_type, exc_value, traceback):
        if exc_type is None:
            self.finish()

    def do_queued_work(self):
        # The work queue consists of three pieces:
        # * _queued_file: The file object we're currently writing to the
        #   Collection.
        # * _queued_dirents: Entries under the current directory
        #   (_queued_trees[0]) that we want to write or recurse through.
        #   This may contain files from subdirectories if
        #   max_manifest_depth == 0 for this directory.
        # * _queued_trees: Directories that should be written as separate
        #   streams to the Collection.
        # This function handles the smallest piece of work currently queued
        # (current file, then current directory, then next directory) until
        # no work remains.  The _work_THING methods each do a unit of work on
        # THING.  _queue_THING methods add a THING to the work queue.
        while True:
            if self._queued_file:
                self._work_file()
            elif self._queued_dirents:
                self._work_dirents()
            elif self._queued_trees:
                self._work_trees()
            else:
                break

    def _work_file(self):
        while True:
            buf = self._queued_file.read(config.KEEP_BLOCK_SIZE)
            if not buf:
                break
            self.write(buf)
        self.finish_current_file()
        if self._close_file:
            self._queued_file.close()
        self._close_file = None
        self._queued_file = None

    def _work_dirents(self):
        path, stream_name, max_manifest_depth = self._queued_trees[0]
        if stream_name != self.current_stream_name():
            self.start_new_stream(stream_name)
        while self._queued_dirents:
            dirent = self._queued_dirents.popleft()
            target = os.path.join(path, dirent)
            if os.path.isdir(target):
                self._queue_tree(target,
                                 os.path.join(stream_name, dirent),
                                 max_manifest_depth - 1)
            else:
                self._queue_file(target, dirent)
                break
        if not self._queued_dirents:
            self._queued_trees.popleft()

    def _work_trees(self):
        path, stream_name, max_manifest_depth = self._queued_trees[0]
        d = util.listdir_recursive(
            path, max_depth = (None if max_manifest_depth == 0 else 0))
        if d:
            self._queue_dirents(stream_name, d)
        else:
            self._queued_trees.popleft()

    def _queue_file(self, source, filename=None):
        assert (self._queued_file is None), "tried to queue more than one file"
        if not hasattr(source, 'read'):
            source = open(source, 'rb')
            self._close_file = True
        else:
            self._close_file = False
        if filename is None:
            filename = os.path.basename(source.name)
        self.start_new_file(filename)
        self._queued_file = source

    def _queue_dirents(self, stream_name, dirents):
        assert (not self._queued_dirents), "tried to queue more than one tree"
        self._queued_dirents = deque(sorted(dirents))

    def _queue_tree(self, path, stream_name, max_manifest_depth):
        self._queued_trees.append((path, stream_name, max_manifest_depth))

    def write_file(self, source, filename=None):
        self._queue_file(source, filename)
        self.do_queued_work()

    def write_directory_tree(self,
                             path, stream_name='.', max_manifest_depth=-1):
        self._queue_tree(path, stream_name, max_manifest_depth)
        self.do_queued_work()

    def write(self, newdata):
        if hasattr(newdata, '__iter__'):
            for s in newdata:
                self.write(s)
            return
        self._data_buffer.append(newdata)
        self._data_buffer_len += len(newdata)
        self._current_stream_length += len(newdata)
        while self._data_buffer_len >= config.KEEP_BLOCK_SIZE:
            self.flush_data()

    def open(self, streampath, filename=None):
        """open(streampath[, filename]) -> file-like object

        Pass in the path of a file to write to the Collection, either as a
        single string or as two separate stream name and file name arguments.
        This method returns a file-like object you can write to add it to the
        Collection.

        You may only have one file object from the Collection open at a time,
        so be sure to close the object when you're done.  Using the object in
        a with statement makes that easy::

          with cwriter.open('./doc/page1.txt') as outfile:
              outfile.write(page1_data)
          with cwriter.open('./doc/page2.txt') as outfile:
              outfile.write(page2_data)
        """
        if filename is None:
            streampath, filename = split(streampath)
        if self._last_open and not self._last_open.closed:
            raise errors.AssertionError(
                "can't open '{}' when '{}' is still open".format(
                    filename, self._last_open.name))
        if streampath != self.current_stream_name():
            self.start_new_stream(streampath)
        self.set_current_file_name(filename)
        self._last_open = _WriterFile(self, filename)
        return self._last_open

    def flush_data(self):
        data_buffer = ''.join(self._data_buffer)
        if data_buffer:
            self._current_stream_locators.append(
                self._my_keep().put(data_buffer[0:config.KEEP_BLOCK_SIZE]))
            self._data_buffer = [data_buffer[config.KEEP_BLOCK_SIZE:]]
            self._data_buffer_len = len(self._data_buffer[0])

    def start_new_file(self, newfilename=None):
        self.finish_current_file()
        self.set_current_file_name(newfilename)

    def set_current_file_name(self, newfilename):
        if re.search(r'[\t\n]', newfilename):
            raise errors.AssertionError(
                "Manifest filenames cannot contain whitespace: %s" %
                newfilename)
        elif re.search(r'\x00', newfilename):
            raise errors.AssertionError(
                "Manifest filenames cannot contain NUL characters: %s" %
                newfilename)
        self._current_file_name = newfilename

    def current_file_name(self):
        return self._current_file_name

    def finish_current_file(self):
        if self._current_file_name is None:
            if self._current_file_pos == self._current_stream_length:
                return
            raise errors.AssertionError(
                "Cannot finish an unnamed file " +
                "(%d bytes at offset %d in '%s' stream)" %
                (self._current_stream_length - self._current_file_pos,
                 self._current_file_pos,
                 self._current_stream_name))
        self._current_stream_files.append([
                self._current_file_pos,
                self._current_stream_length - self._current_file_pos,
                self._current_file_name])
        self._current_file_pos = self._current_stream_length
        self._current_file_name = None

    def start_new_stream(self, newstreamname='.'):
        self.finish_current_stream()
        self.set_current_stream_name(newstreamname)

    def set_current_stream_name(self, newstreamname):
        if re.search(r'[\t\n]', newstreamname):
            raise errors.AssertionError(
                "Manifest stream names cannot contain whitespace")
        self._current_stream_name = '.' if newstreamname=='' else newstreamname

    def current_stream_name(self):
        return self._current_stream_name

    def finish_current_stream(self):
        self.finish_current_file()
        self.flush_data()
        if not self._current_stream_files:
            pass
        elif self._current_stream_name is None:
            raise errors.AssertionError(
                "Cannot finish an unnamed stream (%d bytes in %d files)" %
                (self._current_stream_length, len(self._current_stream_files)))
        else:
            if not self._current_stream_locators:
                self._current_stream_locators.append(config.EMPTY_BLOCK_LOCATOR)
            self._finished_streams.append([self._current_stream_name,
                                           self._current_stream_locators,
                                           self._current_stream_files])
        self._current_stream_files = []
        self._current_stream_length = 0
        self._current_stream_locators = []
        self._current_stream_name = None
        self._current_file_pos = 0
        self._current_file_name = None

    def finish(self):
        # Store the manifest in Keep and return its locator.
        return self._my_keep().put(self.manifest_text())

    def portable_data_hash(self):
        stripped = self.stripped_manifest()
        return hashlib.md5(stripped).hexdigest() + '+' + str(len(stripped))

    def manifest_text(self):
        self.finish_current_stream()
        manifest = ''

        for stream in self._finished_streams:
            if not re.search(r'^\.(/.*)?$', stream[0]):
                manifest += './'
            manifest += stream[0].replace(' ', '\\040')
            manifest += ' ' + ' '.join(stream[1])
            manifest += ' ' + ' '.join("%d:%d:%s" % (sfile[0], sfile[1], sfile[2].replace(' ', '\\040')) for sfile in stream[2])
            manifest += "\n"

        return manifest

    def data_locators(self):
        ret = []
        for name, locators, files in self._finished_streams:
            ret += locators
        return ret


class ResumableCollectionWriter(CollectionWriter):
    STATE_PROPS = ['_current_stream_files', '_current_stream_length',
                   '_current_stream_locators', '_current_stream_name',
                   '_current_file_name', '_current_file_pos', '_close_file',
                   '_data_buffer', '_dependencies', '_finished_streams',
                   '_queued_dirents', '_queued_trees']

    def __init__(self, api_client=None, num_retries=0):
        self._dependencies = {}
        super(ResumableCollectionWriter, self).__init__(
            api_client, num_retries=num_retries)

    @classmethod
    def from_state(cls, state, *init_args, **init_kwargs):
        # Try to build a new writer from scratch with the given state.
        # If the state is not suitable to resume (because files have changed,
        # been deleted, aren't predictable, etc.), raise a
        # StaleWriterStateError.  Otherwise, return the initialized writer.
        # The caller is responsible for calling writer.do_queued_work()
        # appropriately after it's returned.
        writer = cls(*init_args, **init_kwargs)
        for attr_name in cls.STATE_PROPS:
            attr_value = state[attr_name]
            attr_class = getattr(writer, attr_name).__class__
            # Coerce the value into the same type as the initial value, if
            # needed.
            if attr_class not in (type(None), attr_value.__class__):
                attr_value = attr_class(attr_value)
            setattr(writer, attr_name, attr_value)
        # Check dependencies before we try to resume anything.
        if any(KeepLocator(ls).permission_expired()
               for ls in writer._current_stream_locators):
            raise errors.StaleWriterStateError(
                "locators include expired permission hint")
        writer.check_dependencies()
        if state['_current_file'] is not None:
            path, pos = state['_current_file']
            try:
                writer._queued_file = open(path, 'rb')
                writer._queued_file.seek(pos)
            except IOError as error:
                raise errors.StaleWriterStateError(
                    "failed to reopen active file {}: {}".format(path, error))
        return writer

    def check_dependencies(self):
        for path, orig_stat in self._dependencies.items():
            if not S_ISREG(orig_stat[ST_MODE]):
                raise errors.StaleWriterStateError("{} not file".format(path))
            try:
                now_stat = tuple(os.stat(path))
            except OSError as error:
                raise errors.StaleWriterStateError(
                    "failed to stat {}: {}".format(path, error))
            if ((not S_ISREG(now_stat[ST_MODE])) or
                (orig_stat[ST_MTIME] != now_stat[ST_MTIME]) or
                (orig_stat[ST_SIZE] != now_stat[ST_SIZE])):
                raise errors.StaleWriterStateError("{} changed".format(path))

    def dump_state(self, copy_func=lambda x: x):
        state = {attr: copy_func(getattr(self, attr))
                 for attr in self.STATE_PROPS}
        if self._queued_file is None:
            state['_current_file'] = None
        else:
            state['_current_file'] = (os.path.realpath(self._queued_file.name),
                                      self._queued_file.tell())
        return state

    def _queue_file(self, source, filename=None):
        try:
            src_path = os.path.realpath(source)
        except Exception:
            raise errors.AssertionError("{} not a file path".format(source))
        try:
            path_stat = os.stat(src_path)
        except OSError as stat_error:
            path_stat = None
        super(ResumableCollectionWriter, self)._queue_file(source, filename)
        fd_stat = os.fstat(self._queued_file.fileno())
        if not S_ISREG(fd_stat.st_mode):
            # We won't be able to resume from this cache anyway, so don't
            # worry about further checks.
            self._dependencies[source] = tuple(fd_stat)
        elif path_stat is None:
            raise errors.AssertionError(
                "could not stat {}: {}".format(source, stat_error))
        elif path_stat.st_ino != fd_stat.st_ino:
            raise errors.AssertionError(
                "{} changed between open and stat calls".format(source))
        else:
            self._dependencies[src_path] = tuple(fd_stat)

    def write(self, data):
        if self._queued_file is None:
            raise errors.AssertionError(
                "resumable writer can't accept unsourced data")
        return super(ResumableCollectionWriter, self).write(data)

ADD = "add"
DEL = "del"

class SynchronizedCollectionBase(CollectionBase):
    def __init__(self, parent=None):
        self.parent = parent
        self._items = {}

    def _my_api(self):
        raise NotImplementedError()

    def _my_keep(self):
        raise NotImplementedError()

    def _my_block_manager(self):
        raise NotImplementedError()

    def _root_lock(self):
        raise NotImplementedError()

    def _populate(self):
        raise NotImplementedError()

    def sync_mode(self):
        raise NotImplementedError()

    def notify(self, collection, event, name, item):
        raise NotImplementedError()

    @_synchronized
    def find(self, path, create=False, create_collection=False):
        """Recursively search the specified file path.  May return either a Collection
        or ArvadosFile.

        :create:
          If true, create path components (i.e. Collections) that are
          missing.  If "create" is False, return None if a path component is
          not found.

        :create_collection:
          If the path is not found, "create" is True, and
          "create_collection" is False, then create and return a new
          ArvadosFile for the last path component.  If "create_collection" is
          True, then create and return a new Collection for the last path
          component.

        """
        if create and self.sync_mode() == SYNC_READONLY:
            raise IOError((errno.EROFS, "Collection is read only"))

        p = path.split("/")
        if p[0] == '.':
            del p[0]

        if len(p) > 0:
            item = self._items.get(p[0])
            if len(p) == 1:
                # item must be a file
                if item is None and create:
                    # create new file
                    if create_collection:
                        item = Subcollection(self)
                    else:
                        item = ArvadosFile(self)
                    self._items[p[0]] = item
                    self.notify(self, ADD, p[0], item)
                return item
            else:
                if item is None and create:
                    # create new collection
                    item = Subcollection(self)
                    self._items[p[0]] = item
                    self.notify(self, ADD, p[0], item)
                del p[0]
                return item.find("/".join(p), create=create)
        else:
            return self

    def open(self, path, mode):
        """Open a file-like object for access.

        :path:
          path to a file in the collection
        :mode:
          one of "r", "r+", "w", "w+", "a", "a+"
          :"r":
            opens for reading
          :"r+":
            opens for reading and writing.  Reads/writes share a file pointer.
          :"w", "w+":
            truncates to 0 and opens for reading and writing.  Reads/writes share a file pointer.
          :"a", "a+":
            opens for reading and writing.  All writes are appended to
            the end of the file.  Writing does not affect the file pointer for
            reading.
        """
        mode = mode.replace("b", "")
        if len(mode) == 0 or mode[0] not in ("r", "w", "a"):
            raise ArgumentError("Bad mode '%s'" % mode)
        create = (mode != "r")

        if create and self.sync_mode() == SYNC_READONLY:
            raise IOError((errno.EROFS, "Collection is read only"))

        f = self.find(path, create=create)

        if f is None:
            raise IOError((errno.ENOENT, "File not found"))
        if not isinstance(f, ArvadosFile):
            raise IOError((errno.EISDIR, "Path must refer to a file."))

        if mode[0] == "w":
            f.truncate(0)

        if mode == "r":
            return ArvadosFileReader(f, path, mode, num_retries=self.num_retries)
        else:
            return ArvadosFileWriter(f, path, mode, num_retries=self.num_retries)

    @_synchronized
    def modified(self):
        """Test if the collection (or any subcollection or file) has been modified
        since it was created."""
        for k,v in self._items.items():
            if v.modified():
                return True
        return False

    @_synchronized
    def set_unmodified(self):
        """Recursively clear modified flag"""
        for k,v in self._items.items():
            v.set_unmodified()

    @_synchronized
    def __iter__(self):
        """Iterate over names of files and collections contained in this collection."""
        return self._items.keys()

    @_synchronized
    def iterkeys(self):
        """Iterate over names of files and collections directly contained in this collection."""
        return self._items.keys()

    @_synchronized
    def __getitem__(self, k):
        """Get a file or collection that is directly contained by this collection.  If
        you want to search a path, use `find()` instead.
        """
        return self._items[k]

    @_synchronized
    def __contains__(self, k):
        """If there is a file or collection a directly contained by this collection
        with name "k"."""
        return k in self._items

    @_synchronized
    def __len__(self):
        """Get the number of items directly contained in this collection"""
        return len(self._items)

    @_must_be_writable
    @_synchronized
    def __delitem__(self, p):
        """Delete an item by name which is directly contained by this collection."""
        del self._items[p]
        self.notify(self, DEL, p, None)

    @_synchronized
    def keys(self):
        """Get a list of names of files and collections directly contained in this collection."""
        return self._items.keys()

    @_synchronized
    def values(self):
        """Get a list of files and collection objects directly contained in this collection."""
        return self._items.values()

    @_synchronized
    def items(self):
        """Get a list of (name, object) tuples directly contained in this collection."""
        return self._items.items()

    def exists(self, path):
        """Test if there is a file or collection at "path" """
        return self.find(path) != None

    @_must_be_writable
    @_synchronized
    def remove(self, path, rm_r=False):
        """Remove the file or subcollection (directory) at `path`.
        :rm_r:
          Specify whether to remove non-empty subcollections (True), or raise an error (False).
        """
        p = path.split("/")
        if p[0] == '.':
            # Remove '.' from the front of the path
            del p[0]

        if len(p) > 0:
            item = self._items.get(p[0])
            if item is None:
                raise IOError((errno.ENOENT, "File not found"))
            if len(p) == 1:
                if isinstance(self._items[p[0]], SynchronizedCollectionBase) and len(self._items[p[0]]) > 0 and not rm_r:
                    raise IOError((errno.ENOTEMPTY, "Subcollection not empty"))
                del self._items[p[0]]
                self.notify(self, DEL, p[0], None)
            else:
                del p[0]
                item.remove("/".join(p))
        else:
            raise IOError((errno.ENOENT, "File not found"))

    def _cloneinto(self, target):
        for k,v in self._items:
            target._items[k] = v.clone(new_parent=target)

    def clone(self):
        raise NotImplementedError()

    @_must_be_writable
    @_synchronized
    def copyto(self, target_path, source_path, source_collection=None, overwrite=False):
        """
        copyto('/foo', '/bar') will overwrite 'foo' if it exists.
        copyto('/foo/', '/bar') will place 'bar' in subcollection 'foo'
        """
        if source_collection is None:
            source_collection = self

        # Find the object to copy
        sp = source_path.split("/")
        source_obj = source_collection.find(source_path)
        if source_obj is None:
            raise IOError((errno.ENOENT, "File not found"))

        # Find parent collection the target path
        tp = target_path.split("/")
        target_dir = self.find(tp[0:-1].join("/"), create=True, create_collection=True)

        # Determine the name to use.
        target_name = tp[-1] if tp[-1] else sp[-1]

        if target_name in target_dir and not overwrite:
            raise IOError((errno.EEXIST, "File already exists"))

        # Actually make the copy.
        dup = source_obj.clone(target_dir)
        with target_dir.lock:
            target_dir._items[target_name] = dup

        self.notify(target_dir, ADD, target_name, dup)


    @_synchronized
    def manifest_text(self, strip=False, normalize=False):
        """Get the manifest text for this collection, sub collections and files.

        :strip:
          If True, remove signing tokens from block locators if present.
          If False, block locators are left unchanged.

        :normalize:
          If True, always export the manifest text in normalized form
          even if the Collection is not modified.  If False and the collection
          is not modified, return the original manifest text even if it is not
          in normalized form.

        """
        if self.modified() or self._manifest_text is None or normalize:
            return export_manifest(self, stream_name=".", portable_locators=strip)
        else:
            if strip:
                return self.stripped_manifest()
            else:
                return self._manifest_text

    @_must_be_writable
    @_synchronized
    def merge(self, other):
        for k in other.keys():
            if k in self:
                if isinstance(self[k], Subcollection) and isinstance(other[k], Subcollection):
                    self[k].merge(other[k])
                else:
                    if self[k] != other[k]:
                        name = "%s~conflict-%s~" % (k, time.strftime("%Y-%m-%d~%H:%M%:%S",
                                                                     time.gmtime()))
                        self[name] = other[k].clone(self)
                        self.notify(self, name, ADD, self[name])
            else:
                self[k] = other[k].clone(self)
                self.notify(self, k, ADD, self[k])

    def portable_data_hash(self):
        """Get the portable data hash for this collection's manifest."""
        stripped = self.manifest_text(strip=True)
        return hashlib.md5(stripped).hexdigest() + '+' + str(len(stripped))


class Collection(SynchronizedCollectionBase):
    """Store an Arvados collection, consisting of a set of files and
    sub-collections.
    """

    def __init__(self, manifest_locator_or_text=None,
                 parent=None,
                 config=None,
                 api_client=None,
                 keep_client=None,
                 num_retries=None,
                 block_manager=None,
                 sync=SYNC_READONLY):
        """:manifest_locator_or_text:
          One of Arvados collection UUID, block locator of
          a manifest, raw manifest text, or None (to create an empty collection).
        :parent:
          the parent Collection, may be None.
        :config:
          the arvados configuration to get the hostname and api token.
          Prefer this over supplying your own api_client and keep_client (except in testing).
          Will use default config settings if not specified.
        :api_client:
          The API client object to use for requests.  If not specified, create one using `config`.
        :keep_client:
          the Keep client to use for requests.  If not specified, create one using `config`.
        :num_retries:
          the number of retries for API and Keep requests.
        :block_manager:
          the block manager to use.  If not specified, create one.
        :sync:
          Set synchronization policy with API server collection record.
          :SYNC_READONLY:
            Collection is read only.  No synchronization.  This mode will
            also forego locking, which gives better performance.
          :SYNC_EXPLICIT:
            Synchronize on explicit request via `update()` or `save()`
          :SYNC_LIVE:
            Synchronize with server in response to background websocket events,
            on block write, or on file close.

        """
        super(Collection, self).__init__(parent)
        self._api_client = api_client
        self._keep_client = keep_client
        self._block_manager = block_manager
        self._config = config
        self.num_retries = num_retries
        self._manifest_locator = None
        self._manifest_text = None
        self._api_response = None
        self._sync = sync
        self.lock = threading.RLock()
        self.callbacks = []
        self.events = None

        if manifest_locator_or_text:
            if re.match(util.keep_locator_pattern, manifest_locator_or_text):
                self._manifest_locator = manifest_locator_or_text
            elif re.match(util.collection_uuid_pattern, manifest_locator_or_text):
                self._manifest_locator = manifest_locator_or_text
            elif re.match(util.manifest_pattern, manifest_locator_or_text):
                self._manifest_text = manifest_locator_or_text
            else:
                raise errors.ArgumentError(
                    "Argument to CollectionReader must be a manifest or a collection UUID")

            self._populate()

            if self._sync == SYNC_LIVE:
                if not self._manifest_locator or not re.match(util.collection_uuid_pattern, self._manifest_locator):
                    raise errors.ArgumentError("Cannot SYNC_LIVE unless a collection uuid is specified")
                self.events = events.subscribe(arvados.api(), [["object_uuid", "=", self._manifest_locator]], self.on_message)

    @staticmethod
    def create(name, owner_uuid=None, sync=SYNC_EXPLICIT):
        c = Collection(sync=sync)
        c.save_as(name, owner_uuid=owner_uuid, ensure_unique_name=True)
        return c

    def _root_lock(self):
        return self.lock

    def sync_mode(self):
        return self._sync

    def on_message(self):
        self.update()

    @_synchronized
    def update(self):
        n = self._my_api().collections().get(uuid=self._manifest_locator, select=["manifest_text"]).execute()
        other = import_collection(n["manifest_text"])
        self.merge(other)

    @_synchronized
    def _my_api(self):
        if self._api_client is None:
            self._api_client = arvados.api.SafeApi(self._config)
            self._keep_client = self._api_client.keep
        return self._api_client

    @_synchronized
    def _my_keep(self):
        if self._keep_client is None:
            if self._api_client is None:
                self._my_api()
            else:
                self._keep_client = KeepClient(api=self._api_client)
        return self._keep_client

    @_synchronized
    def _my_block_manager(self):
        if self._block_manager is None:
            self._block_manager = BlockManager(self._my_keep())
        return self._block_manager

    def _populate_from_api_server(self):
        # As in KeepClient itself, we must wait until the last
        # possible moment to instantiate an API client, in order to
        # avoid tripping up clients that don't have access to an API
        # server.  If we do build one, make sure our Keep client uses
        # it.  If instantiation fails, we'll fall back to the except
        # clause, just like any other Collection lookup
        # failure. Return an exception, or None if successful.
        try:
            self._api_response = self._my_api().collections().get(
                uuid=self._manifest_locator).execute(
                    num_retries=self.num_retries)
            self._manifest_text = self._api_response['manifest_text']
            return None
        except Exception as e:
            return e

    def _populate_from_keep(self):
        # Retrieve a manifest directly from Keep. This has a chance of
        # working if [a] the locator includes a permission signature
        # or [b] the Keep services are operating in world-readable
        # mode. Return an exception, or None if successful.
        try:
            self._manifest_text = self._my_keep().get(
                self._manifest_locator, num_retries=self.num_retries)
        except Exception as e:
            return e

    def _populate(self):
        if self._manifest_locator is None and self._manifest_text is None:
            return
        error_via_api = None
        error_via_keep = None
        should_try_keep = ((self._manifest_text is None) and
                           util.keep_locator_pattern.match(
                               self._manifest_locator))
        if ((self._manifest_text is None) and
            util.signed_locator_pattern.match(self._manifest_locator)):
            error_via_keep = self._populate_from_keep()
        if self._manifest_text is None:
            error_via_api = self._populate_from_api_server()
            if error_via_api is not None and not should_try_keep:
                raise error_via_api
        if ((self._manifest_text is None) and
            not error_via_keep and
            should_try_keep):
            # Looks like a keep locator, and we didn't already try keep above
            error_via_keep = self._populate_from_keep()
        if self._manifest_text is None:
            # Nothing worked!
            raise arvados.errors.NotFoundError(
                ("Failed to retrieve collection '{}' " +
                 "from either API server ({}) or Keep ({})."
                 ).format(
                    self._manifest_locator,
                    error_via_api,
                    error_via_keep))
        # populate
        import_manifest(self._manifest_text, self)

        if self._sync == SYNC_READONLY:
            # Now that we're populated, knowing that this will be readonly,
            # forego any further locking.
            self.lock = NoopLock()

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        """Support scoped auto-commit in a with: block"""
        if self._sync != SYNC_READONLY:
            self.save(allow_no_locator=True)
        if self._block_manager is not None:
            self._block_manager.stop_threads()

    @_synchronized
    def clone(self, new_parent=None, new_sync=SYNC_READONLY, new_config=None):
        if new_config is None:
            new_config = self.config
        c = Collection(parent=new_parent, config=new_config, sync=new_sync)
        if new_sync == SYNC_READONLY:
            c.lock = NoopLock()
        c._items = {}
        self._cloneinto(c)
        return c

    @_synchronized
    def api_response(self):
        """
        api_response() -> dict or None

        Returns information about this Collection fetched from the API server.
        If the Collection exists in Keep but not the API server, currently
        returns None.  Future versions may provide a synthetic response.
        """
        return self._api_response

    @_must_be_writable
    @_synchronized
    def save(self, allow_no_locator=False):
        """Commit pending buffer blocks to Keep, write the manifest to Keep, and
        update the collection record to Keep.

        :allow_no_locator:
          If there is no collection uuid associated with this
          Collection and `allow_no_locator` is False, raise an error.  If True,
          do not raise an error.
        """
        if self.modified():
            self._my_block_manager().commit_all()
            self._my_keep().put(self.manifest_text(strip=True))
            if self._manifest_locator is not None and re.match(util.collection_uuid_pattern, self._manifest_locator):
                self._api_response = self._my_api().collections().update(
                    uuid=self._manifest_locator,
                    body={'manifest_text': self.manifest_text(strip=False)}
                    ).execute(
                        num_retries=self.num_retries)
            elif not allow_no_locator:
                raise AssertionError("Collection manifest_locator must be a collection uuid.  Use save_as() for new collections.")
            self.set_unmodified()

    @_must_be_writable
    @_synchronized
    def save_as(self, name, owner_uuid=None, ensure_unique_name=False):
        """Save a new collection record.

        :name:
          The collection name.

        :owner_uuid:
          the user, or project uuid that will own this collection.
          If None, defaults to the current user.

        :ensure_unique_name:
          If True, ask the API server to rename the collection
          if it conflicts with a collection with the same name and owner.  If
          False, a name conflict will result in an error.

        """
        self._my_block_manager().commit_all()
        self._my_keep().put(self.manifest_text(strip=True))
        body = {"manifest_text": self.manifest_text(strip=False),
                "name": name}
        if owner_uuid:
            body["owner_uuid"] = owner_uuid
        self._api_response = self._my_api().collections().create(ensure_unique_name=ensure_unique_name, body=body).execute(num_retries=self.num_retries)

        if self.events:
            self.events.unsubscribe(filters=[["object_uuid", "=", self._manifest_locator]])

        self._manifest_locator = self._api_response["uuid"]

        if self.events:
            self.events.subscribe(filters=[["object_uuid", "=", self._manifest_locator]])

        self.set_unmodified()

    @_synchronized
    def subscribe(self, callback):
        self.callbacks.append(callback)

    @_synchronized
    def unsubscribe(self, callback):
        self.callbacks.remove(callback)

    @_synchronized
    def notify(self, collection, event, name, item):
        for c in self.callbacks:
            c(collection, event, name, item)

class Subcollection(SynchronizedCollectionBase):
    """This is a subdirectory within a collection that doesn't have its own API
    server record.  It falls under the umbrella of the root collection."""

    def __init__(self, parent):
        super(Subcollection, self).__init__(parent)
        self.lock = parent._root_lock()

    def _root_lock(self):
        return self.parent._root_lock()

    def sync_mode(self):
        return self.parent.sync_mode()

    def _my_api(self):
        return self.parent._my_api()

    def _my_keep(self):
        return self.parent._my_keep()

    def _my_block_manager(self):
        return self.parent._my_block_manager()

    def _populate(self):
        self.parent._populate()

    def notify(self, collection, event, name, item):
        self.parent.notify(collection, event, name, item)

    @_synchronized
    def clone(self, new_parent):
        c = Subcollection(new_parent)
        c._items = {}
        self._cloneinto(c)
        return c

def import_manifest(manifest_text,
                    into_collection=None,
                    api_client=None,
                    keep=None,
                    num_retries=None,
                    sync=SYNC_READONLY):
    """Import a manifest into a `Collection`.

    :manifest_text:
      The manifest text to import from.

    :into_collection:
      The `Collection` that will be initialized (must be empty).
      If None, create a new `Collection` object.

    :api_client:
      The API client object that will be used when creating a new `Collection` object.

    :keep:
      The keep client object that will be used when creating a new `Collection` object.

    :num_retries:
      the default number of api client and keep retries on error.

    :sync:
      Collection sync mode (only if into_collection is None)
    """
    if into_collection is not None:
        if len(into_collection) > 0:
            raise ArgumentError("Can only import manifest into an empty collection")
        c = into_collection
    else:
        c = Collection(api_client=api_client, keep_client=keep, num_retries=num_retries, sync=sync)

    save_sync = c.sync_mode()
    c._sync = None

    STREAM_NAME = 0
    BLOCKS = 1
    SEGMENTS = 2

    stream_name = None
    state = STREAM_NAME

    for n in re.finditer(r'(\S+)(\s+|$)', manifest_text):
        tok = n.group(1)
        sep = n.group(2)

        if state == STREAM_NAME:
            # starting a new stream
            stream_name = tok.replace('\\040', ' ')
            blocks = []
            segments = []
            streamoffset = 0L
            state = BLOCKS
            continue

        if state == BLOCKS:
            s = re.match(r'[0-9a-f]{32}\+(\d+)(\+\S+)*', tok)
            if s:
                blocksize = long(s.group(1))
                blocks.append(Range(tok, streamoffset, blocksize))
                streamoffset += blocksize
            else:
                state = SEGMENTS

        if state == SEGMENTS:
            s = re.search(r'^(\d+):(\d+):(\S+)', tok)
            if s:
                pos = long(s.group(1))
                size = long(s.group(2))
                name = s.group(3).replace('\\040', ' ')
                f = c.find("%s/%s" % (stream_name, name), create=True)
                f.add_segment(blocks, pos, size)
            else:
                # error!
                raise errors.SyntaxError("Invalid manifest format")

        if sep == "\n":
            stream_name = None
            state = STREAM_NAME

    c.set_unmodified()
    c._sync = save_sync
    return c

def export_manifest(item, stream_name=".", portable_locators=False):
    """
    :item:
      Create a manifest for `item` (must be a `Collection` or `ArvadosFile`).  If
      `item` is a is a `Collection`, this will also export subcollections.

    :stream_name:
      the name of the stream when exporting `item`.

    :portable_locators:
      If True, strip any permission hints on block locators.
      If False, use block locators as-is.
    """
    buf = ""
    if isinstance(item, SynchronizedCollectionBase):
        stream = {}
        sorted_keys = sorted(item.keys())
        for k in [s for s in sorted_keys if isinstance(item[s], ArvadosFile)]:
            v = item[k]
            st = []
            for s in v.segments():
                loc = s.locator
                if loc.startswith("bufferblock"):
                    loc = v.parent._my_block_manager()._bufferblocks[loc].locator()
                if portable_locators:
                    loc = KeepLocator(loc).stripped()
                st.append(LocatorAndRange(loc, locator_block_size(loc),
                                     s.segment_offset, s.range_size))
            stream[k] = st
        if stream:
            buf += ' '.join(normalize_stream(stream_name, stream))
            buf += "\n"
        for k in [s for s in sorted_keys if isinstance(item[s], SynchronizedCollectionBase)]:
            buf += export_manifest(item[k], stream_name=os.path.join(stream_name, k), portable_locators=portable_locators)
    elif isinstance(item, ArvadosFile):
        st = []
        for s in item.segments:
            loc = s.locator
            if loc.startswith("bufferblock"):
                loc = item._bufferblocks[loc].calculate_locator()
            if portable_locators:
                loc = KeepLocator(loc).stripped()
            st.append(LocatorAndRange(loc, locator_block_size(loc),
                                 s.segment_offset, s.range_size))
        stream[stream_name] = st
        buf += ' '.join(normalize_stream(stream_name, stream))
        buf += "\n"
    return buf
