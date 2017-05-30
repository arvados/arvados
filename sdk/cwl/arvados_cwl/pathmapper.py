import re
import logging
import uuid
import os
import urllib

import arvados.commands.run
import arvados.collection

from schema_salad.sourceline import SourceLine

from cwltool.pathmapper import PathMapper, MapperEnt, abspath, adjustFileObjs, adjustDirObjs
from cwltool.workflow import WorkflowException

logger = logging.getLogger('arvados.cwl-runner')

def trim_listing(obj):
    """Remove 'listing' field from Directory objects that are keep references.

    When Directory objects represent Keep references, it is redundant and
    potentially very expensive to pass fully enumerated Directory objects
    between instances of cwl-runner (e.g. a submitting a job, or using the
    RunInSingleContainer feature), so delete the 'listing' field when it is
    safe to do so.

    """

    if obj.get("location", "").startswith("keep:") and "listing" in obj:
        del obj["listing"]


class ArvPathMapper(PathMapper):
    """Convert container-local paths to and from Keep collection ids."""

    pdh_path = re.compile(r'^keep:[0-9a-f]{32}\+\d+/.+$')
    pdh_dirpath = re.compile(r'^keep:[0-9a-f]{32}\+\d+(/.*)?$')

    def __init__(self, arvrunner, referenced_files, input_basedir,
                 collection_pattern, file_pattern, name=None, single_collection=False, **kwargs):
        self.arvrunner = arvrunner
        self.input_basedir = input_basedir
        self.collection_pattern = collection_pattern
        self.file_pattern = file_pattern
        self.name = name
        self.referenced_files = [r["location"] for r in referenced_files]
        self.single_collection = single_collection
        super(ArvPathMapper, self).__init__(referenced_files, input_basedir, None)

    def visit(self, srcobj, uploadfiles):
        src = srcobj["location"]
        if "#" in src:
            src = src[:src.index("#")]

        if isinstance(src, basestring) and ArvPathMapper.pdh_dirpath.match(src):
            self._pathmap[src] = MapperEnt(src, self.collection_pattern % urllib.unquote(src[5:]), srcobj["class"], True)

        if src not in self._pathmap:
            if src.startswith("file:"):
                # Local FS ref, may need to be uploaded or may be on keep
                # mount.
                ab = abspath(src, self.input_basedir)
                st = arvados.commands.run.statfile("", ab,
                                                   fnPattern="keep:%s/%s",
                                                   dirPattern="keep:%s/%s",
                                                   raiseOSError=True)
                with SourceLine(srcobj, "location", WorkflowException):
                    if isinstance(st, arvados.commands.run.UploadFile):
                        uploadfiles.add((src, ab, st))
                    elif isinstance(st, arvados.commands.run.ArvFile):
                        self._pathmap[src] = MapperEnt(st.fn, self.collection_pattern % urllib.unquote(st.fn[5:]), "File", True)
                    else:
                        raise WorkflowException("Input file path '%s' is invalid" % st)
            elif src.startswith("_:"):
                if srcobj["class"] == "File" and "contents" not in srcobj:
                    raise WorkflowException("File literal '%s' is missing `contents`" % src)
                if srcobj["class"] == "Directory" and "listing" not in srcobj:
                    raise WorkflowException("Directory literal '%s' is missing `listing`" % src)
            else:
                self._pathmap[src] = MapperEnt(src, src, srcobj["class"], True)

        with SourceLine(srcobj, "secondaryFiles", WorkflowException):
            for l in srcobj.get("secondaryFiles", []):
                self.visit(l, uploadfiles)
        with SourceLine(srcobj, "listing", WorkflowException):
            for l in srcobj.get("listing", []):
                self.visit(l, uploadfiles)

    def addentry(self, obj, c, path, subdirs):
        if obj["location"] in self._pathmap:
            src, srcpath = self.arvrunner.fs_access.get_collection(self._pathmap[obj["location"]].resolved)
            if srcpath == "":
                srcpath = "."
            c.copy(srcpath, path + "/" + obj["basename"], source_collection=src, overwrite=True)
            for l in obj.get("secondaryFiles", []):
                self.addentry(l, c, path, subdirs)
        elif obj["class"] == "Directory":
            for l in obj.get("listing", []):
                self.addentry(l, c, path + "/" + obj["basename"], subdirs)
            subdirs.append((obj["location"], path + "/" + obj["basename"]))
        elif obj["location"].startswith("_:") and "contents" in obj:
            with c.open(path + "/" + obj["basename"], "w") as f:
                f.write(obj["contents"].encode("utf-8"))
        else:
            raise SourceLine(obj, "location", WorkflowException).makeError("Don't know what to do with '%s'" % obj["location"])

    def setup(self, referenced_files, basedir):
        # type: (List[Any], unicode) -> None
        uploadfiles = set()

        collection = None
        if self.single_collection:
            collection = arvados.collection.Collection(api_client=self.arvrunner.api,
                                                       keep_client=self.arvrunner.keep_client,
                                                       num_retries=self.arvrunner.num_retries)

        already_uploaded = self.arvrunner.get_uploaded()
        copied_files = set()
        for k in referenced_files:
            loc = k["location"]
            if loc in already_uploaded:
                v = already_uploaded[loc]
                self._pathmap[loc] = MapperEnt(v.resolved, self.collection_pattern % urllib.unquote(v.resolved[5:]), v.type, True)
                if self.single_collection:
                    basename = k["basename"]
                    if basename not in collection:
                        self.addentry({"location": loc, "class": v.type, "basename": basename}, collection, ".", [])
                        copied_files.add((loc, basename, v.type))

        for srcobj in referenced_files:
            self.visit(srcobj, uploadfiles)

        arvados.commands.run.uploadfiles([u[2] for u in uploadfiles],
                                         self.arvrunner.api,
                                         dry_run=False,
                                         num_retries=self.arvrunner.num_retries,
                                         fnPattern="keep:%s/%s",
                                         name=self.name,
                                         project=self.arvrunner.project_uuid,
                                         collection=collection)

        for src, ab, st in uploadfiles:
            self._pathmap[src] = MapperEnt(urllib.quote(st.fn, "/:+@"), self.collection_pattern % st.fn[5:],
                                           "Directory" if os.path.isdir(ab) else "File", True)
            self.arvrunner.add_uploaded(src, self._pathmap[src])

        for loc, basename, cls in copied_files:
            fn = "keep:%s/%s" % (collection.portable_data_hash(), basename)
            self._pathmap[loc] = MapperEnt(urllib.quote(fn, "/:+@"), self.collection_pattern % fn[5:], cls, True)

        for srcobj in referenced_files:
            subdirs = []
            if srcobj["class"] == "Directory" and srcobj["location"] not in self._pathmap:
                c = arvados.collection.Collection(api_client=self.arvrunner.api,
                                                  keep_client=self.arvrunner.keep_client,
                                                  num_retries=self.arvrunner.num_retries)
                for l in srcobj.get("listing", []):
                    self.addentry(l, c, ".", subdirs)

                check = self.arvrunner.api.collections().list(filters=[["portable_data_hash", "=", c.portable_data_hash()]], limit=1).execute(num_retries=self.arvrunner.num_retries)
                if not check["items"]:
                    c.save_new(owner_uuid=self.arvrunner.project_uuid)

                ab = self.collection_pattern % c.portable_data_hash()
                self._pathmap[srcobj["location"]] = MapperEnt("keep:"+c.portable_data_hash(), ab, "Directory", True)
            elif srcobj["class"] == "File" and (srcobj.get("secondaryFiles") or
                (srcobj["location"].startswith("_:") and "contents" in srcobj)):

                c = arvados.collection.Collection(api_client=self.arvrunner.api,
                                                  keep_client=self.arvrunner.keep_client,
                                                  num_retries=self.arvrunner.num_retries                                                  )
                self.addentry(srcobj, c, ".", subdirs)

                check = self.arvrunner.api.collections().list(filters=[["portable_data_hash", "=", c.portable_data_hash()]], limit=1).execute(num_retries=self.arvrunner.num_retries)
                if not check["items"]:
                    c.save_new(owner_uuid=self.arvrunner.project_uuid)

                ab = self.file_pattern % (c.portable_data_hash(), srcobj["basename"])
                self._pathmap[srcobj["location"]] = MapperEnt("keep:%s/%s" % (c.portable_data_hash(), srcobj["basename"]),
                                                              ab, "File", True)
                if srcobj.get("secondaryFiles"):
                    ab = self.collection_pattern % c.portable_data_hash()
                    self._pathmap["_:" + unicode(uuid.uuid4())] = MapperEnt("keep:"+c.portable_data_hash(), ab, "Directory", True)

            if subdirs:
                for loc, sub in subdirs:
                    # subdirs will all start with "./", strip it off
                    ab = self.file_pattern % (c.portable_data_hash(), sub[2:])
                    self._pathmap[loc] = MapperEnt("keep:%s/%s" % (c.portable_data_hash(), sub[2:]),
                                                   ab, "Directory", True)

        self.keepdir = None

    def reversemap(self, target):
        if target.startswith("keep:"):
            return (target, target)
        elif self.keepdir and target.startswith(self.keepdir):
            return (target, "keep:" + target[len(self.keepdir)+1:])
        else:
            return super(ArvPathMapper, self).reversemap(target)

