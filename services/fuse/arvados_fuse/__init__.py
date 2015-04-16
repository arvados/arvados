#
# FUSE driver for Arvados Keep
#

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

from fusedir import sanitize_filename, Directory, CollectionDirectory, MagicDirectory, TagsDirectory, ProjectDirectory, SharedDirectory, CollectionDirectoryBase
from fusefile import StringFile, FuseArvadosFile

_logger = logging.getLogger('arvados.arvados_fuse')

log_handler = logging.StreamHandler()
llogger = logging.getLogger('llfuse')
llogger.addHandler(log_handler)
llogger.setLevel(logging.DEBUG)

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
        with llfuse.lock_released:
            return self.obj.flush()


class FileHandle(Handle):
    """Connects a numeric file handle to a File  object that has
    been opened by the client."""
    pass


class DirectoryHandle(Handle):
    """Connects a numeric file handle to a Directory object that has
    been opened by the client."""

    def __init__(self, fh, dirobj, entries):
        super(DirectoryHandle, self).__init__(fh, dirobj)
        self.entries = entries


class InodeCache(object):
    def __init__(self, cap):
        self._entries = collections.OrderedDict()
        self._counter = itertools.count(1)
        self.cap = cap
        self._total = 0

    def _remove(self, obj, clear):
        if clear and not obj.clear():
            _logger.debug("Could not clear %s in_use %s", obj, obj.in_use())
            return False
        self._total -= obj._cache_size
        del self._entries[obj._cache_priority]
        _logger.debug("Cleared %s total now %i", obj, self._total)
        return True

    def cap_cache(self):
        _logger.debug("total is %i cap is %i", self._total, self.cap)
        if self._total > self.cap:
            need_gc = False
            for key in list(self._entries.keys()):
                if self._total < self.cap or len(self._entries) < 4:
                    break
                self._remove(self._entries[key], True)


    def manage(self, obj):
        if obj.persisted():
            obj._cache_priority = next(self._counter)
            obj._cache_size = obj.objsize()
            self._entries[obj._cache_priority] = obj
            self._total += obj.objsize()
            _logger.debug("Managing %s total now %i", obj, self._total)
            self.cap_cache()

    def touch(self, obj):
        if obj.persisted():
            if obj._cache_priority in self._entries:
                self._remove(obj, False)
            self.manage(obj)
            _logger.debug("Touched %s (%i) total now %i", obj, obj.objsize(), self._total)

    def unmanage(self, obj):
        if obj.persisted() and obj._cache_priority in self._entries:
            self._remove(obj, True)


class Inodes(object):
    """Manage the set of inodes.  This is the mapping from a numeric id
    to a concrete File or Directory object"""

    def __init__(self, inode_cache=256*1024*1024):
        self._entries = {}
        self._counter = itertools.count(llfuse.ROOT_INODE)
        self._obj_cache = InodeCache(cap=inode_cache)

    def __getitem__(self, item):
        return self._entries[item]

    def __setitem__(self, key, item):
        self._entries[key] = item

    def __iter__(self):
        return self._entries.iterkeys()

    def items(self):
        return self._entries.items()

    def __contains__(self, k):
        return k in self._entries

    def touch(self, entry):
        entry._atime = time.time()
        self._obj_cache.touch(entry)

    def cap_cache(self):
        self._obj_cache.cap_cache()

    def add_entry(self, entry):
        entry.inode = next(self._counter)
        self._entries[entry.inode] = entry
        self._obj_cache.manage(entry)
        return entry

    def del_entry(self, entry):
        self._obj_cache.unmanage(entry)
        llfuse.invalidate_inode(entry.inode)
        del self._entries[entry.inode]

