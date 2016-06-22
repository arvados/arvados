import os
import urlparse
from functools import partial
import logging
import json

from cwltool.draft2tool import CommandLineTool
import cwltool.workflow
from cwltool.process import get_feature, scandeps, adjustFiles, UnsupportedRequirement
from cwltool.load_tool import fetch_document

import arvados.collection

from .arvdocker import arv_docker_get_image
from .pathmapper import ArvPathMapper

logger = logging.getLogger('arvados.cwl-runner')

class Runner(object):
    def __init__(self, runner, tool, job_order, enable_reuse):
        self.arvrunner = runner
        self.tool = tool
        self.job_order = job_order
        self.running = False
        self.enable_reuse = enable_reuse
        self.uuid = None

    def update_pipeline_component(self, record):
        pass

    def upload_docker(self, tool):
        if isinstance(tool, CommandLineTool):
            (docker_req, docker_is_req) = get_feature(tool, "DockerRequirement")
            if docker_req:
                arv_docker_get_image(self.arvrunner.api, docker_req, True, self.arvrunner.project_uuid)
        elif isinstance(tool, cwltool.workflow.Workflow):
            for s in tool.steps:
                self.upload_docker(s.embedded_tool)


    def arvados_job_spec(self, *args, **kwargs):
        self.upload_docker(self.tool)

        workflowfiles = set()
        jobfiles = set()
        workflowfiles.add(self.tool.tool["id"])

        self.name = os.path.basename(self.tool.tool["id"])

        def visitFiles(files, path):
            files.add(path)
            return path

        document_loader, workflowobj, uri = fetch_document(self.tool.tool["id"])
        loaded = set()
        def loadref(b, u):
            joined = urlparse.urljoin(b, u)
            if joined not in loaded:
                loaded.add(joined)
                return document_loader.fetch(urlparse.urljoin(b, u))
            else:
                return {}

        sc = scandeps(uri, workflowobj,
                      set(("$import", "run")),
                      set(("$include", "$schemas", "path")),
                      loadref)
        adjustFiles(sc, partial(visitFiles, workflowfiles))
        adjustFiles(self.job_order, partial(visitFiles, jobfiles))

        keepprefix = kwargs.get("keepprefix", "")
        workflowmapper = ArvPathMapper(self.arvrunner, workflowfiles, "",
                                       keepprefix+"%s",
                                       keepprefix+"%s/%s",
                                       name=self.name,
                                       **kwargs)

        jobmapper = ArvPathMapper(self.arvrunner, jobfiles, "",
                                  keepprefix+"%s",
                                  keepprefix+"%s/%s",
                                  name=os.path.basename(self.job_order.get("id", "#")),
                                  **kwargs)

        adjustFiles(self.job_order, lambda p: jobmapper.mapper(p)[1])

        if "id" in self.job_order:
            del self.job_order["id"]

        return workflowmapper


    def done(self, record):
        if record["state"] == "Complete":
            if record.get("exit_code") is not None:
                if record["exit_code"] == 33:
                    processStatus = "UnsupportedRequirement"
                elif record["exit_code"] == 0:
                    processStatus = "success"
                else:
                    processStatus = "permanentFail"
            else:
                processStatus = "success"
        else:
            processStatus = "permanentFail"

        outputs = None
        try:
            try:
                outc = arvados.collection.Collection(record["output"])
                with outc.open("cwl.output.json") as f:
                    outputs = json.load(f)
                def keepify(path):
                    if not path.startswith("keep:"):
                        return "keep:%s/%s" % (record["output"], path)
                    else:
                        return path
                adjustFiles(outputs, keepify)
            except Exception as e:
                logger.error("While getting final output object: %s", e)
            self.arvrunner.output_callback(outputs, processStatus)
        finally:
            del self.arvrunner.processes[record["uuid"]]
