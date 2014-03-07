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

from time import time
from llfuse import FUSEError

class Directory(object):
    '''Generic directory object, backed by a dict.
    Consists of a set of entries with the key representing the filename
    and the value referencing a File or Directory object.
    '''

    def __init__(self, parent_inode):
        self.inode = None
        self.parent_inode = parent_inode
        self._entries = {}

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

    def size(self):
        return 0

class MagicDirectory(Directory):
    '''A special directory that logically contains the set of all extant
    keep locators.  When a file is referenced by lookup(), it is tested
    to see if it is a valid keep locator to a manifest, and if so, loads the manifest
    contents as a subdirectory of this directory with the locator as the directory name.
    Since querying a list of all extant keep locators is impractical, only loaded collections 
    are visible to readdir().'''

    def __init__(self, parent_inode, inodes):
        super(MagicDirectory, self).__init__(parent_inode)
        self.inodes = inodes

    def __contains__(self, k):
        if k in self._entries:
            return True
        try:
            if arvados.Keep.get(k):
                return True
            else:
                return False
        except Exception as e:
            #print 'exception keep', e
            return False

    def __getitem__(self, item):
        if item not in self._entries:
            collection = arvados.CollectionReader(arvados.Keep.get(item))
            self._entries[item] = self.inodes.add_entry(Directory(self.inode))
            self.inodes.load_collection(self._entries[item], collection)
        return self._entries[item]

class File(object):
    '''Wraps a StreamFileReader for use by Directory.'''

    def __init__(self, parent_inode, reader):
        self.inode = None
        self.parent_inode = parent_inode
        self.reader = reader

    def size(self):
        return self.reader.size()

class FileHandle(object):
    '''Connects a numeric file handle to a File or Directory object that has 
    been opened by the client.'''

    def __init__(self, fh, entry):
        self.fh = fh
        self.entry = entry

class Inodes(object):
    '''Manage the set of inodes.  This is the mapping from a numeric id 
    to a concrete File or Directory object'''

    def __init__(self):
        self._entries = {}
        self._counter = llfuse.ROOT_INODE

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

    def load_collection(self, parent_dir, collection):
        '''parent_dir is the Directory object that will be populated by the collection.
        collection is the arvados.CollectionReader to use as the source'''
        for s in collection.all_streams():
            cwd = parent_dir
            for part in s.name().split('/'):
                if part != '' and part != '.':
                    if part not in cwd:
                        cwd[part] = self.add_entry(Directory(cwd.inode))
                    cwd = cwd[part]
            for k, v in s.files().items():
                cwd[k] = self.add_entry(File(cwd.inode, v))

    def add_entry(self, entry):
        entry.inode = self._counter
        self._entries[entry.inode] = entry
        self._counter += 1
        return entry    

class Operations(llfuse.Operations):
    '''This is the main interface with llfuse.  The methods on this object are
    called by llfuse threads to service FUSE events to query and read from 
    the file system.

    llfuse has its own global lock which is acquired before calling a request handler,
    so request handlers do not run concurrently unless the lock is explicitly released 
    with llfuse.lock_released.'''

    def __init__(self, uid, gid):
        super(Operations, self).__init__()

        self.inodes = Inodes()
        self.uid = uid
        self.gid = gid
        
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

        entry.st_nlink = 1
        entry.st_uid = self.uid
        entry.st_gid = self.gid
        entry.st_rdev = 0

        entry.st_size = e.size()

        entry.st_blksize = 1024
        entry.st_blocks = e.size()/1024
        if e.size()/1024 != 0:
            entry.st_blocks += 1
        entry.st_atime = 0
        entry.st_mtime = 0
        entry.st_ctime = 0

        return entry

    def lookup(self, parent_inode, name):
        #print "lookup: parent_inode", parent_inode, "name", name
        inode = None

        if name == '.':
            inode = parent_inode
        else:
            if parent_inode in self.inodes:
                p = self.inodes[parent_inode]
                if name == '..':
                    inode = p.parent_inode
                elif name in p:
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
        #print "read", fh, off, size
        if fh in self._filehandles:
            handle = self._filehandles[fh]
        else:
            raise llfuse.FUSEError(errno.EBADF)

        try:
            with llfuse.lock_released:
                return handle.entry.reader.readfrom(off, size)
        except:
            raise llfuse.FUSEError(errno.EIO)

    def release(self, fh):
        if fh in self._filehandles:
            del self._filehandles[fh]

    def opendir(self, inode):
        #print "opendir: inode", inode

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
            parent = None
        self._filehandles[fh] = FileHandle(fh, [('.', p), ('..', parent)] + list(p.items()))
        return fh

    def readdir(self, fh, off):
        #print "readdir: fh", fh, "off", off

        if fh in self._filehandles:
            handle = self._filehandles[fh]
        else:
            raise llfuse.FUSEError(errno.EBADF)

        #print "handle.entry", handle.entry

        e = off
        while e < len(handle.entry):
            yield (handle.entry[e][0], self.getattr(handle.entry[e][1].inode), e+1)
            e += 1

    def releasedir(self, fh):
        del self._filehandles[fh]

    def statfs(self):
        st = llfuse.StatvfsData()
        st.f_bsize = 1024 * 1024
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
    def create(self, p1, p2, p3, p4, p5):
        raise llfuse.FUSEError(errno.EROFS)
