# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import absolute_import
from future.utils import listitems, listvalues, viewkeys
from builtins import str
from past.builtins import basestring
from builtins import object
import ciso8601
import datetime
import errno
import functools
import hashlib
import io
import logging
import os
import re
import sys
import threading
import time

from collections import deque
from stat import *

from .arvfile import split, _FileLikeObjectBase, ArvadosFile, ArvadosFileWriter, ArvadosFileReader, WrappableFile, _BlockManager, synchronized, must_be_writable, NoopLock
from .keep import KeepLocator, KeepClient
from .stream import StreamReader
from ._normalize_stream import normalize_stream, escape
from ._ranges import Range, LocatorAndRange
from .safeapi import ThreadSafeApiCache
import arvados.config as config
import arvados.errors as errors
import arvados.util
import arvados.events as events
from arvados.retry import retry_method

_logger = logging.getLogger('arvados.collection')


if sys.version_info >= (3, 0):
    TextIOWrapper = io.TextIOWrapper
else:
    class TextIOWrapper(io.TextIOWrapper):
        """To maintain backward compatibility, cast str to unicode in
        write('foo').

        """
        def write(self, data):
            if isinstance(data, basestring):
                data = unicode(data)
            return super(TextIOWrapper, self).write(data)


class CollectionBase(object):
    """Abstract base class for Collection classes."""

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
        """Get the manifest with locator hints stripped.

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
                     if re.match(arvados.util.keep_locator_pattern, x)
                     else x)
                    for x in fields[1:]]
                clean += [' '.join(clean_fields), "\n"]
        return ''.join(clean)


class _WriterFile(_FileLikeObjectBase):
    def __init__(self, coll_writer, name):
        super(_WriterFile, self).__init__(name, 'wb')
        self.dest = coll_writer

    def close(self):
        super(_WriterFile, self).close()
        self.dest.finish_current_file()

    @_FileLikeObjectBase._before_close
    def write(self, data):
        self.dest.write(data)

    @_FileLikeObjectBase._before_close
    def writelines(self, seq):
        for data in seq:
            self.write(data)

    @_FileLikeObjectBase._before_close
    def flush(self):
        self.dest.flush_data()


class CollectionWriter(CollectionBase):
    """Deprecated, use Collection instead."""

    def __init__(self, api_client=None, num_retries=0, replication=None):
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
        * replication: The number of copies of each block to store.
          If this argument is None or not supplied, replication is
          the server-provided default if available, otherwise 2.
        """
        self._api_client = api_client
        self.num_retries = num_retries
        self.replication = (2 if replication is None else replication)
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
        d = arvados.util.listdir_recursive(
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
        if isinstance(newdata, bytes):
            pass
        elif isinstance(newdata, str):
            newdata = newdata.encode()
        elif hasattr(newdata, '__iter__'):
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
                u"can't open '{}' when '{}' is still open".format(
                    filename, self._last_open.name))
        if streampath != self.current_stream_name():
            self.start_new_stream(streampath)
        self.set_current_file_name(filename)
        self._last_open = _WriterFile(self, filename)
        return self._last_open

    def flush_data(self):
        data_buffer = b''.join(self._data_buffer)
        if data_buffer:
            self._current_stream_locators.append(
                self._my_keep().put(
                    data_buffer[0:config.KEEP_BLOCK_SIZE],
                    copies=self.replication))
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
                "Manifest stream names cannot contain whitespace: '%s'" %
                (newstreamname))
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
        """Store the manifest in Keep and return its locator.

        This is useful for storing manifest fragments (task outputs)
        temporarily in Keep during a Crunch job.

        In other cases you should make a collection instead, by
        sending manifest_text() to the API server's "create
        collection" endpoint.
        """
        return self._my_keep().put(self.manifest_text().encode(),
                                   copies=self.replication)

    def portable_data_hash(self):
        stripped = self.stripped_manifest().encode()
        return '{}+{}'.format(hashlib.md5(stripped).hexdigest(), len(stripped))

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

    def save_new(self, name=None):
        return self._api_client.collections().create(
            ensure_unique_name=True,
            body={
                'name': name,
                'manifest_text': self.manifest_text(),
            }).execute(num_retries=self.num_retries)


class ResumableCollectionWriter(CollectionWriter):
    """Deprecated, use Collection instead."""

    STATE_PROPS = ['_current_stream_files', '_current_stream_length',
                   '_current_stream_locators', '_current_stream_name',
                   '_current_file_name', '_current_file_pos', '_close_file',
                   '_data_buffer', '_dependencies', '_finished_streams',
                   '_queued_dirents', '_queued_trees']

    def __init__(self, api_client=None, **kwargs):
        self._dependencies = {}
        super(ResumableCollectionWriter, self).__init__(api_client, **kwargs)

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
                    u"failed to reopen active file {}: {}".format(path, error))
        return writer

    def check_dependencies(self):
        for path, orig_stat in listitems(self._dependencies):
            if not S_ISREG(orig_stat[ST_MODE]):
                raise errors.StaleWriterStateError(u"{} not file".format(path))
            try:
                now_stat = tuple(os.stat(path))
            except OSError as error:
                raise errors.StaleWriterStateError(
                    u"failed to stat {}: {}".format(path, error))
            if ((not S_ISREG(now_stat[ST_MODE])) or
                (orig_stat[ST_MTIME] != now_stat[ST_MTIME]) or
                (orig_stat[ST_SIZE] != now_stat[ST_SIZE])):
                raise errors.StaleWriterStateError(u"{} changed".format(path))

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
            raise errors.AssertionError(u"{} not a file path".format(source))
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
                u"could not stat {}: {}".format(source, stat_error))
        elif path_stat.st_ino != fd_stat.st_ino:
            raise errors.AssertionError(
                u"{} changed between open and stat calls".format(source))
        else:
            self._dependencies[src_path] = tuple(fd_stat)

    def write(self, data):
        if self._queued_file is None:
            raise errors.AssertionError(
                "resumable writer can't accept unsourced data")
        return super(ResumableCollectionWriter, self).write(data)


ADD = "add"
DEL = "del"
MOD = "mod"
TOK = "tok"
FILE = "file"
COLLECTION = "collection"

