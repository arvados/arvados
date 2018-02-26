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
from cwltool.workflow import Workflow, WorkflowException
from cwltool.pathmapper import adjustFileObjs, adjustDirObjs, visit_class
from cwltool.builder import Builder

import ruamel.yaml as yaml

from .runner import (upload_dependencies, packed_workflow, upload_workflow_collection,
                     trim_anonymous_location, remove_redundant_fields, discover_secondary_files)
from .pathmapper import ArvPathMapper, trim_listing
from .arvtool import ArvadosCommandTool
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

    # TODO nowhere for submit_runner_ram to go.

    body = {
        "workflow": {
            "name": name,
            "description": tool.tool.get("doc", ""),
            "definition":yaml.round_trip_dump(packed)
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
                    msg = SourceLine(res_req).makeError(
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

class ArvadosWorkflow(Workflow):
    """Wrap cwltool Workflow to override selected methods."""

    def __init__(self, arvrunner, toolpath_object, **kwargs):
        super(ArvadosWorkflow, self).__init__(toolpath_object, **kwargs)
        self.arvrunner = arvrunner
        self.work_api = kwargs["work_api"]
        self.wf_pdh = None

    def job(self, joborder, output_callback, **kwargs):
        kwargs["work_api"] = self.work_api
        req, _ = self.get_requirement("http://arvados.org/cwl#RunInSingleContainer")
        if req:
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

                    builder = Builder()
                    builder.job = joborder
                    builder.requirements = workflowobj["requirements"]
                    builder.hints = workflowobj["hints"]
                    builder.resources = {}

                    res_reqs = {"requirements": [], "hints": []}
                    for t in ("requirements", "hints"):
                        for item in packed["$graph"]:
                            if t in item:
                                if item["id"] == "#main": # evaluate potential expressions in the top-level requirements/hints
                                    for req in item[t]:
                                        if req["class"] == "ResourceRequirement":
                                            eval_req = {"class": "ResourceRequirement"}
                                            for a in max_res_pars + sum_res_pars:
                                                if a in req:
                                                    eval_req[a] = builder.do_eval(req[a])
                                            res_reqs[t].append(eval_req)
                                else:
                                    for req in item[t]:
                                        if req["class"] == "ResourceRequirement":
                                            res_reqs[t].append(req)
                    overall_res_req = {"requirements": get_overall_res_req(res_reqs["requirements"]),
                                       "hints": get_overall_res_req(res_reqs["hints"])}

                    new_spec = {"requirements": self.requirements, "hints": self.hints}
                    for t in ("requirements", "hints"):
                        for req in new_spec[t]:
                            if req["class"] == "ResourceRequirement":
                                new_spec[t].remove(req)
                        if overall_res_req[t]:
                            new_spec[t].append(overall_res_req[t])

                    upload_dependencies(self.arvrunner,
                                        kwargs.get("name", ""),
                                        document_loader,
                                        packed,
                                        uri,
                                        False)

            with Perf(metrics, "subworkflow adjust"):
                joborder_resolved = copy.deepcopy(joborder)
                joborder_keepmount = copy.deepcopy(joborder)

                reffiles = []
                visit_class(joborder_keepmount, ("File", "Directory"), lambda x: reffiles.append(x))

                mapper = ArvPathMapper(self.arvrunner, reffiles, kwargs["basedir"],
                                 "/keep/%s",
                                 "/keep/%s/%s",
                                 **kwargs)

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
                "requirements": self.requirements+[
                    {
                    "class": "InitialWorkDirRequirement",
                    "listing": [{
                            "entryname": "workflow.cwl",
                            "entry": {
                                "class": "File",
                                "location": "keep:%s/workflow.cwl" % self.wf_pdh
                            }
                        }, {
                            "entryname": "cwl.input.yml",
                            "entry": json.dumps(joborder_keepmount, indent=2, sort_keys=True, separators=(',',': ')).replace("\\", "\\\\").replace('$(', '\$(').replace('${', '\${')
                        }]
                }],
                "hints": self.hints,
                "arguments": ["--no-container", "--move-outputs", "--preserve-entire-environment", "workflow.cwl#main", "cwl.input.yml"],
                "id": "#"
            })
            kwargs["loader"] = self.doc_loader
            kwargs["avsc_names"] = self.doc_schema
            return ArvadosCommandTool(self.arvrunner, wf_runner, **kwargs).job(joborder_resolved, output_callback, **kwargs)
        else:
            return super(ArvadosWorkflow, self).job(joborder, output_callback, **kwargs)
