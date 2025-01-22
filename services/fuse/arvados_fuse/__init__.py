# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

"""FUSE driver for Arvados Keep

Architecture:

There is one `Operations` object per mount point.  It is the entry point for all
read and write requests from the llfuse module.

The operations object owns an `Inodes` object.  The inodes object stores the
mapping from numeric inode (used throughout the file system API to uniquely
identify files) to the Python objects that implement files and directories.

The `Inodes` object owns an `InodeCache` object.  The inode cache records the
memory footprint of file system objects and when they are last used.  When the
cache limit is exceeded, the least recently used objects are cleared.

File system objects inherit from `fresh.FreshBase` which manages the object lifecycle.

File objects inherit from `fusefile.File`.  Key methods are `readfrom` and `writeto`
which implement actual reads and writes.

Directory objects inherit from `fusedir.Directory`.  The directory object wraps
a Python dict which stores the mapping from filenames to directory entries.
Directory contents can be accessed through the Python operators such as `[]`
and `in`.  These methods automatically check if the directory is fresh (up to
date) or stale (needs update) and will call `update` if necessary before
returing a result.

The general FUSE operation flow is as follows:

- The request handler is called with either an inode or file handle that is the
  subject of the operation.

- Look up the inode using the Inodes table or the file handle in the
  filehandles table to get the file system object.

- For methods that alter files or directories, check that the operation is
  valid and permitted using _check_writable().

- Call the relevant method on the file system object.

- Return the result.

The FUSE driver supports the Arvados event bus.  When an event is received for
an object that is live in the inode cache, that object is immediately updated.

Implementation note: in the code, the terms 'object', 'entry' and
'inode' are used somewhat interchangeably, but generally mean an
arvados_fuse.File or arvados_fuse.Directory object which has numeric
inode assigned to it and appears in the Inodes._entries dictionary.

"""

import os
import llfuse
import errno
import stat
import threading
import arvados
import arvados.events
import logging
import time
import threading
import itertools
import collections
import functools
import arvados.keep
from prometheus_client import Summary
import queue
from dataclasses import dataclass
import typing

from .fusedir import Directory, CollectionDirectory, TmpCollectionDirectory, MagicDirectory, TagsDirectory, ProjectDirectory, SharedDirectory, CollectionDirectoryBase
from .fusefile import File, StringFile, FuseArvadosFile

_logger = logging.getLogger('arvados.arvados_fuse')

# Uncomment this to enable llfuse debug logging.
# log_handler = logging.StreamHandler()
# llogger = logging.getLogger('llfuse')
# llogger.addHandler(log_handler)
# llogger.setLevel(logging.DEBUG)

class Handle(object):
    """Connects a numeric file handle to a File or Directory object that has
    been opened by the client."""

    def __init__(self, fh, obj):
        self.fh = fh
        self.obj = obj
        self.obj.inc_use()

    def release(self):
        self.obj.dec_use()

    def flush(self, force):
        pass


class FileHandle(Handle):
    """Connects a numeric file handle to a File  object that has
    been opened by the client."""

    def __init__(self, fh, obj, parent_obj, open_for_writing):
        super(FileHandle, self).__init__(fh, obj)
        self.parent_obj = parent_obj
        if self.parent_obj is not None:
            self.parent_obj.inc_use()
        self.open_for_writing = open_for_writing

    def release(self):
        super(FileHandle, self).release()
        if self.parent_obj is not None:
            self.parent_obj.dec_use()

    def flush(self, force):
        if not self.open_for_writing and not force:
            return
        self.obj.flush()
        if self.parent_obj is not None:
            self.parent_obj.flush()

class DirectoryHandle(Handle):
    """Connects a numeric file handle to a Directory object that has
    been opened by the client.

    DirectoryHandle is used by opendir() and readdir() to get
    directory listings.  Entries returned by readdir() don't increment
    the lookup count (kernel references), so increment our internal
    "use count" to avoid having an item being removed mid-read.

    """

    def __init__(self, fh, dirobj, entries):
        super(DirectoryHandle, self).__init__(fh, dirobj)
        self.entries = entries

        for ent in self.entries:
            ent[1].inc_use()

    def release(self):
        for ent in self.entries:
            ent[1].dec_use()
        super(DirectoryHandle, self).release()

    def flush(self, force):
        self.obj.flush()


