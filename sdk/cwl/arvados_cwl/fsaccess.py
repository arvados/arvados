import fnmatch
import os
import errno
import urlparse
import re

import ruamel.yaml as yaml

import cwltool.stdfsaccess
from cwltool.pathmapper import abspath
import cwltool.resolver

import arvados.util
import arvados.collection
import arvados.arvfile

from schema_salad.ref_resolver import DefaultFetcher

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

class CollectionFetcher(DefaultFetcher):
    def __init__(self, cache, session, api_client=None, keep_client=None):
        super(CollectionFetcher, self).__init__(cache, session)
        self.api_client = api_client
        self.fsaccess = CollectionFsAccess("", api_client=api_client, keep_client=keep_client)

    def fetch_text(self, url):
        if url.startswith("keep:"):
            with self.fsaccess.open(url, "r") as f:
                return f.read()
        if url.startswith("arvwf:"):
            return self.api_client.workflows().get(uuid=url[6:]).execute()["definition"]
        return super(CollectionFetcher, self).fetch_text(url)

    def check_exists(self, url):
        if url.startswith("keep:"):
            return self.fsaccess.exists(url)
        if url.startswith("arvwf:"):
            if self.fetch_text(url):
                return True
        return super(CollectionFetcher, self).check_exists(url)

    def urljoin(self, base_url, url):
        if not url:
            return base_url

        urlsp = urlparse.urlsplit(url)
        if urlsp.scheme or not base_url:
            return url

        basesp = urlparse.urlsplit(base_url)
        if basesp.scheme in ("keep", "arvwf"):
            if not basesp.path:
                raise IOError(errno.EINVAL, "Invalid Keep locator", base_url)

            baseparts = basesp.path.split("/")
            urlparts = urlsp.path.split("/") if urlsp.path else []

            pdh = baseparts.pop(0)

            if basesp.scheme == "keep" and not arvados.util.keep_locator_pattern.match(pdh):
                raise IOError(errno.EINVAL, "Invalid Keep locator", base_url)

            if urlsp.path.startswith("/"):
                baseparts = []
                urlparts.pop(0)

            if baseparts and urlsp.path:
                baseparts.pop()

            path = "/".join([pdh] + baseparts + urlparts)
            return urlparse.urlunsplit((basesp.scheme, "", path, "", urlsp.fragment))

        return super(CollectionFetcher, self).urljoin(base_url, url)

workflow_uuid_pattern = re.compile(r'[a-z0-9]{5}-7fd4e-[a-z0-9]{15}')
pipeline_template_uuid_pattern = re.compile(r'[a-z0-9]{5}-p5p6p-[a-z0-9]{15}')

def collectionResolver(api_client, document_loader, uri):
    if workflow_uuid_pattern.match(uri):
        return "arvwf:%s#main" % (uri)

    if pipeline_template_uuid_pattern.match(uri):
        pt = api_client.pipeline_templates().get(uuid=uri).execute()
        return "keep:" + pt["components"].values()[0]["script_parameters"]["cwl:tool"]

    p = uri.split("/")
    if arvados.util.keep_locator_pattern.match(p[0]):
        return "keep:%s" % (uri)

    if arvados.util.collection_uuid_pattern.match(p[0]):
        return "keep:%s%s" % (api_client.collections().
                              get(uuid=p[0]).execute()["portable_data_hash"],
                              uri[len(p[0]):])

    return cwltool.resolver.tool_resolver(document_loader, uri)
