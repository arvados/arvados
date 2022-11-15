# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from future import standard_library
standard_library.install_aliases()
from builtins import str
from past.builtins import basestring
from future.utils import viewitems

import re
import logging
import uuid
import os
import urllib.request, urllib.parse, urllib.error

import arvados_cwl.util
import arvados.commands.run
import arvados.collection

from schema_salad.sourceline import SourceLine

from arvados.errors import ApiError
from cwltool.pathmapper import PathMapper, MapperEnt
from cwltool.utils import adjustFileObjs, adjustDirObjs
from cwltool.stdfsaccess import abspath
from cwltool.workflow import WorkflowException

from .http import http_to_keep

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

collection_pdh_path = re.compile(r'^keep:[0-9a-f]{32}\+\d+/.+$')
collection_pdh_pattern = re.compile(r'^keep:([0-9a-f]{32}\+\d+)(/.*)?')
collection_uuid_pattern = re.compile(r'^keep:([a-z0-9]{5}-4zz18-[a-z0-9]{15})(/.*)?$')

class ArvPathMapper(PathMapper):
    """Convert container-local paths to and from Keep collection ids."""

    def __init__(self, arvrunner, referenced_files, input_basedir,
                 collection_pattern, file_pattern, name=None, single_collection=False,
                 optional_deps=None):
        self.arvrunner = arvrunner
        self.input_basedir = input_basedir
        self.collection_pattern = collection_pattern
        self.file_pattern = file_pattern
        self.name = name
        self.referenced_files = [r["location"] for r in referenced_files]
        self.single_collection = single_collection
        self.pdh_to_uuid = {}
        self.optional_deps = optional_deps or []
        super(ArvPathMapper, self).__init__(referenced_files, input_basedir, None)

    def visit(self, srcobj, uploadfiles):
        src = srcobj["location"]
        if "#" in src:
            src = src[:src.index("#")]

        debug = logger.isEnabledFor(logging.DEBUG)

        if isinstance(src, basestring) and src.startswith("keep:"):
            if collection_pdh_pattern.match(src):
                self._pathmap[src] = MapperEnt(src, self.collection_pattern % urllib.parse.unquote(src[5:]), srcobj["class"], True)

                if arvados_cwl.util.collectionUUID in srcobj:
                    self.pdh_to_uuid[src.split("/", 1)[0][5:]] = srcobj[arvados_cwl.util.collectionUUID]
            elif not collection_uuid_pattern.match(src):
                with SourceLine(srcobj, "location", WorkflowException, debug):
                    raise WorkflowException("Invalid keep reference '%s'" % src)

        if src not in self._pathmap:
            if src.startswith("file:"):
                # Local FS ref, may need to be uploaded or may be on keep
                # mount.
                ab = abspath(src, self.input_basedir)
                st = arvados.commands.run.statfile("", ab,
                                                   fnPattern="keep:%s/%s",
                                                   dirPattern="keep:%s/%s",
                                                   raiseOSError=True)
                with SourceLine(srcobj, "location", WorkflowException, debug):
                    if isinstance(st, arvados.commands.run.UploadFile):
                        uploadfiles.add((src, ab, st))
                    elif isinstance(st, arvados.commands.run.ArvFile):
                        self._pathmap[src] = MapperEnt(st.fn, self.collection_pattern % urllib.parse.unquote(st.fn[5:]), "File", True)
                    else:
                        raise WorkflowException("Input file path '%s' is invalid" % st)
            elif src.startswith("_:"):
                if srcobj["class"] == "File" and "contents" not in srcobj:
                    raise WorkflowException("File literal '%s' is missing `contents`" % src)
                if srcobj["class"] == "Directory" and "listing" not in srcobj:
                    raise WorkflowException("Directory literal '%s' is missing `listing`" % src)
            elif src.startswith("http:") or src.startswith("https:"):
                try:
                    if self.arvrunner.defer_downloads:
                        # passthrough, we'll download it later.
                        self._pathmap[src] = MapperEnt(src, src, srcobj["class"], True)
                    else:
                        keepref = http_to_keep(self.arvrunner.api, self.arvrunner.project_uuid, src,
                                               varying_url_params=self.arvrunner.toplevel_runtimeContext.varying_url_params,
                                               prefer_cached_downloads=self.arvrunner.toplevel_runtimeContext.prefer_cached_downloads)
                        logger.info("%s is %s", src, keepref)
                        self._pathmap[src] = MapperEnt(keepref, keepref, srcobj["class"], True)
                except Exception as e:
                    logger.warning(str(e))
            else:
                self._pathmap[src] = MapperEnt(src, src, srcobj["class"], True)

        with SourceLine(srcobj, "secondaryFiles", WorkflowException, debug):
            for l in srcobj.get("secondaryFiles", []):
                self.visit(l, uploadfiles)
        with SourceLine(srcobj, "listing", WorkflowException, debug):
            for l in srcobj.get("listing", []):
                self.visit(l, uploadfiles)

    def addentry(self, obj, c, path, remap):
        if obj["location"] in self._pathmap:
            src, srcpath = self.arvrunner.fs_access.get_collection(self._pathmap[obj["location"]].resolved)
            if srcpath == "":
                srcpath = "."
            c.copy(srcpath, path + "/" + obj["basename"], source_collection=src, overwrite=True)
            remap.append((obj["location"], path + "/" + obj["basename"]))
            for l in obj.get("secondaryFiles", []):
                self.addentry(l, c, path, remap)
        elif obj["class"] == "Directory":
            for l in obj.get("listing", []):
                self.addentry(l, c, path + "/" + obj["basename"], remap)
            remap.append((obj["location"], path + "/" + obj["basename"]))
        elif obj["location"].startswith("_:") and "contents" in obj:
            with c.open(path + "/" + obj["basename"], "w") as f:
                f.write(obj["contents"])
            remap.append((obj["location"], path + "/" + obj["basename"]))
        else:
            for opt in self.optional_deps:
                if obj["location"] == opt["location"]:
                    return
            raise SourceLine(obj, "location", WorkflowException).makeError("Don't know what to do with '%s'" % obj["location"])

    def needs_new_collection(self, srcobj, prefix=""):
        """Check if files need to be staged into a new collection.

        If all the files are in the same collection and in the same
        paths they would be staged to, return False.  Otherwise, a new
        collection is needed with files copied/created in the
        appropriate places.
        """

        loc = srcobj["location"]
        if loc.startswith("_:"):
            return True

        if self.arvrunner.defer_downloads and (loc.startswith("http:") or loc.startswith("https:")):
            return False

        i = loc.rfind("/")
        if i > -1:
            loc_prefix = loc[:i+1]
            if not prefix:
                prefix = loc_prefix
            # quote/unquote to ensure consistent quoting
            suffix = urllib.parse.quote(urllib.parse.unquote(loc[i+1:]), "/+@")
        else:
            # no '/' found
            loc_prefix = loc+"/"
            prefix = loc+"/"
            suffix = ""

        if prefix != loc_prefix:
            return True

        if "basename" in srcobj and suffix != urllib.parse.quote(srcobj["basename"], "/+@"):
            return True

        if srcobj["class"] == "File" and loc not in self._pathmap:
            return True
        for s in srcobj.get("secondaryFiles", []):
            if self.needs_new_collection(s, prefix):
                return True
        if srcobj.get("listing"):
            prefix = "%s%s/" % (prefix, urllib.parse.quote(srcobj.get("basename", suffix), "/+@"))
            for l in srcobj["listing"]:
                if self.needs_new_collection(l, prefix):
                    return True
        return False

    def setup(self, referenced_files, basedir):
        # type: (List[Any], unicode) -> None
        uploadfiles = set()

        collection = None
        if self.single_collection:
            collection = arvados.collection.Collection(api_client=self.arvrunner.api,
                                                       keep_client=self.arvrunner.keep_client,
                                                       num_retries=self.arvrunner.num_retries)

        for srcobj in referenced_files:
            self.visit(srcobj, uploadfiles)

        arvados.commands.run.uploadfiles([u[2] for u in uploadfiles],
                                         self.arvrunner.api,
                                         dry_run=False,
                                         num_retries=self.arvrunner.num_retries,
                                         fnPattern="keep:%s/%s",
                                         name=self.name,
                                         project=self.arvrunner.project_uuid,
                                         collection=collection,
                                         packed=False)

        for src, ab, st in uploadfiles:
            self._pathmap[src] = MapperEnt(urllib.parse.quote(st.fn, "/:+@"), urllib.parse.quote(self.collection_pattern % st.fn[5:], "/:+@"),
                                           "Directory" if os.path.isdir(ab) else "File", True)

        for srcobj in referenced_files:
            remap = []
            if srcobj["class"] == "Directory" and srcobj["location"] not in self._pathmap:
                c = arvados.collection.Collection(api_client=self.arvrunner.api,
                                                  keep_client=self.arvrunner.keep_client,
                                                  num_retries=self.arvrunner.num_retries)
                for l in srcobj.get("listing", []):
                    self.addentry(l, c, ".", remap)

                container = arvados_cwl.util.get_current_container(self.arvrunner.api, self.arvrunner.num_retries, logger)
                info = arvados_cwl.util.get_intermediate_collection_info(None, container, self.arvrunner.intermediate_output_ttl)

                c.save_new(name=info["name"],
                           owner_uuid=self.arvrunner.project_uuid,
                           ensure_unique_name=True,
                           trash_at=info["trash_at"],
                           properties=info["properties"])

                ab = self.collection_pattern % c.portable_data_hash()
                self._pathmap[srcobj["location"]] = MapperEnt("keep:"+c.portable_data_hash(), ab, "Directory", True)
            elif srcobj["class"] == "File" and self.needs_new_collection(srcobj):
                c = arvados.collection.Collection(api_client=self.arvrunner.api,
                                                  keep_client=self.arvrunner.keep_client,
                                                  num_retries=self.arvrunner.num_retries)
                self.addentry(srcobj, c, ".", remap)

                container = arvados_cwl.util.get_current_container(self.arvrunner.api, self.arvrunner.num_retries, logger)
                info = arvados_cwl.util.get_intermediate_collection_info(None, container, self.arvrunner.intermediate_output_ttl)

                c.save_new(name=info["name"],
                           owner_uuid=self.arvrunner.project_uuid,
                           ensure_unique_name=True,
                           trash_at=info["trash_at"],
                           properties=info["properties"])

                ab = self.file_pattern % (c.portable_data_hash(), srcobj["basename"])
                self._pathmap[srcobj["location"]] = MapperEnt("keep:%s/%s" % (c.portable_data_hash(), srcobj["basename"]),
                                                              ab, "File", True)
                if srcobj.get("secondaryFiles"):
                    ab = self.collection_pattern % c.portable_data_hash()
                    self._pathmap["_:" + str(uuid.uuid4())] = MapperEnt("keep:"+c.portable_data_hash(), ab, "Directory", True)

            if remap:
                for loc, sub in remap:
                    # subdirs start with "./", strip it off
                    if sub.startswith("./"):
                        ab = self.file_pattern % (c.portable_data_hash(), sub[2:])
                    else:
                        ab = self.file_pattern % (c.portable_data_hash(), sub)
                    self._pathmap[loc] = MapperEnt("keep:%s/%s" % (c.portable_data_hash(), sub[2:]),
                                                   ab, "Directory", True)

        self.keepdir = None

    def reversemap(self, target):
        p = super(ArvPathMapper, self).reversemap(target)
        if p:
            return p
        elif target.startswith("keep:"):
            return (target, target)
        elif self.keepdir and target.startswith(self.keepdir):
            kp = "keep:" + target[len(self.keepdir)+1:]
            return (kp, kp)
        else:
            return None


