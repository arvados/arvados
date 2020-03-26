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

"""

from __future__ import absolute_import
from __future__ import division
from future.utils import viewitems
from future.utils import native
from future.utils import listvalues
from future.utils import listitems
from future import standard_library
standard_library.install_aliases()
from builtins import next
from builtins import str
from builtins import object
import os
import sys
import llfuse
import errno
import stat
import threading
import arvados
import pprint
import arvados.events
import re
import apiclient
import json
import logging
import time
import _strptime
import calendar
import threading
import itertools
import ciso8601
import collections
import functools
import arvados.keep
from prometheus_client import Summary
import queue

# Default _notify_queue has a limit of 1000 items, but it really needs to be
# unlimited to avoid deadlocks, see https://arvados.org/issues/3198#note-43 for
# details.

if hasattr(llfuse, 'capi'):
    # llfuse < 0.42
    llfuse.capi._notify_queue = queue.Queue()
else:
    # llfuse >= 0.42
    llfuse._notify_queue = queue.Queue()

LLFUSE_VERSION_0 = llfuse.__version__.startswith('0')

from .fusedir import Directory, CollectionDirectory, TmpCollectionDirectory, MagicDirectory, TagsDirectory, ProjectDirectory, SharedDirectory, CollectionDirectoryBase
from .fusefile import StringFile, FuseArvadosFile

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

    def flush(self):
        pass


class FileHandle(Handle):
    """Connects a numeric file handle to a File  object that has
    been opened by the client."""

    def flush(self):
        if self.obj.writable():
            return self.obj.flush()


class DirectoryHandle(Handle):
    """Connects a numeric file handle to a Directory object that has
    been opened by the client."""

    def __init__(self, fh, dirobj, entries):
        super(DirectoryHandle, self).__init__(fh, dirobj)
        self.entries = entries


class InodeCache(object):
    """Records the memory footprint of objects and when they are last used.

    When the cache limit is exceeded, the least recently used objects are
    cleared.  Clearing the object means discarding its contents to release
    memory.  The next time the object is accessed, it must be re-fetched from
    the server.  Note that the inode cache limit is a soft limit; the cache
    limit may be exceeded if necessary to load very large objects, it may also
    be exceeded if open file handles prevent objects from being cleared.

    """

    def __init__(self, cap, min_entries=4):
        self._entries = collections.OrderedDict()
        self._by_uuid = {}
        self.cap = cap
        self._total = 0
        self.min_entries = min_entries

    def total(self):
        return self._total

    def _remove(self, obj, clear):
        if clear:
            # Kernel behavior seems to be that if a file is
            # referenced, its parents remain referenced too. This
            # means has_ref() exits early when a collection is not
            # candidate for eviction.
            #
            # By contrast, in_use() doesn't increment references on
            # parents, so it requires a full tree walk to determine if
            # a collection is a candidate for eviction.  This takes
            # .07s for 240000 files, which becomes a major drag when
            # cap_cache is being called several times a second and
            # there are multiple non-evictable collections in the
            # cache.
            #
            # So it is important for performance that we do the
            # has_ref() check first.

            if obj.has_ref(True):
                _logger.debug("InodeCache cannot clear inode %i, still referenced", obj.inode)
                return

            if obj.in_use():
                _logger.debug("InodeCache cannot clear inode %i, in use", obj.inode)
                return

            obj.kernel_invalidate()
            _logger.debug("InodeCache sent kernel invalidate inode %i", obj.inode)
            obj.clear()

        # The llfuse lock is released in del_entry(), which is called by
        # Directory.clear().  While the llfuse lock is released, it can happen
        # that a reentrant call removes this entry before this call gets to it.
        # Ensure that the entry is still valid before trying to remove it.
        if obj.inode not in self._entries:
            return

        self._total -= obj.cache_size
        del self._entries[obj.inode]
        if obj.cache_uuid:
            self._by_uuid[obj.cache_uuid].remove(obj)
            if not self._by_uuid[obj.cache_uuid]:
                del self._by_uuid[obj.cache_uuid]
            obj.cache_uuid = None
        if clear:
            _logger.debug("InodeCache cleared inode %i total now %i", obj.inode, self._total)

    def cap_cache(self):
        if self._total > self.cap:
            for ent in listvalues(self._entries):
                if self._total < self.cap or len(self._entries) < self.min_entries:
                    break
                self._remove(ent, True)

    def manage(self, obj):
        if obj.persisted():
            obj.cache_size = obj.objsize()
            self._entries[obj.inode] = obj
            obj.cache_uuid = obj.uuid()
            if obj.cache_uuid:
                if obj.cache_uuid not in self._by_uuid:
                    self._by_uuid[obj.cache_uuid] = [obj]
                else:
                    if obj not in self._by_uuid[obj.cache_uuid]:
                        self._by_uuid[obj.cache_uuid].append(obj)
            self._total += obj.objsize()
            _logger.debug("InodeCache touched inode %i (size %i) (uuid %s) total now %i (%i entries)",
                          obj.inode, obj.objsize(), obj.cache_uuid, self._total, len(self._entries))
            self.cap_cache()

    def touch(self, obj):
        if obj.persisted():
            if obj.inode in self._entries:
                self._remove(obj, False)
            self.manage(obj)

    def unmanage(self, obj):
        if obj.persisted() and obj.inode in self._entries:
            self._remove(obj, True)

    def find_by_uuid(self, uuid):
        return self._by_uuid.get(uuid, [])

    def clear(self):
        self._entries.clear()
        self._by_uuid.clear()
        self._total = 0

class Inodes(object):
    """Manage the set of inodes.  This is the mapping from a numeric id
    to a concrete File or Directory object"""

    def __init__(self, inode_cache, encoding="utf-8"):
        self._entries = {}
        self._counter = itertools.count(llfuse.ROOT_INODE)
        self.inode_cache = inode_cache
        self.encoding = encoding
        self.deferred_invalidations = []

    def __getitem__(self, item):
        return self._entries[item]

    def __setitem__(self, key, item):
        self._entries[key] = item

    def __iter__(self):
        return iter(self._entries.keys())

    def items(self):
        return viewitems(self._entries.items())

    def __contains__(self, k):
        return k in self._entries

    def touch(self, entry):
        entry._atime = time.time()
        self.inode_cache.touch(entry)

    def add_entry(self, entry):
        entry.inode = next(self._counter)
        if entry.inode == llfuse.ROOT_INODE:
            entry.inc_ref()
        self._entries[entry.inode] = entry
        self.inode_cache.manage(entry)
        return entry

    def del_entry(self, entry):
        if entry.ref_count == 0:
            self.inode_cache.unmanage(entry)
            del self._entries[entry.inode]
            with llfuse.lock_released:
                entry.finalize()
            entry.inode = None
        else:
            entry.dead = True
            _logger.debug("del_entry on inode %i with refcount %i", entry.inode, entry.ref_count)

    def invalidate_inode(self, entry):
        if entry.has_ref(False):
            # Only necessary if the kernel has previously done a lookup on this
            # inode and hasn't yet forgotten about it.
            llfuse.invalidate_inode(entry.inode)

    def invalidate_entry(self, entry, name):
        if entry.has_ref(False):
            # Only necessary if the kernel has previously done a lookup on this
            # inode and hasn't yet forgotten about it.
            llfuse.invalidate_entry(entry.inode, native(name.encode(self.encoding)))

    def clear(self):
        self.inode_cache.clear()

        for k,v in viewitems(self._entries):
            try:
                v.finalize()
            except Exception as e:
                _logger.exception("Error during finalize of inode %i", k)

        self._entries.clear()


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

    def __init__(self, uid, gid, api_client, encoding="utf-8", inode_cache=None, num_retries=4, enable_write=False):
        super(Operations, self).__init__()

        self._api_client = api_client

        if not inode_cache:
            inode_cache = InodeCache(cap=256*1024*1024)
        self.inodes = Inodes(inode_cache, encoding=encoding)
        self.uid = uid
        self.gid = gid
        self.enable_write = enable_write

        # dict of inode to filehandle
        self._filehandles = {}
        self._filehandles_counter = itertools.count(0)

        # Other threads that need to wait until the fuse driver
        # is fully initialized should wait() on this event object.
        self.initlock = threading.Event()

        # If we get overlapping shutdown events (e.g., fusermount -u
        # -z and operations.destroy()) llfuse calls forget() on inodes
        # that have already been deleted. To avoid this, we make
        # forget() a no-op if called after destroy().
        self._shutdown_started = threading.Event()

        self.num_retries = num_retries

        self.read_counter = arvados.keep.Counter()
        self.write_counter = arvados.keep.Counter()
        self.read_ops_counter = arvados.keep.Counter()
        self.write_ops_counter = arvados.keep.Counter()

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

    @destroy_time.time()
    @catch_exceptions
    def destroy(self):
        self._shutdown_started.set()
        if self.events:
            self.events.close()
            self.events = None

        # Different versions of llfuse require and forbid us to
        # acquire the lock here. See #8345#note-37, #10805#note-9.
        if LLFUSE_VERSION_0 and llfuse.lock.acquire():
            # llfuse < 0.42
            self.inodes.clear()
            llfuse.lock.release()
        else:
            # llfuse >= 0.42
            self.inodes.clear()

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

            for item in self.inodes.inode_cache.find_by_uuid(ev["object_uuid"]):
                item.invalidate()
                if ev.get("object_kind") == "arvados#collection":
                    pdh = new_attrs.get("portable_data_hash")
                    # new_attributes.modified_at currently lacks
                    # subsecond precision (see #6347) so use event_at
                    # which should always be the same.
                    stamp = ev.get("event_at")
                    if (stamp and pdh and item.writable() and
                        item.collection is not None and
                        item.collection.modified() and
                        new_attrs.get("is_trashed") is not True):
                        item.update(to_record_version=(stamp, pdh))

            oldowner = old_attrs.get("owner_uuid")
            newowner = ev.get("object_owner_uuid")
            for parent in (
                    self.inodes.inode_cache.find_by_uuid(oldowner) +
                    self.inodes.inode_cache.find_by_uuid(newowner)):
                parent.child_event(ev)

    @getattr_time.time()
    @catch_exceptions
    def getattr(self, inode, ctx=None):
        if inode not in self.inodes:
            raise llfuse.FUSEError(errno.ENOENT)

        e = self.inodes[inode]

        entry = llfuse.EntryAttributes()
        entry.st_ino = inode
        entry.generation = 0
        entry.entry_timeout = 0
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
        else:
            if parent_inode in self.inodes:
                p = self.inodes[parent_inode]
                self.inodes.touch(p)
                if name == '..':
                    inode = p.parent_inode
                elif isinstance(p, Directory) and name in p:
                    inode = p[name].inode

        if inode != None:
            _logger.debug("arv-mount lookup: parent_inode %i name '%s' inode %i",
                      parent_inode, name, inode)
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
            if ent.dec_ref(nlookup) == 0 and ent.dead:
                self.inodes.del_entry(ent)

    @open_time.time()
    @catch_exceptions
    def open(self, inode, flags, ctx=None):
        if inode in self.inodes:
            p = self.inodes[inode]
        else:
            raise llfuse.FUSEError(errno.ENOENT)

        if isinstance(p, Directory):
            raise llfuse.FUSEError(errno.EISDIR)

        if ((flags & os.O_WRONLY) or (flags & os.O_RDWR)) and not p.writable():
            raise llfuse.FUSEError(errno.EPERM)

        fh = next(self._filehandles_counter)
        self._filehandles[fh] = FileHandle(fh, p)
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
                self._filehandles[fh].flush()
            except Exception:
                raise
            finally:
                self._filehandles[fh].release()
                del self._filehandles[fh]
        self.inodes.inode_cache.cap_cache()

    def releasedir(self, fh):
        self.release(fh)

    @opendir_time.time()
    @catch_exceptions
    def opendir(self, inode, ctx=None):
        _logger.debug("arv-mount opendir: inode %i", inode)

        if inode in self.inodes:
            p = self.inodes[inode]
        else:
            raise llfuse.FUSEError(errno.ENOENT)

        if not isinstance(p, Directory):
            raise llfuse.FUSEError(errno.ENOTDIR)

        fh = next(self._filehandles_counter)
        if p.parent_inode in self.inodes:
            parent = self.inodes[p.parent_inode]
        else:
            raise llfuse.FUSEError(errno.EIO)

        # update atime
        self.inodes.touch(p)
        self._filehandles[fh] = DirectoryHandle(fh, p, [('.', p), ('..', parent)] + listitems(p))
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
            if handle.entries[e][1].inode in self.inodes:
                yield (handle.entries[e][0].encode(self.inodes.encoding), self.getattr(handle.entries[e][1].inode), e+1)
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
        self._filehandles[fh] = FileHandle(fh, f)
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
            self._filehandles[fh].flush()

    def fsync(self, fh, datasync):
        self.flush(fh)

    def fsyncdir(self, fh, datasync):
        self.flush(fh)