class RichCollectionBase(CollectionBase):
    """Base class for Collections and Subcollections.

    Implements the majority of functionality relating to accessing items in the
    Collection.

    """

    def __init__(self, parent=None):
        self.parent = parent
        self._committed = False
        self._has_remote_blocks = False
        self._callback = None
        self._items = {}

    def _my_api(self):
        raise NotImplementedError()

    def _my_keep(self):
        raise NotImplementedError()

    def _my_block_manager(self):
        raise NotImplementedError()

    def writable(self):
        raise NotImplementedError()

    def root_collection(self):
        raise NotImplementedError()

    def notify(self, event, collection, name, item):
        raise NotImplementedError()

    def stream_name(self):
        raise NotImplementedError()


    @synchronized
    def has_remote_blocks(self):
        """Recursively check for a +R segment locator signature."""

        if self._has_remote_blocks:
            return True
        for item in self:
            if self[item].has_remote_blocks():
                return True
        return False

    @synchronized
    def set_has_remote_blocks(self, val):
        self._has_remote_blocks = val
        if self.parent:
            self.parent.set_has_remote_blocks(val)

    @must_be_writable
    @synchronized
    def find_or_create(self, path, create_type):
        """Recursively search the specified file path.

        May return either a `Collection` or `ArvadosFile`.  If not found, will
        create a new item at the specified path based on `create_type`.  Will
        create intermediate subcollections needed to contain the final item in
        the path.

        :create_type:
          One of `arvados.collection.FILE` or
          `arvados.collection.COLLECTION`.  If the path is not found, and value
          of create_type is FILE then create and return a new ArvadosFile for
          the last path component.  If COLLECTION, then create and return a new
          Collection for the last path component.

        """

        pathcomponents = path.split("/", 1)
        if pathcomponents[0]:
            item = self._items.get(pathcomponents[0])
            if len(pathcomponents) == 1:
                if item is None:
                    # create new file
                    if create_type == COLLECTION:
                        item = Subcollection(self, pathcomponents[0])
                    else:
                        item = ArvadosFile(self, pathcomponents[0])
                    self._items[pathcomponents[0]] = item
                    self.set_committed(False)
                    self.notify(ADD, self, pathcomponents[0], item)
                return item
            else:
                if item is None:
                    # create new collection
                    item = Subcollection(self, pathcomponents[0])
                    self._items[pathcomponents[0]] = item
                    self.set_committed(False)
                    self.notify(ADD, self, pathcomponents[0], item)
                if isinstance(item, RichCollectionBase):
                    return item.find_or_create(pathcomponents[1], create_type)
                else:
                    raise IOError(errno.ENOTDIR, "Not a directory", pathcomponents[0])
        else:
            return self

    @synchronized
    def find(self, path):
        """Recursively search the specified file path.

        May return either a Collection or ArvadosFile. Return None if not
        found.
        If path is invalid (ex: starts with '/'), an IOError exception will be
        raised.

        """
        if not path:
            raise errors.ArgumentError("Parameter 'path' is empty.")

        pathcomponents = path.split("/", 1)
        if pathcomponents[0] == '':
            raise IOError(errno.ENOTDIR, "Not a directory", pathcomponents[0])

        item = self._items.get(pathcomponents[0])
        if item is None:
            return None
        elif len(pathcomponents) == 1:
            return item
        else:
            if isinstance(item, RichCollectionBase):
                if pathcomponents[1]:
                    return item.find(pathcomponents[1])
                else:
                    return item
            else:
                raise IOError(errno.ENOTDIR, "Not a directory", pathcomponents[0])

    @synchronized
    def mkdirs(self, path):
        """Recursive subcollection create.

        Like `os.makedirs()`.  Will create intermediate subcollections needed
        to contain the leaf subcollection path.

        """

        if self.find(path) != None:
            raise IOError(errno.EEXIST, "Directory or file exists", path)

        return self.find_or_create(path, COLLECTION)

    def open(self, path, mode="r", encoding=None):
        """Open a file-like object for access.

        :path:
          path to a file in the collection
        :mode:
          a string consisting of "r", "w", or "a", optionally followed
          by "b" or "t", optionally followed by "+".
          :"b":
            binary mode: write() accepts bytes, read() returns bytes.
          :"t":
            text mode (default): write() accepts strings, read() returns strings.
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

        if not re.search(r'^[rwa][bt]?\+?$', mode):
            raise errors.ArgumentError("Invalid mode {!r}".format(mode))

        if mode[0] == 'r' and '+' not in mode:
            fclass = ArvadosFileReader
            arvfile = self.find(path)
        elif not self.writable():
            raise IOError(errno.EROFS, "Collection is read only")
        else:
            fclass = ArvadosFileWriter
            arvfile = self.find_or_create(path, FILE)

        if arvfile is None:
            raise IOError(errno.ENOENT, "File not found", path)
        if not isinstance(arvfile, ArvadosFile):
            raise IOError(errno.EISDIR, "Is a directory", path)

        if mode[0] == 'w':
            arvfile.truncate(0)

        binmode = mode[0] + 'b' + re.sub('[bt]', '', mode[1:])
        f = fclass(arvfile, mode=binmode, num_retries=self.num_retries)
        if 'b' not in mode:
            bufferclass = io.BufferedRandom if f.writable() else io.BufferedReader
            f = TextIOWrapper(bufferclass(WrappableFile(f)), encoding=encoding)
        return f

    def modified(self):
        """Determine if the collection has been modified since last commited."""
        return not self.committed()

    @synchronized
    def committed(self):
        """Determine if the collection has been committed to the API server."""
        return self._committed

    @synchronized
    def set_committed(self, value=True):
        """Recursively set committed flag.

        If value is True, set committed to be True for this and all children.

        If value is False, set committed to be False for this and all parents.
        """
        if value == self._committed:
            return
        if value:
            for k,v in listitems(self._items):
                v.set_committed(True)
            self._committed = True
        else:
            self._committed = False
            if self.parent is not None:
                self.parent.set_committed(False)

    @synchronized
    def __iter__(self):
        """Iterate over names of files and collections contained in this collection."""
        return iter(viewkeys(self._items))

    @synchronized
    def __getitem__(self, k):
        """Get a file or collection that is directly contained by this collection.

        If you want to search a path, use `find()` instead.

        """
        return self._items[k]

    @synchronized
    def __contains__(self, k):
        """Test if there is a file or collection a directly contained by this collection."""
        return k in self._items

    @synchronized
    def __len__(self):
        """Get the number of items directly contained in this collection."""
        return len(self._items)

    @must_be_writable
    @synchronized
    def __delitem__(self, p):
        """Delete an item by name which is directly contained by this collection."""
        del self._items[p]
        self.set_committed(False)
        self.notify(DEL, self, p, None)

    @synchronized
    def keys(self):
        """Get a list of names of files and collections directly contained in this collection."""
        return self._items.keys()

    @synchronized
    def values(self):
        """Get a list of files and collection objects directly contained in this collection."""
        return listvalues(self._items)

    @synchronized
    def items(self):
        """Get a list of (name, object) tuples directly contained in this collection."""
        return listitems(self._items)

    def exists(self, path):
        """Test if there is a file or collection at `path`."""
        return self.find(path) is not None

    @must_be_writable
    @synchronized
    def remove(self, path, recursive=False):
        """Remove the file or subcollection (directory) at `path`.

        :recursive:
          Specify whether to remove non-empty subcollections (True), or raise an error (False).
        """

        if not path:
            raise errors.ArgumentError("Parameter 'path' is empty.")

        pathcomponents = path.split("/", 1)
        item = self._items.get(pathcomponents[0])
        if item is None:
            raise IOError(errno.ENOENT, "File not found", path)
        if len(pathcomponents) == 1:
            if isinstance(self._items[pathcomponents[0]], RichCollectionBase) and len(self._items[pathcomponents[0]]) > 0 and not recursive:
                raise IOError(errno.ENOTEMPTY, "Directory not empty", path)
            deleteditem = self._items[pathcomponents[0]]
            del self._items[pathcomponents[0]]
            self.set_committed(False)
            self.notify(DEL, self, pathcomponents[0], deleteditem)
        else:
            item.remove(pathcomponents[1], recursive=recursive)

    def _clonefrom(self, source):
        for k,v in listitems(source):
            self._items[k] = v.clone(self, k)

    def clone(self):
        raise NotImplementedError()

    @must_be_writable
    @synchronized
    def add(self, source_obj, target_name, overwrite=False, reparent=False):
        """Copy or move a file or subcollection to this collection.

        :source_obj:
          An ArvadosFile, or Subcollection object

        :target_name:
          Destination item name.  If the target name already exists and is a
          file, this will raise an error unless you specify `overwrite=True`.

        :overwrite:
          Whether to overwrite target file if it already exists.

        :reparent:
          If True, source_obj will be moved from its parent collection to this collection.
          If False, source_obj will be copied and the parent collection will be
          unmodified.

        """

        if target_name in self and not overwrite:
            raise IOError(errno.EEXIST, "File already exists", target_name)

        modified_from = None
        if target_name in self:
            modified_from = self[target_name]

        # Actually make the move or copy.
        if reparent:
            source_obj._reparent(self, target_name)
            item = source_obj
        else:
            item = source_obj.clone(self, target_name)

        self._items[target_name] = item
        self.set_committed(False)
        if not self._has_remote_blocks and source_obj.has_remote_blocks():
            self.set_has_remote_blocks(True)

        if modified_from:
            self.notify(MOD, self, target_name, (modified_from, item))
        else:
            self.notify(ADD, self, target_name, item)

    def _get_src_target(self, source, target_path, source_collection, create_dest):
        if source_collection is None:
            source_collection = self

        # Find the object
        if isinstance(source, basestring):
            source_obj = source_collection.find(source)
            if source_obj is None:
                raise IOError(errno.ENOENT, "File not found", source)
            sourcecomponents = source.split("/")
        else:
            source_obj = source
            sourcecomponents = None

        # Find parent collection the target path
        targetcomponents = target_path.split("/")

        # Determine the name to use.
        target_name = targetcomponents[-1] if targetcomponents[-1] else sourcecomponents[-1]

        if not target_name:
            raise errors.ArgumentError("Target path is empty and source is an object.  Cannot determine destination filename to use.")

        if create_dest:
            target_dir = self.find_or_create("/".join(targetcomponents[0:-1]), COLLECTION)
        else:
            if len(targetcomponents) > 1:
                target_dir = self.find("/".join(targetcomponents[0:-1]))
            else:
                target_dir = self

        if target_dir is None:
            raise IOError(errno.ENOENT, "Target directory not found", target_name)

        if target_name in target_dir and isinstance(target_dir[target_name], RichCollectionBase) and sourcecomponents:
            target_dir = target_dir[target_name]
            target_name = sourcecomponents[-1]

        return (source_obj, target_dir, target_name)

    @must_be_writable
    @synchronized
    def copy(self, source, target_path, source_collection=None, overwrite=False):
        """Copy a file or subcollection to a new path in this collection.

        :source:
          A string with a path to source file or subcollection, or an actual ArvadosFile or Subcollection object.

        :target_path:
          Destination file or path.  If the target path already exists and is a
          subcollection, the item will be placed inside the subcollection.  If
          the target path already exists and is a file, this will raise an error
          unless you specify `overwrite=True`.

        :source_collection:
          Collection to copy `source_path` from (default `self`)

        :overwrite:
          Whether to overwrite target file if it already exists.
        """

        source_obj, target_dir, target_name = self._get_src_target(source, target_path, source_collection, True)
        target_dir.add(source_obj, target_name, overwrite, False)

    @must_be_writable
    @synchronized
    def rename(self, source, target_path, source_collection=None, overwrite=False):
        """Move a file or subcollection from `source_collection` to a new path in this collection.

        :source:
          A string with a path to source file or subcollection.

        :target_path:
          Destination file or path.  If the target path already exists and is a
          subcollection, the item will be placed inside the subcollection.  If
          the target path already exists and is a file, this will raise an error
          unless you specify `overwrite=True`.

        :source_collection:
          Collection to copy `source_path` from (default `self`)

        :overwrite:
          Whether to overwrite target file if it already exists.
        """

        source_obj, target_dir, target_name = self._get_src_target(source, target_path, source_collection, False)
        if not source_obj.writable():
            raise IOError(errno.EROFS, "Source collection is read only", source)
        target_dir.add(source_obj, target_name, overwrite, True)

    def portable_manifest_text(self, stream_name="."):
        """Get the manifest text for this collection, sub collections and files.

        This method does not flush outstanding blocks to Keep.  It will return
        a normalized manifest with access tokens stripped.

        :stream_name:
          Name to use for this stream (directory)

        """
        return self._get_manifest_text(stream_name, True, True)

    @synchronized
    def manifest_text(self, stream_name=".", strip=False, normalize=False,
                      only_committed=False):
        """Get the manifest text for this collection, sub collections and files.

        This method will flush outstanding blocks to Keep.  By default, it will
        not normalize an unmodified manifest or strip access tokens.

        :stream_name:
          Name to use for this stream (directory)

        :strip:
          If True, remove signing tokens from block locators if present.
          If False (default), block locators are left unchanged.

        :normalize:
          If True, always export the manifest text in normalized form
          even if the Collection is not modified.  If False (default) and the collection
          is not modified, return the original manifest text even if it is not
          in normalized form.

        :only_committed:
          If True, don't commit pending blocks.

        """

        if not only_committed:
            self._my_block_manager().commit_all()
        return self._get_manifest_text(stream_name, strip, normalize,
                                       only_committed=only_committed)

    @synchronized
    def _get_manifest_text(self, stream_name, strip, normalize, only_committed=False):
        """Get the manifest text for this collection, sub collections and files.

        :stream_name:
          Name to use for this stream (directory)

        :strip:
          If True, remove signing tokens from block locators if present.
          If False (default), block locators are left unchanged.

        :normalize:
          If True, always export the manifest text in normalized form
          even if the Collection is not modified.  If False (default) and the collection
          is not modified, return the original manifest text even if it is not
          in normalized form.

        :only_committed:
          If True, only include blocks that were already committed to Keep.

        """

        if not self.committed() or self._manifest_text is None or normalize:
            stream = {}
            buf = []
            sorted_keys = sorted(self.keys())
            for filename in [s for s in sorted_keys if isinstance(self[s], ArvadosFile)]:
                # Create a stream per file `k`
                arvfile = self[filename]
                filestream = []
                for segment in arvfile.segments():
                    loc = segment.locator
                    if arvfile.parent._my_block_manager().is_bufferblock(loc):
                        if only_committed:
                            continue
                        loc = arvfile.parent._my_block_manager().get_bufferblock(loc).locator()
                    if strip:
                        loc = KeepLocator(loc).stripped()
                    filestream.append(LocatorAndRange(loc, KeepLocator(loc).size,
                                         segment.segment_offset, segment.range_size))
                stream[filename] = filestream
            if stream:
                buf.append(" ".join(normalize_stream(stream_name, stream)) + "\n")
            for dirname in [s for s in sorted_keys if isinstance(self[s], RichCollectionBase)]:
                buf.append(self[dirname].manifest_text(
                    stream_name=os.path.join(stream_name, dirname),
                    strip=strip, normalize=True, only_committed=only_committed))
            return "".join(buf)
        else:
            if strip:
                return self.stripped_manifest()
            else:
                return self._manifest_text

    @synchronized
    def _copy_remote_blocks(self, remote_blocks={}):
        """Scan through the entire collection and ask Keep to copy remote blocks.

        When accessing a remote collection, blocks will have a remote signature
        (+R instead of +A). Collect these signatures and request Keep to copy the
        blocks to the local cluster, returning local (+A) signatures.

        :remote_blocks:
          Shared cache of remote to local block mappings. This is used to avoid
          doing extra work when blocks are shared by more than one file in
          different subdirectories.

        """
        for item in self:
            remote_blocks = self[item]._copy_remote_blocks(remote_blocks)
        return remote_blocks

    @synchronized
    def diff(self, end_collection, prefix=".", holding_collection=None):
        """Generate list of add/modify/delete actions.

        When given to `apply`, will change `self` to match `end_collection`

        """
        changes = []
        if holding_collection is None:
            holding_collection = Collection(api_client=self._my_api(), keep_client=self._my_keep())
        for k in self:
            if k not in end_collection:
               changes.append((DEL, os.path.join(prefix, k), self[k].clone(holding_collection, "")))
        for k in end_collection:
            if k in self:
                if isinstance(end_collection[k], Subcollection) and isinstance(self[k], Subcollection):
                    changes.extend(self[k].diff(end_collection[k], os.path.join(prefix, k), holding_collection))
                elif end_collection[k] != self[k]:
                    changes.append((MOD, os.path.join(prefix, k), self[k].clone(holding_collection, ""), end_collection[k].clone(holding_collection, "")))
                else:
                    changes.append((TOK, os.path.join(prefix, k), self[k].clone(holding_collection, ""), end_collection[k].clone(holding_collection, "")))
            else:
                changes.append((ADD, os.path.join(prefix, k), end_collection[k].clone(holding_collection, "")))
        return changes

    @must_be_writable
    @synchronized
    def apply(self, changes):
        """Apply changes from `diff`.

        If a change conflicts with a local change, it will be saved to an
        alternate path indicating the conflict.

        """
        if changes:
            self.set_committed(False)
        for change in changes:
            event_type = change[0]
            path = change[1]
            initial = change[2]
            local = self.find(path)
            conflictpath = "%s~%s~conflict~" % (path, time.strftime("%Y%m%d-%H%M%S",
                                                                    time.gmtime()))
            if event_type == ADD:
                if local is None:
                    # No local file at path, safe to copy over new file
                    self.copy(initial, path)
                elif local is not None and local != initial:
                    # There is already local file and it is different:
                    # save change to conflict file.
                    self.copy(initial, conflictpath)
            elif event_type == MOD or event_type == TOK:
                final = change[3]
                if local == initial:
                    # Local matches the "initial" item so it has not
                    # changed locally and is safe to update.
                    if isinstance(local, ArvadosFile) and isinstance(final, ArvadosFile):
                        # Replace contents of local file with new contents
                        local.replace_contents(final)
                    else:
                        # Overwrite path with new item; this can happen if
                        # path was a file and is now a collection or vice versa
                        self.copy(final, path, overwrite=True)
                else:
                    # Local is missing (presumably deleted) or local doesn't
                    # match the "start" value, so save change to conflict file
                    self.copy(final, conflictpath)
            elif event_type == DEL:
                if local == initial:
                    # Local item matches "initial" value, so it is safe to remove.
                    self.remove(path, recursive=True)
                # else, the file is modified or already removed, in either
                # case we don't want to try to remove it.

    def portable_data_hash(self):
        """Get the portable data hash for this collection's manifest."""
        if self._manifest_locator and self.committed():
            # If the collection is already saved on the API server, and it's committed
            # then return API server's PDH response.
            return self._portable_data_hash
        else:
            stripped = self.portable_manifest_text().encode()
            return '{}+{}'.format(hashlib.md5(stripped).hexdigest(), len(stripped))

    @synchronized
    def subscribe(self, callback):
        if self._callback is None:
            self._callback = callback
        else:
            raise errors.ArgumentError("A callback is already set on this collection.")

    @synchronized
    def unsubscribe(self):
        if self._callback is not None:
            self._callback = None

    @synchronized
    def notify(self, event, collection, name, item):
        if self._callback:
            self._callback(event, collection, name, item)
        self.root_collection().notify(event, collection, name, item)

    @synchronized
    def __eq__(self, other):
        if other is self:
            return True
        if not isinstance(other, RichCollectionBase):
            return False
        if len(self._items) != len(other):
            return False
        for k in self._items:
            if k not in other:
                return False
            if self._items[k] != other[k]:
                return False
        return True

    def __ne__(self, other):
        return not self.__eq__(other)

    @synchronized
    def flush(self):
        """Flush bufferblocks to Keep."""
        for e in listvalues(self):
            e.flush()


