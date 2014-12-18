import functools
import logging
import os
import re

from collections import deque
from stat import *

from .arvfile import ArvadosFileBase, split, ArvadosFile
from keep import *
from .stream import StreamReader, normalize_stream
import config
import errors
import util

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
                    streams[streamname][filename].extend(s.locators_and_ranges(r[0], r[1]))

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


class Collection(object):
    def __init__(self):
        self.items = {}

    def find_or_create(self, path):
        p = path.split("/")
        if p[0] == '.':
            del p[0]

        if len(p) > 0:
            item = self.items.get(p[0])
            if len(p) == 1:
                # item must be a file
                if item is None:
                    # create new file
                    item = ArvadosFile(p[0], 'wb', [], [])
                    self.items[p[0]] = item
                return item
            else:
                if item is None:
                    # create new collection
                    item = Collection()
                    self.items[p[0]] = item
                del p[0]
                return item.find_or_create("/".join(p))
        else:
            return self


def import_manifest(manifest_text):
    c = Collection()

    STREAM_NAME = 0
    BLOCKS = 1
    SEGMENTS = 2

    stream_name = None
    state = STREAM_NAME

    for n in re.finditer(r'([^ \n]+)([ \n])', manifest_text):
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
            s = re.match(r'[0-9a-f]{32}\+(\d)+(\+\S+)*', tok)
            if s:
                blocksize = long(s.group(1))
                blocks.append([tok, blocksize, streamoffset])
                streamoffset += blocksize
            else:
                state = SEGMENTS

        if state == SEGMENTS:
            s = re.search(r'^(\d+):(\d+):(\S+)', tok)
            if s:
                pos = long(s.group(1))
                size = long(s.group(2))
                name = s.group(3).replace('\\040', ' ')
                f = c.find_or_create("%s/%s" % (stream_name, name))
                f.add_segment(blocks, pos, size)
            else:
                # error!
                raise errors.SyntaxError("Invalid manifest format")

        if sep == "\n":
            stream_name = None
            state = STREAM_NAME

    return c

def export_manifest(item, stream_name="."):
    buf = ""
    print item
    if isinstance(item, Collection):
        for i, j in item.items.values():
            buf += export_manifest(j, stream_name)
    else:
        buf += stream_name
        buf += " "
        buf += item.segments
    return buf
