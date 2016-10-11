import fnmatch
import os
import errno

import cwltool.stdfsaccess
from cwltool.pathmapper import abspath

import arvados.util
import arvados.collection
import arvados.arvfile

class CollectionFsAccess(cwltool.stdfsaccess.StdFsAccess):
    """Implement the cwltool FsAccess interface for Arvados Collections."""

    def __init__(self, basedir, api_client=None, keep_client=None):
        super(CollectionFsAccess, self).__init__(basedir)
        self.api_client = api_client
        self.keep_client = keep_client
        self.collections = {}

    def get_collection(self, path):
        p = path.split("/")
        if p[0].startswith("keep:") and arvados.util.keep_locator_pattern.match(p[0][5:]):
            pdh = p[0][5:]
            if pdh not in self.collections:
                self.collections[pdh] = arvados.collection.CollectionReader(pdh, api_client=self.api_client,
                                                                            keep_client=self.keep_client)
            return (self.collections[pdh], "/".join(p[1:]))
        else:
            return (None, path)

    def _match(self, collection, patternsegments, parent):
        if not patternsegments:
            return []

        if not isinstance(collection, arvados.collection.RichCollectionBase):
            return []

        ret = []
        # iterate over the files and subcollections in 'collection'
        for filename in collection:
            if patternsegments[0] == '.':
                # Pattern contains something like "./foo" so just shift
                # past the "./"
                ret.extend(self._match(collection, patternsegments[1:], parent))
            elif fnmatch.fnmatch(filename, patternsegments[0]):
                cur = os.path.join(parent, filename)
                if len(patternsegments) == 1:
                    ret.append(cur)
                else:
                    ret.extend(self._match(collection[filename], patternsegments[1:], cur))
        return ret

    def glob(self, pattern):
        collection, rest = self.get_collection(pattern)
        if collection and not rest:
            return [pattern]
        patternsegments = rest.split("/")
        return self._match(collection, patternsegments, "keep:" + collection.manifest_locator())

    def open(self, fn, mode):
        collection, rest = self.get_collection(fn)
        if collection:
            return collection.open(rest, mode)
        else:
            return super(CollectionFsAccess, self).open(self._abs(fn), mode)

    def exists(self, fn):
        collection, rest = self.get_collection(fn)
        if collection:
            return collection.exists(rest)
        else:
            return super(CollectionFsAccess, self).exists(fn)

    def isfile(self, fn):  # type: (unicode) -> bool
        collection, rest = self.get_collection(fn)
        if collection:
            if rest:
                return isinstance(collection.find(rest), arvados.arvfile.ArvadosFile)
            else:
                return False
        else:
            return super(CollectionFsAccess, self).isfile(fn)

    def isdir(self, fn):  # type: (unicode) -> bool
        collection, rest = self.get_collection(fn)
        if collection:
            if rest:
                return isinstance(collection.find(rest), arvados.collection.RichCollectionBase)
            else:
                return True
        else:
            return super(CollectionFsAccess, self).isdir(fn)

    def listdir(self, fn):  # type: (unicode) -> List[unicode]
        collection, rest = self.get_collection(fn)
        if collection:
            if rest:
                dir = collection.find(rest)
            else:
                dir = collection
            if dir is None:
                raise IOError(errno.ENOENT, "Directory '%s' in '%s' not found" % (rest, collection.portable_data_hash()))
            if not isinstance(dir, arvados.collection.RichCollectionBase):
                raise IOError(errno.ENOENT, "Path '%s' in '%s' is not a Directory" % (rest, collection.portable_data_hash()))
            return [abspath(l, fn) for l in dir.keys()]
        else:
            return super(CollectionFsAccess, self).listdir(fn)

    def join(self, path, *paths): # type: (unicode, *unicode) -> unicode
        if paths and paths[-1].startswith("keep:") and arvados.util.keep_locator_pattern.match(paths[-1][5:]):
            return paths[-1]
        return os.path.join(path, *paths)

    def realpath(self, path):
        if path.startswith("$(task.tmpdir)") or path.startswith("$(task.outdir)"):
            return path
        collection, rest = self.get_collection(path)
        if collection:
            return path
        else:
            return os.path.realpath(path)