class StagingPathMapper(PathMapper):
    _follow_dirs = True

    def visit(self, obj, stagedir, basedir, copy=False, staged=False):
        # type: (Dict[unicode, Any], unicode, unicode, bool) -> None
        loc = obj["location"]
        tgt = os.path.join(stagedir, obj["basename"])
        if obj["class"] == "Directory":
            self._pathmap[loc] = MapperEnt(loc, tgt, "Directory", staged)
            if loc.startswith("_:") or self._follow_dirs:
                self.visitlisting(obj.get("listing", []), tgt, basedir)
        elif obj["class"] == "File":
            if loc in self._pathmap:
                return
            if "contents" in obj and loc.startswith("_:"):
                self._pathmap[loc] = MapperEnt(obj["contents"], tgt, "CreateFile", staged)
            else:
                if copy:
                    self._pathmap[loc] = MapperEnt(loc, tgt, "WritableFile", staged)
                else:
                    self._pathmap[loc] = MapperEnt(loc, tgt, "File", staged)
                self.visitlisting(obj.get("secondaryFiles", []), stagedir, basedir)


class VwdPathMapper(StagingPathMapper):
    def setup(self, referenced_files, basedir):
        # type: (List[Any], unicode) -> None

        # Go through each file and set the target to its own directory along
        # with any secondary files.
        self.visitlisting(referenced_files, self.stagedir, basedir)

        for path, (ab, tgt, type, staged) in self._pathmap.items():
            if type in ("File", "Directory") and ab.startswith("keep:"):
                self._pathmap[path] = MapperEnt("$(task.keep)/%s" % ab[5:], tgt, type, staged)


class NoFollowPathMapper(StagingPathMapper):
    _follow_dirs = False
    def setup(self, referenced_files, basedir):
        # type: (List[Any], unicode) -> None
        self.visitlisting(referenced_files, self.stagedir, basedir)
