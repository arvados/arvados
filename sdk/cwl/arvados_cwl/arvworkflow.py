# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from past.builtins import basestring
from future.utils import viewitems

import os
import json
import copy
import logging
import urllib
from io import StringIO
import sys
import re

from typing import (MutableSequence, MutableMapping)

from ruamel.yaml import YAML
from ruamel.yaml.comments import CommentedMap, CommentedSeq

from schema_salad.sourceline import SourceLine, cmap
import schema_salad.ref_resolver

import arvados.collection

from cwltool.pack import pack
from cwltool.load_tool import fetch_document, resolve_and_validate_document
from cwltool.process import shortname, uniquename
from cwltool.workflow import Workflow, WorkflowException, WorkflowStep
from cwltool.utils import adjustFileObjs, adjustDirObjs, visit_class, normalizeFilesDirs
from cwltool.context import LoadingContext

from schema_salad.ref_resolver import file_uri, uri_file_path

import ruamel.yaml as yaml

from .runner import (upload_dependencies, packed_workflow, upload_workflow_collection,
                     trim_anonymous_location, remove_redundant_fields, discover_secondary_files,
                     make_builder, arvados_jobs_image, FileUpdates)
from .pathmapper import ArvPathMapper, trim_listing
from .arvtool import ArvadosCommandTool, set_cluster_target
from ._version import __version__

from .perf import Perf

logger = logging.getLogger('arvados.cwl-runner')
metrics = logging.getLogger('arvados.cwl-runner.metrics')

max_res_pars = ("coresMin", "coresMax", "ramMin", "ramMax", "tmpdirMin", "tmpdirMax")
sum_res_pars = ("outdirMin", "outdirMax")

def make_wrapper_workflow(arvRunner, main, packed, project_uuid, name, git_info, tool):
    col = arvados.collection.Collection(api_client=arvRunner.api,
                                        keep_client=arvRunner.keep_client)

    with col.open("workflow.json", "wt") as f:
        json.dump(packed, f, sort_keys=True, indent=4, separators=(',',': '))

    pdh = col.portable_data_hash()

    toolname = tool.tool.get("label") or tool.metadata.get("label") or os.path.basename(tool.tool["id"])
    if git_info and git_info.get("http://arvados.org/cwl#gitDescribe"):
        toolname = "%s (%s)" % (toolname, git_info.get("http://arvados.org/cwl#gitDescribe"))

    existing = arvRunner.api.collections().list(filters=[["portable_data_hash", "=", pdh], ["owner_uuid", "=", project_uuid]]).execute(num_retries=arvRunner.num_retries)
    if len(existing["items"]) == 0:
        col.save_new(name=toolname, owner_uuid=project_uuid, ensure_unique_name=True)

    # now construct the wrapper

    step = {
        "id": "#main/" + toolname,
        "in": [],
        "out": [],
        "run": "keep:%s/workflow.json#main" % pdh,
        "label": name
    }

    newinputs = []
    for i in main["inputs"]:
        inp = {}
        # Make sure to only copy known fields that are meaningful at
        # the workflow level. In practice this ensures that if we're
        # wrapping a CommandLineTool we don't grab inputBinding.
        # Right now also excludes extension fields, which is fine,
        # Arvados doesn't currently look for any extension fields on
        # input parameters.
        for f in ("type", "label", "secondaryFiles", "streamable",
                  "doc", "id", "format", "loadContents",
                  "loadListing", "default"):
            if f in i:
                inp[f] = i[f]
        newinputs.append(inp)

    wrapper = {
        "class": "Workflow",
        "id": "#main",
        "inputs": newinputs,
        "outputs": [],
        "steps": [step]
    }

    for i in main["inputs"]:
        step["in"].append({
            "id": "#main/step/%s" % shortname(i["id"]),
            "source": i["id"]
        })

    for i in main["outputs"]:
        step["out"].append({"id": "#main/step/%s" % shortname(i["id"])})
        wrapper["outputs"].append({"outputSource": "#main/step/%s" % shortname(i["id"]),
                                   "type": i["type"],
                                   "id": i["id"]})

    wrapper["requirements"] = [{"class": "SubworkflowFeatureRequirement"}]

    if main.get("requirements"):
        wrapper["requirements"].extend(main["requirements"])
    if main.get("hints"):
        wrapper["hints"] = main["hints"]

    doc = {"cwlVersion": "v1.2", "$graph": [wrapper]}

    if git_info:
        for g in git_info:
            doc[g] = git_info[g]

    return json.dumps(doc, sort_keys=True, indent=4, separators=(',',': '))


