# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import logging
import re
import time
import llfuse
import arvados
import apiclient
import functools
import threading
from apiclient import errors as apiclient_errors
import errno
import time

from fusefile import StringFile, ObjectFile, FuncToJSONFile, FuseArvadosFile
from fresh import FreshBase, convertTime, use_counter, check_update

import arvados.collection
from arvados.util import portable_data_hash_pattern, uuid_pattern, collection_uuid_pattern, group_uuid_pattern, user_uuid_pattern, link_uuid_pattern

_logger = logging.getLogger('arvados.arvados_fuse')


# Match any character which FUSE or Linux cannot accommodate as part
# of a filename. (If present in a collection filename, they will
# appear as underscores in the fuse mount.)
_disallowed_filename_characters = re.compile('[\x00/]')

# '.' and '..' are not reachable if API server is newer than #6277
def sanitize_filename(dirty):
    """Replace disallowed filename characters with harmless "_"."""
    if dirty is None:
        return None
    elif dirty == '':
        return '_'
    elif dirty == '.':
        return '_'
    elif dirty == '..':
        return '__'
    else:
        return _disallowed_filename_characters.sub('_', dirty)


class Directory(FreshBase):
    """Generic directory object, backed by a dict.

    Consists of a set of entries with the key representing the filename
    and the value referencing a File or Directory object.
    """

    def __init__(self, parent_inode, inodes):
        """parent_inode is the integer inode number"""

        super(Directory, self).__init__()

        self.inode = None
        if not isinstance(parent_inode, int):
            raise Exception("parent_inode should be an int")
        self.parent_inode = parent_inode
        self.inodes = inodes
        self._entries = {}
        self._mtime = time.time()

    #  Overriden by subclasses to implement logic to update the entries dict
    #  when the directory is stale
    @use_counter
    def update(self):
        pass

    # Only used when computing the size of the disk footprint of the directory
    # (stub)
    def size(self):
        return 0

    def persisted(self):
        return False

    def checkupdate(self):
        if self.stale():
            try:
                self.update()
            except apiclient.errors.HttpError as e:
                _logger.warn(e)

    @use_counter
    @check_update
    def __getitem__(self, item):
        return self._entries[item]

    @use_counter
    @check_update
    def items(self):
        return list(self._entries.items())

    @use_counter
    @check_update
    def __contains__(self, k):
        return k in self._entries

    @use_counter
    @check_update
    def __len__(self):
        return len(self._entries)

    def fresh(self):
        self.inodes.touch(self)
        super(Directory, self).fresh()

    def merge(self, items, fn, same, new_entry):
        """Helper method for updating the contents of the directory.

        Takes a list describing the new contents of the directory, reuse
        entries that are the same in both the old and new lists, create new
        entries, and delete old entries missing from the new list.

        :items: iterable with new directory contents

        :fn: function to take an entry in 'items' and return the desired file or
        directory name, or None if this entry should be skipped

        :same: function to compare an existing entry (a File or Directory
        object) with an entry in the items list to determine whether to keep
        the existing entry.

        :new_entry: function to create a new directory entry (File or Directory
        object) from an entry in the items list.

        """

        oldentries = self._entries
        self._entries = {}
        changed = False
        for i in items:
            name = sanitize_filename(fn(i))
            if name:
                if name in oldentries and same(oldentries[name], i):
                    # move existing directory entry over
                    self._entries[name] = oldentries[name]
                    del oldentries[name]
                else:
                    _logger.debug("Adding entry '%s' to inode %i", name, self.inode)
                    # create new directory entry
                    ent = new_entry(i)
                    if ent is not None:
                        self._entries[name] = self.inodes.add_entry(ent)
                        changed = True

        # delete any other directory entries that were not in found in 'items'
        for i in oldentries:
            _logger.debug("Forgetting about entry '%s' on inode %i", i, self.inode)
            self.inodes.invalidate_entry(self, i)
            self.inodes.del_entry(oldentries[i])
            changed = True

        if changed:
            self.inodes.invalidate_inode(self)
            self._mtime = time.time()

        self.fresh()

    def in_use(self):
        if super(Directory, self).in_use():
            return True
        for v in self._entries.itervalues():
            if v.in_use():
                return True
        return False

    def has_ref(self, only_children):
        if super(Directory, self).has_ref(only_children):
            return True
        for v in self._entries.itervalues():
            if v.has_ref(False):
                return True
        return False

    def clear(self):
        """Delete all entries"""
        oldentries = self._entries
        self._entries = {}
        for n in oldentries:
            oldentries[n].clear()
            self.inodes.del_entry(oldentries[n])
        self.invalidate()

    def kernel_invalidate(self):
        # Invalidating the dentry on the parent implies invalidating all paths
        # below it as well.
        parent = self.inodes[self.parent_inode]

        # Find self on the parent in order to invalidate this path.
        # Calling the public items() method might trigger a refresh,
        # which we definitely don't want, so read the internal dict directly.
        for k,v in parent._entries.items():
            if v is self:
                self.inodes.invalidate_entry(parent, k)
                break

    def mtime(self):
        return self._mtime

    def writable(self):
        return False

    def flush(self):
        pass

    def want_event_subscribe(self):
        raise NotImplementedError()

    def create(self, name):
        raise NotImplementedError()

    def mkdir(self, name):
        raise NotImplementedError()

    def unlink(self, name):
        raise NotImplementedError()

    def rmdir(self, name):
        raise NotImplementedError()

    def rename(self, name_old, name_new, src):
        raise NotImplementedError()


