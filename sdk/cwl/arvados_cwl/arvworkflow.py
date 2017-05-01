import os
import json
import copy
import logging

from schema_salad.sourceline import SourceLine, cmap

from cwltool.pack import pack
from cwltool.load_tool import fetch_document
from cwltool.process import shortname
from cwltool.workflow import Workflow, WorkflowException
from cwltool.pathmapper import adjustFileObjs, adjustDirObjs

import ruamel.yaml as yaml

from .runner import upload_dependencies, packed_workflow, upload_workflow_collection, trim_anonymous_location
from .pathmapper import trim_listing
from .arvtool import ArvadosCommandTool
from .perf import Perf

logger = logging.getLogger('arvados.cwl-runner')
metrics = logging.getLogger('arvados.cwl-runner.metrics')

def upload_workflow(arvRunner, tool, job_order, project_uuid, uuid=None,
                    submit_runner_ram=0, name=None):

    packed = packed_workflow(arvRunner, tool)

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
            with SourceLine(self.tool, None, WorkflowException):
                if "id" not in self.tool:
                    raise WorkflowException("%s object must have 'id'" % (self.tool["class"]))
            document_loader, workflowobj, uri = (self.doc_loader, self.doc_loader.fetch(self.tool["id"]), self.tool["id"])

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

                    upload_dependencies(self.arvrunner,
                                        kwargs.get("name", ""),
                                        document_loader,
                                        packed,
                                        uri,
                                        False)

            with Perf(metrics, "subworkflow adjust"):
                joborder_keepmount = copy.deepcopy(joborder)

                def keepmount(obj):
                    with SourceLine(obj, None, WorkflowException):
                        if "location" not in obj:
                            raise WorkflowException("%s object is missing required 'location' field: %s" % (obj["class"], obj))
                    with SourceLine(obj, "location", WorkflowException):
                        if obj["location"].startswith("keep:"):
                            obj["location"] = "/keep/" + obj["location"][5:]
                            if "listing" in obj:
                                del obj["listing"]
                        elif obj["location"].startswith("_:"):
                            del obj["location"]
                        else:
                            raise WorkflowException("Location is not a keep reference or a literal: '%s'" % obj["location"])

                adjustFileObjs(joborder_keepmount, keepmount)
                adjustDirObjs(joborder_keepmount, keepmount)

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
                "requirements": workflowobj["requirements"]+[
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
                "hints": workflowobj["hints"],
                "arguments": ["--no-container", "--move-outputs", "--preserve-entire-environment", "workflow.cwl#main", "cwl.input.yml"]
            })
            kwargs["loader"] = self.doc_loader
            kwargs["avsc_names"] = self.doc_schema
            return ArvadosCommandTool(self.arvrunner, wf_runner, **kwargs).job(joborder, output_callback, **kwargs)
        else:
            return super(ArvadosWorkflow, self).job(joborder, output_callback, **kwargs)
