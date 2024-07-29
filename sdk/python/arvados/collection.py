# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
"""Tools to work with Arvados collections

This module provides high-level interfaces to create, read, and update
Arvados collections. Most users will want to instantiate `Collection`
objects, and use methods like `Collection.open` and `Collection.mkdirs` to
read and write data in the collection. Refer to the Arvados Python SDK
cookbook for [an introduction to using the Collection class][cookbook].

[cookbook]: https://doc.arvados.org/sdk/python/cookbook.html#working-with-collections
"""

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

from ._internal import streams
from .api import ThreadSafeAPIClient
from .arvfile import split, _FileLikeObjectBase, ArvadosFile, ArvadosFileWriter, ArvadosFileReader, WrappableFile, _BlockManager, synchronized, must_be_writable, NoopLock
from .keep import KeepLocator, KeepClient
import arvados.config as config
import arvados.errors as errors
import arvados.util
import arvados.events as events
from arvados.retry import retry_method

from typing import (
    Any,
    Callable,
    Dict,
    IO,
    Iterator,
    List,
    Mapping,
    Optional,
    Tuple,
    Union,
)

if sys.version_info < (3, 8):
    from typing_extensions import Literal
else:
    from typing import Literal

_logger = logging.getLogger('arvados.collection')

ADD = "add"
"""Argument value for `Collection` methods to represent an added item"""
DEL = "del"
"""Argument value for `Collection` methods to represent a removed item"""
MOD = "mod"
"""Argument value for `Collection` methods to represent a modified item"""
TOK = "tok"
"""Argument value for `Collection` methods to represent an item with token differences"""
FILE = "file"
"""`create_type` value for `Collection.find_or_create`"""
COLLECTION = "collection"
"""`create_type` value for `Collection.find_or_create`"""

ChangeList = List[Union[
    Tuple[Literal[ADD, DEL], str, 'Collection'],
    Tuple[Literal[MOD, TOK], str, 'Collection', 'Collection'],
]]
ChangeType = Literal[ADD, DEL, MOD, TOK]
CollectionItem = Union[ArvadosFile, 'Collection']
ChangeCallback = Callable[[ChangeType, 'Collection', str, CollectionItem], object]
CreateType = Literal[COLLECTION, FILE]
Properties = Dict[str, Any]
StorageClasses = List[str]