class CollectionDirectoryBase(Directory):
    """Represent an Arvados Collection as a directory.

    This class is used for Subcollections, and is also the base class for
    CollectionDirectory, which implements collection loading/saving on
    Collection records.

    Most operations act only the underlying Arvados `Collection` object.  The
    `Collection` object signals via a notify callback to
    `CollectionDirectoryBase.on_event` that an item was added, removed or
    modified.  FUSE inodes and directory entries are created, deleted or
    invalidated in response to these events.

    """

    def __init__(self, parent_inode, inodes, collection):
        super(CollectionDirectoryBase, self).__init__(parent_inode, inodes)
        self.collection = collection

    def new_entry(self, name, item, mtime):
        name = sanitize_filename(name)
        if hasattr(item, "fuse_entry") and item.fuse_entry is not None:
            if item.fuse_entry.dead is not True:
                raise Exception("Can only reparent dead inode entry")
            if item.fuse_entry.inode is None:
                raise Exception("Reparented entry must still have valid inode")
            item.fuse_entry.dead = False
            self._entries[name] = item.fuse_entry
        elif isinstance(item, arvados.collection.RichCollectionBase):
            self._entries[name] = self.inodes.add_entry(CollectionDirectoryBase(self.inode, self.inodes, item))
            self._entries[name].populate(mtime)
        else:
            self._entries[name] = self.inodes.add_entry(FuseArvadosFile(self.inode, item, mtime))
        item.fuse_entry = self._entries[name]

    def on_event(self, event, collection, name, item):
        if collection == self.collection:
            name = sanitize_filename(name)
            _logger.debug("collection notify %s %s %s %s", event, collection, name, item)
            with llfuse.lock:
                if event == arvados.collection.ADD:
                    self.new_entry(name, item, self.mtime())
                elif event == arvados.collection.DEL:
                    ent = self._entries[name]
                    del self._entries[name]
                    self.inodes.invalidate_entry(self, name)
                    self.inodes.del_entry(ent)
                elif event == arvados.collection.MOD:
                    if hasattr(item, "fuse_entry") and item.fuse_entry is not None:
                        self.inodes.invalidate_inode(item.fuse_entry)
                    elif name in self._entries:
                        self.inodes.invalidate_inode(self._entries[name])

    def populate(self, mtime):
        self._mtime = mtime
        self.collection.subscribe(self.on_event)
        for entry, item in self.collection.items():
            self.new_entry(entry, item, self.mtime())

    def writable(self):
        return self.collection.writable()

    @use_counter
    def flush(self):
        with llfuse.lock_released:
            self.collection.root_collection().save()

    @use_counter
    @check_update
    def create(self, name):
        with llfuse.lock_released:
            self.collection.open(name, "w").close()

    @use_counter
    @check_update
    def mkdir(self, name):
        with llfuse.lock_released:
            self.collection.mkdirs(name)

    @use_counter
    @check_update
    def unlink(self, name):
        with llfuse.lock_released:
            self.collection.remove(name)
        self.flush()

    @use_counter
    @check_update
    def rmdir(self, name):
        with llfuse.lock_released:
            self.collection.remove(name)
        self.flush()

    @use_counter
    @check_update
    def rename(self, name_old, name_new, src):
        if not isinstance(src, CollectionDirectoryBase):
            raise llfuse.FUSEError(errno.EPERM)

        if name_new in self:
            ent = src[name_old]
            tgt = self[name_new]
            if isinstance(ent, FuseArvadosFile) and isinstance(tgt, FuseArvadosFile):
                pass
            elif isinstance(ent, CollectionDirectoryBase) and isinstance(tgt, CollectionDirectoryBase):
                if len(tgt) > 0:
                    raise llfuse.FUSEError(errno.ENOTEMPTY)
            elif isinstance(ent, CollectionDirectoryBase) and isinstance(tgt, FuseArvadosFile):
                raise llfuse.FUSEError(errno.ENOTDIR)
            elif isinstance(ent, FuseArvadosFile) and isinstance(tgt, CollectionDirectoryBase):
                raise llfuse.FUSEError(errno.EISDIR)

        with llfuse.lock_released:
            self.collection.rename(name_old, name_new, source_collection=src.collection, overwrite=True)
        self.flush()
        src.flush()

    def clear(self):
        super(CollectionDirectoryBase, self).clear()
        self.collection = None


