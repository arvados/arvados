import os
import urlparse
from functools import partial
import logging
import json
import re
from cStringIO import StringIO

from schema_salad.sourceline import SourceLine

import cwltool.draft2tool
from cwltool.draft2tool import CommandLineTool
import cwltool.workflow
from cwltool.process import get_feature, scandeps, UnsupportedRequirement, normalizeFilesDirs, shortname
from cwltool.load_tool import fetch_document
from cwltool.pathmapper import adjustFileObjs, adjustDirObjs
from cwltool.utils import aslist
from cwltool.builder import substitute

import arvados.collection
import ruamel.yaml as yaml

from .arvdocker import arv_docker_get_image
from .pathmapper import ArvPathMapper
from ._version import __version__
from . import done

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
        joined = document_loader.fetcher.urljoin(b, u)
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
                  loadref, urljoin=document_loader.fetcher.urljoin)

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
            if docker_req.get("dockerOutputDirectory"):
                # TODO: can be supported by containers API, but not jobs API.
                raise SourceLine(docker_req, "dockerOutputDirectory", UnsupportedRequirement).makeError(
                    "Option 'dockerOutputDirectory' of DockerRequirement not supported.")
            arv_docker_get_image(arvrunner.api, docker_req, True, arvrunner.project_uuid)
    elif isinstance(tool, cwltool.workflow.Workflow):
        for s in tool.steps:
            upload_docker(arvrunner, s.embedded_tool)

def upload_instance(arvrunner, name, tool, job_order):
        upload_docker(arvrunner, tool)

        for t in tool.tool["inputs"]:
            def setSecondary(fileobj):
                if isinstance(fileobj, dict) and fileobj.get("class") == "File":
                    if "secondaryFiles" not in fileobj:
                        fileobj["secondaryFiles"] = [{"location": substitute(fileobj["location"], sf), "class": "File"} for sf in t["secondaryFiles"]]

                if isinstance(fileobj, list):
                    for e in fileobj:
                        setSecondary(e)

            if shortname(t["id"]) in job_order and t.get("secondaryFiles"):
                setSecondary(job_order[shortname(t["id"])])

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

def arvados_jobs_image(arvrunner):
    img = "arvados/jobs:"+__version__
    try:
        arv_docker_get_image(arvrunner.api, {"dockerPull": img}, True, arvrunner.project_uuid)
    except Exception as e:
        raise Exception("Docker image %s is not available\n%s" % (img, e) )
    return img

class Runner(object):
    def __init__(self, runner, tool, job_order, enable_reuse,
                 output_name, output_tags, submit_runner_ram=0,
                 name=None):
        self.arvrunner = runner
        self.tool = tool
        self.job_order = job_order
        self.running = False
        self.enable_reuse = enable_reuse
        self.uuid = None
        self.final_output = None
        self.output_name = output_name
        self.output_tags = output_tags
        self.name = name

        if submit_runner_ram:
            self.submit_runner_ram = submit_runner_ram
        else:
            self.submit_runner_ram = 1024

        if self.submit_runner_ram <= 0:
            raise Exception("Value of --submit-runner-ram must be greater than zero")

    def update_pipeline_component(self, record):
        pass

    def arvados_job_spec(self, *args, **kwargs):
        if self.name is None:
            self.name = self.tool.tool.get("label") or os.path.basename(self.tool.tool["id"])

        # Need to filter this out, gets added by cwltool when providing
        # parameters on the command line.
        if "job_order" in self.job_order:
            del self.job_order["job_order"]

        workflowmapper = upload_instance(self.arvrunner, self.name, self.tool, self.job_order)
        adjustDirObjs(self.job_order, trim_listing)
        return workflowmapper

    def done(self, record):
        try:
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

            outputs = {}

            if processStatus == "permanentFail":
                logc = arvados.collection.CollectionReader(record["log"],
                                                           api_client=self.arvrunner.api,
                                                           keep_client=self.arvrunner.keep_client,
                                                           num_retries=self.arvrunner.num_retries)
                done.logtail(logc, logger, "%s error log:" % self.arvrunner.label(self), maxlen=40)

            self.final_output = record["output"]
            outc = arvados.collection.CollectionReader(self.final_output,
                                                       api_client=self.arvrunner.api,
                                                       keep_client=self.arvrunner.keep_client,
                                                       num_retries=self.arvrunner.num_retries)
            if "cwl.output.json" in outc:
                with outc.open("cwl.output.json") as f:
                    if f.size() > 0:
                        outputs = json.load(f)
            def keepify(fileobj):
                path = fileobj["location"]
                if not path.startswith("keep:"):
                    fileobj["location"] = "keep:%s/%s" % (record["output"], path)
            adjustFileObjs(outputs, keepify)
            adjustDirObjs(outputs, keepify)
        except Exception as e:
            logger.exception("[%s] While getting final output object: %s", self.name, e)
            self.arvrunner.output_callback({}, "permanentFail")
        else:
            self.arvrunner.output_callback(outputs, processStatus)
        finally:
            if record["uuid"] in self.arvrunner.processes:
                del self.arvrunner.processes[record["uuid"]]
