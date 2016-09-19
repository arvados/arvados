import os
import json
import copy

from cwltool.pack import pack
from cwltool.load_tool import fetch_document
from cwltool.process import shortname

import ruamel.yaml as yaml

from .runner import upload_docker, upload_dependencies

def make_workflow(arvRunner, tool, job_order, project_uuid, update_uuid):
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