class CollectionDirectory(CollectionDirectoryBase):
    """Represents the root of a directory tree representing a collection."""

    def __init__(self, parent_inode, inodes, api, num_retries, collection_record=None, explicit_collection=None):
        super(CollectionDirectory, self).__init__(parent_inode, inodes, None)
        self.api = api
        self.num_retries = num_retries
        self.collection_record_file = None
        self.collection_record = None
        self._poll = True
        try:
            self._poll_time = (api._rootDesc.get('blobSignatureTtl', 60*60*2)/2)
        except:
            _logger.debug("Error getting blobSignatureTtl from discovery document: %s", sys.exc_info()[0])
            self._poll_time = 60*60

        if isinstance(collection_record, dict):
            self.collection_locator = collection_record['uuid']
            self._mtime = convertTime(collection_record.get('modified_at'))
        else:
            self.collection_locator = collection_record
            self._mtime = 0
        self._manifest_size = 0
        if self.collection_locator:
            self._writable = (uuid_pattern.match(self.collection_locator) is not None)
        self._updating_lock = threading.Lock()

    def same(self, i):
        return i['uuid'] == self.collection_locator or i['portable_data_hash'] == self.collection_locator

    def writable(self):
        return self.collection.writable() if self.collection is not None else self._writable

    def want_event_subscribe(self):
        return (uuid_pattern.match(self.collection_locator) is not None)

    # Used by arv-web.py to switch the contents of the CollectionDirectory
    def change_collection(self, new_locator):
        """Switch the contents of the CollectionDirectory.

        Must be called with llfuse.lock held.
        """

        self.collection_locator = new_locator
        self.collection_record = None
        self.update()

    def new_collection(self, new_collection_record, coll_reader):
        if self.inode:
            self.clear()

        self.collection_record = new_collection_record

        if self.collection_record:
            self._mtime = convertTime(self.collection_record.get('modified_at'))
            self.collection_locator = self.collection_record["uuid"]
            if self.collection_record_file is not None:
                self.collection_record_file.update(self.collection_record)

        self.collection = coll_reader
        self.populate(self.mtime())

    def uuid(self):
        return self.collection_locator

    @use_counter
    def update(self, to_record_version=None):
        try:
            if self.collection_record is not None and portable_data_hash_pattern.match(self.collection_locator):
                return True

            if self.collection_locator is None:
                self.fresh()
                return True

            try:
                with llfuse.lock_released:
                    self._updating_lock.acquire()
                    if not self.stale():
                        return

                    _logger.debug("Updating collection %s inode %s to record version %s", self.collection_locator, self.inode, to_record_version)
                    if self.collection is not None:
                        if self.collection.known_past_version(to_record_version):
                            _logger.debug("%s already processed %s", self.collection_locator, to_record_version)
                        else:
                            self.collection.update()
                    else:
                        if uuid_pattern.match(self.collection_locator):
                            coll_reader = arvados.collection.Collection(
                                self.collection_locator, self.api, self.api.keep,
                                num_retries=self.num_retries)
                        else:
                            coll_reader = arvados.collection.CollectionReader(
                                self.collection_locator, self.api, self.api.keep,
                                num_retries=self.num_retries)
                        new_collection_record = coll_reader.api_response() or {}
                        # If the Collection only exists in Keep, there will be no API
                        # response.  Fill in the fields we need.
                        if 'uuid' not in new_collection_record:
                            new_collection_record['uuid'] = self.collection_locator
                        if "portable_data_hash" not in new_collection_record:
                            new_collection_record["portable_data_hash"] = new_collection_record["uuid"]
                        if 'manifest_text' not in new_collection_record:
                            new_collection_record['manifest_text'] = coll_reader.manifest_text()

                        if self.collection_record is None or self.collection_record["portable_data_hash"] != new_collection_record.get("portable_data_hash"):
                            self.new_collection(new_collection_record, coll_reader)

                        self._manifest_size = len(coll_reader.manifest_text())
                        _logger.debug("%s manifest_size %i", self, self._manifest_size)
                # end with llfuse.lock_released, re-acquire lock

                self.fresh()
                return True
            finally:
                self._updating_lock.release()
        except arvados.errors.NotFoundError as e:
            _logger.error("Error fetching collection '%s': %s", self.collection_locator, e)
        except arvados.errors.ArgumentError as detail:
            _logger.warning("arv-mount %s: error %s", self.collection_locator, detail)
            if self.collection_record is not None and "manifest_text" in self.collection_record:
                _logger.warning("arv-mount manifest_text is: %s", self.collection_record["manifest_text"])
        except Exception:
            _logger.exception("arv-mount %s: error", self.collection_locator)
            if self.collection_record is not None and "manifest_text" in self.collection_record:
                _logger.error("arv-mount manifest_text is: %s", self.collection_record["manifest_text"])
        self.invalidate()
        return False

    @use_counter
    @check_update
    def __getitem__(self, item):
        if item == '.arvados#collection':
            if self.collection_record_file is None:
                self.collection_record_file = ObjectFile(self.inode, self.collection_record)
                self.inodes.add_entry(self.collection_record_file)
            return self.collection_record_file
        else:
            return super(CollectionDirectory, self).__getitem__(item)

    def __contains__(self, k):
        if k == '.arvados#collection':
            return True
        else:
            return super(CollectionDirectory, self).__contains__(k)

    def invalidate(self):
        self.collection_record = None
        self.collection_record_file = None
        super(CollectionDirectory, self).invalidate()

    def persisted(self):
        return (self.collection_locator is not None)

    def objsize(self):
        # This is an empirically-derived heuristic to estimate the memory used
        # to store this collection's metadata.  Calculating the memory
        # footprint directly would be more accurate, but also more complicated.
        return self._manifest_size * 128

    def finalize(self):
        if self.collection is not None:
            if self.writable():
                self.collection.save()
            self.collection.stop_threads()

    def clear(self):
        if self.collection is not None:
            self.collection.stop_threads()
        super(CollectionDirectory, self).clear()
        self._manifest_size = 0