class InodeCache(object):
    """Records the memory footprint of objects and when they are last used.

    When the cache limit is exceeded, the least recently used objects
    are cleared.  Clearing the object means discarding its contents to
    release memory.  The next time the object is accessed, it must be
    re-fetched from the server.  Note that the inode cache limit is a
    soft limit; the cache limit may be exceeded if necessary to load
    very large projects or collections, it may also be exceeded if an
    inode can't be safely discarded based on kernel lookups
    (has_ref()) or internal use count (in_use()).

    """

    def __init__(self, cap, min_entries=4):
        # Standard dictionaries are ordered, but OrderedDict is still better here, see
        # https://docs.python.org/3.11/library/collections.html#ordereddict-objects
        # specifically we use move_to_end() which standard dicts don't have.
        self._cache_entries = collections.OrderedDict()
        self.cap = cap
        self._total = 0
        self.min_entries = min_entries

    def total(self):
        return self._total

    def evict_candidates(self):
        """Yield entries that are candidates to be evicted
        and stop when the cache total has shrunk sufficiently.

        Implements a LRU cache, when an item is added or touch()ed it
        goes to the back of the OrderedDict, so items in the front are
        oldest.  The Inodes._remove() function determines if the entry
        can actually be removed safely.

        """

        if self._total <= self.cap:
            return

        _logger.debug("InodeCache evict_candidates total %i cap %i entries %i", self._total, self.cap, len(self._cache_entries))

        # Copy this into a deque for two reasons:
        #
        # 1. _cache_entries is modified by unmanage() which is called
        # by _remove
        #
        # 2. popping off the front means the reference goes away
        # immediately intead of sticking around for the lifetime of
        # "values"
        values = collections.deque(self._cache_entries.values())

        while values:
            if self._total < self.cap or len(self._cache_entries) < self.min_entries:
                break
            yield values.popleft()

    def unmanage(self, entry):
        """Stop managing an object in the cache.

        This happens when an object is being removed from the inode
        entries table.

        """

        if entry.inode not in self._cache_entries:
            return

        # manage cache size running sum
        self._total -= entry.cache_size
        entry.cache_size = 0

        # Now forget about it
        del self._cache_entries[entry.inode]

    def update_cache_size(self, obj):
        """Update the cache total in response to the footprint of an
        object changing (usually because it has been loaded or
        cleared).

        Adds or removes entries to the cache list based on the object
        cache size.

        """

        if not obj.persisted():
            return

        if obj.inode in self._cache_entries:
            self._total -= obj.cache_size

        obj.cache_size = obj.objsize()

        if obj.cache_size > 0 or obj.parent_inode is None:
            self._total += obj.cache_size
            self._cache_entries[obj.inode] = obj
        elif obj.cache_size == 0 and obj.inode in self._cache_entries:
            del self._cache_entries[obj.inode]

    def touch(self, obj):
        """Indicate an object was used recently, making it low
        priority to be removed from the cache.

        """
        if obj.inode in self._cache_entries:
            self._cache_entries.move_to_end(obj.inode)
            return True
        return False

    def clear(self):
        self._cache_entries.clear()
        self._total = 0

@dataclass
class RemoveInode:
    entry: typing.Union[Directory, File]
    def inode_op(self, inodes, locked_ops):
        if locked_ops is None:
            inodes._remove(self.entry)
            return True
        else:
            locked_ops.append(self)
            return False

@dataclass
class InvalidateInode:
    inode: int
    def inode_op(self, inodes, locked_ops):
        llfuse.invalidate_inode(self.inode)
        return True

@dataclass
class InvalidateEntry:
    inode: int
    name: str
    def inode_op(self, inodes, locked_ops):
        llfuse.invalidate_entry(self.inode, self.name)
        return True

@dataclass
class EvictCandidates:
    def inode_op(self, inodes, locked_ops):
        return True