def rel_ref(s, baseuri, urlexpander, merged_map, jobmapper):
    if s.startswith("keep:"):
        return s

    uri = urlexpander(s, baseuri)

    if uri.startswith("keep:"):
        return uri

    fileuri = urllib.parse.urldefrag(baseuri)[0]

    for u in (baseuri, fileuri):
        if u in merged_map:
            replacements = merged_map[u].resolved
            if uri in replacements:
                return replacements[uri]

    if uri in jobmapper:
        return jobmapper.mapper(uri).target

    p1 = os.path.dirname(uri_file_path(fileuri))
    p2 = os.path.dirname(uri_file_path(uri))
    p3 = os.path.basename(uri_file_path(uri))

    r = os.path.relpath(p2, p1)
    if r == ".":
        r = ""

    return os.path.join(r, p3)

def is_basetype(tp):
    basetypes = ("null", "boolean", "int", "long", "float", "double", "string", "File", "Directory", "record", "array", "enum")
    for b in basetypes:
        if re.match(b+"(\[\])?\??", tp):
            return True
    return False


def update_refs(d, baseuri, urlexpander, merged_map, jobmapper, set_block_style, runtimeContext, prefix, replacePrefix):
    if set_block_style and (isinstance(d, CommentedSeq) or isinstance(d, CommentedMap)):
        d.fa.set_block_style()

    if isinstance(d, MutableSequence):
        for i, s in enumerate(d):
            if prefix and isinstance(s, str):
                if s.startswith(prefix):
                    d[i] = replacePrefix+s[len(prefix):]
            else:
                update_refs(s, baseuri, urlexpander, merged_map, jobmapper, set_block_style, runtimeContext, prefix, replacePrefix)
    elif isinstance(d, MutableMapping):
        for field in ("id", "name"):
            if isinstance(d.get(field), str) and d[field].startswith("_:"):
                # blank node reference, was added in automatically, can get rid of it.
                del d[field]

        if "id" in d:
            baseuri = urlexpander(d["id"], baseuri, scoped_id=True)
        elif "name" in d and isinstance(d["name"], str):
            baseuri = urlexpander(d["name"], baseuri, scoped_id=True)

        if d.get("class") == "DockerRequirement":
            dockerImageId = d.get("dockerImageId") or d.get("dockerPull")
            d["http://arvados.org/cwl#dockerCollectionPDH"] = runtimeContext.cached_docker_lookups.get(dockerImageId)

        for field in d:
            if field in ("location", "run", "name") and isinstance(d[field], str):
                d[field] = rel_ref(d[field], baseuri, urlexpander, merged_map, jobmapper)
                continue

            if field in ("$include", "$import") and isinstance(d[field], str):
                d[field] = rel_ref(d[field], baseuri, urlexpander, {}, jobmapper)
                continue

            for t in ("type", "items"):
                if (field == t and
                    isinstance(d[t], str) and
                    not is_basetype(d[t])):
                    d[t] = rel_ref(d[t], baseuri, urlexpander, merged_map, jobmapper)
                    continue

            if field == "inputs" and isinstance(d["inputs"], MutableMapping):
                for inp in d["inputs"]:
                    if isinstance(d["inputs"][inp], str) and not is_basetype(d["inputs"][inp]):
                        d["inputs"][inp] = rel_ref(d["inputs"][inp], baseuri, urlexpander, merged_map, jobmapper)
                    if isinstance(d["inputs"][inp], MutableMapping):
                        update_refs(d["inputs"][inp], baseuri, urlexpander, merged_map, jobmapper, set_block_style, runtimeContext, prefix, replacePrefix)
                continue

            if field == "$schemas":
                for n, s in enumerate(d["$schemas"]):
                    d["$schemas"][n] = rel_ref(d["$schemas"][n], baseuri, urlexpander, merged_map, jobmapper)
                continue

            update_refs(d[field], baseuri, urlexpander, merged_map, jobmapper, set_block_style, runtimeContext, prefix, replacePrefix)