class TmpCollectionDirectory(CollectionDirectoryBase):
    """A directory backed by an Arvados collection that never gets saved.

    This supports using Keep as scratch space. A userspace program can
    read the .arvados#collection file to get a current manifest in
    order to save a snapshot of the scratch data or use it as a crunch
    job output.
    """

    class UnsaveableCollection(arvados.collection.Collection):
        def save(self):
            pass
        def save_new(self):
            pass

    def __init__(self, parent_inode, inodes, api_client, num_retries):
        collection = self.UnsaveableCollection(
            api_client=api_client,
            keep_client=api_client.keep,
            num_retries=num_retries)
        super(TmpCollectionDirectory, self).__init__(
            parent_inode, inodes, collection)
        self.collection_record_file = None
        self.populate(self.mtime())

    def on_event(self, *args, **kwargs):
        super(TmpCollectionDirectory, self).on_event(*args, **kwargs)
        if self.collection_record_file:
            with llfuse.lock:
                self.collection_record_file.invalidate()
            self.inodes.invalidate_inode(self.collection_record_file)
            _logger.debug("%s invalidated collection record", self)

    def collection_record(self):
        with llfuse.lock_released:
            return {
                "uuid": None,
                "manifest_text": self.collection.manifest_text(),
                "portable_data_hash": self.collection.portable_data_hash(),
            }

    def __contains__(self, k):
        return (k == '.arvados#collection' or
                super(TmpCollectionDirectory, self).__contains__(k))

    @use_counter
    def __getitem__(self, item):
        if item == '.arvados#collection':
            if self.collection_record_file is None:
                self.collection_record_file = FuncToJSONFile(
                    self.inode, self.collection_record)
                self.inodes.add_entry(self.collection_record_file)
            return self.collection_record_file
        return super(TmpCollectionDirectory, self).__getitem__(item)

    def persisted(self):
        return False

    def writable(self):
        return True

    def want_event_subscribe(self):
        return False

    def finalize(self):
        self.collection.stop_threads()

    def invalidate(self):
        if self.collection_record_file:
            self.collection_record_file.invalidate()
        super(TmpCollectionDirectory, self).invalidate()