class Inodes(object):
    """Manage the set of inodes.

    This is the mapping from a numeric id to a concrete File or
    Directory object

    """

    def __init__(self, inode_cache, encoding="utf-8", fsns=None, shutdown_started=None):
        self._entries = {}
        self._counter = itertools.count(llfuse.ROOT_INODE)
        self.inode_cache = inode_cache
        self.encoding = encoding
        self._fsns = fsns
        self._shutdown_started = shutdown_started or threading.Event()

        self._inode_remove_queue = queue.Queue()
        self._inode_remove_thread = threading.Thread(None, self._inode_remove)
        self._inode_remove_thread.daemon = True
        self._inode_remove_thread.start()

        self._by_uuid = collections.defaultdict(list)

    def __getitem__(self, item):
        return self._entries[item]

    def __setitem__(self, key, item):
        self._entries[key] = item

    def __iter__(self):
        return iter(self._entries.keys())

    def items(self):
        return self._entries.items()

    def __contains__(self, k):
        return k in self._entries

    def touch(self, entry):
        """Update the access time, adjust the cache position, and
        notify the _inode_remove thread to recheck the cache.

        """

        entry._atime = time.time()
        if self.inode_cache.touch(entry):
            self.cap_cache()

    def cap_cache(self):
        """Notify the _inode_remove thread to recheck the cache."""
        if self._inode_remove_queue.empty():
            self._inode_remove_queue.put(EvictCandidates())

    def update_uuid(self, entry):
        """Update the Arvados uuid associated with an inode entry.

        This is used to look up inodes that need to be invalidated
        when a websocket event indicates the object has changed on the
        API server.

        """
        if entry.cache_uuid and entry in self._by_uuid[entry.cache_uuid]:
            self._by_uuid[entry.cache_uuid].remove(entry)

        entry.cache_uuid = entry.uuid()
        if entry.cache_uuid and entry not in self._by_uuid[entry.cache_uuid]:
            self._by_uuid[entry.cache_uuid].append(entry)

        if not self._by_uuid[entry.cache_uuid]:
            del self._by_uuid[entry.cache_uuid]

    def add_entry(self, entry):
        """Assign a numeric inode to a new entry."""

        entry.inode = next(self._counter)
        if entry.inode == llfuse.ROOT_INODE:
            entry.inc_ref()
        self._entries[entry.inode] = entry

        self.update_uuid(entry)
        self.inode_cache.update_cache_size(entry)
        self.cap_cache()
        return entry

    def del_entry(self, entry):
        """Remove entry from the inode table.

        Indicate this inode entry is pending deletion by setting
        parent_inode to None.  Notify the _inode_remove thread to try
        and remove it.

        """

        entry.parent_inode = None
        self._inode_remove_queue.put(RemoveInode(entry))
        _logger.debug("del_entry on inode %i with refcount %i", entry.inode, entry.ref_count)

    def _inode_remove(self):
        """Background thread to handle tasks related to invalidating
        inodes in the kernel, and removing objects from the inodes
        table entirely.

        """

        locked_ops = collections.deque()
        shutting_down = False
        while not shutting_down:
            tasks_done = 0
            blocking_get = True
            while True:
                try:
                    qentry = self._inode_remove_queue.get(blocking_get)
                except queue.Empty:
                    break

                blocking_get = False
                if qentry is None:
                    shutting_down = True
                    continue

                # Process (or defer) this entry
                qentry.inode_op(self, locked_ops)
                tasks_done += 1

                # Give up the reference
                qentry = None

            with llfuse.lock:
                while locked_ops:
                    locked_ops.popleft().inode_op(self, None)
                for entry in self.inode_cache.evict_candidates():
                    self._remove(entry)

            # Unblock _inode_remove_queue.join() only when all of the
            # deferred work is done, i.e., after calling inode_op()
            # and then evict_candidates().
            for _ in range(tasks_done):
                self._inode_remove_queue.task_done()

    def wait_remove_queue_empty(self):
        # used by tests
        self._inode_remove_queue.join()

    def _remove(self, entry):
        """Remove an inode entry if possible.

        If the entry is still referenced or in use, don't do anything.
        If this is not referenced but the parent is still referenced,
        clear any data held by the object (which may include directory
        entries under the object) but don't remove it from the inode
        table.

        """
        try:
            if entry.inode is None:
                # Removed already
                return

            if entry.inode == llfuse.ROOT_INODE:
                return

            if entry.in_use():
                # referenced internally, stay pinned
                #_logger.debug("InodeCache cannot clear inode %i, in use", entry.inode)
                return

            # Tell the kernel it should forget about it
            entry.kernel_invalidate()

            if entry.has_ref():
                # has kernel reference, could still be accessed.
                # when the kernel forgets about it, we can delete it.
                #_logger.debug("InodeCache cannot clear inode %i, is referenced", entry.inode)
                return

            # commit any pending changes
            with llfuse.lock_released:
                entry.finalize()

            # Clear the contents
            entry.clear()

            if entry.parent_inode is None:
                _logger.debug("InodeCache forgetting inode %i, object cache_size %i, cache total %i, forget_inode True, inode entries %i, type %s",
                              entry.inode, entry.cache_size, self.inode_cache.total(),
                              len(self._entries), type(entry))

                if entry.cache_uuid:
                    self._by_uuid[entry.cache_uuid].remove(entry)
                    if not self._by_uuid[entry.cache_uuid]:
                        del self._by_uuid[entry.cache_uuid]
                    entry.cache_uuid = None

                self.inode_cache.unmanage(entry)

                del self._entries[entry.inode]
                entry.inode = None

        except Exception as e:
            _logger.exception("failed remove")

    def invalidate_inode(self, entry):
        if entry.has_ref():
            # Only necessary if the kernel has previously done a lookup on this
            # inode and hasn't yet forgotten about it.
            self._inode_remove_queue.put(InvalidateInode(entry.inode))

    def invalidate_entry(self, entry, name):
        if entry.has_ref():
            # Only necessary if the kernel has previously done a lookup on this
            # inode and hasn't yet forgotten about it.
            self._inode_remove_queue.put(InvalidateEntry(entry.inode, name.encode(self.encoding)))

    def begin_shutdown(self):
        self._inode_remove_queue.put(None)
        if self._inode_remove_thread is not None:
            self._inode_remove_thread.join()
        self._inode_remove_thread = None

    def clear(self):
        with llfuse.lock_released:
            self.begin_shutdown()

        self.inode_cache.clear()
        self._by_uuid.clear()

        for k,v in self._entries.items():
            try:
                v.finalize()
            except Exception as e:
                _logger.exception("Error during finalize of inode %i", k)

        self._entries.clear()

    def forward_slash_subst(self):
        return self._fsns

    def find_by_uuid(self, uuid):
        """Return a list of zero or more inode entries corresponding
        to this Arvados UUID."""
        return self._by_uuid.get(uuid, [])