class Collection(RichCollectionBase):
    """Represents the root of an Arvados Collection.

    This class is threadsafe.  The root collection object, all subcollections
    and files are protected by a single lock (i.e. each access locks the entire
    collection).

    Brief summary of
    useful methods:

    :To read an existing file:
      `c.open("myfile", "r")`

    :To write a new file:
      `c.open("myfile", "w")`

    :To determine if a file exists:
      `c.find("myfile") is not None`

    :To copy a file:
      `c.copy("source", "dest")`

    :To delete a file:
      `c.remove("myfile")`

    :To save to an existing collection record:
      `c.save()`

    :To save a new collection record:
    `c.save_new()`

    :To merge remote changes into this object:
      `c.update()`

    Must be associated with an API server Collection record (during
    initialization, or using `save_new`) to use `save` or `update`

    """

    def __init__(self, manifest_locator_or_text=None,
                 api_client=None,
                 keep_client=None,
                 num_retries=10,
                 parent=None,
                 apiconfig=None,
                 block_manager=None,
                 replication_desired=None,
                 storage_classes_desired=None,
                 put_threads=None,
                 get_threads=None):
        """Collection constructor.

        :manifest_locator_or_text:
          An Arvados collection UUID, portable data hash, raw manifest
          text, or (if creating an empty collection) None.

        :parent:
          the parent Collection, may be None.

        :apiconfig:
          A dict containing keys for ARVADOS_API_HOST and ARVADOS_API_TOKEN.
          Prefer this over supplying your own api_client and keep_client (except in testing).
          Will use default config settings if not specified.

        :api_client:
          The API client object to use for requests.  If not specified, create one using `apiconfig`.

        :keep_client:
          the Keep client to use for requests.  If not specified, create one using `apiconfig`.

        :num_retries:
          the number of retries for API and Keep requests.

        :block_manager:
          the block manager to use.  If not specified, create one.

        :replication_desired:
          How many copies should Arvados maintain. If None, API server default
          configuration applies. If not None, this value will also be used
          for determining the number of block copies being written.

        :storage_classes_desired:
          A list of storage class names where to upload the data. If None,
          the keep client is expected to store the data into the cluster's
          default storage class(es).

        """

        if storage_classes_desired and type(storage_classes_desired) is not list:
            raise errors.ArgumentError("storage_classes_desired must be list type.")

        super(Collection, self).__init__(parent)
        self._api_client = api_client
        self._keep_client = keep_client

        # Use the keep client from ThreadSafeApiCache
        if self._keep_client is None and isinstance(self._api_client, ThreadSafeApiCache):
            self._keep_client = self._api_client.keep

        self._block_manager = block_manager
        self.replication_desired = replication_desired
        self._storage_classes_desired = storage_classes_desired
        self.put_threads = put_threads
        self.get_threads = get_threads

        if apiconfig:
            self._config = apiconfig
        else:
            self._config = config.settings()

        self.num_retries = num_retries
        self._manifest_locator = None
        self._manifest_text = None
        self._portable_data_hash = None
        self._api_response = None
        self._past_versions = set()

        self.lock = threading.RLock()
        self.events = None

        if manifest_locator_or_text:
            if re.match(arvados.util.keep_locator_pattern, manifest_locator_or_text):
                self._manifest_locator = manifest_locator_or_text
            elif re.match(arvados.util.collection_uuid_pattern, manifest_locator_or_text):
                self._manifest_locator = manifest_locator_or_text
                if not self._has_local_collection_uuid():
                    self._has_remote_blocks = True
            elif re.match(arvados.util.manifest_pattern, manifest_locator_or_text):
                self._manifest_text = manifest_locator_or_text
                if '+R' in self._manifest_text:
                    self._has_remote_blocks = True
            else:
                raise errors.ArgumentError(
                    "Argument to CollectionReader is not a manifest or a collection UUID")

            try:
                self._populate()
            except errors.SyntaxError as e:
                raise errors.ArgumentError("Error processing manifest text: %s", str(e)) from None

    def storage_classes_desired(self):
        return self._storage_classes_desired or []

    def root_collection(self):
        return self

    def get_properties(self):
        if self._api_response and self._api_response["properties"]:
            return self._api_response["properties"]
        else:
            return {}

    def get_trash_at(self):
        if self._api_response and self._api_response["trash_at"]:
            try:
                return ciso8601.parse_datetime(self._api_response["trash_at"])
            except ValueError:
                return None
        else:
            return None

    def stream_name(self):
        return "."

    def writable(self):
        return True

    @synchronized
    def known_past_version(self, modified_at_and_portable_data_hash):
        return modified_at_and_portable_data_hash in self._past_versions

    @synchronized
    @retry_method
    def update(self, other=None, num_retries=None):
        """Merge the latest collection on the API server with the current collection."""

        if other is None:
            if self._manifest_locator is None:
                raise errors.ArgumentError("`other` is None but collection does not have a manifest_locator uuid")
            response = self._my_api().collections().get(uuid=self._manifest_locator).execute(num_retries=num_retries)
            if (self.known_past_version((response.get("modified_at"), response.get("portable_data_hash"))) and
                response.get("portable_data_hash") != self.portable_data_hash()):
                # The record on the server is different from our current one, but we've seen it before,
                # so ignore it because it's already been merged.
                # However, if it's the same as our current record, proceed with the update, because we want to update
                # our tokens.
                return
            else:
                self._remember_api_response(response)
            other = CollectionReader(response["manifest_text"])
        baseline = CollectionReader(self._manifest_text)
        self.apply(baseline.diff(other))
        self._manifest_text = self.manifest_text()

    @synchronized
    def _my_api(self):
        if self._api_client is None:
            self._api_client = ThreadSafeApiCache(self._config, version='v1')
            if self._keep_client is None:
                self._keep_client = self._api_client.keep
        return self._api_client

    @synchronized
    def _my_keep(self):
        if self._keep_client is None:
            if self._api_client is None:
                self._my_api()
            else:
                self._keep_client = KeepClient(api_client=self._api_client)
        return self._keep_client

    @synchronized
    def _my_block_manager(self):
        if self._block_manager is None:
            copies = (self.replication_desired or
                      self._my_api()._rootDesc.get('defaultCollectionReplication',
                                                   2))
            self._block_manager = _BlockManager(self._my_keep(),
                                                copies=copies,
                                                put_threads=self.put_threads,
                                                num_retries=self.num_retries,
                                                storage_classes_func=self.storage_classes_desired,
                                                get_threads=self.get_threads,)
        return self._block_manager

    def _remember_api_response(self, response):
        self._api_response = response
        self._past_versions.add((response.get("modified_at"), response.get("portable_data_hash")))

    def _populate_from_api_server(self):
        # As in KeepClient itself, we must wait until the last
        # possible moment to instantiate an API client, in order to
        # avoid tripping up clients that don't have access to an API
        # server.  If we do build one, make sure our Keep client uses
        # it.  If instantiation fails, we'll fall back to the except
        # clause, just like any other Collection lookup
        # failure. Return an exception, or None if successful.
        self._remember_api_response(self._my_api().collections().get(
            uuid=self._manifest_locator).execute(
                num_retries=self.num_retries))
        self._manifest_text = self._api_response['manifest_text']
        self._portable_data_hash = self._api_response['portable_data_hash']
        # If not overriden via kwargs, we should try to load the
        # replication_desired and storage_classes_desired from the API server
        if self.replication_desired is None:
            self.replication_desired = self._api_response.get('replication_desired', None)
        if self._storage_classes_desired is None:
            self._storage_classes_desired = self._api_response.get('storage_classes_desired', None)

    def _populate(self):
        if self._manifest_text is None:
            if self._manifest_locator is None:
                return
            else:
                self._populate_from_api_server()
        self._baseline_manifest = self._manifest_text
        self._import_manifest(self._manifest_text)

    def _has_collection_uuid(self):
        return self._manifest_locator is not None and re.match(arvados.util.collection_uuid_pattern, self._manifest_locator)

    def _has_local_collection_uuid(self):
        return self._has_collection_uuid and \
            self._my_api()._rootDesc['uuidPrefix'] == self._manifest_locator.split('-')[0]

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        """Support scoped auto-commit in a with: block."""
        if exc_type is None:
            if self.writable() and self._has_collection_uuid():
                self.save()
        self.stop_threads()

    def stop_threads(self):
        if self._block_manager is not None:
            self._block_manager.stop_threads()

    @synchronized
    def manifest_locator(self):
        """Get the manifest locator, if any.

        The manifest locator will be set when the collection is loaded from an
        API server record or the portable data hash of a manifest.

        The manifest locator will be None if the collection is newly created or
        was created directly from manifest text.  The method `save_new()` will
        assign a manifest locator.

        """
        return self._manifest_locator

    @synchronized
    def clone(self, new_parent=None, new_name=None, readonly=False, new_config=None):
        if new_config is None:
            new_config = self._config
        if readonly:
            newcollection = CollectionReader(parent=new_parent, apiconfig=new_config)
        else:
            newcollection = Collection(parent=new_parent, apiconfig=new_config)

        newcollection._clonefrom(self)
        return newcollection

    @synchronized
    def api_response(self):
        """Returns information about this Collection fetched from the API server.

        If the Collection exists in Keep but not the API server, currently
        returns None.  Future versions may provide a synthetic response.

        """
        return self._api_response

    def find_or_create(self, path, create_type):
        """See `RichCollectionBase.find_or_create`"""
        if path == ".":
            return self
        else:
            return super(Collection, self).find_or_create(path[2:] if path.startswith("./") else path, create_type)

    def find(self, path):
        """See `RichCollectionBase.find`"""
        if path == ".":
            return self
        else:
            return super(Collection, self).find(path[2:] if path.startswith("./") else path)

    def remove(self, path, recursive=False):
        """See `RichCollectionBase.remove`"""
        if path == ".":
            raise errors.ArgumentError("Cannot remove '.'")
        else:
            return super(Collection, self).remove(path[2:] if path.startswith("./") else path, recursive)

    @must_be_writable
    @synchronized
    @retry_method
    def save(self,
             properties=None,
             storage_classes=None,
             trash_at=None,
             merge=True,
             num_retries=None,
             preserve_version=False):
        """Save collection to an existing collection record.

        Commit pending buffer blocks to Keep, merge with remote record (if
        merge=True, the default), and update the collection record. Returns
        the current manifest text.

        Will raise AssertionError if not associated with a collection record on
        the API server.  If you want to save a manifest to Keep only, see
        `save_new()`.

        :properties:
          Additional properties of collection. This value will replace any existing
          properties of collection.

        :storage_classes:
          Specify desirable storage classes to be used when writing data to Keep.

        :trash_at:
          A collection is *expiring* when it has a *trash_at* time in the future.
          An expiring collection can be accessed as normal,
          but is scheduled to be trashed automatically at the *trash_at* time.

        :merge:
          Update and merge remote changes before saving.  Otherwise, any
          remote changes will be ignored and overwritten.

        :num_retries:
          Retry count on API calls (if None,  use the collection default)

        :preserve_version:
          If True, indicate that the collection content being saved right now
          should be preserved in a version snapshot if the collection record is
          updated in the future. Requires that the API server has
          Collections.CollectionVersioning enabled, if not, setting this will
          raise an exception.

        """
        if properties and type(properties) is not dict:
            raise errors.ArgumentError("properties must be dictionary type.")

        if storage_classes and type(storage_classes) is not list:
            raise errors.ArgumentError("storage_classes must be list type.")
        if storage_classes:
            self._storage_classes_desired = storage_classes

        if trash_at and type(trash_at) is not datetime.datetime:
            raise errors.ArgumentError("trash_at must be datetime type.")

        if preserve_version and not self._my_api().config()['Collections'].get('CollectionVersioning', False):
            raise errors.ArgumentError("preserve_version is not supported when CollectionVersioning is not enabled.")

        body={}
        if properties:
            body["properties"] = properties
        if self.storage_classes_desired():
            body["storage_classes_desired"] = self.storage_classes_desired()
        if trash_at:
            t = trash_at.strftime("%Y-%m-%dT%H:%M:%S.%fZ")
            body["trash_at"] = t
        if preserve_version:
            body["preserve_version"] = preserve_version

        if not self.committed():
            if self._has_remote_blocks:
                # Copy any remote blocks to the local cluster.
                self._copy_remote_blocks(remote_blocks={})
                self._has_remote_blocks = False
            if not self._has_collection_uuid():
                raise AssertionError("Collection manifest_locator is not a collection uuid.  Use save_new() for new collections.")
            elif not self._has_local_collection_uuid():
                raise AssertionError("Collection manifest_locator is from a remote cluster. Use save_new() to save it on the local cluster.")

            self._my_block_manager().commit_all()

            if merge:
                self.update()

            text = self.manifest_text(strip=False)
            body['manifest_text'] = text

            self._remember_api_response(self._my_api().collections().update(
                uuid=self._manifest_locator,
                body=body
                ).execute(num_retries=num_retries))
            self._manifest_text = self._api_response["manifest_text"]
            self._portable_data_hash = self._api_response["portable_data_hash"]
            self.set_committed(True)
        elif body:
            self._remember_api_response(self._my_api().collections().update(
                uuid=self._manifest_locator,
                body=body
                ).execute(num_retries=num_retries))

        return self._manifest_text


    @must_be_writable
    @synchronized
    @retry_method
    def save_new(self, name=None,
                 create_collection_record=True,
                 owner_uuid=None,
                 properties=None,
                 storage_classes=None,
                 trash_at=None,
                 ensure_unique_name=False,
                 num_retries=None,
                 preserve_version=False):
        """Save collection to a new collection record.

        Commit pending buffer blocks to Keep and, when create_collection_record
        is True (default), create a new collection record.  After creating a
        new collection record, this Collection object will be associated with
        the new record used by `save()`. Returns the current manifest text.

        :name:
          The collection name.

        :create_collection_record:
           If True, create a collection record on the API server.
           If False, only commit blocks to Keep and return the manifest text.

        :owner_uuid:
          the user, or project uuid that will own this collection.
          If None, defaults to the current user.

        :properties:
          Additional properties of collection. This value will replace any existing
          properties of collection.

        :storage_classes:
          Specify desirable storage classes to be used when writing data to Keep.

        :trash_at:
          A collection is *expiring* when it has a *trash_at* time in the future.
          An expiring collection can be accessed as normal,
          but is scheduled to be trashed automatically at the *trash_at* time.

        :ensure_unique_name:
          If True, ask the API server to rename the collection
          if it conflicts with a collection with the same name and owner.  If
          False, a name conflict will result in an error.

        :num_retries:
          Retry count on API calls (if None,  use the collection default)

        :preserve_version:
          If True, indicate that the collection content being saved right now
          should be preserved in a version snapshot if the collection record is
          updated in the future. Requires that the API server has
          Collections.CollectionVersioning enabled, if not, setting this will
          raise an exception.

        """
        if properties and type(properties) is not dict:
            raise errors.ArgumentError("properties must be dictionary type.")

        if storage_classes and type(storage_classes) is not list:
            raise errors.ArgumentError("storage_classes must be list type.")

        if trash_at and type(trash_at) is not datetime.datetime:
            raise errors.ArgumentError("trash_at must be datetime type.")

        if preserve_version and not self._my_api().config()['Collections'].get('CollectionVersioning', False):
            raise errors.ArgumentError("preserve_version is not supported when CollectionVersioning is not enabled.")

        if self._has_remote_blocks:
            # Copy any remote blocks to the local cluster.
            self._copy_remote_blocks(remote_blocks={})
            self._has_remote_blocks = False

        if storage_classes:
            self._storage_classes_desired = storage_classes

        self._my_block_manager().commit_all()
        text = self.manifest_text(strip=False)

        if create_collection_record:
            if name is None:
                name = "New collection"
                ensure_unique_name = True

            body = {"manifest_text": text,
                    "name": name,
                    "replication_desired": self.replication_desired}
            if owner_uuid:
                body["owner_uuid"] = owner_uuid
            if properties:
                body["properties"] = properties
            if self.storage_classes_desired():
                body["storage_classes_desired"] = self.storage_classes_desired()
            if trash_at:
                t = trash_at.strftime("%Y-%m-%dT%H:%M:%S.%fZ")
                body["trash_at"] = t
            if preserve_version:
                body["preserve_version"] = preserve_version

            self._remember_api_response(self._my_api().collections().create(ensure_unique_name=ensure_unique_name, body=body).execute(num_retries=num_retries))
            text = self._api_response["manifest_text"]

            self._manifest_locator = self._api_response["uuid"]
            self._portable_data_hash = self._api_response["portable_data_hash"]

            self._manifest_text = text
            self.set_committed(True)

        return text

    _token_re = re.compile(r'(\S+)(\s+|$)')
    _block_re = re.compile(r'[0-9a-f]{32}\+(\d+)(\+\S+)*')
    _segment_re = re.compile(r'(\d+):(\d+):(\S+)')

    def _unescape_manifest_path(self, path):
        return re.sub('\\\\([0-3][0-7][0-7])', lambda m: chr(int(m.group(1), 8)), path)

    @synchronized
    def _import_manifest(self, manifest_text):
        """Import a manifest into a `Collection`.

        :manifest_text:
          The manifest text to import from.

        """
        if len(self) > 0:
            raise ArgumentError("Can only import manifest into an empty collection")

        STREAM_NAME = 0
        BLOCKS = 1
        SEGMENTS = 2

        stream_name = None
        state = STREAM_NAME

        for token_and_separator in self._token_re.finditer(manifest_text):
            tok = token_and_separator.group(1)
            sep = token_and_separator.group(2)

            if state == STREAM_NAME:
                # starting a new stream
                stream_name = self._unescape_manifest_path(tok)
                blocks = []
                segments = []
                streamoffset = 0
                state = BLOCKS
                self.find_or_create(stream_name, COLLECTION)
                continue

            if state == BLOCKS:
                block_locator = self._block_re.match(tok)
                if block_locator:
                    blocksize = int(block_locator.group(1))
                    blocks.append(Range(tok, streamoffset, blocksize, 0))
                    streamoffset += blocksize
                else:
                    state = SEGMENTS

            if state == SEGMENTS:
                file_segment = self._segment_re.match(tok)
                if file_segment:
                    pos = int(file_segment.group(1))
                    size = int(file_segment.group(2))
                    name = self._unescape_manifest_path(file_segment.group(3))
                    if name.split('/')[-1] == '.':
                        # placeholder for persisting an empty directory, not a real file
                        if len(name) > 2:
                            self.find_or_create(os.path.join(stream_name, name[:-2]), COLLECTION)
                    else:
                        filepath = os.path.join(stream_name, name)
                        try:
                            afile = self.find_or_create(filepath, FILE)
                        except IOError as e:
                            if e.errno == errno.ENOTDIR:
                                raise errors.SyntaxError("Dir part of %s conflicts with file of the same name.", filepath) from None
                            else:
                                raise e from None
                        if isinstance(afile, ArvadosFile):
                            afile.add_segment(blocks, pos, size)
                        else:
                            raise errors.SyntaxError("File %s conflicts with stream of the same name.", filepath)
                else:
                    # error!
                    raise errors.SyntaxError("Invalid manifest format, expected file segment but did not match format: '%s'" % tok)

            if sep == "\n":
                stream_name = None
                state = STREAM_NAME

        self.set_committed(True)

    @synchronized
    def notify(self, event, collection, name, item):
        if self._callback:
            self._callback(event, collection, name, item)


