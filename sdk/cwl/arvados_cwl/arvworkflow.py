import os
import json
import copy
import logging

from cwltool.pack import pack
from cwltool.load_tool import fetch_document
from cwltool.process import shortname
from cwltool.workflow import Workflow
from cwltool.pathmapper import adjustDirObjs

import ruamel.yaml as yaml

from .runner import upload_docker, upload_dependencies
from .arvtool import ArvadosCommandTool

logger = logging.getLogger('arvados.cwl-runner')

def upload_workflow(arvRunner, tool, job_order, project_uuid, update_uuid):
    upload_docker(arvRunner, tool)

    document_loader, workflowobj, uri = (tool.doc_loader, tool.doc_loader.fetch(tool.tool["id"]), tool.tool["id"])

    packed = pack(document_loader, workflowobj, uri, tool.metadata)

    main = [p for p in packed["$graph"] if p["id"] == "#main"][0]
    for inp in main["inputs"]:
        sn = shortname(inp["id"])
        if sn in job_order:
            inp["default"] = job_order[sn]

    name = os.path.basename(tool.tool["id"])
    upload_dependencies(arvRunner, name, document_loader,
                        packed, uri, False)

    body = {
        "workflow": {
            "owner_uuid": project_uuid,
            "name": tool.tool.get("label", name),
            "description": tool.tool.get("doc", ""),
            "definition":yaml.safe_dump(packed)
        }}

    if update_uuid:
        return arvRunner.api.workflows().update(uuid=update_uuid, body=body).execute(num_retries=arvRunner.num_retries)["uuid"]
    else:
        return arvRunner.api.workflows().create(body=body).execute(num_retries=arvRunner.num_retries)["uuid"]

class ArvadosWorkflow(Workflow):
    """Wrap cwltool Workflow to override selected methods."""

    def __init__(self, arvrunner, toolpath_object, **kwargs):
        super(ArvadosWorkflow, self).__init__(toolpath_object, **kwargs)
        self.arvrunner = arvrunner
        self.work_api = kwargs["work_api"]

    def job(self, joborder, output_callback, **kwargs):
        kwargs["work_api"] = self.work_api
        req, _ = self.get_requirement("http://arvados.org/cwl#RunInSingleContainer")
        if req:
            document_loader, workflowobj, uri = (self.doc_loader, self.doc_loader.fetch(self.tool["id"]), self.tool["id"])
            workflowobj["requirements"] = self.requirements + workflowobj.get("requirements", [])
            workflowobj["hints"] = self.hints + workflowobj.get("hints", [])
            packed = pack(document_loader, workflowobj, uri, self.metadata)

            def prune_directories(obj):
                if obj["location"].startswith("keep:") and "listing" in obj:
                    del obj["listing"]
            adjustDirObjs(joborder, prune_directories)

            wf_runner = {
                "class": "CommandLineTool",
                "baseCommand": "cwltool",
                "inputs": self.tool["inputs"],
                "outputs": self.tool["outputs"],
                "stdout": "cwl.output.json",
                "requirements": workflowobj["requirements"]+[
                    {"class": "InlineJavascriptRequirement"},
                    {
                    "class": "InitialWorkDirRequirement",
                    "listing": [{
                            "entryname": "workflow.cwl",
                            "entry": yaml.safe_dump(packed).replace("\\", "\\\\").replace('$(', '\$(').replace('${', '\${')
                        }, {
                            "entryname": "cwl.input.json",
                            "entry": yaml.safe_dump(joborder).replace("\\", "\\\\").replace('$(', '\$(').replace('${', '\${')
                        }]
                }],
                "hints": workflowobj["hints"],
                "arguments": ["--no-container", "--move-outputs", "workflow.cwl", "cwl.input.json"]
            }
            kwargs["loader"] = self.doc_loader
            kwargs["avsc_names"] = self.doc_schema
            return ArvadosCommandTool(self.arvrunner, wf_runner, **kwargs).job(joborder, output_callback, **kwargs)
        else:
            return super(ArvadosWorkflow, self).job(joborder, output_callback, **kwargs)