def catch_exceptions(orig_func):
    """Catch uncaught exceptions and log them consistently."""

    @functools.wraps(orig_func)
    def catch_exceptions_wrapper(self, *args, **kwargs):
        try:
            return orig_func(self, *args, **kwargs)
        except llfuse.FUSEError:
            raise
        except EnvironmentError as e:
            raise llfuse.FUSEError(e.errno)
        except NotImplementedError:
            raise llfuse.FUSEError(errno.ENOTSUP)
        except arvados.errors.KeepWriteError as e:
            _logger.error("Keep write error: " + str(e))
            raise llfuse.FUSEError(errno.EIO)
        except arvados.errors.NotFoundError as e:
            _logger.error("Block not found error: " + str(e))
            raise llfuse.FUSEError(errno.EIO)
        except:
            _logger.exception("Unhandled exception during FUSE operation")
            raise llfuse.FUSEError(errno.EIO)

    return catch_exceptions_wrapper


class Operations(llfuse.Operations):
    """This is the main interface with llfuse.

    The methods on this object are called by llfuse threads to service FUSE
    events to query and read from the file system.

    llfuse has its own global lock which is acquired before calling a request handler,
    so request handlers do not run concurrently unless the lock is explicitly released
    using 'with llfuse.lock_released:'

    """

    fuse_time = Summary('arvmount_fuse_operations_seconds', 'Time spent during FUSE operations', labelnames=['op'])
    read_time = fuse_time.labels(op='read')
    write_time = fuse_time.labels(op='write')
    destroy_time = fuse_time.labels(op='destroy')
    on_event_time = fuse_time.labels(op='on_event')
    getattr_time = fuse_time.labels(op='getattr')
    setattr_time = fuse_time.labels(op='setattr')
    lookup_time = fuse_time.labels(op='lookup')
    forget_time = fuse_time.labels(op='forget')
    open_time = fuse_time.labels(op='open')
    release_time = fuse_time.labels(op='release')
    opendir_time = fuse_time.labels(op='opendir')
    readdir_time = fuse_time.labels(op='readdir')
    statfs_time = fuse_time.labels(op='statfs')
    create_time = fuse_time.labels(op='create')
    mkdir_time = fuse_time.labels(op='mkdir')
    unlink_time = fuse_time.labels(op='unlink')
    rmdir_time = fuse_time.labels(op='rmdir')
    rename_time = fuse_time.labels(op='rename')
    flush_time = fuse_time.labels(op='flush')

    def __init__(self, uid, gid, api_client, encoding="utf-8", inode_cache=None, num_retries=4, enable_write=False, fsns=None):
        super(Operations, self).__init__()

        self._api_client = api_client

        if not inode_cache:
            inode_cache = InodeCache(cap=256*1024*1024)

        if fsns is None:
            try:
                fsns = self._api_client.config()["Collections"]["ForwardSlashNameSubstitution"]
            except KeyError:
                # old API server with no FSNS config
                fsns = '_'
            else:
                if fsns == '' or fsns == '/':
                    fsns = None

        # If we get overlapping shutdown events (e.g., fusermount -u
        # -z and operations.destroy()) llfuse calls forget() on inodes
        # that have already been deleted. To avoid this, we make
        # forget() a no-op if called after destroy().
        self._shutdown_started = threading.Event()

        self.inodes = Inodes(inode_cache, encoding=encoding, fsns=fsns,
                             shutdown_started=self._shutdown_started)
        self.uid = uid
        self.gid = gid
        self.enable_write = enable_write

        # dict of inode to filehandle
        self._filehandles = {}
        self._filehandles_counter = itertools.count(0)

        # Other threads that need to wait until the fuse driver
        # is fully initialized should wait() on this event object.
        self.initlock = threading.Event()

        self.num_retries = num_retries

        self.read_counter = arvados.keep._Counter()
        self.write_counter = arvados.keep._Counter()
        self.read_ops_counter = arvados.keep._Counter()
        self.write_ops_counter = arvados.keep._Counter()

        self.events = None

    def init(self):
        # Allow threads that are waiting for the driver to be finished
        # initializing to continue
        self.initlock.set()

    def metric_samples(self):
        return self.fuse_time.collect()[0].samples

    def metric_op_names(self):
        ops = []
        for cur_op in [sample.labels['op'] for sample in self.metric_samples()]:
            if cur_op not in ops:
                ops.append(cur_op)
        return ops

    def metric_value(self, opname, metric):
        op_value = [sample.value for sample in self.metric_samples()
                    if sample.name == metric and sample.labels['op'] == opname]
        return op_value[0] if len(op_value) == 1 else None

    def metric_sum_func(self, opname):
        return lambda: self.metric_value(opname, "arvmount_fuse_operations_seconds_sum")

    def metric_count_func(self, opname):
        return lambda: int(self.metric_value(opname, "arvmount_fuse_operations_seconds_count"))

    def begin_shutdown(self):
        self._shutdown_started.set()
        self.inodes.begin_shutdown()

    @destroy_time.time()
    @catch_exceptions
    def destroy(self):
        _logger.debug("arv-mount destroy: start")

        with llfuse.lock_released:
            self.begin_shutdown()

        if self.events:
            self.events.close()
            self.events = None

        self.inodes.clear()

        _logger.debug("arv-mount destroy: complete")


    def access(self, inode, mode, ctx):
        return True

    def listen_for_events(self):
        self.events = arvados.events.subscribe(
            self._api_client,
            [["event_type", "in", ["create", "update", "delete"]]],
            self.on_event)

    @on_event_time.time()
    @catch_exceptions
    def on_event(self, ev):
        if 'event_type' not in ev or ev["event_type"] not in ("create", "update", "delete"):
            return
        with llfuse.lock:
            properties = ev.get("properties") or {}
            old_attrs = properties.get("old_attributes") or {}
            new_attrs = properties.get("new_attributes") or {}

            for item in self.inodes.find_by_uuid(ev["object_uuid"]):
                item.invalidate()

            oldowner = old_attrs.get("owner_uuid")
            newowner = ev.get("object_owner_uuid")
            for parent in (
                    self.inodes.find_by_uuid(oldowner) +
                    self.inodes.find_by_uuid(newowner)):
                parent.invalidate()

    @getattr_time.time()
    @catch_exceptions
    def getattr(self, inode, ctx=None):
        if inode not in self.inodes:
            _logger.debug("arv-mount getattr: inode %i missing", inode)
            raise llfuse.FUSEError(errno.ENOENT)

        e = self.inodes[inode]
        self.inodes.touch(e)
        parent = None
        if e.parent_inode:
            parent = self.inodes[e.parent_inode]
            self.inodes.touch(parent)

        entry = llfuse.EntryAttributes()
        entry.st_ino = inode
        entry.generation = 0
        entry.entry_timeout = parent.time_to_next_poll() if parent is not None else 0
        entry.attr_timeout = e.time_to_next_poll() if e.allow_attr_cache else 0

        entry.st_mode = stat.S_IRUSR | stat.S_IRGRP | stat.S_IROTH
        if isinstance(e, Directory):
            entry.st_mode |= stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH | stat.S_IFDIR
        else:
            entry.st_mode |= stat.S_IFREG
            if isinstance(e, FuseArvadosFile):
                entry.st_mode |= stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH

        if self.enable_write and e.writable():
            entry.st_mode |= stat.S_IWUSR | stat.S_IWGRP | stat.S_IWOTH

        entry.st_nlink = 1
        entry.st_uid = self.uid
        entry.st_gid = self.gid
        entry.st_rdev = 0

        entry.st_size = e.size()

        entry.st_blksize = 512
        entry.st_blocks = (entry.st_size // 512) + 1
        if hasattr(entry, 'st_atime_ns'):
            # llfuse >= 0.42
            entry.st_atime_ns = int(e.atime() * 1000000000)
            entry.st_mtime_ns = int(e.mtime() * 1000000000)
            entry.st_ctime_ns = int(e.mtime() * 1000000000)
        else:
            # llfuse < 0.42
            entry.st_atime = int(e.atime)
            entry.st_mtime = int(e.mtime)
            entry.st_ctime = int(e.mtime)

        return entry

    @setattr_time.time()
    @catch_exceptions
    def setattr(self, inode, attr, fields=None, fh=None, ctx=None):
        entry = self.getattr(inode)

        if fh is not None and fh in self._filehandles:
            handle = self._filehandles[fh]
            e = handle.obj
        else:
            e = self.inodes[inode]

        if fields is None:
            # llfuse < 0.42
            update_size = attr.st_size is not None
        else:
            # llfuse >= 0.42
            update_size = fields.update_size
        if update_size and isinstance(e, FuseArvadosFile):
            with llfuse.lock_released:
                e.arvfile.truncate(attr.st_size)
                entry.st_size = e.arvfile.size()

        return entry

    @lookup_time.time()
    @catch_exceptions
    def lookup(self, parent_inode, name, ctx=None):
        name = str(name, self.inodes.encoding)
        inode = None

        if name == '.':
            inode = parent_inode
        elif parent_inode in self.inodes:
            p = self.inodes[parent_inode]
            self.inodes.touch(p)
            if name == '..':
                inode = p.parent_inode
            elif isinstance(p, Directory) and name in p:
                if p[name].inode is None:
                    _logger.debug("arv-mount lookup: parent_inode %i name '%s' found but inode was None",
                                  parent_inode, name)
                    raise llfuse.FUSEError(errno.ENOENT)

                inode = p[name].inode

        if inode != None:
            _logger.debug("arv-mount lookup: parent_inode %i name '%s' inode %i",
                      parent_inode, name, inode)
            self.inodes.touch(self.inodes[inode])
            self.inodes[inode].inc_ref()
            return self.getattr(inode)
        else:
            _logger.debug("arv-mount lookup: parent_inode %i name '%s' not found",
                      parent_inode, name)
            raise llfuse.FUSEError(errno.ENOENT)

    @forget_time.time()
    @catch_exceptions
    def forget(self, inodes):
        if self._shutdown_started.is_set():
            return
        for inode, nlookup in inodes:
            ent = self.inodes[inode]
            _logger.debug("arv-mount forget: inode %i nlookup %i ref_count %i", inode, nlookup, ent.ref_count)
            if ent.dec_ref(nlookup) == 0 and ent.parent_inode is None:
                self.inodes.del_entry(ent)

    @open_time.time()
    @catch_exceptions
    def open(self, inode, flags, ctx=None):
        if inode in self.inodes:
            p = self.inodes[inode]
        else:
            _logger.debug("arv-mount open: inode %i missing", inode)
            raise llfuse.FUSEError(errno.ENOENT)

        if isinstance(p, Directory):
            raise llfuse.FUSEError(errno.EISDIR)

        open_for_writing = (flags & os.O_WRONLY) or (flags & os.O_RDWR)
        if open_for_writing and not p.writable():
            raise llfuse.FUSEError(errno.EPERM)

        fh = next(self._filehandles_counter)

        if p.stale():
            p.checkupdate()
            self.inodes.invalidate_inode(p)

        parent_inode = self.inodes[p.parent_inode] if p.parent_inode in self.inodes else None
        self._filehandles[fh] = FileHandle(fh, p, parent_inode, open_for_writing)
        self.inodes.touch(p)

        # Normally, we will have received an "update" event if the
        # parent collection is stale here. However, even if the parent
        # collection hasn't changed, the manifest might have been
        # fetched so long ago that the signatures on the data block
        # locators have expired. Calling checkupdate() on all
        # ancestors ensures the signatures will be refreshed if
        # necessary.
        while p.parent_inode in self.inodes:
            if p == self.inodes[p.parent_inode]:
                break
            p = self.inodes[p.parent_inode]
            self.inodes.touch(p)
            p.checkupdate()

        _logger.debug("arv-mount open inode %i flags %x fh %i", inode, flags, fh)

        return fh

    @read_time.time()
    @catch_exceptions
    def read(self, fh, off, size):
        _logger.debug("arv-mount read fh %i off %i size %i", fh, off, size)
        self.read_ops_counter.add(1)

        if fh in self._filehandles:
            handle = self._filehandles[fh]
        else:
            raise llfuse.FUSEError(errno.EBADF)

        self.inodes.touch(handle.obj)

        r = handle.obj.readfrom(off, size, self.num_retries)
        if r:
            self.read_counter.add(len(r))
        return r

    @write_time.time()
    @catch_exceptions
    def write(self, fh, off, buf):
        _logger.debug("arv-mount write %i %i %i", fh, off, len(buf))
        self.write_ops_counter.add(1)

        if fh in self._filehandles:
            handle = self._filehandles[fh]
        else:
            raise llfuse.FUSEError(errno.EBADF)

        if not handle.obj.writable():
            raise llfuse.FUSEError(errno.EPERM)

        self.inodes.touch(handle.obj)

        w = handle.obj.writeto(off, buf, self.num_retries)
        if w:
            self.write_counter.add(w)
        return w

    @release_time.time()
    @catch_exceptions
    def release(self, fh):
        if fh in self._filehandles:
            _logger.debug("arv-mount release fh %i", fh)
            try:
                self._filehandles[fh].flush(False)
            except Exception:
                raise
            finally:
                self._filehandles[fh].release()
                del self._filehandles[fh]
        self.inodes.cap_cache()

    def releasedir(self, fh):
        self.release(fh)

    @opendir_time.time()
    @catch_exceptions
    def opendir(self, inode, ctx=None):
        _logger.debug("arv-mount opendir: inode %i", inode)

        if inode in self.inodes:
            p = self.inodes[inode]
        else:
            _logger.debug("arv-mount opendir: called with unknown or removed inode %i", inode)
            raise llfuse.FUSEError(errno.ENOENT)

        if not isinstance(p, Directory):
            raise llfuse.FUSEError(errno.ENOTDIR)

        fh = next(self._filehandles_counter)
        if p.parent_inode in self.inodes:
            parent = self.inodes[p.parent_inode]
        else:
            _logger.warning("arv-mount opendir: parent inode %i of %i is missing", p.parent_inode, inode)
            raise llfuse.FUSEError(errno.EIO)

        _logger.debug("arv-mount opendir: inode %i fh %i ", inode, fh)

        # update atime
        p.inc_use()
        self._filehandles[fh] = DirectoryHandle(fh, p, [('.', p), ('..', parent)] + p.items())
        p.dec_use()
        self.inodes.touch(p)
        return fh

    @readdir_time.time()
    @catch_exceptions
    def readdir(self, fh, off):
        _logger.debug("arv-mount readdir: fh %i off %i", fh, off)

        if fh in self._filehandles:
            handle = self._filehandles[fh]
        else:
            raise llfuse.FUSEError(errno.EBADF)

        e = off
        while e < len(handle.entries):
            ent = handle.entries[e]
            if ent[1].inode in self.inodes:
                yield (ent[0].encode(self.inodes.encoding), self.getattr(ent[1].inode), e+1)
            e += 1

    @statfs_time.time()
    @catch_exceptions
    def statfs(self, ctx=None):
        st = llfuse.StatvfsData()
        st.f_bsize = 128 * 1024
        st.f_blocks = 0
        st.f_files = 0

        st.f_bfree = 0
        st.f_bavail = 0

        st.f_ffree = 0
        st.f_favail = 0

        st.f_frsize = 0
        return st

    def _check_writable(self, inode_parent):
        if not self.enable_write:
            raise llfuse.FUSEError(errno.EROFS)

        if inode_parent in self.inodes:
            p = self.inodes[inode_parent]
        else:
            raise llfuse.FUSEError(errno.ENOENT)

        if not isinstance(p, Directory):
            raise llfuse.FUSEError(errno.ENOTDIR)

        if not p.writable():
            raise llfuse.FUSEError(errno.EPERM)

        return p

    @create_time.time()
    @catch_exceptions
    def create(self, inode_parent, name, mode, flags, ctx=None):
        name = name.decode(encoding=self.inodes.encoding)
        _logger.debug("arv-mount create: parent_inode %i '%s' %o", inode_parent, name, mode)

        p = self._check_writable(inode_parent)
        p.create(name)

        # The file entry should have been implicitly created by callback.
        f = p[name]
        fh = next(self._filehandles_counter)
        self._filehandles[fh] = FileHandle(fh, f, p, True)
        self.inodes.touch(p)

        f.inc_ref()
        return (fh, self.getattr(f.inode))

    @mkdir_time.time()
    @catch_exceptions
    def mkdir(self, inode_parent, name, mode, ctx=None):
        name = name.decode(encoding=self.inodes.encoding)
        _logger.debug("arv-mount mkdir: parent_inode %i '%s' %o", inode_parent, name, mode)

        p = self._check_writable(inode_parent)
        p.mkdir(name)

        # The dir entry should have been implicitly created by callback.
        d = p[name]

        d.inc_ref()
        return self.getattr(d.inode)

    @unlink_time.time()
    @catch_exceptions
    def unlink(self, inode_parent, name, ctx=None):
        name = name.decode(encoding=self.inodes.encoding)
        _logger.debug("arv-mount unlink: parent_inode %i '%s'", inode_parent, name)
        p = self._check_writable(inode_parent)
        p.unlink(name)

    @rmdir_time.time()
    @catch_exceptions
    def rmdir(self, inode_parent, name, ctx=None):
        name = name.decode(encoding=self.inodes.encoding)
        _logger.debug("arv-mount rmdir: parent_inode %i '%s'", inode_parent, name)
        p = self._check_writable(inode_parent)
        p.rmdir(name)

    @rename_time.time()
    @catch_exceptions
    def rename(self, inode_parent_old, name_old, inode_parent_new, name_new, ctx=None):
        name_old = name_old.decode(encoding=self.inodes.encoding)
        name_new = name_new.decode(encoding=self.inodes.encoding)
        _logger.debug("arv-mount rename: old_parent_inode %i '%s' new_parent_inode %i '%s'", inode_parent_old, name_old, inode_parent_new, name_new)
        src = self._check_writable(inode_parent_old)
        dest = self._check_writable(inode_parent_new)
        dest.rename(name_old, name_new, src)

    @flush_time.time()
    @catch_exceptions
    def flush(self, fh):
        if fh in self._filehandles:
            self._filehandles[fh].flush(False)

    def fsync(self, fh, datasync):
        if fh in self._filehandles:
            self._filehandles[fh].flush(True)
            self.inodes.invalidate_inode(self._filehandles[fh].obj)

    def fsyncdir(self, fh, datasync):
        if fh in self._filehandles:
            self._filehandles[fh].flush(True)

    @catch_exceptions
    def mknod(self, parent_inode, name, mode, rdev, ctx=None):
        if not stat.S_ISREG(mode):
            # Can only be used to create regular files.
            raise NotImplementedError()

        name = name.decode(encoding=self.inodes.encoding)
        _logger.debug("arv-mount mknod: parent_inode %i '%s' %o", parent_inode, name, mode)

        p = self._check_writable(parent_inode)
        p.create(name)

        # The file entry should have been implicitly created by callback.
        f = p[name]
        self.inodes.touch(p)

        f.inc_ref()
        return self.getattr(f.inode)