class Subcollection(RichCollectionBase):
    """This is a subdirectory within a collection that doesn't have its own API
    server record.

    Subcollection locking falls under the umbrella lock of its root collection.

    """

    def __init__(self, parent, name):
        super(Subcollection, self).__init__(parent)
        self.lock = self.root_collection().lock
        self._manifest_text = None
        self.name = name
        self.num_retries = parent.num_retries

    def root_collection(self):
        return self.parent.root_collection()

    def writable(self):
        return self.root_collection().writable()

    def _my_api(self):
        return self.root_collection()._my_api()

    def _my_keep(self):
        return self.root_collection()._my_keep()

    def _my_block_manager(self):
        return self.root_collection()._my_block_manager()

    def stream_name(self):
        return os.path.join(self.parent.stream_name(), self.name)

    @synchronized
    def clone(self, new_parent, new_name):
        c = Subcollection(new_parent, new_name)
        c._clonefrom(self)
        return c

    @must_be_writable
    @synchronized
    def _reparent(self, newparent, newname):
        self.set_committed(False)
        self.flush()
        self.parent.remove(self.name, recursive=True)
        self.parent = newparent
        self.name = newname
        self.lock = self.parent.root_collection().lock

    @synchronized
    def _get_manifest_text(self, stream_name, strip, normalize, only_committed=False):
        """Encode empty directories by using an \056-named (".") empty file"""
        if len(self._items) == 0:
            return "%s %s 0:0:\\056\n" % (
                escape(stream_name), config.EMPTY_BLOCK_LOCATOR)
        return super(Subcollection, self)._get_manifest_text(stream_name,
                                                             strip, normalize,
                                                             only_committed)


