import logging
import re
import time
import llfuse
import arvados
import apiclient

from fusefile import StringFile, StreamReaderFile, ObjectFile
from fresh import FreshBase, convertTime

from arvados.util import portable_data_hash_pattern, uuid_pattern, collection_uuid_pattern, group_uuid_pattern, user_uuid_pattern, link_uuid_pattern

_logger = logging.getLogger('arvados.arvados_fuse')


# Match any character which FUSE or Linux cannot accommodate as part
# of a filename. (If present in a collection filename, they will
# appear as underscores in the fuse mount.)
_disallowed_filename_characters = re.compile('[\x00/]')

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

    def __init__(self, parent_inode):
        super(Directory, self).__init__()

        """parent_inode is the integer inode number"""
        self.inode = None
        if not isinstance(parent_inode, int):
            raise Exception("parent_inode should be an int")
        self.parent_inode = parent_inode
        self._entries = {}
        self._mtime = time.time()

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
                    # create new directory entry
                    ent = new_entry(i)
                    if ent is not None:
                        self._entries[name] = self.inodes.add_entry(ent)
                        changed = True

        # delete any other directory entries that were not in found in 'items'
        for i in oldentries:
            llfuse.invalidate_entry(self.inode, str(i))
            self.inodes.del_entry(oldentries[i])
            changed = True

        if changed:
            self._mtime = time.time()

        self.fresh()

    def clear(self):
        """Delete all entries"""
        oldentries = self._entries
        self._entries = {}
        for n in oldentries:
            if isinstance(n, Directory):
                n.clear()
            llfuse.invalidate_entry(self.inode, str(n))
            self.inodes.del_entry(oldentries[n])
        llfuse.invalidate_inode(self.inode)
        self.invalidate()

    def mtime(self):
        return self._mtime


class CollectionDirectory(Directory):
    """Represents the root of a directory tree holding a collection."""

    def __init__(self, parent_inode, inodes, api, num_retries, collection):
        super(CollectionDirectory, self).__init__(parent_inode)
        self.inodes = inodes
        self.api = api
        self.num_retries = num_retries
        self.collection_object_file = None
        self.collection_object = None
        if isinstance(collection, dict):
            self.collection_locator = collection['uuid']
            self._mtime = convertTime(collection.get('modified_at'))
        else:
            self.collection_locator = collection
            self._mtime = 0

    def same(self, i):
        return i['uuid'] == self.collection_locator or i['portable_data_hash'] == self.collection_locator

    # Used by arv-web.py to switch the contents of the CollectionDirectory
    def change_collection(self, new_locator):
        """Switch the contents of the CollectionDirectory.

        Must be called with llfuse.lock held.
        """

        self.collection_locator = new_locator
        self.collection_object = None
        self.update()

    def new_collection(self, new_collection_object, coll_reader):
        self.collection_object = new_collection_object

        self._mtime = convertTime(self.collection_object.get('modified_at'))

        if self.collection_object_file is not None:
            self.collection_object_file.update(self.collection_object)

        self.clear()
        for s in coll_reader.all_streams():
            cwd = self
            for part in s.name().split('/'):
                if part != '' and part != '.':
                    partname = sanitize_filename(part)
                    if partname not in cwd._entries:
                        cwd._entries[partname] = self.inodes.add_entry(Directory(cwd.inode))
                    cwd = cwd._entries[partname]
            for k, v in s.files().items():
                cwd._entries[sanitize_filename(k)] = self.inodes.add_entry(StreamReaderFile(cwd.inode, v, self.mtime()))

    def update(self):
        try:
            if self.collection_object is not None and portable_data_hash_pattern.match(self.collection_locator):
                return True

            if self.collection_locator is None:
                self.fresh()
                return True

            with llfuse.lock_released:
                coll_reader = arvados.CollectionReader(
                    self.collection_locator, self.api, self.api.keep,
                    num_retries=self.num_retries)
                new_collection_object = coll_reader.api_response() or {}
                # If the Collection only exists in Keep, there will be no API
                # response.  Fill in the fields we need.
                if 'uuid' not in new_collection_object:
                    new_collection_object['uuid'] = self.collection_locator
                if "portable_data_hash" not in new_collection_object:
                    new_collection_object["portable_data_hash"] = new_collection_object["uuid"]
                if 'manifest_text' not in new_collection_object:
                    new_collection_object['manifest_text'] = coll_reader.manifest_text()
                coll_reader.normalize()
            # end with llfuse.lock_released, re-acquire lock

            if self.collection_object is None or self.collection_object["portable_data_hash"] != new_collection_object["portable_data_hash"]:
                self.new_collection(new_collection_object, coll_reader)

            self.fresh()
            return True
        except arvados.errors.NotFoundError:
            _logger.exception("arv-mount %s: error", self.collection_locator)
        except arvados.errors.ArgumentError as detail:
            _logger.warning("arv-mount %s: error %s", self.collection_locator, detail)
            if self.collection_object is not None and "manifest_text" in self.collection_object:
                _logger.warning("arv-mount manifest_text is: %s", self.collection_object["manifest_text"])
        except Exception:
            _logger.exception("arv-mount %s: error", self.collection_locator)
            if self.collection_object is not None and "manifest_text" in self.collection_object:
                _logger.error("arv-mount manifest_text is: %s", self.collection_object["manifest_text"])
        return False

    def __getitem__(self, item):
        self.checkupdate()
        if item == '.arvados#collection':
            if self.collection_object_file is None:
                self.collection_object_file = ObjectFile(self.inode, self.collection_object)
                self.inodes.add_entry(self.collection_object_file)
            return self.collection_object_file
        else:
            return super(CollectionDirectory, self).__getitem__(item)

    def __contains__(self, k):
        if k == '.arvados#collection':
            return True
        else:
            return super(CollectionDirectory, self).__contains__(k)


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
the form '1234567890abcdefghijklmnopqrstuv+123').

