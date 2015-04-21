import logging
import re
import json

from fresh import FreshBase, convertTime

_logger = logging.getLogger('arvados.arvados_fuse')

class File(FreshBase):
    """Base for file objects."""

    def __init__(self, parent_inode, _mtime=0):
        super(File, self).__init__()
        self.inode = None
        self.parent_inode = parent_inode
        self._mtime = _mtime

    def size(self):
        return 0

    def readfrom(self, off, size, num_retries=0):
        return ''

    def writeto(self, off, size, num_retries=0):
        raise Exception("Not writable")

    def mtime(self):
        return self._mtime

    def clear(self, force=False):
        return True

    def writable(self):
        return False

    def flush(self):
        pass

class FuseArvadosFile(File):
    """Wraps a ArvadosFile."""

    def __init__(self, parent_inode, arvfile, _mtime):
        super(FuseArvadosFile, self).__init__(parent_inode, _mtime)
        self.arvfile = arvfile

    def size(self):
        return self.arvfile.size()

    def readfrom(self, off, size, num_retries=0):
        return self.arvfile.readfrom(off, size, num_retries, exact=True)

    def writeto(self, off, buf, num_retries=0):
        return self.arvfile.writeto(off, buf, num_retries)

    def stale(self):
        return False

    def writable(self):
        return self.arvfile.writable()

    def flush(self):
        if self.writable():
            self.arvfile.parent.root_collection().save()


class StringFile(File):
    """Wrap a simple string as a file"""
    def __init__(self, parent_inode, contents, _mtime):
        super(StringFile, self).__init__(parent_inode, _mtime)
        self.contents = contents

    def size(self):
        return len(self.contents)

    def readfrom(self, off, size, num_retries=0):
        return self.contents[off:(off+size)]


class ObjectFile(StringFile):
    """Wrap a dict as a serialized json object."""

    def __init__(self, parent_inode, obj):
        super(ObjectFile, self).__init__(parent_inode, "", 0)
        self.uuid = obj['uuid']
        self.update(obj)

    def update(self, obj):
        self._mtime = convertTime(obj['modified_at']) if 'modified_at' in obj else 0
        self.contents = json.dumps(obj, indent=4, sort_keys=True) + "\n"