class CollectionReader(Collection):
    """A read-only collection object.

    Initialize from a collection UUID or portable data hash, or raw
    manifest text.  See `Collection` constructor for detailed options.

    """
    def __init__(self, manifest_locator_or_text, *args, **kwargs):
        self._in_init = True
        super(CollectionReader, self).__init__(manifest_locator_or_text, *args, **kwargs)
        self._in_init = False

        # Forego any locking since it should never change once initialized.
        self.lock = NoopLock()

        # Backwards compatability with old CollectionReader
        # all_streams() and all_files()
        self._streams = None

    def writable(self):
        return self._in_init

    def _populate_streams(orig_func):
        @functools.wraps(orig_func)
        def populate_streams_wrapper(self, *args, **kwargs):
            # Defer populating self._streams until needed since it creates a copy of the manifest.
            if self._streams is None:
                if self._manifest_text:
                    self._streams = [sline.split()
                                     for sline in self._manifest_text.split("\n")
                                     if sline]
                else:
                    self._streams = []
            return orig_func(self, *args, **kwargs)
        return populate_streams_wrapper

    @_populate_streams
    def normalize(self):
        """Normalize the streams returned by `all_streams`.

        This method is kept for backwards compatability and only affects the
        behavior of `all_streams()` and `all_files()`

        """

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
    @_populate_streams
    def all_streams(self):
        return [StreamReader(s, self._my_keep(), num_retries=self.num_retries)
                for s in self._streams]

    @_populate_streams
    def all_files(self):
        for s in self.all_streams():
            for f in s.all_files():
                yield f
