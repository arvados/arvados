# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import os
import json
import copy
import logging

from schema_salad.sourceline import SourceLine, cmap

from cwltool.pack import pack
from cwltool.load_tool import fetch_document
from cwltool.process import shortname
from cwltool.workflow import Workflow, WorkflowException, WorkflowStep
from cwltool.pathmapper import adjustFileObjs, adjustDirObjs, visit_class
from cwltool.context import LoadingContext

import ruamel.yaml as yaml

from .runner import (upload_dependencies, packed_workflow, upload_workflow_collection,
                     trim_anonymous_location, remove_redundant_fields, discover_secondary_files)
from .pathmapper import ArvPathMapper, trim_listing
from .arvtool import ArvadosCommandTool, set_cluster_target, make_builder
from .perf import Perf

logger = logging.getLogger('arvados.cwl-runner')
metrics = logging.getLogger('arvados.cwl-runner.metrics')

max_res_pars = ("coresMin", "coresMax", "ramMin", "ramMax", "tmpdirMin", "tmpdirMax")
sum_res_pars = ("outdirMin", "outdirMax")

def upload_workflow(arvRunner, tool, job_order, project_uuid, uuid=None,
                    submit_runner_ram=0, name=None, merged_map=None):

    packed = packed_workflow(arvRunner, tool, merged_map)

    adjustDirObjs(job_order, trim_listing)
    adjustFileObjs(job_order, trim_anonymous_location)
    adjustDirObjs(job_order, trim_anonymous_location)

    main = [p for p in packed["$graph"] if p["id"] == "#main"][0]
    for inp in main["inputs"]:
        sn = shortname(inp["id"])
        if sn in job_order:
            inp["default"] = job_order[sn]

    if not name:
        name = tool.tool.get("label", os.path.basename(tool.tool["id"]))

    upload_dependencies(arvRunner, name, tool.doc_loader,
                        packed, tool.tool["id"], False)

    if submit_runner_ram:
        hints = main.get("hints", [])
        found = False
        for h in hints:
            if h["class"] == "http://arvados.org/cwl#WorkflowRunnerResources":
                h["ramMin"] = submit_runner_ram
                found = True
                break
        if not found:
            hints.append({"class": "http://arvados.org/cwl#WorkflowRunnerResources",
                          "ramMin": submit_runner_ram})
        main["hints"] = hints

    body = {
        "workflow": {
            "name": name,
            "description": tool.tool.get("doc", ""),
            "definition":json.dumps(packed, sort_keys=True, indent=4, separators=(',',': '))
        }}
    if project_uuid:
        body["workflow"]["owner_uuid"] = project_uuid

    if uuid:
        call = arvRunner.api.workflows().update(uuid=uuid, body=body)
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

        super(ArvadosWorkflowStep, self).__init__(toolpath_object, pos, loadingContext, *argc, **argv)
        self.tool["class"] = "WorkflowStep"
        self.arvrunner = arvrunner

    def job(self, joborder, output_callback, runtimeContext):
        runtimeContext = runtimeContext.copy()
        runtimeContext.toplevel = True  # Preserve behavior for #13365

        builder = make_builder({shortname(k): v for k,v in joborder.items()}, self.hints, self.requirements, runtimeContext)
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

        builder = make_builder(joborder, self.hints, self.requirements, runtimeContext)
        runtimeContext = set_cluster_target(self.tool, self.arvrunner, builder, runtimeContext)

        req, _ = self.get_requirement("http://arvados.org/cwl#RunInSingleContainer")
        if not req:
            return super(ArvadosWorkflow, self).job(joborder, output_callback, runtimeContext)

        # RunInSingleContainer is true

        with SourceLine(self.tool, None, WorkflowException, logger.isEnabledFor(logging.DEBUG)):
            if "id" not in self.tool:
                raise WorkflowException("%s object must have 'id'" % (self.tool["class"]))
        document_loader, workflowobj, uri = (self.doc_loader, self.doc_loader.fetch(self.tool["id"]), self.tool["id"])

        discover_secondary_files(self.tool["inputs"], joborder)

        with Perf(metrics, "subworkflow upload_deps"):
            upload_dependencies(self.arvrunner,
                                os.path.basename(joborder.get("id", "#")),
                                document_loader,
                                joborder,
                                joborder.get("id", "#"),
                                False)

            if self.wf_pdh is None:
                workflowobj["requirements"] = dedup_reqs(self.requirements)
                workflowobj["hints"] = dedup_reqs(self.hints)

                packed = pack(document_loader, workflowobj, uri, self.metadata)

                def visit(item):
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
                            if req["class"] == "DockerRequirement":
                                if "http://arvados.org/cwl#dockerCollectionPDH" in req:
                                    del req["http://arvados.org/cwl#dockerCollectionPDH"]

                visit_class(packed["$graph"], ("Workflow", "CommandLineTool"), visit)

                if self.static_resource_req:
                    self.static_resource_req = [get_overall_res_req(self.static_resource_req)]

                upload_dependencies(self.arvrunner,
                                    runtimeContext.name,
                                    document_loader,
                                    packed,
                                    uri,
                                    False)

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
                self.wf_pdh = upload_workflow_collection(self.arvrunner, shortname(self.tool["id"]), packed)

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
            "arguments": ["--no-container", "--move-outputs", "--preserve-entire-environment", "workflow.cwl#main", "cwl.input.yml"],
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