Note that this directory will appear empty until you attempt to access a
specific collection subdirectory (such as trying to 'cd' into it), at which
point the collection will actually be looked up on the server and the directory
will appear if it exists.
""".lstrip()

    def __init__(self, parent_inode, inodes, api, num_retries):
        super(MagicDirectory, self).__init__(parent_inode)
        self.inodes = inodes
        self.api = api
        self.num_retries = num_retries

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
                        self.inode, self.inodes, self.api, self.num_retries))

    def __contains__(self, k):
        if k in self._entries:
            return True

        if not portable_data_hash_pattern.match(k) and not uuid_pattern.match(k):
            return False

        try:
            e = self.inodes.add_entry(CollectionDirectory(
                    self.inode, self.inodes, self.api, self.num_retries, k))
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
        if self.inode == llfuse.ROOT_INODE:
            llfuse.lock.acquire()
        try:
            super(RecursiveInvalidateDirectory, self).invalidate()
            for a in self._entries:
                self._entries[a].invalidate()
        except Exception:
            _logger.exception()
        finally:
            if self.inode == llfuse.ROOT_INODE:
                llfuse.lock.release()


class TagsDirectory(RecursiveInvalidateDirectory):
    """A special directory that contains as subdirectories all tags visible to the user."""

    def __init__(self, parent_inode, inodes, api, num_retries, poll_time=60):
        super(TagsDirectory, self).__init__(parent_inode)
        self.inodes = inodes
        self.api = api
        self.num_retries = num_retries
        self._poll = True
        self._poll_time = poll_time

    def update(self):
        with llfuse.lock_released:
            tags = self.api.links().list(
                filters=[['link_class', '=', 'tag']],
                select=['name'], distinct=True
                ).execute(num_retries=self.num_retries)
        if "items" in tags:
            self.merge(tags['items'],
                       lambda i: i['name'],
                       lambda a, i: a.tag == i['name'],
                       lambda i: TagDirectory(self.inode, self.inodes, self.api, self.num_retries, i['name'], poll=self._poll, poll_time=self._poll_time))


class TagDirectory(Directory):
    """A special directory that contains as subdirectories all collections visible
    to the user that are tagged with a particular tag.
    """

    def __init__(self, parent_inode, inodes, api, num_retries, tag,
                 poll=False, poll_time=60):
        super(TagDirectory, self).__init__(parent_inode)
        self.inodes = inodes
        self.api = api
        self.num_retries = num_retries
        self.tag = tag
        self._poll = poll
        self._poll_time = poll_time

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
        super(ProjectDirectory, self).__init__(parent_inode)
        self.inodes = inodes
        self.api = api
        self.num_retries = num_retries
        self.project_object = project_object
        self.project_object_file = None
        self.uuid = project_object['uuid']
        self._poll = poll
        self._poll_time = poll_time

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

    def update(self):
        if self.project_object_file == None:
            self.project_object_file = ObjectFile(self.inode, self.project_object)
            self.inodes.add_entry(self.project_object_file)

        def namefn(i):
            if 'name' in i:
                if i['name'] is None or len(i['name']) == 0:
                    return None
                elif collection_uuid_pattern.match(i['uuid']) or group_uuid_pattern.match(i['uuid']):
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

        def samefn(a, i):
            if isinstance(a, CollectionDirectory):
                return a.collection_locator == i['uuid']
            elif isinstance(a, ProjectDirectory):
                return a.uuid == i['uuid']
            elif isinstance(a, ObjectFile):
                return a.uuid == i['uuid'] and not a.stale()
            return False

        with llfuse.lock_released:
            if group_uuid_pattern.match(self.uuid):
                self.project_object = self.api.groups().get(
                    uuid=self.uuid).execute(num_retries=self.num_retries)
            elif user_uuid_pattern.match(self.uuid):
                self.project_object = self.api.users().get(
                    uuid=self.uuid).execute(num_retries=self.num_retries)

            contents = arvados.util.list_all(self.api.groups().contents,
                                             self.num_retries, uuid=self.uuid)
            # Name links will be obsolete soon, take this out when there are no more pre-#3036 in use.
            contents += arvados.util.list_all(
                self.api.links().list, self.num_retries,
                filters=[['tail_uuid', '=', self.uuid],
                         ['link_class', '=', 'name']])

        # end with llfuse.lock_released, re-acquire lock

        self.merge(contents,
                   namefn,
                   samefn,
                   self.createDirectory)

    def __getitem__(self, item):
        self.checkupdate()
        if item == '.arvados#project':
            return self.project_object_file
        else:
            return super(ProjectDirectory, self).__getitem__(item)

    def __contains__(self, k):
        if k == '.arvados#project':
            return True
        else:
            return super(ProjectDirectory, self).__contains__(k)


class SharedDirectory(Directory):
    """A special directory that represents users or groups who have shared projects with me."""

    def __init__(self, parent_inode, inodes, api, num_retries, exclude,
                 poll=False, poll_time=60):
        super(SharedDirectory, self).__init__(parent_inode)
        self.inodes = inodes
        self.api = api
        self.num_retries = num_retries
        self.current_user = api.users().current().execute(num_retries=num_retries)
        self._poll = True
        self._poll_time = poll_time

    def update(self):
        with llfuse.lock_released:
            all_projects = arvados.util.list_all(
                self.api.groups().list, self.num_retries,
                filters=[['group_class','=','project']])
            objects = {}
            for ob in all_projects:
                objects[ob['uuid']] = ob

            roots = []
            root_owners = {}
            for ob in all_projects:
                if ob['owner_uuid'] != self.current_user['uuid'] and ob['owner_uuid'] not in objects:
                    roots.append(ob)
                    root_owners[ob['owner_uuid']] = True

            lusers = arvados.util.list_all(
                self.api.users().list, self.num_retries,
                filters=[['uuid','in', list(root_owners)]])
            lgroups = arvados.util.list_all(
                self.api.groups().list, self.num_retries,
                filters=[['uuid','in', list(root_owners)]])

            users = {}
            groups = {}

            for l in lusers:
                objects[l["uuid"]] = l
            for l in lgroups:
                objects[l["uuid"]] = l

            contents = {}
            for r in root_owners:
                if r in objects:
                    obr = objects[r]
                    if "name" in obr:
                        contents[obr["name"]] = obr
                    if "first_name" in obr:
                        contents[u"{} {}".format(obr["first_name"], obr["last_name"])] = obr

            for r in roots:
                if r['owner_uuid'] not in objects:
                    contents[r['name']] = r

        # end with llfuse.lock_released, re-acquire lock

        try:
            self.merge(contents.items(),
                       lambda i: i[0],
                       lambda a, i: a.uuid == i[1]['uuid'],
                       lambda i: ProjectDirectory(self.inode, self.inodes, self.api, self.num_retries, i[1], poll=self._poll, poll_time=self._poll_time))
        except Exception:
            _logger.exception()
