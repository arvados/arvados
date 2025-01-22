# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import json
import llfuse
import logging
import re
import time

from .fresh import FreshBase, convertTime, check_update

_logger = logging.getLogger('arvados.arvados_fuse')

class File(FreshBase):
    """Base for file objects."""

    __slots__ = ("inode", "parent_inode", "_mtime")

    def __init__(self, parent_inode, _mtime=0, poll=False, poll_time=0):
        super(File, self).__init__()
        self.inode = None
        self.parent_inode = parent_inode
        self._mtime = _mtime
        self._poll = poll
        self._poll_time = poll_time

    def size(self):
        return 0

    def readfrom(self, off, size, num_retries=0):
        return ''

    def writeto(self, off, size, num_retries=0):
        raise Exception("Not writable")

    def mtime(self):
        return self._mtime

    def clear(self):
        pass

    def writable(self):
        return False

    def flush(self):
        pass


class FuseArvadosFile(File):
    """Wraps a ArvadosFile."""

    __slots__ = ('arvfile', '_enable_write')

    def __init__(self, parent_inode, arvfile, _mtime, enable_write, poll, poll_time):
        super(FuseArvadosFile, self).__init__(parent_inode, _mtime, poll=poll, poll_time=poll_time)
        self.arvfile = arvfile
        self._enable_write = enable_write

    def size(self):
        with llfuse.lock_released:
            return self.arvfile.size()

    def readfrom(self, off, size, num_retries=0):
        with llfuse.lock_released:
            return self.arvfile.readfrom(off, size, num_retries, exact=True, return_memoryview=True)

    def writeto(self, off, buf, num_retries=0):
        with llfuse.lock_released:
            return self.arvfile.writeto(off, buf, num_retries)

    def writable(self):
        return self._enable_write and self.arvfile.writable()

    def flush(self):
        with llfuse.lock_released:
            if self.writable():
                self.arvfile.parent.root_collection().save()

    def clear(self):
        if self.parent_inode is None:
            self.arvfile.fuse_entry = None
            self.arvfile = None


class StringFile(File):
    """Wrap a simple string as a file"""

    __slots__ = ("contents",)

    def __init__(self, parent_inode, contents, _mtime):
        super(StringFile, self).__init__(parent_inode, _mtime)
        self.contents = contents

    def size(self):
        return len(self.contents)

    def readfrom(self, off, size, num_retries=0):
        return bytes(self.contents[off:(off+size)], encoding='utf-8')


class ObjectFile(StringFile):
    """Wrap a dict as a serialized json object."""

    __slots__ = ("object_uuid",)

    def __init__(self, parent_inode, obj):
        super(ObjectFile, self).__init__(parent_inode, "", 0)
        self.object_uuid = obj['uuid']
        self.update(obj)

    def uuid(self):
        return self.object_uuid

    def update(self, obj=None):
        if obj is None:
            # TODO: retrieve the current record for self.object_uuid
            # from the server. For now, at least don't crash when
            # someone tells us it's a good time to update but doesn't
            # pass us a fresh obj. See #8345
            return
        self._mtime = convertTime(obj['modified_at']) if 'modified_at' in obj else 0
        self.contents = json.dumps(obj, indent=4, sort_keys=True) + "\n"

    def persisted(self):
        return True


class FuncToJSONFile(StringFile):
    """File content is the return value of a given function, encoded as JSON.

    The function is called at the time the file is read. The result is
    cached until invalidate() is called.
    """

    __slots__ = ("func",)

    def __init__(self, parent_inode, func):
        super(FuncToJSONFile, self).__init__(parent_inode, "", 0)
        self.func = func

        # invalidate_inode() is asynchronous with no callback to wait for. In
        # order to guarantee userspace programs don't get stale data that was
        # generated before the last invalidate(), we must disallow inode
        # caching entirely.
        self.allow_attr_cache = False

    @check_update
    def size(self):
        return super(FuncToJSONFile, self).size()

    @check_update
    def readfrom(self, *args, **kwargs):
        return super(FuncToJSONFile, self).readfrom(*args, **kwargs)

    def update(self):
        self._mtime = time.time()
        obj = self.func()
        self.contents = json.dumps(obj, indent=4, sort_keys=True) + "\n"
        self.fresh()
