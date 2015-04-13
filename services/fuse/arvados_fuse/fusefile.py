import logging
import re

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

    def readfrom(self, off, size):
        return ''

    def mtime(self):
        return self._mtime


class StreamReaderFile(File):
    """Wraps a StreamFileReader as a file."""

    def __init__(self, parent_inode, reader, _mtime):
        super(StreamReaderFile, self).__init__(parent_inode, _mtime)
        self.reader = reader

    def size(self):
        return self.reader.size()

    def readfrom(self, off, size):
        return self.reader.readfrom(off, size)

    def stale(self):
        return False


class StringFile(File):
    """Wrap a simple string as a file"""
    def __init__(self, parent_inode, contents, _mtime):
        super(StringFile, self).__init__(parent_inode, _mtime)
        self.contents = contents

    def size(self):
        return len(self.contents)

    def readfrom(self, off, size):
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