class StagingPathMapper(PathMapper):
    # Note that StagingPathMapper internally maps files from target to source.
    # Specifically, the 'self._pathmap' dict keys are the target location and the
    # values are 'MapperEnt' named tuples from which we use the 'resolved' attribute
    # as the file identifier. This makes it possible to map an input file to multiple
    # target directories. The exception is for file literals, which store the contents of
    # the file in 'MapperEnt.resolved' and are therefore still mapped from source to target.

    _follow_dirs = True

    def __init__(self, referenced_files, basedir, stagedir, separateDirs=True):
        self.targets = set()
        super(StagingPathMapper, self).__init__(referenced_files, basedir, stagedir, separateDirs)

    def visit(self, obj, stagedir, basedir, copy=False, staged=False):
        # type: (Dict[unicode, Any], unicode, unicode, bool) -> None
        loc = obj["location"]
        stagedir = obj.get("dirname") or stagedir
        tgt = os.path.join(stagedir, obj["basename"])
        basetgt, baseext = os.path.splitext(tgt)

        def targetExists():
            return tgt in self.targets and ("contents" not in obj) and (self._pathmap[tgt].resolved != loc)
        def literalTargetExists():
            return tgt in self.targets and "contents" in obj

        n = 1
        if targetExists() or literalTargetExists():
            while tgt in self.targets:
                n += 1
                tgt = "%s_%i%s" % (basetgt, n, baseext)
        self.targets.add(tgt)
        if obj["class"] == "Directory":
            if obj.get("writable"):
                self._pathmap[tgt] = MapperEnt(loc, tgt, "WritableDirectory", staged)
            else:
                self._pathmap[tgt] = MapperEnt(loc, tgt, "Directory", staged)
            if loc.startswith("_:") or self._follow_dirs:
                self.visitlisting(obj.get("listing", []), tgt, basedir)
        elif obj["class"] == "File":
            if tgt in self._pathmap:
                return
            if "contents" in obj and loc.startswith("_:"):
                self._pathmap[loc] = MapperEnt(obj["contents"], tgt, "CreateFile", staged)
            else:
                if copy or obj.get("writable"):
                    self._pathmap[tgt] = MapperEnt(loc, tgt, "WritableFile", staged)
                else:
                    self._pathmap[tgt] = MapperEnt(loc, tgt, "File", staged)
                self.visitlisting(obj.get("secondaryFiles", []), stagedir, basedir)

    def mapper(self, src):  # type: (Text) -> MapperEnt.
        # Overridden to maintain the use case of mapping by source (identifier) to
        # target regardless of how the map is structured interally.
        def getMapperEnt(src):
            for k,v in viewitems(self._pathmap):
                if (v.type != "CreateFile" and v.resolved == src) or (v.type == "CreateFile" and k == src):
                    return v

        if u"#" in src:
            i = src.index(u"#")
            v = getMapperEnt(src[i:])
            return MapperEnt(v.resolved, v.target + src[i:], v.type, v.staged)
        return getMapperEnt(src)


class VwdPathMapper(StagingPathMapper):
    def setup(self, referenced_files, basedir):
        # type: (List[Any], unicode) -> None

        # Go through each file and set the target to its own directory along
        # with any secondary files.
        self.visitlisting(referenced_files, self.stagedir, basedir)

        for path, (ab, tgt, type, staged) in viewitems(self._pathmap):
            if type in ("File", "Directory") and ab.startswith("keep:"):
                self._pathmap[path] = MapperEnt("$(task.keep)/%s" % ab[5:], tgt, type, staged)


class NoFollowPathMapper(StagingPathMapper):
    _follow_dirs = False
    def setup(self, referenced_files, basedir):
        # type: (List[Any], unicode) -> None
        self.visitlisting(referenced_files, self.stagedir, basedir)