def fix_schemadef(req, baseuri, urlexpander, merged_map, jobmapper, pdh):
    req = copy.deepcopy(req)

    for f in req["types"]:
        r = f["name"]
        path, frag = urllib.parse.urldefrag(r)
        rel = rel_ref(r, baseuri, urlexpander, merged_map, jobmapper)
        merged_map.setdefault(path, FileUpdates({}, {}))
        rename = "keep:%s/%s" %(pdh, rel)
        for mm in merged_map:
            merged_map[mm].resolved[r] = rename
    return req

def drop_ids(d):
    if isinstance(d, MutableSequence):
        for i, s in enumerate(d):
            drop_ids(s)
    elif isinstance(d, MutableMapping):
        if "id" in d and d["id"].startswith("file:"):
            del d["id"]

        for field in d:
            drop_ids(d[field])


def upload_workflow(arvRunner, tool, job_order, project_uuid,
                        runtimeContext,
                        uuid=None,
                        submit_runner_ram=0, name=None, merged_map=None,
                        submit_runner_image=None,
                        git_info=None,
                        set_defaults=False,
                        jobmapper=None):

    firstfile = None
    workflow_files = set()
    import_files = set()
    include_files = set()

    for w in tool.doc_loader.idx:
        if w.startswith("file://"):
            workflow_files.add(urllib.parse.urldefrag(w)[0])
            if firstfile is None:
                firstfile = urllib.parse.urldefrag(w)[0]
        if w.startswith("import:file://"):
            import_files.add(urllib.parse.urldefrag(w[7:])[0])
        if w.startswith("include:file://"):
            include_files.add(urllib.parse.urldefrag(w[8:])[0])

    all_files = workflow_files | import_files | include_files

    n = 7
    allmatch = True
    if firstfile:
        while allmatch:
            n += 1
            for f in all_files:
                if len(f)-1 < n:
                    n -= 1
                    allmatch = False
                    break
                if f[n] != firstfile[n]:
                    allmatch = False
                    break

        while firstfile[n] != "/":
            n -= 1

    col = arvados.collection.Collection(api_client=arvRunner.api)

    for w in workflow_files | import_files:
        # 1. load YAML

        text = tool.doc_loader.fetch_text(w)
        if isinstance(text, bytes):
            textIO = StringIO(text.decode('utf-8'))
        else:
            textIO = StringIO(text)

        yamlloader = schema_salad.utils.yaml_no_ts()
        result = yamlloader.load(textIO)

        set_block_style = False
        if result.fa.flow_style():
            set_block_style = True

        # 2. find $import, $include, $schema, run, location
        # 3. update field value
        update_refs(result, w, tool.doc_loader.expand_url, merged_map, jobmapper, set_block_style, runtimeContext, "", "")

        with col.open(w[n+1:], "wt") as f:
            # yamlloader.dump(result, stream=sys.stdout)
            yamlloader.dump(result, stream=f)

        with col.open(os.path.join("original", w[n+1:]), "wt") as f:
            f.write(text)


    for w in include_files:
        with col.open(w[n+1:], "wb") as f1:
            with col.open(os.path.join("original", w[n+1:]), "wb") as f3:
                with open(uri_file_path(w), "rb") as f2:
                    dat = f2.read(65536)
                    while dat:
                        f1.write(dat)
                        f3.write(dat)
                        dat = f2.read(65536)


    toolname = tool.tool.get("label") or tool.metadata.get("label") or os.path.basename(tool.tool["id"])
    if git_info and git_info.get("http://arvados.org/cwl#gitDescribe"):
        toolname = "%s (%s)" % (toolname, git_info.get("http://arvados.org/cwl#gitDescribe"))

    toolfile = tool.tool["id"][n+1:]

    properties = {
        "type": "workflow",
        "arv:workflowMain": toolfile,
    }

    if git_info:
        for g in git_info:
            p = g.split("#", 1)[1]
            properties["arv:"+p] = git_info[g]

    col.save_new(name=toolname, owner_uuid=arvRunner.project_uuid, ensure_unique_name=True, properties=properties)

    logger.info("Workflow uploaded to %s", col.manifest_locator())

    adjustDirObjs(job_order, trim_listing)
    adjustFileObjs(job_order, trim_anonymous_location)
    adjustDirObjs(job_order, trim_anonymous_location)

    # now construct the wrapper

    runfile = "keep:%s/%s" % (col.portable_data_hash(), toolfile)

    step = {
        "id": "#main/" + toolname,
        "in": [],
        "out": [],
        "run": runfile,
        "label": name
    }

    main = tool.tool

    wf_runner_resources = None

    hints = main.get("hints", [])
    found = False
    for h in hints:
        if h["class"] == "http://arvados.org/cwl#WorkflowRunnerResources":
            wf_runner_resources = h
            found = True
            break
    if not found:
        wf_runner_resources = {"class": "http://arvados.org/cwl#WorkflowRunnerResources"}
        hints.append(wf_runner_resources)

    wf_runner_resources["acrContainerImage"] = arvados_jobs_image(arvRunner,
                                                                  submit_runner_image or "arvados/jobs:"+__version__,
                                                                  runtimeContext)

    if submit_runner_ram:
        wf_runner_resources["ramMin"] = submit_runner_ram

    newinputs = []
    for i in main["inputs"]:
        inp = {}
        # Make sure to only copy known fields that are meaningful at
        # the workflow level. In practice this ensures that if we're
        # wrapping a CommandLineTool we don't grab inputBinding.
        # Right now also excludes extension fields, which is fine,
        # Arvados doesn't currently look for any extension fields on
        # input parameters.
        for f in ("type", "label", "secondaryFiles", "streamable",
                  "doc", "format", "loadContents",
                  "loadListing", "default"):
            if f in i:
                inp[f] = i[f]

        if set_defaults:
            sn = shortname(i["id"])
            if sn in job_order:
                inp["default"] = job_order[sn]

        inp["id"] = "#main/%s" % shortname(i["id"])
        newinputs.append(inp)

    wrapper = {
        "class": "Workflow",
        "id": "#main",
        "inputs": newinputs,
        "outputs": [],
        "steps": [step]
    }

    for i in main["inputs"]:
        step["in"].append({
            "id": "#main/step/%s" % shortname(i["id"]),
            "source": "#main/%s" % shortname(i["id"])
        })

    for i in main["outputs"]:
        step["out"].append({"id": "#main/step/%s" % shortname(i["id"])})
        wrapper["outputs"].append({"outputSource": "#main/step/%s" % shortname(i["id"]),
                                   "type": i["type"],
                                   "id": "#main/%s" % shortname(i["id"])})

    wrapper["requirements"] = [{"class": "SubworkflowFeatureRequirement"}]

    if main.get("requirements"):
        wrapper["requirements"].extend(main["requirements"])
    if hints:
        wrapper["hints"] = hints

    # 1. check for SchemaDef
    # 2. do what pack does
    # 3. fix inputs

    doc = {"cwlVersion": "v1.2", "$graph": [wrapper]}

    if git_info:
        for g in git_info:
            doc[g] = git_info[g]

    for i, r in enumerate(wrapper["requirements"]):
        if r["class"] == "SchemaDefRequirement":
            wrapper["requirements"][i] = fix_schemadef(r, main["id"], tool.doc_loader.expand_url, merged_map, jobmapper, col.portable_data_hash())

    update_refs(wrapper, main["id"], tool.doc_loader.expand_url, merged_map, jobmapper, False, runtimeContext, main["id"]+"#", "#main/")

    # Remove any lingering file references.
    drop_ids(wrapper)

    return doc


