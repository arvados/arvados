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

from fusedir import sanitize_filename, Directory, CollectionDirectory, MagicDirectory, TagsDirectory, ProjectDirectory, SharedDirectory
from fusefile import StreamReaderFile

_logger = logging.getLogger('arvados.arvados_fuse')


class FileHandle(object):
    """Connects a numeric file handle to a File or Directory object that has
    been opened by the client."""

    def __init__(self, fh, entry):
        self.fh = fh
        self.entry = entry


class Inodes(object):
    """Manage the set of inodes.  This is the mapping from a numeric id
    to a concrete File or Directory object"""

    def __init__(self):
        self._entries = {}
        self._counter = itertools.count(llfuse.ROOT_INODE)

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

    def add_entry(self, entry):
        entry.inode = next(self._counter)
        self._entries[entry.inode] = entry
        return entry

    def del_entry(self, entry):
        llfuse.invalidate_inode(entry.inode)
        del self._entries[entry.inode]


class Operations(llfuse.Operations):
    """This is the main interface with llfuse.

    The methods on this object are called by llfuse threads to service FUSE
    events to query and read from the file system.

    llfuse has its own global lock which is acquired before calling a request handler,
    so request handlers do not run concurrently unless the lock is explicitly released
    using 'with llfuse.lock_released:'

    """

    def __init__(self, uid, gid, encoding="utf-8"):
        super(Operations, self).__init__()

        self.inodes = Inodes()
        self.uid = uid
        self.gid = gid
        self.encoding = encoding

        # dict of inode to filehandle
        self._filehandles = {}
        self._filehandles_counter = 1

        # Other threads that need to wait until the fuse driver
        # is fully initialized should wait() on this event object.
        self.initlock = threading.Event()

    def init(self):
        # Allow threads that are waiting for the driver to be finished
        # initializing to continue
        self.initlock.set()

    def access(self, inode, mode, ctx):
        return True

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
        elif isinstance(e, StreamReaderFile):
            entry.st_mode |= stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH | stat.S_IFREG
        else:
            entry.st_mode |= stat.S_IFREG

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

    def open(self, inode, flags):
        if inode in self.inodes:
            p = self.inodes[inode]
        else:
            raise llfuse.FUSEError(errno.ENOENT)

        if (flags & os.O_WRONLY) or (flags & os.O_RDWR):
            raise llfuse.FUSEError(errno.EROFS)

        if isinstance(p, Directory):
            raise llfuse.FUSEError(errno.EISDIR)

        fh = self._filehandles_counter
        self._filehandles_counter += 1
        self._filehandles[fh] = FileHandle(fh, p)
        return fh

    def read(self, fh, off, size):
        _logger.debug("arv-mount read %i %i %i", fh, off, size)
        if fh in self._filehandles:
            handle = self._filehandles[fh]
        else:
            raise llfuse.FUSEError(errno.EBADF)

        # update atime
        handle.entry._atime = time.time()

        try:
            with llfuse.lock_released:
                return handle.entry.readfrom(off, size)
        except arvados.errors.NotFoundError as e:
            _logger.warning("Block not found: " + str(e))
            raise llfuse.FUSEError(errno.EIO)
        except Exception:
            _logger.exception()
            raise llfuse.FUSEError(errno.EIO)

    def release(self, fh):
        if fh in self._filehandles:
            del self._filehandles[fh]

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
        p._atime = time.time()

        self._filehandles[fh] = FileHandle(fh, [('.', p), ('..', parent)] + list(p.items()))
        return fh

    def readdir(self, fh, off):
        _logger.debug("arv-mount readdir: fh %i off %i", fh, off)

        if fh in self._filehandles:
            handle = self._filehandles[fh]
        else:
            raise llfuse.FUSEError(errno.EBADF)

        _logger.debug("arv-mount handle.entry %s", handle.entry)

        e = off
        while e < len(handle.entry):
            if handle.entry[e][1].inode in self.inodes:
                try:
                    yield (handle.entry[e][0].encode(self.encoding), self.getattr(handle.entry[e][1].inode), e+1)
                except UnicodeEncodeError:
                    pass
            e += 1

    def releasedir(self, fh):
        del self._filehandles[fh]

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

    # The llfuse documentation recommends only overloading functions that
    # are actually implemented, as the default implementation will raise ENOSYS.
    # However, there is a bug in the llfuse default implementation of create()
    # "create() takes exactly 5 positional arguments (6 given)" which will crash
    # arv-mount.
    # The workaround is to implement it with the proper number of parameters,
    # and then everything works out.
    def create(self, inode_parent, name, mode, flags, ctx):
        raise llfuse.FUSEError(errno.EROFS)
