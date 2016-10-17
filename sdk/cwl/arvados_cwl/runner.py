import os
import urlparse
from functools import partial
import logging
import json
import re
from cStringIO import StringIO

import cwltool.draft2tool
from cwltool.draft2tool import CommandLineTool
import cwltool.workflow
from cwltool.process import get_feature, scandeps, UnsupportedRequirement, normalizeFilesDirs
from cwltool.load_tool import fetch_document
from cwltool.pathmapper import adjustFileObjs, adjustDirObjs

import arvados.collection
import ruamel.yaml as yaml

from .arvdocker import arv_docker_get_image
from .pathmapper import ArvPathMapper

logger = logging.getLogger('arvados.cwl-runner')

cwltool.draft2tool.ACCEPTLIST_RE = re.compile(r".*")

def trim_listing(obj):
    """Remove 'listing' field from Directory objects that are keep references.

    When Directory objects represent Keep references, it redundant and
    potentially very expensive to pass fully enumerated Directory objects
    between instances of cwl-runner (e.g. a submitting a job, or using the
    RunInSingleContainer feature), so delete the 'listing' field when it is
    safe to do so.
    """

    if obj.get("location", "").startswith("keep:") and "listing" in obj:
        del obj["listing"]
    if obj.get("location", "").startswith("_:"):
        del obj["location"]

def upload_dependencies(arvrunner, name, document_loader,
                        workflowobj, uri, loadref_run):
    """Upload the dependencies of the workflowobj document to Keep.

    Returns a pathmapper object mapping local paths to keep references.  Also
    does an in-place update of references in "workflowobj".

    Use scandeps to find $import, $include, $schemas, run, File and Directory
    fields that represent external references.

    If workflowobj has an "id" field, this will reload the document to ensure
    it is scanning the raw document prior to preprocessing.
    """

    loaded = set()
    def loadref(b, u):
        joined = urlparse.urljoin(b, u)
        defrg, _ = urlparse.urldefrag(joined)
        if defrg not in loaded:
            loaded.add(defrg)
            # Use fetch_text to get raw file (before preprocessing).
            text = document_loader.fetch_text(defrg)
            if isinstance(text, bytes):
                textIO = StringIO(text.decode('utf-8'))
            else:
                textIO = StringIO(text)
            return yaml.safe_load(textIO)
        else:
            return {}

    if loadref_run:
        loadref_fields = set(("$import", "run"))
    else:
        loadref_fields = set(("$import",))

    scanobj = workflowobj
    if "id" in workflowobj:
        # Need raw file content (before preprocessing) to ensure
        # that external references in $include and $mixin are captured.
        scanobj = loadref("", workflowobj["id"])

    sc = scandeps(uri, scanobj,
                  loadref_fields,
                  set(("$include", "$schemas", "location")),
                  loadref)

    normalizeFilesDirs(sc)

    if "id" in workflowobj:
        sc.append({"class": "File", "location": workflowobj["id"]})

    mapper = ArvPathMapper(arvrunner, sc, "",
                           "keep:%s",
                           "keep:%s/%s",
                           name=name)

    def setloc(p):
        if "location" in p and (not p["location"].startswith("_:")) and (not p["location"].startswith("keep:")):
            p["location"] = mapper.mapper(p["location"]).resolved
    adjustFileObjs(workflowobj, setloc)
    adjustDirObjs(workflowobj, setloc)

    return mapper


def upload_docker(arvrunner, tool):
    if isinstance(tool, CommandLineTool):
        (docker_req, docker_is_req) = get_feature(tool, "DockerRequirement")
        if docker_req:
            arv_docker_get_image(arvrunner.api, docker_req, True, arvrunner.project_uuid)
    elif isinstance(tool, cwltool.workflow.Workflow):
        for s in tool.steps:
            upload_docker(arvrunner, s.embedded_tool)

def upload_instance(arvrunner, name, tool, job_order):
        upload_docker(arvrunner, tool)

        workflowmapper = upload_dependencies(arvrunner,
                                             name,
                                             tool.doc_loader,
                                             tool.tool,
                                             tool.tool["id"],
                                             True)

        jobmapper = upload_dependencies(arvrunner,
                                        os.path.basename(job_order.get("id", "#")),
                                        tool.doc_loader,
                                        job_order,
                                        job_order.get("id", "#"),
                                        False)

        if "id" in job_order:
            del job_order["id"]

        return workflowmapper


class Runner(object):
    def __init__(self, runner, tool, job_order, enable_reuse, output_name):
        self.arvrunner = runner
        self.tool = tool
        self.job_order = job_order
        self.running = False
        self.enable_reuse = enable_reuse
        self.uuid = None
        self.final_output = None
        self.output_name = output_name

    def update_pipeline_component(self, record):
        pass

    def arvados_job_spec(self, *args, **kwargs):
        self.name = os.path.basename(self.tool.tool["id"])
        workflowmapper = upload_instance(self.arvrunner, self.name, self.tool, self.job_order)
        adjustDirObjs(self.job_order, trim_listing)
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
                self.final_output = record["output"]
                outc = arvados.collection.CollectionReader(self.final_output,
                                                           api_client=self.arvrunner.api,
                                                           keep_client=self.arvrunner.keep_client,
                                                           num_retries=self.arvrunner.num_retries)
                with outc.open("cwl.output.json") as f:
                    outputs = json.load(f)
                def keepify(fileobj):
                    path = fileobj["location"]
                    if not path.startswith("keep:"):
                        fileobj["location"] = "keep:%s/%s" % (record["output"], path)
                adjustFileObjs(outputs, keepify)
                adjustDirObjs(outputs, keepify)
            except Exception as e:
                logger.error("While getting final output object: %s", e)
            self.arvrunner.output_callback(outputs, processStatus)
        finally:
            del self.arvrunner.processes[record["uuid"]]
