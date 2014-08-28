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

_logger = logging.getLogger('arvados.arvados_fuse')

from time import time
from llfuse import FUSEError

class FreshBase(object):
    '''Base class for maintaining fresh/stale state to determine when to update.'''
    def __init__(self):
        self._stale = True
        self._poll = False
        self._last_update = time()
        self._poll_time = 60

    # Mark the value as stale
    def invalidate(self):
        self._stale = True

    # Test if the entries dict is stale
    def stale(self):
        if self._stale:
            return True
        if self._poll:
            return (self._last_update + self._poll_time) < time()
        return False

    def fresh(self):
        self._stale = False
        self._last_update = time()

    def ctime():
        return 0

    def mtime():
        return 0


class File(FreshBase):
    '''Base for file objects.'''

    def __init__(self, parent_inode):
        super(File, self).__init__()
        self.inode = None
        self.parent_inode = parent_inode

    def size(self):
        return 0

    def readfrom(self, off, size):
        return ''


class StreamReaderFile(File):
    '''Wraps a StreamFileReader as a file.'''

    def __init__(self, parent_inode, reader, collection):
        super(StreamReaderFile, self).__init__(parent_inode)
        self.reader = reader
        self.collection = collection

    def size(self):
        return self.reader.size()

    def readfrom(self, off, size):
        return self.reader.readfrom(off, size)

    def stale(self):
        return False

    def ctime():
        return collection["created_at"]

    def mtime():
        return collection["modified_at"]


class ObjectFile(File):
    '''Wraps a dict as a serialized json object.'''

    def __init__(self, parent_inode, contents):
        super(ObjectFile, self).__init__(parent_inode)
        self.contentsdict = contents
        self.uuid = self.contentsdict['uuid']
        self.contents = json.dumps(self.contentsdict, indent=4, sort_keys=True)

    def size(self):
        return len(self.contents)

    def readfrom(self, off, size):
        return self.contents[off:(off+size)]


class Directory(FreshBase):
    '''Generic directory object, backed by a dict.
    Consists of a set of entries with the key representing the filename
    and the value referencing a File or Directory object.
    '''

    def __init__(self, parent_inode):
        super(Directory, self).__init__()

        '''parent_inode is the integer inode number'''
        self.inode = None
        if not isinstance(parent_inode, int):
            raise Exception("parent_inode should be an int")
        self.parent_inode = parent_inode
        self._entries = {}

    #  Overriden by subclasses to implement logic to update the entries dict
    #  when the directory is stale
    def update(self):
        pass

    # Only used when computing the size of the disk footprint of the directory
    # (stub)
    def size(self):
        return 0

    def checkupdate(self):
        if self.stale():
            try:
                self.update()
            except apiclient.errors.HttpError as e:
                _logger.debug(e)

    def __getitem__(self, item):
        self.checkupdate()
        return self._entries[item]

    def items(self):
        self.checkupdate()
        return self._entries.items()

    def __iter__(self):
        self.checkupdate()
        return self._entries.iterkeys()

    def __contains__(self, k):
        self.checkupdate()
        return k in self._entries

    def merge(self, items, fn, same, new_entry):
        '''Helper method for updating the contents of the directory.

        items: array with new directory contents

        fn: function to take an entry in 'items' and return the desired file or
        directory name

        same: function to compare an existing entry with an entry in the items
        list to determine whether to keep the existing entry.

        new_entry: function to create a new directory entry from array entry.
        '''

        oldentries = self._entries
        self._entries = {}
        for i in items:
            n = fn(i)
            if n in oldentries and same(oldentries[n], i):
                self._entries[n] = oldentries[n]
                del oldentries[n]
            else:
                ent = new_entry(i)
                if ent is not None:
                    self._entries[n] = self.inodes.add_entry(ent)
        for n in oldentries:
            llfuse.invalidate_entry(self.inode, str(n))
            self.inodes.del_entry(oldentries[n])
        self.fresh()


class CollectionDirectory(Directory):
    '''Represents the root of a directory tree holding a collection.'''

    def __init__(self, parent_inode, inodes, collection_locator):
        super(CollectionDirectory, self).__init__(parent_inode)
        self.inodes = inodes
        self.collection_locator = collection_locator

    def same(self, i):
        return i['uuid'] == self.collection_locator or i['portable_data_hash'] == self.collection_locator

    def update(self):
        try:
            collection = arvados.CollectionReader(self.collection_locator)
            for s in collection.all_streams():
                cwd = self
                for part in s.name().split('/'):
                    if part != '' and part != '.':
                        if part not in cwd._entries:
                            cwd._entries[part] = self.inodes.add_entry(Directory(cwd.inode))
                        cwd = cwd._entries[part]
                for k, v in s.files().items():
                    cwd._entries[k] = self.inodes.add_entry(StreamReaderFile(cwd.inode, v))
            self.fresh()
            return True
        except Exception as detail:
            _logger.debug("arv-mount %s: error: %s",
                          self.collection_locator, detail)
            return False

class MagicDirectory(Directory):
    '''A special directory that logically contains the set of all extant keep
    locators.  When a file is referenced by lookup(), it is tested to see if it
    is a valid keep locator to a manifest, and if so, loads the manifest
    contents as a subdirectory of this directory with the locator as the
    directory name.  Since querying a list of all extant keep locators is
    impractical, only collections that have already been accessed are visible
    to readdir().
    '''

    def __init__(self, parent_inode, inodes):
        super(MagicDirectory, self).__init__(parent_inode)
        self.inodes = inodes

    def __contains__(self, k):
        if k in self._entries:
            return True
        try:
            e = self.inodes.add_entry(CollectionDirectory(self.inode, self.inodes, k))
            if e.update():
                self._entries[k] = e
                return True
            else:
                return False
        except Exception as e:
            _logger.debug('arv-mount exception keep %s', e)
            return False

    def __getitem__(self, item):
        if item in self:
            return self._entries[item]
        else:
            raise KeyError("No collection with id " + item)