def catch_exceptions(orig_func):
    @functools.wraps(orig_func)
    def catch_exceptions_wrapper(self, *args, **kwargs):
        try:
            return orig_func(self, *args, **kwargs)
        except llfuse.FUSEError:
            raise
        except EnvironmentError as e:
            raise llfuse.FUSEError(e.errno)
        except Exception:
            _logger.exception("")
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

    def __init__(self, uid, gid, encoding="utf-8", inode_cache=1000, num_retries=7):
        super(Operations, self).__init__()

        self.inodes = Inodes(inode_cache)
        self.uid = uid
        self.gid = gid
        self.encoding = encoding

        # dict of inode to filehandle
        self._filehandles = {}
        self._filehandles_counter = 1

        # Other threads that need to wait until the fuse driver
        # is fully initialized should wait() on this event object.
        self.initlock = threading.Event()

        self.num_retries = num_retries

    def init(self):
        # Allow threads that are waiting for the driver to be finished
        # initializing to continue
        self.initlock.set()

    def access(self, inode, mode, ctx):
        return True

    @catch_exceptions
    def getattr(self, inode):
        if inode not in self.inodes:
            raise llfuse.FUSEError(errno.ENOENT)

        e = self.inodes[inode]

        entry = llfuse.EntryAttributes()
        entry.st_ino = inode
        entry.generation = 0
        entry.entry_timeout = 300
        entry.attr_timeout = 300

        entry.st_mode = stat.S_IRUSR | stat.S_IRGRP | stat.S_IROTH
        if isinstance(e, Directory):
            entry.st_mode |= stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH | stat.S_IFDIR
        else:
            entry.st_mode |= stat.S_IFREG
            if isinstance(e, FuseArvadosFile):
                entry.st_mode |= stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH

        if e.writable():
            entry.st_mode |= stat.S_IWUSR | stat.S_IWGRP | stat.S_IWOTH

        entry.st_nlink = 1
        entry.st_uid = self.uid
        entry.st_gid = self.gid
        entry.st_rdev = 0

        entry.st_size = e.size()

        entry.st_blksize = 512
        entry.st_blocks = (e.size()/512)+1
        entry.st_atime = int(e.atime())
        entry.st_mtime = int(e.mtime())
        entry.st_ctime = int(e.mtime())

        return entry

    @catch_exceptions
    def lookup(self, parent_inode, name):
        name = unicode(name, self.encoding)
        _logger.debug("arv-mount lookup: parent_inode %i name %s",
                      parent_inode, name)
        inode = None

        if name == '.':
            inode = parent_inode
        else:
            if parent_inode in self.inodes:
                p = self.inodes[parent_inode]
                if name == '..':
                    inode = p.parent_inode
                elif isinstance(p, Directory) and name in p:
                    inode = p[name].inode

        if inode != None:
            return self.getattr(inode)
        else:
            raise llfuse.FUSEError(errno.ENOENT)

    @catch_exceptions
    def open(self, inode, flags):
        if inode in self.inodes:
            p = self.inodes[inode]
        else:
            raise llfuse.FUSEError(errno.ENOENT)

        if isinstance(p, Directory):
            raise llfuse.FUSEError(errno.EISDIR)

        if ((flags & os.O_WRONLY) or (flags & os.O_RDWR)) and not p.writable():
            raise llfuse.FUSEError(errno.EPERM)

        fh = self._filehandles_counter
        self._filehandles_counter += 1
        self._filehandles[fh] = FileHandle(fh, p)
        self.inodes.touch(p)
        return fh

    @catch_exceptions
    def read(self, fh, off, size):
        _logger.debug("arv-mount read %i %i %i", fh, off, size)
        if fh in self._filehandles:
            handle = self._filehandles[fh]
        else:
            raise llfuse.FUSEError(errno.EBADF)

        self.inodes.touch(handle.obj)

        try:
            with llfuse.lock_released:
                return handle.obj.readfrom(off, size, self.num_retries)
        except arvados.errors.NotFoundError as e:
            _logger.warning("Block not found: " + str(e))
            raise llfuse.FUSEError(errno.EIO)

    @catch_exceptions
    def write(self, fh, off, buf):
        _logger.debug("arv-mount write %i %i %i", fh, off, len(buf))
        if fh in self._filehandles:
            handle = self._filehandles[fh]
        else:
            raise llfuse.FUSEError(errno.EBADF)

        if not handle.obj.writable():
            raise llfuse.FUSEError(errno.EPERM)

        self.inodes.touch(handle.obj)

        with llfuse.lock_released:
            return handle.obj.writeto(off, buf, self.num_retries)

    @catch_exceptions
    def release(self, fh):
        if fh in self._filehandles:
            try:
                self._filehandles[fh].flush()
            except EnvironmentError as e:
                raise llfuse.FUSEError(e.errno)
            except Exception:
                _logger.exception("Flush error")
            self._filehandles[fh].release()
            del self._filehandles[fh]
        self.inodes.cap_cache()

    def releasedir(self, fh):
        self.release(fh)

    @catch_exceptions
    def opendir(self, inode):
        _logger.debug("arv-mount opendir: inode %i", inode)

        if inode in self.inodes:
            p = self.inodes[inode]
        else:
            raise llfuse.FUSEError(errno.ENOENT)

        if not isinstance(p, Directory):
            raise llfuse.FUSEError(errno.ENOTDIR)

        fh = self._filehandles_counter
        self._filehandles_counter += 1
        if p.parent_inode in self.inodes:
            parent = self.inodes[p.parent_inode]
        else:
            raise llfuse.FUSEError(errno.EIO)

        # update atime
        self.inodes.touch(p)

        self._filehandles[fh] = DirectoryHandle(fh, p, [('.', p), ('..', parent)] + list(p.items()))
        return fh

    @catch_exceptions
    def readdir(self, fh, off):
        _logger.debug("arv-mount readdir: fh %i off %i", fh, off)

        if fh in self._filehandles:
            handle = self._filehandles[fh]
        else:
            raise llfuse.FUSEError(errno.EBADF)

        _logger.debug("arv-mount handle.dirobj %s", handle.obj)

        e = off
        while e < len(handle.entries):
            if handle.entries[e][1].inode in self.inodes:
                try:
                    yield (handle.entries[e][0].encode(self.encoding), self.getattr(handle.entries[e][1].inode), e+1)
                except UnicodeEncodeError:
                    pass
            e += 1

    @catch_exceptions
    def statfs(self):
        st = llfuse.StatvfsData()
        st.f_bsize = 64 * 1024
        st.f_blocks = 0
        st.f_files = 0

        st.f_bfree = 0
        st.f_bavail = 0

        st.f_ffree = 0
        st.f_favail = 0

        st.f_frsize = 0
        return st

    def _check_writable(self, inode_parent):
        if inode_parent in self.inodes:
            p = self.inodes[inode_parent]
        else:
            raise llfuse.FUSEError(errno.ENOENT)

        if not isinstance(p, Directory):
            raise llfuse.FUSEError(errno.ENOTDIR)

        if not p.writable():
            raise llfuse.FUSEError(errno.EPERM)

        if not isinstance(p, CollectionDirectoryBase):
            raise llfuse.FUSEError(errno.EPERM)

        return p

    @catch_exceptions
    def create(self, inode_parent, name, mode, flags, ctx):
        p = self._check_writable(inode_parent)

        with llfuse.lock_released:
            p.collection.open(name, "w")

        # The file entry should have been implicitly created by callback.
        f = p[name]

        fh = self._filehandles_counter
        self._filehandles_counter += 1
        self._filehandles[fh] = FileHandle(fh, f)
        self.inodes.touch(p)

        return (fh, self.getattr(f.inode))

    @catch_exceptions
    def mkdir(self, inode_parent, name, mode, ctx):
        p = self._check_writable(inode_parent)

        with llfuse.lock_released:
            p.collection.mkdirs(name)

        # The dir entry should have been implicitly created by callback.
        d = p[name]

        return self.getattr(d.inode)

    @catch_exceptions
    def unlink(self, inode_parent, name):
        p = self._check_writable(inode_parent)

        with llfuse.lock_released:
            p.collection.remove(name)

    def rmdir(self, inode_parent, name):
        self.unlink(inode_parent, name)

    @catch_exceptions
    def rename(self, inode_parent_old, name_old, inode_parent_new, name_new):
        src = self._check_writable(inode_parent_old)
        dest = self._check_writable(inode_parent_new)

        with llfuse.lock_released:
            dest.collection.copy(name_old, name_new, source_collection=src.collection, overwrite=True)
            src.collection.remove(name_old)

    @catch_exceptions
    def flush(self, fh):
        if fh in self._filehandles:
            self._filehandles[fh].flush()

    def fsync(self, fh, datasync):
        self.flush(fh)

    def fsyncdir(self, fh, datasync):
        self.flush(fh)