def make_workflow_record(arvRunner, doc, name, tool, project_uuid, update_uuid):

    wrappertext = json.dumps(doc, sort_keys=True, indent=4, separators=(',',': '))

    body = {
        "workflow": {
            "name": name,
            "description": tool.tool.get("doc", ""),
            "definition": wrappertext
        }}
    if project_uuid:
        body["workflow"]["owner_uuid"] = project_uuid

    if update_uuid:
        call = arvRunner.api.workflows().update(uuid=update_uuid, body=body)
    else:
        call = arvRunner.api.workflows().create(body=body)
    return call.execute(num_retries=arvRunner.num_retries)["uuid"]


def dedup_reqs(reqs):
    dedup = {}
    for r in reversed(reqs):
        if r["class"] not in dedup and not r["class"].startswith("http://arvados.org/cwl#"):
            dedup[r["class"]] = r
    return [dedup[r] for r in sorted(dedup.keys())]

def get_overall_res_req(res_reqs):
    """Take the overall of a list of ResourceRequirement,
    i.e., the max of coresMin, coresMax, ramMin, ramMax, tmpdirMin, tmpdirMax
    and the sum of outdirMin, outdirMax."""

    all_res_req = {}
    exception_msgs = []
    for a in max_res_pars + sum_res_pars:
        all_res_req[a] = []
        for res_req in res_reqs:
            if a in res_req:
                if isinstance(res_req[a], int): # integer check
                    all_res_req[a].append(res_req[a])
                else:
                    msg = SourceLine(res_req, a).makeError(
                    "Non-top-level ResourceRequirement in single container cannot have expressions")
                    exception_msgs.append(msg)
    if exception_msgs:
        raise WorkflowException("\n".join(exception_msgs))
    else:
        overall_res_req = {}
        for a in all_res_req:
            if all_res_req[a]:
                if a in max_res_pars:
                    overall_res_req[a] = max(all_res_req[a])
                elif a in sum_res_pars:
                    overall_res_req[a] = sum(all_res_req[a])
        if overall_res_req:
            overall_res_req["class"] = "ResourceRequirement"
        return cmap(overall_res_req)