class MagicDirectory(Directory):
    """A special directory that logically contains the set of all extant keep locators.

    When a file is referenced by lookup(), it is tested to see if it is a valid
    keep locator to a manifest, and if so, loads the manifest contents as a
    subdirectory of this directory with the locator as the directory name.
    Since querying a list of all extant keep locators is impractical, only
    collections that have already been accessed are visible to readdir().

    """

    README_TEXT = """
This directory provides access to Arvados collections as subdirectories listed
by uuid (in the form 'zzzzz-4zz18-1234567890abcde') or portable data hash (in
the form '1234567890abcdef0123456789abcdef+123'), and Arvados projects by uuid
(in the form 'zzzzz-j7d0g-1234567890abcde').

Note that this directory will appear empty until you attempt to access a
specific collection or project subdirectory (such as trying to 'cd' into it),
at which point the collection or project will actually be looked up on the server
and the directory will appear if it exists.

""".lstrip()

    def __init__(self, parent_inode, inodes, api, num_retries, pdh_only=False):
        super(MagicDirectory, self).__init__(parent_inode, inodes)
        self.api = api
        self.num_retries = num_retries
        self.pdh_only = pdh_only

    def __setattr__(self, name, value):
        super(MagicDirectory, self).__setattr__(name, value)
        # When we're assigned an inode, add a README.
        if ((name == 'inode') and (self.inode is not None) and
              (not self._entries)):
            self._entries['README'] = self.inodes.add_entry(
                StringFile(self.inode, self.README_TEXT, time.time()))
            # If we're the root directory, add an identical by_id subdirectory.
            if self.inode == llfuse.ROOT_INODE:
                self._entries['by_id'] = self.inodes.add_entry(MagicDirectory(
                        self.inode, self.inodes, self.api, self.num_retries, self.pdh_only))

    def __contains__(self, k):
        if k in self._entries:
            return True

        if not portable_data_hash_pattern.match(k) and (self.pdh_only or not uuid_pattern.match(k)):
            return False

        try:
            e = None

            if group_uuid_pattern.match(k):
                project = self.api.groups().list(
                    filters=[['group_class', '=', 'project'], ["uuid", "=", k]]).execute(num_retries=self.num_retries)
                if project[u'items_available'] == 0:
                    return False
                e = self.inodes.add_entry(ProjectDirectory(
                    self.inode, self.inodes, self.api, self.num_retries, project[u'items'][0]))
            else:
                e = self.inodes.add_entry(CollectionDirectory(
                        self.inode, self.inodes, self.api, self.num_retries, k))

            if e.update():
                if k not in self._entries:
                    self._entries[k] = e
                else:
                    self.inodes.del_entry(e)
                return True
            else:
                self.inodes.invalidate_entry(self, k)
                self.inodes.del_entry(e)
                return False
        except Exception as ex:
            _logger.exception("arv-mount lookup '%s':", k)
            if e is not None:
                self.inodes.del_entry(e)
            return False

    def __getitem__(self, item):
        if item in self:
            return self._entries[item]
        else:
            raise KeyError("No collection with id " + item)

    def clear(self):
        pass

    def want_event_subscribe(self):
        return not self.pdh_only


class TagsDirectory(Directory):
    """A special directory that contains as subdirectories all tags visible to the user."""

    def __init__(self, parent_inode, inodes, api, num_retries, poll_time=60):
        super(TagsDirectory, self).__init__(parent_inode, inodes)
        self.api = api
        self.num_retries = num_retries
        self._poll = True
        self._poll_time = poll_time
        self._extra = set()

    def want_event_subscribe(self):
        return True

    @use_counter
    def update(self):
        with llfuse.lock_released:
            tags = self.api.links().list(
                filters=[['link_class', '=', 'tag'], ["name", "!=", ""]],
                select=['name'], distinct=True, limit=1000
                ).execute(num_retries=self.num_retries)
        if "items" in tags:
            self.merge(tags['items']+[{"name": n} for n in self._extra],
                       lambda i: i['name'],
                       lambda a, i: a.tag == i['name'],
                       lambda i: TagDirectory(self.inode, self.inodes, self.api, self.num_retries, i['name'], poll=self._poll, poll_time=self._poll_time))

    @use_counter
    @check_update
    def __getitem__(self, item):
        if super(TagsDirectory, self).__contains__(item):
            return super(TagsDirectory, self).__getitem__(item)
        with llfuse.lock_released:
            tags = self.api.links().list(
                filters=[['link_class', '=', 'tag'], ['name', '=', item]], limit=1
            ).execute(num_retries=self.num_retries)
        if tags["items"]:
            self._extra.add(item)
            self.update()
        return super(TagsDirectory, self).__getitem__(item)

    @use_counter
    @check_update
    def __contains__(self, k):
        if super(TagsDirectory, self).__contains__(k):
            return True
        try:
            self[k]
            return True
        except KeyError:
            pass
        return False


