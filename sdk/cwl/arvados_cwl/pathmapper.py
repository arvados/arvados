# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import re
import logging
import uuid
import os
import urllib

import arvados_cwl.util
import arvados.commands.run
import arvados.collection

from schema_salad.sourceline import SourceLine

from arvados.errors import ApiError
from cwltool.pathmapper import PathMapper, MapperEnt, abspath, adjustFileObjs, adjustDirObjs
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


class ArvPathMapper(PathMapper):
    """Convert container-local paths to and from Keep collection ids."""

    pdh_path = re.compile(r'^keep:[0-9a-f]{32}\+\d+/.+$')
    pdh_dirpath = re.compile(r'^keep:[0-9a-f]{32}\+\d+(/.*)?$')

    def __init__(self, arvrunner, referenced_files, input_basedir,
                 collection_pattern, file_pattern, name=None, single_collection=False):
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

        debug = logger.isEnabledFor(logging.DEBUG)

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
                        self._pathmap[src] = MapperEnt(st.fn, self.collection_pattern % urllib.unquote(st.fn[5:]), "File", True)
                    else:
                        raise WorkflowException("Input file path '%s' is invalid" % st)
            elif src.startswith("_:"):
                if srcobj["class"] == "File" and "contents" not in srcobj:
                    raise WorkflowException("File literal '%s' is missing `contents`" % src)
                if srcobj["class"] == "Directory" and "listing" not in srcobj:
                    raise WorkflowException("Directory literal '%s' is missing `listing`" % src)
            elif src.startswith("http:") or src.startswith("https:"):
                keepref = http_to_keep(self.arvrunner.api, self.arvrunner.project_uuid, src)
                logger.info("%s is %s", src, keepref)
                self._pathmap[src] = MapperEnt(keepref, keepref, srcobj["class"], True)
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
                f.write(obj["contents"].encode("utf-8"))
            remap.append((obj["location"], path + "/" + obj["basename"]))
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
            self._pathmap[src] = MapperEnt(urllib.quote(st.fn, "/:+@"), self.collection_pattern % st.fn[5:],
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
            elif srcobj["class"] == "File" and (srcobj.get("secondaryFiles") or
                (srcobj["location"].startswith("_:") and "contents" in srcobj)):

                c = arvados.collection.Collection(api_client=self.arvrunner.api,
                                                  keep_client=self.arvrunner.keep_client,
                                                  num_retries=self.arvrunner.num_retries                                                  )
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
                    self._pathmap["_:" + unicode(uuid.uuid4())] = MapperEnt("keep:"+c.portable_data_hash(), ab, "Directory", True)

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
    _follow_dirs = True

    def __init__(self, referenced_files, basedir, stagedir, separateDirs=True):
        self.targets = set()
        super(StagingPathMapper, self).__init__(referenced_files, basedir, stagedir, separateDirs)

    def visit(self, obj, stagedir, basedir, copy=False, staged=False):
        # type: (Dict[unicode, Any], unicode, unicode, bool) -> None
        loc = obj["location"]
        tgt = os.path.join(stagedir, obj["basename"])
        basetgt, baseext = os.path.splitext(tgt)
        n = 1
        if tgt in self.targets and (self.reversemap(tgt)[0] != loc):
            while tgt in self.targets:
                n += 1
                tgt = "%s_%i%s" % (basetgt, n, baseext)
        self.targets.add(tgt)
        if obj["class"] == "Directory":
            if obj.get("writable"):
                self._pathmap[loc] = MapperEnt(loc, tgt, "WritableDirectory", staged)
            else:
                self._pathmap[loc] = MapperEnt(loc, tgt, "Directory", staged)
            if loc.startswith("_:") or self._follow_dirs:
                self.visitlisting(obj.get("listing", []), tgt, basedir)
        elif obj["class"] == "File":
            if loc in self._pathmap:
                return
            if "contents" in obj and loc.startswith("_:"):
                self._pathmap[loc] = MapperEnt(obj["contents"], tgt, "CreateFile", staged)
            else:
                if copy or obj.get("writable"):
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