class CollectionBase(object):
    """Abstract base class for Collection classes

    .. ATTENTION:: Internal
       This class is meant to be used by other parts of the SDK. User code
       should instantiate or subclass `Collection` or one of its subclasses
       directly.
    """

    def __enter__(self):
        """Enter a context block with this collection instance"""
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        """Exit a context block with this collection instance"""
        pass

    def _my_keep(self):
        if self._keep_client is None:
            self._keep_client = KeepClient(api_client=self._api_client,
                                           num_retries=self.num_retries)
        return self._keep_client

    def stripped_manifest(self) -> str:
        """Create a copy of the collection manifest with only size hints

        This method returns a string with the current collection's manifest
        text with all non-portable locator hints like permission hints and
        remote cluster hints removed. The only hints in the returned manifest
        will be size hints.
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


class RichCollectionBase(CollectionBase):
    """Base class for Collection classes

    .. ATTENTION:: Internal
       This class is meant to be used by other parts of the SDK. User code
       should instantiate or subclass `Collection` or one of its subclasses
       directly.
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

    def writable(self) -> bool:
        """Indicate whether this collection object can be modified

        This method returns `False` if this object is a `CollectionReader`,
        else `True`.
        """
        raise NotImplementedError()

    def root_collection(self) -> 'Collection':
        """Get this collection's root collection object

        If you open a subcollection with `Collection.find`, calling this method
        on that subcollection returns the source Collection object.
        """
        raise NotImplementedError()

    def stream_name(self) -> str:
        """Get the name of the manifest stream represented by this collection

        If you open a subcollection with `Collection.find`, calling this method
        on that subcollection returns the name of the stream you opened.
        """
        raise NotImplementedError()

    @synchronized
    def has_remote_blocks(self) -> bool:
        """Indiciate whether the collection refers to remote data

        Returns `True` if the collection manifest includes any Keep locators
        with a remote hint (`+R`), else `False`.
        """
        if self._has_remote_blocks:
            return True
        for item in self:
            if self[item].has_remote_blocks():
                return True
        return False

    @synchronized
    def set_has_remote_blocks(self, val: bool) -> None:
        """Cache whether this collection refers to remote blocks

        .. ATTENTION:: Internal
           This method is only meant to be used by other Collection methods.

        Set this collection's cached "has remote blocks" flag to the given
        value.
        """
        self._has_remote_blocks = val
        if self.parent:
            self.parent.set_has_remote_blocks(val)

    @must_be_writable
    @synchronized
    def find_or_create(
            self,
            path: str,
            create_type: CreateType,
    ) -> CollectionItem:
        """Get the item at the given path, creating it if necessary

        If `path` refers to a stream in this collection, returns a
        corresponding `Subcollection` object. If `path` refers to a file in
        this collection, returns a corresponding
        `arvados.arvfile.ArvadosFile` object. If `path` does not exist in
        this collection, then this method creates a new object and returns
        it, creating parent streams as needed. The type of object created is
        determined by the value of `create_type`.

        Arguments:

        * path: str --- The path to find or create within this collection.

        * create_type: Literal[COLLECTION, FILE] --- The type of object to
          create at `path` if one does not exist. Passing `COLLECTION`
          creates a stream and returns the corresponding
          `Subcollection`. Passing `FILE` creates a new file and returns the
          corresponding `arvados.arvfile.ArvadosFile`.
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
    def find(self, path: str) -> CollectionItem:
        """Get the item at the given path

        If `path` refers to a stream in this collection, returns a
        corresponding `Subcollection` object. If `path` refers to a file in
        this collection, returns a corresponding
        `arvados.arvfile.ArvadosFile` object. If `path` does not exist in
        this collection, then this method raises `NotADirectoryError`.

        Arguments:

        * path: str --- The path to find or create within this collection.
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
    def mkdirs(self, path: str) -> 'Subcollection':
        """Create and return a subcollection at `path`

        If `path` exists within this collection, raises `FileExistsError`.
        Otherwise, creates a stream at that path and returns the
        corresponding `Subcollection`.
        """
        if self.find(path) != None:
            raise IOError(errno.EEXIST, "Directory or file exists", path)

        return self.find_or_create(path, COLLECTION)

    def open(
            self,
            path: str,
            mode: str="r",
            encoding: Optional[str]=None
    ) -> IO:
        """Open a file-like object within the collection

        This method returns a file-like object that can read and/or write the
        file located at `path` within the collection. If you attempt to write
        a `path` that does not exist, the file is created with `find_or_create`.
        If the file cannot be opened for any other reason, this method raises
        `OSError` with an appropriate errno.

        Arguments:

        * path: str --- The path of the file to open within this collection

        * mode: str --- The mode to open this file. Supports all the same
          values as `builtins.open`.

        * encoding: str | None --- The text encoding of the file. Only used
          when the file is opened in text mode. The default is
          platform-dependent.

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
            f = io.TextIOWrapper(bufferclass(WrappableFile(f)), encoding=encoding)
        return f

    def modified(self) -> bool:
        """Indicate whether this collection has an API server record

        Returns `False` if this collection corresponds to a record loaded from
        the API server, `True` otherwise.
        """
        return not self.committed()

    @synchronized
    def committed(self):
        """Indicate whether this collection has an API server record

        Returns `True` if this collection corresponds to a record loaded from
        the API server, `False` otherwise.
        """
        return self._committed

    @synchronized
    def set_committed(self, value: bool=True):
        """Cache whether this collection has an API server record

        .. ATTENTION:: Internal
           This method is only meant to be used by other Collection methods.

        Set this collection's cached "committed" flag to the given
        value and propagates it as needed.
        """
        if value == self._committed:
            return
        if value:
            for k,v in self._items.items():
                v.set_committed(True)
            self._committed = True
        else:
            self._committed = False
            if self.parent is not None:
                self.parent.set_committed(False)

    @synchronized
    def __iter__(self) -> Iterator[str]:
        """Iterate names of streams and files in this collection

        This method does not recurse. It only iterates the contents of this
        collection's corresponding stream.
        """
        return iter(self._items)

    @synchronized
    def __getitem__(self, k: str) -> CollectionItem:
        """Get a `arvados.arvfile.ArvadosFile` or `Subcollection` in this collection

        This method does not recurse. If you want to search a path, use
        `RichCollectionBase.find` instead.
        """
        return self._items[k]

    @synchronized
    def __contains__(self, k: str) -> bool:
        """Indicate whether this collection has an item with this name

        This method does not recurse. It you want to check a path, use
        `RichCollectionBase.exists` instead.
        """
        return k in self._items

    @synchronized
    def __len__(self):
        """Get the number of items directly contained in this collection

        This method does not recurse. It only counts the streams and files
        in this collection's corresponding stream.
        """
        return len(self._items)

    @must_be_writable
    @synchronized
    def __delitem__(self, p: str) -> None:
        """Delete an item from this collection's stream

        This method does not recurse. If you want to remove an item by a
        path, use `RichCollectionBase.remove` instead.
        """
        del self._items[p]
        self.set_committed(False)
        self.notify(DEL, self, p, None)

    @synchronized
    def keys(self) -> Iterator[str]:
        """Iterate names of streams and files in this collection

        This method does not recurse. It only iterates the contents of this
        collection's corresponding stream.
        """
        return self._items.keys()

    @synchronized
    def values(self) -> List[CollectionItem]:
        """Get a list of objects in this collection's stream

        The return value includes a `Subcollection` for every stream, and an
        `arvados.arvfile.ArvadosFile` for every file, directly within this
        collection's stream.  This method does not recurse.
        """
        return list(self._items.values())

    @synchronized
    def items(self) -> List[Tuple[str, CollectionItem]]:
        """Get a list of `(name, object)` tuples from this collection's stream

        The return value includes a `Subcollection` for every stream, and an
        `arvados.arvfile.ArvadosFile` for every file, directly within this
        collection's stream.  This method does not recurse.
        """
        return list(self._items.items())

    def exists(self, path: str) -> bool:
        """Indicate whether this collection includes an item at `path`

        This method returns `True` if `path` refers to a stream or file within
        this collection, else `False`.

        Arguments:

        * path: str --- The path to check for existence within this collection
        """
        return self.find(path) is not None

    @must_be_writable
    @synchronized
    def remove(self, path: str, recursive: bool=False) -> None:
        """Remove the file or stream at `path`

        Arguments:

        * path: str --- The path of the item to remove from the collection

        * recursive: bool --- Controls the method's behavior if `path` refers
          to a nonempty stream. If `False` (the default), this method raises
          `OSError` with errno `ENOTEMPTY`. If `True`, this method removes all
          items under the stream.
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
        for k,v in source.items():
            self._items[k] = v.clone(self, k)

    def clone(self):
        raise NotImplementedError()

    @must_be_writable
    @synchronized
    def add(
            self,
            source_obj: CollectionItem,
            target_name: str,
            overwrite: bool=False,
            reparent: bool=False,
    ) -> None:
        """Copy or move a file or subcollection object to this collection

        Arguments:

        * source_obj: arvados.arvfile.ArvadosFile | Subcollection --- The file or subcollection
          to add to this collection

        * target_name: str --- The path inside this collection where
          `source_obj` should be added.

        * overwrite: bool --- Controls the behavior of this method when the
          collection already contains an object at `target_name`. If `False`
          (the default), this method will raise `FileExistsError`. If `True`,
          the object at `target_name` will be replaced with `source_obj`.

        * reparent: bool --- Controls whether this method copies or moves
          `source_obj`. If `False` (the default), `source_obj` is copied into
          this collection. If `True`, `source_obj` is moved into this
          collection.
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
        if isinstance(source, str):
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
    def copy(
            self,
            source: Union[str, CollectionItem],
            target_path: str,
            source_collection: Optional['RichCollectionBase']=None,
            overwrite: bool=False,
    ) -> None:
        """Copy a file or subcollection object to this collection

        Arguments:

        * source: str | arvados.arvfile.ArvadosFile |
          arvados.collection.Subcollection --- The file or subcollection to
          add to this collection. If `source` is a str, the object will be
          found by looking up this path from `source_collection` (see
          below).

        * target_path: str --- The path inside this collection where the
          source object should be added.

        * source_collection: arvados.collection.Collection | None --- The
          collection to find the source object from when `source` is a
          path. Defaults to the current collection (`self`).

        * overwrite: bool --- Controls the behavior of this method when the
          collection already contains an object at `target_path`. If `False`
          (the default), this method will raise `FileExistsError`. If `True`,
          the object at `target_path` will be replaced with `source_obj`.
        """
        source_obj, target_dir, target_name = self._get_src_target(source, target_path, source_collection, True)
        target_dir.add(source_obj, target_name, overwrite, False)

    @must_be_writable
    @synchronized
    def rename(
            self,
            source: Union[str, CollectionItem],
            target_path: str,
            source_collection: Optional['RichCollectionBase']=None,
            overwrite: bool=False,
    ) -> None:
        """Move a file or subcollection object to this collection

        Arguments:

        * source: str | arvados.arvfile.ArvadosFile |
          arvados.collection.Subcollection --- The file or subcollection to
          add to this collection. If `source` is a str, the object will be
          found by looking up this path from `source_collection` (see
          below).

        * target_path: str --- The path inside this collection where the
          source object should be added.

        * source_collection: arvados.collection.Collection | None --- The
          collection to find the source object from when `source` is a
          path. Defaults to the current collection (`self`).

        * overwrite: bool --- Controls the behavior of this method when the
          collection already contains an object at `target_path`. If `False`
          (the default), this method will raise `FileExistsError`. If `True`,
          the object at `target_path` will be replaced with `source_obj`.
        """
        source_obj, target_dir, target_name = self._get_src_target(source, target_path, source_collection, False)
        if not source_obj.writable():
            raise IOError(errno.EROFS, "Source collection is read only", source)
        target_dir.add(source_obj, target_name, overwrite, True)

    def portable_manifest_text(self, stream_name: str=".") -> str:
        """Get the portable manifest text for this collection

        The portable manifest text is normalized, and does not include access
        tokens. This method does not flush outstanding blocks to Keep.

        Arguments:

        * stream_name: str --- The name to use for this collection's stream in
          the generated manifest. Default `'.'`.
        """
        return self._get_manifest_text(stream_name, True, True)

    @synchronized
    def manifest_text(
            self,
            stream_name: str=".",
            strip: bool=False,
            normalize: bool=False,
            only_committed: bool=False,
    ) -> str:
        """Get the manifest text for this collection

        Arguments:

        * stream_name: str --- The name to use for this collection's stream in
          the generated manifest. Default `'.'`.

        * strip: bool --- Controls whether or not the returned manifest text
          includes access tokens. If `False` (the default), the manifest text
          will include access tokens. If `True`, the manifest text will not
          include access tokens.

        * normalize: bool --- Controls whether or not the returned manifest
          text is normalized. Default `False`.

        * only_committed: bool --- Controls whether or not this method uploads
          pending data to Keep before building and returning the manifest text.
          If `False` (the default), this method will finish uploading all data
          to Keep, then return the final manifest. If `True`, this method will
          build and return a manifest that only refers to the data that has
          finished uploading at the time this method was called.
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
                    filestream.append(streams.LocatorAndRange(
                        loc,
                        KeepLocator(loc).size,
                        segment.segment_offset,
                        segment.range_size,
                    ))
                stream[filename] = filestream
            if stream:
                buf.append(" ".join(streams.normalize_stream(stream_name, stream)) + "\n")
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
    def diff(
            self,
            end_collection: 'RichCollectionBase',
            prefix: str=".",
            holding_collection: Optional['Collection']=None,
    ) -> ChangeList:
        """Build a list of differences between this collection and another

        Arguments:

        * end_collection: arvados.collection.RichCollectionBase --- A
          collection object with the desired end state. The returned diff
          list will describe how to go from the current collection object
          `self` to `end_collection`.

        * prefix: str --- The name to use for this collection's stream in
          the diff list. Default `'.'`.

        * holding_collection: arvados.collection.Collection | None --- A
          collection object used to hold objects for the returned diff
          list. By default, a new empty collection is created.
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
    def apply(self, changes: ChangeList) -> None:
        """Apply a list of changes from to this collection

        This method takes a list of changes generated by
        `RichCollectionBase.diff` and applies it to this
        collection. Afterward, the state of this collection object will
        match the state of `end_collection` passed to `diff`. If a change
        conflicts with a local change, it will be saved to an alternate path
        indicating the conflict.

        Arguments:

        * changes: arvados.collection.ChangeList --- The list of differences
          generated by `RichCollectionBase.diff`.
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

    def portable_data_hash(self) -> str:
        """Get the portable data hash for this collection's manifest"""
        if self._manifest_locator and self.committed():
            # If the collection is already saved on the API server, and it's committed
            # then return API server's PDH response.
            return self._portable_data_hash
        else:
            stripped = self.portable_manifest_text().encode()
            return '{}+{}'.format(hashlib.md5(stripped).hexdigest(), len(stripped))

    @synchronized
    def subscribe(self, callback: ChangeCallback) -> None:
        """Set a notify callback for changes to this collection

        Arguments:

        * callback: arvados.collection.ChangeCallback --- The callable to
          call each time the collection is changed.
        """
        if self._callback is None:
            self._callback = callback
        else:
            raise errors.ArgumentError("A callback is already set on this collection.")

    @synchronized
    def unsubscribe(self) -> None:
        """Remove any notify callback set for changes to this collection"""
        if self._callback is not None:
            self._callback = None

    @synchronized
    def notify(
            self,
            event: ChangeType,
            collection: 'RichCollectionBase',
            name: str,
            item: CollectionItem,
    ) -> None:
        """Notify any subscribed callback about a change to this collection

        .. ATTENTION:: Internal
           This method is only meant to be used by other Collection methods.

        If a callback has been registered with `RichCollectionBase.subscribe`,
        it will be called with information about a change to this collection.
        Then this notification will be propagated to this collection's root.

        Arguments:

        * event: Literal[ADD, DEL, MOD, TOK] --- The type of modification to
          the collection.

        * collection: arvados.collection.RichCollectionBase --- The
          collection that was modified.

        * name: str --- The name of the file or stream within `collection` that
          was modified.

        * item: arvados.arvfile.ArvadosFile |
          arvados.collection.Subcollection --- The new contents at `name`
          within `collection`.
        """
        if self._callback:
            self._callback(event, collection, name, item)
        self.root_collection().notify(event, collection, name, item)

    @synchronized
    def __eq__(self, other: Any) -> bool:
        """Indicate whether this collection object is equal to another"""
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

    def __ne__(self, other: Any) -> bool:
        """Indicate whether this collection object is not equal to another"""
        return not self.__eq__(other)

    @synchronized
    def flush(self) -> None:
        """Upload any pending data to Keep"""
        for e in self.values():
            e.flush()