class TagDirectory(Directory):
    """A special directory that contains as subdirectories all collections visible
    to the user that are tagged with a particular tag.
    """

    def __init__(self, parent_inode, inodes, api, num_retries, tag,
                 poll=False, poll_time=60):
        super(TagDirectory, self).__init__(parent_inode, inodes)
        self.api = api
        self.num_retries = num_retries
        self.tag = tag
        self._poll = poll
        self._poll_time = poll_time

    def want_event_subscribe(self):
        return True

    @use_counter
    def update(self):
        with llfuse.lock_released:
            taggedcollections = self.api.links().list(
                filters=[['link_class', '=', 'tag'],
                         ['name', '=', self.tag],
                         ['head_uuid', 'is_a', 'arvados#collection']],
                select=['head_uuid']
                ).execute(num_retries=self.num_retries)
        self.merge(taggedcollections['items'],
                   lambda i: i['head_uuid'],
                   lambda a, i: a.collection_locator == i['head_uuid'],
                   lambda i: CollectionDirectory(self.inode, self.inodes, self.api, self.num_retries, i['head_uuid']))


class ProjectDirectory(Directory):
    """A special directory that contains the contents of a project."""

    def __init__(self, parent_inode, inodes, api, num_retries, project_object,
                 poll=False, poll_time=60):
        super(ProjectDirectory, self).__init__(parent_inode, inodes)
        self.api = api
        self.num_retries = num_retries
        self.project_object = project_object
        self.project_object_file = None
        self.project_uuid = project_object['uuid']
        self._poll = poll
        self._poll_time = poll_time
        self._updating_lock = threading.Lock()
        self._current_user = None
        self._full_listing = False

    def want_event_subscribe(self):
        return True

    def createDirectory(self, i):
        if collection_uuid_pattern.match(i['uuid']):
            return CollectionDirectory(self.inode, self.inodes, self.api, self.num_retries, i)
        elif group_uuid_pattern.match(i['uuid']):
            return ProjectDirectory(self.inode, self.inodes, self.api, self.num_retries, i, self._poll, self._poll_time)
        elif link_uuid_pattern.match(i['uuid']):
            if i['head_kind'] == 'arvados#collection' or portable_data_hash_pattern.match(i['head_uuid']):
                return CollectionDirectory(self.inode, self.inodes, self.api, self.num_retries, i['head_uuid'])
            else:
                return None
        elif uuid_pattern.match(i['uuid']):
            return ObjectFile(self.parent_inode, i)
        else:
            return None

    def uuid(self):
        return self.project_uuid

    def items(self):
        self._full_listing = True
        return super(ProjectDirectory, self).items()

    def namefn(self, i):
        if 'name' in i:
            if i['name'] is None or len(i['name']) == 0:
                return None
            elif "uuid" in i and (collection_uuid_pattern.match(i['uuid']) or group_uuid_pattern.match(i['uuid'])):
                # collection or subproject
                return i['name']
            elif link_uuid_pattern.match(i['uuid']) and i['head_kind'] == 'arvados#collection':
                # name link
                return i['name']
            elif 'kind' in i and i['kind'].startswith('arvados#'):
                # something else
                return "{}.{}".format(i['name'], i['kind'][8:])
        else:
            return None


    @use_counter
    def update(self):
        if self.project_object_file == None:
            self.project_object_file = ObjectFile(self.inode, self.project_object)
            self.inodes.add_entry(self.project_object_file)

        if not self._full_listing:
            return True

        def samefn(a, i):
            if isinstance(a, CollectionDirectory) or isinstance(a, ProjectDirectory):
                return a.uuid() == i['uuid']
            elif isinstance(a, ObjectFile):
                return a.uuid() == i['uuid'] and not a.stale()
            return False

        try:
            with llfuse.lock_released:
                self._updating_lock.acquire()
                if not self.stale():
                    return

                if group_uuid_pattern.match(self.project_uuid):
                    self.project_object = self.api.groups().get(
                        uuid=self.project_uuid).execute(num_retries=self.num_retries)
                elif user_uuid_pattern.match(self.project_uuid):
                    self.project_object = self.api.users().get(
                        uuid=self.project_uuid).execute(num_retries=self.num_retries)

                contents = arvados.util.list_all(self.api.groups().list,
                                                 self.num_retries,
                                                 filters=[["owner_uuid", "=", self.project_uuid],
                                                          ["group_class", "=", "project"]])
                contents.extend(arvados.util.list_all(self.api.collections().list,
                                                      self.num_retries,
                                                      filters=[["owner_uuid", "=", self.project_uuid]]))

            # end with llfuse.lock_released, re-acquire lock

            self.merge(contents,
                       self.namefn,
                       samefn,
                       self.createDirectory)
            return True
        finally:
            self._updating_lock.release()

    def _add_entry(self, i, name):
        ent = self.createDirectory(i)
        self._entries[name] = self.inodes.add_entry(ent)
        return self._entries[name]

    @use_counter
    @check_update
    def __getitem__(self, k):
        if k == '.arvados#project':
            return self.project_object_file
        elif self._full_listing or super(ProjectDirectory, self).__contains__(k):
            return super(ProjectDirectory, self).__getitem__(k)
        with llfuse.lock_released:
            contents = self.api.groups().list(filters=[["owner_uuid", "=", self.project_uuid],
                                                       ["group_class", "=", "project"],
                                                       ["name", "=", k]],
                                              limit=1).execute(num_retries=self.num_retries)["items"]
            if not contents:
                contents = self.api.collections().list(filters=[["owner_uuid", "=", self.project_uuid],
                                                                ["name", "=", k]],
                                                       limit=1).execute(num_retries=self.num_retries)["items"]
        if contents:
            name = sanitize_filename(self.namefn(contents[0]))
            if name != k:
                raise KeyError(k)
            return self._add_entry(contents[0], name)

        # Didn't find item
        raise KeyError(k)

    def __contains__(self, k):
        if k == '.arvados#project':
            return True
        try:
            self[k]
            return True
        except KeyError:
            pass
        return False

    @use_counter
    @check_update
    def writable(self):
        with llfuse.lock_released:
            if not self._current_user:
                self._current_user = self.api.users().current().execute(num_retries=self.num_retries)
            return self._current_user["uuid"] in self.project_object.get("writable_by", [])

    def persisted(self):
        return True

    @use_counter
    @check_update
    def mkdir(self, name):
        try:
            with llfuse.lock_released:
                self.api.collections().create(body={"owner_uuid": self.project_uuid,
                                                    "name": name,
                                                    "manifest_text": ""}).execute(num_retries=self.num_retries)
            self.invalidate()
        except apiclient_errors.Error as error:
            _logger.error(error)
            raise llfuse.FUSEError(errno.EEXIST)

    @use_counter
    @check_update
    def rmdir(self, name):
        if name not in self:
            raise llfuse.FUSEError(errno.ENOENT)
        if not isinstance(self[name], CollectionDirectory):
            raise llfuse.FUSEError(errno.EPERM)
        if len(self[name]) > 0:
            raise llfuse.FUSEError(errno.ENOTEMPTY)
        with llfuse.lock_released:
            self.api.collections().delete(uuid=self[name].uuid()).execute(num_retries=self.num_retries)
        self.invalidate()

    @use_counter
    @check_update
    def rename(self, name_old, name_new, src):
        if not isinstance(src, ProjectDirectory):
            raise llfuse.FUSEError(errno.EPERM)

        ent = src[name_old]

        if not isinstance(ent, CollectionDirectory):
            raise llfuse.FUSEError(errno.EPERM)

        if name_new in self:
            # POSIX semantics for replacing one directory with another is
            # tricky (the target directory must be empty, the operation must be
            # atomic which isn't possible with the Arvados API as of this
            # writing) so don't support that.
            raise llfuse.FUSEError(errno.EPERM)

        self.api.collections().update(uuid=ent.uuid(),
                                      body={"owner_uuid": self.uuid(),
                                            "name": name_new}).execute(num_retries=self.num_retries)

        # Acually move the entry from source directory to this directory.
        del src._entries[name_old]
        self._entries[name_new] = ent
        self.inodes.invalidate_entry(src, name_old)

    @use_counter
    def child_event(self, ev):
        properties = ev.get("properties") or {}
        old_attrs = properties.get("old_attributes") or {}
        new_attrs = properties.get("new_attributes") or {}
        old_attrs["uuid"] = ev["object_uuid"]
        new_attrs["uuid"] = ev["object_uuid"]
        old_name = sanitize_filename(self.namefn(old_attrs))
        new_name = sanitize_filename(self.namefn(new_attrs))

        # create events will have a new name, but not an old name
        # delete events will have an old name, but not a new name
        # update events will have an old and new name, and they may be same or different
        # if they are the same, an unrelated field changed and there is nothing to do.

        if old_attrs.get("owner_uuid") != self.project_uuid:
            # Was moved from somewhere else, so don't try to remove entry.
            old_name = None
        if ev.get("object_owner_uuid") != self.project_uuid:
            # Was moved to somewhere else, so don't try to add entry
            new_name = None

        if old_attrs.get("is_trashed"):
            # Was previously deleted
            old_name = None
        if new_attrs.get("is_trashed"):
            # Has been deleted
            new_name = None

        if new_name != old_name:
            ent = None
            if old_name in self._entries:
                ent = self._entries[old_name]
                del self._entries[old_name]
                self.inodes.invalidate_entry(self, old_name)

            if new_name:
                if ent is not None:
                    self._entries[new_name] = ent
                else:
                    self._add_entry(new_attrs, new_name)
            elif ent is not None:
                self.inodes.del_entry(ent)