class ArvadosWorkflowStep(WorkflowStep):
    def __init__(self,
                 toolpath_object,      # type: Dict[Text, Any]
                 pos,                  # type: int
                 loadingContext,       # type: LoadingContext
                 arvrunner,
                 *argc,
                 **argv
                ):  # type: (...) -> None

        if arvrunner.fast_submit:
            self.tool = toolpath_object
            self.tool["inputs"] = []
            self.tool["outputs"] = []
        else:
            super(ArvadosWorkflowStep, self).__init__(toolpath_object, pos, loadingContext, *argc, **argv)
            self.tool["class"] = "WorkflowStep"
        self.arvrunner = arvrunner

    def job(self, joborder, output_callback, runtimeContext):
        runtimeContext = runtimeContext.copy()
        runtimeContext.toplevel = True  # Preserve behavior for #13365

        builder = make_builder({shortname(k): v for k,v in viewitems(joborder)}, self.hints, self.requirements,
                               runtimeContext, self.metadata)
        runtimeContext = set_cluster_target(self.tool, self.arvrunner, builder, runtimeContext)
        return super(ArvadosWorkflowStep, self).job(joborder, output_callback, runtimeContext)


class ArvadosWorkflow(Workflow):
    """Wrap cwltool Workflow to override selected methods."""

    def __init__(self, arvrunner, toolpath_object, loadingContext):
        self.arvrunner = arvrunner
        self.wf_pdh = None
        self.dynamic_resource_req = []
        self.static_resource_req = []
        self.wf_reffiles = []
        self.loadingContext = loadingContext
        super(ArvadosWorkflow, self).__init__(toolpath_object, loadingContext)
        self.cluster_target_req, _ = self.get_requirement("http://arvados.org/cwl#ClusterTarget")

    def job(self, joborder, output_callback, runtimeContext):

        builder = make_builder(joborder, self.hints, self.requirements, runtimeContext, self.metadata)
        runtimeContext = set_cluster_target(self.tool, self.arvrunner, builder, runtimeContext)

        req, _ = self.get_requirement("http://arvados.org/cwl#RunInSingleContainer")
        if not req:
            return super(ArvadosWorkflow, self).job(joborder, output_callback, runtimeContext)

        # RunInSingleContainer is true

        with SourceLine(self.tool, None, WorkflowException, logger.isEnabledFor(logging.DEBUG)):
            if "id" not in self.tool:
                raise WorkflowException("%s object must have 'id'" % (self.tool["class"]))

        discover_secondary_files(self.arvrunner.fs_access, builder,
                                 self.tool["inputs"], joborder)

        normalizeFilesDirs(joborder)

        with Perf(metrics, "subworkflow upload_deps"):
            upload_dependencies(self.arvrunner,
                                os.path.basename(joborder.get("id", "#")),
                                self.doc_loader,
                                joborder,
                                joborder.get("id", "#"),
                                runtimeContext)

            if self.wf_pdh is None:
                packed = pack(self.loadingContext, self.tool["id"], loader=self.doc_loader)

                for p in packed["$graph"]:
                    if p["id"] == "#main":
                        p["requirements"] = dedup_reqs(self.requirements)
                        p["hints"] = dedup_reqs(self.hints)

                def visit(item):
                    if "requirements" in item:
                        item["requirements"] = [i for i in item["requirements"] if i["class"] != "DockerRequirement"]
                    for t in ("hints", "requirements"):
                        if t not in item:
                            continue
                        for req in item[t]:
                            if req["class"] == "ResourceRequirement":
                                dyn = False
                                for k in max_res_pars + sum_res_pars:
                                    if k in req:
                                        if isinstance(req[k], basestring):
                                            if item["id"] == "#main":
                                                # only the top-level requirements/hints may contain expressions
                                                self.dynamic_resource_req.append(req)
                                                dyn = True
                                                break
                                            else:
                                                with SourceLine(req, k, WorkflowException):
                                                    raise WorkflowException("Non-top-level ResourceRequirement in single container cannot have expressions")
                                if not dyn:
                                    self.static_resource_req.append(req)

                visit_class(packed["$graph"], ("Workflow", "CommandLineTool"), visit)

                if self.static_resource_req:
                    self.static_resource_req = [get_overall_res_req(self.static_resource_req)]

                upload_dependencies(self.arvrunner,
                                    runtimeContext.name,
                                    self.doc_loader,
                                    packed,
                                    self.tool["id"],
                                    runtimeContext)

                # Discover files/directories referenced by the
                # workflow (mainly "default" values)
                visit_class(packed, ("File", "Directory"), self.wf_reffiles.append)


        if self.dynamic_resource_req:
            # Evaluate dynamic resource requirements using current builder
            rs = copy.copy(self.static_resource_req)
            for dyn_rs in self.dynamic_resource_req:
                eval_req = {"class": "ResourceRequirement"}
                for a in max_res_pars + sum_res_pars:
                    if a in dyn_rs:
                        eval_req[a] = builder.do_eval(dyn_rs[a])
                rs.append(eval_req)
            job_res_reqs = [get_overall_res_req(rs)]
        else:
            job_res_reqs = self.static_resource_req

        with Perf(metrics, "subworkflow adjust"):
            joborder_resolved = copy.deepcopy(joborder)
            joborder_keepmount = copy.deepcopy(joborder)

            reffiles = []
            visit_class(joborder_keepmount, ("File", "Directory"), reffiles.append)

            mapper = ArvPathMapper(self.arvrunner, reffiles+self.wf_reffiles, runtimeContext.basedir,
                                   "/keep/%s",
                                   "/keep/%s/%s")

            # For containers API, we need to make sure any extra
            # referenced files (ie referenced by the workflow but
            # not in the inputs) are included in the mounts.
            if self.wf_reffiles:
                runtimeContext = runtimeContext.copy()
                runtimeContext.extra_reffiles = copy.deepcopy(self.wf_reffiles)

            def keepmount(obj):
                remove_redundant_fields(obj)
                with SourceLine(obj, None, WorkflowException, logger.isEnabledFor(logging.DEBUG)):
                    if "location" not in obj:
                        raise WorkflowException("%s object is missing required 'location' field: %s" % (obj["class"], obj))
                with SourceLine(obj, "location", WorkflowException, logger.isEnabledFor(logging.DEBUG)):
                    if obj["location"].startswith("keep:"):
                        obj["location"] = mapper.mapper(obj["location"]).target
                        if "listing" in obj:
                            del obj["listing"]
                    elif obj["location"].startswith("_:"):
                        del obj["location"]
                    else:
                        raise WorkflowException("Location is not a keep reference or a literal: '%s'" % obj["location"])

            visit_class(joborder_keepmount, ("File", "Directory"), keepmount)

            def resolved(obj):
                if obj["location"].startswith("keep:"):
                    obj["location"] = mapper.mapper(obj["location"]).resolved

            visit_class(joborder_resolved, ("File", "Directory"), resolved)

            if self.wf_pdh is None:
                adjustFileObjs(packed, keepmount)
                adjustDirObjs(packed, keepmount)
                self.wf_pdh = upload_workflow_collection(self.arvrunner, shortname(self.tool["id"]), packed, runtimeContext)

        self.loadingContext = self.loadingContext.copy()
        self.loadingContext.metadata = self.loadingContext.metadata.copy()
        self.loadingContext.metadata["http://commonwl.org/cwltool#original_cwlVersion"] = "v1.0"

        if len(job_res_reqs) == 1:
            # RAM request needs to be at least 128 MiB or the workflow
            # runner itself won't run reliably.
            if job_res_reqs[0].get("ramMin", 1024) < 128:
                job_res_reqs[0]["ramMin"] = 128

        arguments = ["--no-container", "--move-outputs", "--preserve-entire-environment", "workflow.cwl", "cwl.input.yml"]
        if runtimeContext.debug:
            arguments.insert(0, '--debug')

        wf_runner = cmap({
            "class": "CommandLineTool",
            "baseCommand": "cwltool",
            "inputs": self.tool["inputs"],
            "outputs": self.tool["outputs"],
            "stdout": "cwl.output.json",
            "requirements": self.requirements+job_res_reqs+[
                {"class": "InlineJavascriptRequirement"},
                {
                "class": "InitialWorkDirRequirement",
                "listing": [{
                        "entryname": "workflow.cwl",
                        "entry": '$({"class": "File", "location": "keep:%s/workflow.cwl"})' % self.wf_pdh
                    }, {
                        "entryname": "cwl.input.yml",
                        "entry": json.dumps(joborder_keepmount, indent=2, sort_keys=True, separators=(',',': ')).replace("\\", "\\\\").replace('$(', '\$(').replace('${', '\${')
                    }]
            }],
            "hints": self.hints,
            "arguments": arguments,
            "id": "#"
        })
        return ArvadosCommandTool(self.arvrunner, wf_runner, self.loadingContext).job(joborder_resolved, output_callback, runtimeContext)

    def make_workflow_step(self,
                           toolpath_object,      # type: Dict[Text, Any]
                           pos,                  # type: int
                           loadingContext,       # type: LoadingContext
                           *argc,
                           **argv
    ):
        # (...) -> WorkflowStep
        return ArvadosWorkflowStep(toolpath_object, pos, loadingContext, self.arvrunner, *argc, **argv)