class Collection(RichCollectionBase):
    """Read and manipulate an Arvados collection

    This class provides a high-level interface to create, read, and update
    Arvados collections and their contents. Refer to the Arvados Python SDK
    cookbook for [an introduction to using the Collection class][cookbook].

    [cookbook]: https://doc.arvados.org/sdk/python/cookbook.html#working-with-collections
    """

    def __init__(self, manifest_locator_or_text: Optional[str]=None,
                 api_client: Optional['arvados.api_resources.ArvadosAPIClient']=None,
                 keep_client: Optional['arvados.keep.KeepClient']=None,
                 num_retries: int=10,
                 parent: Optional['Collection']=None,
                 apiconfig: Optional[Mapping[str, str]]=None,
                 block_manager: Optional['arvados.arvfile._BlockManager']=None,
                 replication_desired: Optional[int]=None,
                 storage_classes_desired: Optional[List[str]]=None,
                 put_threads: Optional[int]=None):
        """Initialize a Collection object

        Arguments:

        * manifest_locator_or_text: str | None --- This string can contain a
          collection manifest text, portable data hash, or UUID. When given a
          portable data hash or UUID, this instance will load a collection
          record from the API server. Otherwise, this instance will represent a
          new collection without an API server record. The default value `None`
          instantiates a new collection with an empty manifest.

        * api_client: arvados.api_resources.ArvadosAPIClient | None --- The
          Arvados API client object this instance uses to make requests. If
          none is given, this instance creates its own client using the
          settings from `apiconfig` (see below). If your client instantiates
          many Collection objects, you can help limit memory utilization by
          calling `arvados.api.api` to construct an
          `arvados.api.ThreadSafeAPIClient`, and use that as the `api_client`
          for every Collection.

        * keep_client: arvados.keep.KeepClient | None --- The Keep client
          object this instance uses to make requests. If none is given, this
          instance creates its own client using its `api_client`.

        * num_retries: int --- The number of times that client requests are
          retried. Default 10.

        * parent: arvados.collection.Collection | None --- The parent Collection
          object of this instance, if any. This argument is primarily used by
          other Collection methods; user client code shouldn't need to use it.

        * apiconfig: Mapping[str, str] | None --- A mapping with entries for
          `ARVADOS_API_HOST`, `ARVADOS_API_TOKEN`, and optionally
          `ARVADOS_API_HOST_INSECURE`. When no `api_client` is provided, the
          Collection object constructs one from these settings. If no
          mapping is provided, calls `arvados.config.settings` to get these
          parameters from user configuration.

        * block_manager: arvados.arvfile._BlockManager | None --- The
          _BlockManager object used by this instance to coordinate reading
          and writing Keep data blocks. If none is given, this instance
          constructs its own. This argument is primarily used by other
          Collection methods; user client code shouldn't need to use it.

        * replication_desired: int | None --- This controls both the value of
          the `replication_desired` field on API collection records saved by
          this class, as well as the number of Keep services that the object
          writes new data blocks to. If none is given, uses the default value
          configured for the cluster.

        * storage_classes_desired: list[str] | None --- This controls both
          the value of the `storage_classes_desired` field on API collection
          records saved by this class, as well as selecting which specific
          Keep services the object writes new data blocks to. If none is
          given, defaults to an empty list.

        * put_threads: int | None --- The number of threads to run
          simultaneously to upload data blocks to Keep. This value is used when
          building a new `block_manager`. It is unused when a `block_manager`
          is provided.
        """

        if storage_classes_desired and type(storage_classes_desired) is not list:
            raise errors.ArgumentError("storage_classes_desired must be list type.")

        super(Collection, self).__init__(parent)
        self._api_client = api_client
        self._keep_client = keep_client

        # Use the keep client from ThreadSafeAPIClient
        if self._keep_client is None and isinstance(self._api_client, ThreadSafeAPIClient):
            self._keep_client = self._api_client.keep

        self._block_manager = block_manager
        self.replication_desired = replication_desired
        self._storage_classes_desired = storage_classes_desired
        self.put_threads = put_threads

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

    def storage_classes_desired(self) -> List[str]:
        """Get this collection's `storage_classes_desired` value"""
        return self._storage_classes_desired or []

    def root_collection(self) -> 'Collection':
        return self

    def get_properties(self) -> Properties:
        """Get this collection's properties

        This method always returns a dict. If this collection object does not
        have an associated API record, or that record does not have any
        properties set, this method returns an empty dict.
        """
        if self._api_response and self._api_response["properties"]:
            return self._api_response["properties"]
        else:
            return {}

    def get_trash_at(self) -> Optional[datetime.datetime]:
        """Get this collection's `trash_at` field

        This method parses the `trash_at` field of the collection's API
        record and returns a datetime from it. If that field is not set, or
        this collection object does not have an associated API record,
        returns None.
        """
        if self._api_response and self._api_response["trash_at"]:
            try:
                return ciso8601.parse_datetime(self._api_response["trash_at"])
            except ValueError:
                return None
        else:
            return None

    def stream_name(self) -> str:
        return "."

    def writable(self) -> bool:
        return True

    @synchronized
    def known_past_version(
            self,
            modified_at_and_portable_data_hash: Tuple[Optional[str], Optional[str]]
    ) -> bool:
        """Indicate whether an API record for this collection has been seen before

        As this collection object loads records from the API server, it records
        their `modified_at` and `portable_data_hash` fields. This method accepts
        a 2-tuple with values for those fields, and returns `True` if the
        combination was previously loaded.
        """
        return modified_at_and_portable_data_hash in self._past_versions

    @synchronized
    @retry_method
    def update(
            self,
            other: Optional['Collection']=None,
            num_retries: Optional[int]=None,
    ) -> None:
        """Merge another collection's contents into this one

        This method compares the manifest of this collection instance with
        another, then updates this instance's manifest with changes from the
        other, renaming files to flag conflicts where necessary.

        When called without any arguments, this method reloads the collection's
        API record, and updates this instance with any changes that have
        appeared server-side. If this instance does not have a corresponding
        API record, this method raises `arvados.errors.ArgumentError`.

        Arguments:

        * other: arvados.collection.Collection | None --- The collection
          whose contents should be merged into this instance. When not
          provided, this method reloads this collection's API record and
          constructs a Collection object from it.  If this instance does not
          have a corresponding API record, this method raises
          `arvados.errors.ArgumentError`.

        * num_retries: int | None --- The number of times to retry reloading
          the collection's API record from the API server. If not specified,
          uses the `num_retries` provided when this instance was constructed.
        """
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
            self._api_client = ThreadSafeAPIClient(self._config, version='v1')
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
                                                storage_classes_func=self.storage_classes_desired)
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
        """Exit a context with this collection instance

        If no exception was raised inside the context block, and this
        collection is writable and has a corresponding API record, that
        record will be updated to match the state of this instance at the end
        of the block.
        """
        if exc_type is None:
            if self.writable() and self._has_collection_uuid():
                self.save()
        self.stop_threads()

    def stop_threads(self) -> None:
        """Stop background Keep upload/download threads"""
        if self._block_manager is not None:
            self._block_manager.stop_threads()

    @synchronized
    def manifest_locator(self) -> Optional[str]:
        """Get this collection's manifest locator, if any

        * If this collection instance is associated with an API record with a
          UUID, return that.
        * Otherwise, if this collection instance was loaded from an API record
          by portable data hash, return that.
        * Otherwise, return `None`.
        """
        return self._manifest_locator

    @synchronized
    def clone(
            self,
            new_parent: Optional['Collection']=None,
            new_name: Optional[str]=None,
            readonly: bool=False,
            new_config: Optional[Mapping[str, str]]=None,
    ) -> 'Collection':
        """Create a Collection object with the same contents as this instance

        This method creates a new Collection object with contents that match
        this instance's. The new collection will not be associated with any API
        record.

        Arguments:

        * new_parent: arvados.collection.Collection | None --- This value is
          passed to the new Collection's constructor as the `parent`
          argument.

        * new_name: str | None --- This value is unused.

        * readonly: bool --- If this value is true, this method constructs and
          returns a `CollectionReader`. Otherwise, it returns a mutable
          `Collection`. Default `False`.

        * new_config: Mapping[str, str] | None --- This value is passed to the
          new Collection's constructor as `apiconfig`. If no value is provided,
          defaults to the configuration passed to this instance's constructor.
        """
        if new_config is None:
            new_config = self._config
        if readonly:
            newcollection = CollectionReader(parent=new_parent, apiconfig=new_config)
        else:
            newcollection = Collection(parent=new_parent, apiconfig=new_config)

        newcollection._clonefrom(self)
        return newcollection

    @synchronized
    def api_response(self) -> Optional[Dict[str, Any]]:
        """Get this instance's associated API record

        If this Collection instance has an associated API record, return it.
        Otherwise, return `None`.
        """
        return self._api_response

    def find_or_create(
            self,
            path: str,
            create_type: CreateType,
    ) -> CollectionItem:
        if path == ".":
            return self
        else:
            return super(Collection, self).find_or_create(path[2:] if path.startswith("./") else path, create_type)

    def find(self, path: str) -> CollectionItem:
        if path == ".":
            return self
        else:
            return super(Collection, self).find(path[2:] if path.startswith("./") else path)

    def remove(self, path: str, recursive: bool=False) -> None:
        if path == ".":
            raise errors.ArgumentError("Cannot remove '.'")
        else:
            return super(Collection, self).remove(path[2:] if path.startswith("./") else path, recursive)

    @must_be_writable
    @synchronized
    @retry_method
    def save(
            self,
            properties: Optional[Properties]=None,
            storage_classes: Optional[StorageClasses]=None,
            trash_at: Optional[datetime.datetime]=None,
            merge: bool=True,
            num_retries: Optional[int]=None,
            preserve_version: bool=False,
    ) -> str:
        """Save collection to an existing API record

        This method updates the instance's corresponding API record to match
        the instance's state. If this instance does not have a corresponding API
        record yet, raises `AssertionError`. (To create a new API record, use
        `Collection.save_new`.) This method returns the saved collection
        manifest.

        Arguments:

        * properties: dict[str, Any] | None --- If provided, the API record will
          be updated with these properties. Note this will completely replace
          any existing properties.

        * storage_classes: list[str] | None --- If provided, the API record will
          be updated with this value in the `storage_classes_desired` field.
          This value will also be saved on the instance and used for any
          changes that follow.

        * trash_at: datetime.datetime | None --- If provided, the API record
          will be updated with this value in the `trash_at` field.

        * merge: bool --- If `True` (the default), this method will first
          reload this collection's API record, and merge any new contents into
          this instance before saving changes. See `Collection.update` for
          details.

        * num_retries: int | None --- The number of times to retry reloading
          the collection's API record from the API server. If not specified,
          uses the `num_retries` provided when this instance was constructed.

        * preserve_version: bool --- This value will be passed to directly
          to the underlying API call. If `True`, the Arvados API will
          preserve the versions of this collection both immediately before
          and after the update. If `True` when the API server is not
          configured with collection versioning, this method raises
          `arvados.errors.ArgumentError`.
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
    def save_new(
            self,
            name: Optional[str]=None,
            create_collection_record: bool=True,
            owner_uuid: Optional[str]=None,
            properties: Optional[Properties]=None,
            storage_classes: Optional[StorageClasses]=None,
            trash_at: Optional[datetime.datetime]=None,
            ensure_unique_name: bool=False,
            num_retries: Optional[int]=None,
            preserve_version: bool=False,
    ):
        """Save collection to a new API record

        This method finishes uploading new data blocks and (optionally)
        creates a new API collection record with the provided data. If a new
        record is created, this instance becomes associated with that record
        for future updates like `save()`. This method returns the saved
        collection manifest.

        Arguments:

        * name: str | None --- The `name` field to use on the new collection
          record. If not specified, a generic default name is generated.

        * create_collection_record: bool --- If `True` (the default), creates a
          collection record on the API server. If `False`, the method finishes
          all data uploads and only returns the resulting collection manifest
          without sending it to the API server.

        * owner_uuid: str | None --- The `owner_uuid` field to use on the
          new collection record.

        * properties: dict[str, Any] | None --- The `properties` field to use on
          the new collection record.

        * storage_classes: list[str] | None --- The
          `storage_classes_desired` field to use on the new collection record.

        * trash_at: datetime.datetime | None --- The `trash_at` field to use
          on the new collection record.

        * ensure_unique_name: bool --- This value is passed directly to the
          Arvados API when creating the collection record. If `True`, the API
          server may modify the submitted `name` to ensure the collection's
          `name`+`owner_uuid` combination is unique. If `False` (the default),
          if a collection already exists with this same `name`+`owner_uuid`
          combination, creating a collection record will raise a validation
          error.

        * num_retries: int | None --- The number of times to retry reloading
          the collection's API record from the API server. If not specified,
          uses the `num_retries` provided when this instance was constructed.

        * preserve_version: bool --- This value will be passed to directly
          to the underlying API call. If `True`, the Arvados API will
          preserve the versions of this collection both immediately before
          and after the update. If `True` when the API server is not
          configured with collection versioning, this method raises
          `arvados.errors.ArgumentError`.
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
        return re.sub(r'\\([0-3][0-7][0-7])', lambda m: chr(int(m.group(1), 8)), path)

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
                    blocks.append(streams.Range(tok, streamoffset, blocksize, 0))
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
    def notify(
            self,
            event: ChangeType,
            collection: 'RichCollectionBase',
            name: str,
            item: CollectionItem,
    ) -> None:
        if self._callback:
            self._callback(event, collection, name, item)