class RecursiveInvalidateDirectory(Directory):
    def invalidate(self):
        try:
            if self.parent_inode == llfuse.ROOT_INODE:
                llfuse.lock.acquire()
            super(RecursiveInvalidateDirectory, self).invalidate()
            for a in self._entries:
                self._entries[a].invalidate()
        finally:
            if self.parent_inode == llfuse.ROOT_INODE:
                llfuse.lock.release()

class TagsDirectory(RecursiveInvalidateDirectory):
    '''A special directory that contains as subdirectories all tags visible to the user.'''

    def __init__(self, parent_inode, inodes, api, poll_time=60):
        super(TagsDirectory, self).__init__(parent_inode)
        self.inodes = inodes
        self.api = api
        try:
            arvados.events.subscribe(self.api, [['object_uuid', 'is_a', 'arvados#link']], lambda ev: self.invalidate())
        except:
            self._poll = True
            self._poll_time = poll_time

    def update(self):
        tags = self.api.links().list(filters=[['link_class', '=', 'tag']], select=['name'], distinct = True).execute()
        if "items" in tags:
            self.merge(tags['items'],
                       lambda i: i['name'] if 'name' in i else i['uuid'],
                       lambda a, i: a.tag == i,
                       lambda i: TagDirectory(self.inode, self.inodes, self.api, i['name'], poll=self._poll, poll_time=self._poll_time))

class TagDirectory(Directory):
    '''A special directory that contains as subdirectories all collections visible
    to the user that are tagged with a particular tag.
    '''

    def __init__(self, parent_inode, inodes, api, tag, poll=False, poll_time=60):
        super(TagDirectory, self).__init__(parent_inode)
        self.inodes = inodes
        self.api = api
        self.tag = tag
        self._poll = poll
        self._poll_time = poll_time

    def update(self):
        taggedcollections = self.api.links().list(filters=[['link_class', '=', 'tag'],
                                               ['name', '=', self.tag],
                                               ['head_uuid', 'is_a', 'arvados#collection']],
                                      select=['head_uuid']).execute()
        self.merge(taggedcollections['items'],
                   lambda i: i['head_uuid'],
                   lambda a, i: a.collection_locator == i['head_uuid'],
                   lambda i: CollectionDirectory(self.inode, self.inodes, i['head_uuid']))


class ProjectDirectory(RecursiveInvalidateDirectory):
    '''A special directory that contains the contents of a project.'''

    def __init__(self, parent_inode, inodes, api, uuid, poll=False, poll_time=60):
        super(ProjectDirectory, self).__init__(parent_inode)
        self.inodes = inodes
        self.api = api
        self.uuid = uuid['uuid']

        if parent_inode == llfuse.ROOT_INODE:
            try:
                arvados.events.subscribe(self.api, [], lambda ev: self.invalidate())
            except:
                self._poll = True
                self._poll_time = poll_time
        else:
            self._poll = poll
            self._poll_time = poll_time


    def createDirectory(self, i):
        if re.match(r'[a-z0-9]{5}-4zz18-[a-z0-9]{15}', i['uuid']) and i['name'] is not None:
            return CollectionDirectory(self.inode, self.inodes, i['uuid'])
        elif re.match(r'[a-z0-9]{5}-j7d0g-[a-z0-9]{15}', i['uuid']):
            return ProjectDirectory(self.parent_inode, self.inodes, self.api, i, self._poll, self._poll_time)
        #elif re.match(r'[a-z0-9]{5}-8i9sb-[a-z0-9]{15}', i['uuid']):
        #    return None
        #elif re.match(r'[a-z0-9]{5}-[a-z0-9]{5}-[a-z0-9]{15}', i['uuid']):
        #    return ObjectFile(self.parent_inode, i)
        else:
            return None

    def contents(self):
        return arvados.util.all_contents(self.api, self.uuid)

    def update(self):
        def same(a, i):
            if isinstance(a, CollectionDirectory):
                return a.collection_locator == i['uuid']
            elif isinstance(a, ProjectDirectory):
                return a.uuid == i['uuid']
            elif isinstance(a, ObjectFile):
                return a.uuid == i['uuid'] and not a.stale()
            return False

        self.merge(self.contents(),
                   lambda i: i['name'] if 'name' in i and i['name'] is not None and len(i['name']) > 0 else i['uuid'],
                   same,
                   self.createDirectory)


class HomeDirectory(ProjectDirectory):
    '''A special directory that represents the "home" project.'''

    def __init__(self, parent_inode, inodes, api, poll=False, poll_time=60):
        super(HomeDirectory, self).__init__(parent_inode, inodes, api, api.users().current().execute())

    #def contents(self):
    #    return self.api.groups().contents(uuid=self.uuid).execute()['items']

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

    def add_entry(self, entry):
        entry.inode = self._counter
        self._entries[entry.inode] = entry
        self._counter += 1
        return entry

    def del_entry(self, entry):
        llfuse.invalidate_inode(entry.inode)
        del self._entries[entry.inode]

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
        _logger.debug("arv-mount read %i %i %i", fh, off, size)
        if fh in self._filehandles:
            handle = self._filehandles[fh]
        else:
            raise llfuse.FUSEError(errno.EBADF)

        try:
            with llfuse.lock_released:
                return handle.entry.readfrom(off, size)
        except:
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