class SharedDirectory(Directory):
    """A special directory that represents users or groups who have shared projects with me."""

    def __init__(self, parent_inode, inodes, api, num_retries, exclude,
                 poll=False, poll_time=60):
        super(SharedDirectory, self).__init__(parent_inode, inodes)
        self.api = api
        self.num_retries = num_retries
        self.current_user = api.users().current().execute(num_retries=num_retries)
        self._poll = True
        self._poll_time = poll_time
        self._updating_lock = threading.Lock()

    @use_counter
    def update(self):
        try:
            with llfuse.lock_released:
                self._updating_lock.acquire()
                if not self.stale():
                    return

                contents = {}
                roots = []
                root_owners = set()
                objects = {}

                methods = self.api._rootDesc.get('resources')["groups"]['methods']
                if 'httpMethod' in methods.get('shared', {}):
                    page = []
                    while True:
                        resp = self.api.groups().shared(filters=[['group_class', '=', 'project']]+page,
                                                        order="uuid",
                                                        limit=10000,
                                                        count="none",
                                                        include="owner_uuid").execute()
                        if not resp["items"]:
                            break
                        page = [["uuid", ">", resp["items"][len(resp["items"])-1]["uuid"]]]
                        for r in resp["items"]:
                            objects[r["uuid"]] = r
                            roots.append(r["uuid"])
                        for r in resp["included"]:
                            objects[r["uuid"]] = r
                            root_owners.add(r["uuid"])
                else:
                    all_projects = arvados.util.list_all(
                        self.api.groups().list, self.num_retries,
                        filters=[['group_class','=','project']],
                        select=["uuid", "owner_uuid"])
                    for ob in all_projects:
                        objects[ob['uuid']] = ob

                    current_uuid = self.current_user['uuid']
                    for ob in all_projects:
                        if ob['owner_uuid'] != current_uuid and ob['owner_uuid'] not in objects:
                            roots.append(ob['uuid'])
                            root_owners.add(ob['owner_uuid'])

                    lusers = arvados.util.list_all(
                        self.api.users().list, self.num_retries,
                        filters=[['uuid','in', list(root_owners)]])
                    lgroups = arvados.util.list_all(
                        self.api.groups().list, self.num_retries,
                        filters=[['uuid','in', list(root_owners)+roots]])

                    for l in lusers:
                        objects[l["uuid"]] = l
                    for l in lgroups:
                        objects[l["uuid"]] = l

                for r in root_owners:
                    if r in objects:
                        obr = objects[r]
                        if obr.get("name"):
                            contents[obr["name"]] = obr
                        #elif obr.get("username"):
                        #    contents[obr["username"]] = obr
                        elif "first_name" in obr:
                            contents[u"{} {}".format(obr["first_name"], obr["last_name"])] = obr

                for r in roots:
                    if r in objects:
                        obr = objects[r]
                        if obr['owner_uuid'] not in objects:
                            contents[obr["name"]] = obr

            # end with llfuse.lock_released, re-acquire lock

            self.merge(contents.items(),
                       lambda i: i[0],
                       lambda a, i: a.uuid() == i[1]['uuid'],
                       lambda i: ProjectDirectory(self.inode, self.inodes, self.api, self.num_retries, i[1], poll=self._poll, poll_time=self._poll_time))
        except Exception:
            _logger.exception("arv-mount shared dir error")
        finally:
            self._updating_lock.release()

    def want_event_subscribe(self):
        return True