class Subcollection(RichCollectionBase):
    """Read and manipulate a stream/directory within an Arvados collection

    This class represents a single stream (like a directory) within an Arvados
    `Collection`. It is returned by `Collection.find` and provides the same API.
    Operations that work on the API collection record propagate to the parent
    `Collection` object.
    """

    def __init__(self, parent, name):
        super(Subcollection, self).__init__(parent)
        self.lock = self.root_collection().lock
        self._manifest_text = None
        self.name = name
        self.num_retries = parent.num_retries

    def root_collection(self) -> 'Collection':
        return self.parent.root_collection()

    def writable(self) -> bool:
        return self.root_collection().writable()

    def _my_api(self):
        return self.root_collection()._my_api()

    def _my_keep(self):
        return self.root_collection()._my_keep()

    def _my_block_manager(self):
        return self.root_collection()._my_block_manager()

    def stream_name(self) -> str:
        return os.path.join(self.parent.stream_name(), self.name)

    @synchronized
    def clone(
            self,
            new_parent: Optional['Collection']=None,
            new_name: Optional[str]=None,
    ) -> 'Subcollection':
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
                streams.escape(stream_name), config.EMPTY_BLOCK_LOCATOR)
        return super(Subcollection, self)._get_manifest_text(stream_name,
                                                             strip, normalize,
                                                             only_committed)


class CollectionReader(Collection):
    """Read-only `Collection` subclass

    This class will never create or update any API collection records. You can
    use this class for additional code safety when you only need to read
    existing collections.
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

    def writable(self) -> bool:
        return self._in_init
