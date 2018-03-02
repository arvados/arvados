# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import os
import urlparse
from functools import partial
import logging
import json
import subprocess

from StringIO import StringIO

from schema_salad.sourceline import SourceLine

from cwltool.command_line_tool import CommandLineTool
import cwltool.workflow
from cwltool.process import get_feature, scandeps, UnsupportedRequirement, normalizeFilesDirs, shortname
from cwltool.load_tool import fetch_document
from cwltool.pathmapper import adjustFileObjs, adjustDirObjs, visit_class
from cwltool.utils import aslist
from cwltool.builder import substitute
from cwltool.pack import pack

import arvados.collection
import ruamel.yaml as yaml

from .arvdocker import arv_docker_get_image
from .pathmapper import ArvPathMapper, trim_listing
from ._version import __version__
from . import done

logger = logging.getLogger('arvados.cwl-runner')

def trim_anonymous_location(obj):
    """Remove 'location' field from File and Directory literals.

    To make internal handling easier, literals are assigned a random id for
    'location'.  However, when writing the record back out, this can break
    reproducibility.  Since it is valid for literals not have a 'location'
    field, remove it.

    """

    if obj.get("location", "").startswith("_:"):
        del obj["location"]

def remove_redundant_fields(obj):
    for field in ("path", "nameext", "nameroot", "dirname"):
        if field in obj:
            del obj[field]

def find_defaults(d, op):
    if isinstance(d, list):
        for i in d:
            find_defaults(i, op)
    elif isinstance(d, dict):
        if "default" in d:
            op(d)
        else:
            for i in d.itervalues():
                find_defaults(i, op)

def upload_dependencies(arvrunner, name, document_loader,
                        workflowobj, uri, loadref_run, include_primary=True):
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

    if include_primary and "id" in workflowobj:
        sc.append({"class": "File", "location": workflowobj["id"]})

    if "$schemas" in workflowobj:
        for s in workflowobj["$schemas"]:
            sc.append({"class": "File", "location": s})

    def capture_default(obj):
        remove = [False]
        def add_default(f):
            if "location" not in f and "path" in f:
                f["location"] = f["path"]
                del f["path"]
            if "location" in f and not arvrunner.fs_access.exists(f["location"]):
                # Remove from sc
                sc[:] = [x for x in sc if x["location"] != f["location"]]
                # Delete "default" from workflowobj
                remove[0] = True
        visit_class(obj["default"], ("File", "Directory"), add_default)
        if remove[0]:
            del obj["default"]

    find_defaults(workflowobj, capture_default)

    mapper = ArvPathMapper(arvrunner, sc, "",
                           "keep:%s",
                           "keep:%s/%s",
                           name=name,
                           single_collection=True)

    def setloc(p):
        if "location" in p and (not p["location"].startswith("_:")) and (not p["location"].startswith("keep:")):
            p["location"] = mapper.mapper(p["location"]).resolved
    adjustFileObjs(workflowobj, setloc)
    adjustDirObjs(workflowobj, setloc)

    if "$schemas" in workflowobj:
        sch = []
        for s in workflowobj["$schemas"]:
            sch.append(mapper.mapper(s).resolved)
        workflowobj["$schemas"] = sch

    return mapper


def upload_docker(arvrunner, tool):
    """Uploads Docker images used in CommandLineTool objects."""

    if isinstance(tool, CommandLineTool):
        (docker_req, docker_is_req) = get_feature(tool, "DockerRequirement")
        if docker_req:
            if docker_req.get("dockerOutputDirectory") and arvrunner.work_api != "containers":
                # TODO: can be supported by containers API, but not jobs API.
                raise SourceLine(docker_req, "dockerOutputDirectory", UnsupportedRequirement).makeError(
                    "Option 'dockerOutputDirectory' of DockerRequirement not supported.")
            arv_docker_get_image(arvrunner.api, docker_req, True, arvrunner.project_uuid)
        else:
            arv_docker_get_image(arvrunner.api, {"dockerPull": "arvados/jobs"}, True, arvrunner.project_uuid)
    elif isinstance(tool, cwltool.workflow.Workflow):
        for s in tool.steps:
            upload_docker(arvrunner, s.embedded_tool)

def packed_workflow(arvrunner, tool, merged_map):
    """Create a packed workflow.

    A "packed" workflow is one where all the components have been combined into a single document."""

    rewrites = {}
    packed = pack(tool.doc_loader, tool.doc_loader.fetch(tool.tool["id"]),
                  tool.tool["id"], tool.metadata, rewrite_out=rewrites)

    rewrite_to_orig = {}
    for k,v in rewrites.items():
        rewrite_to_orig[v] = k

    def visit(v, cur_id):
        if isinstance(v, dict):
            if v.get("class") in ("CommandLineTool", "Workflow"):
                cur_id = rewrite_to_orig.get(v["id"], v["id"])
            if "location" in v and not v["location"].startswith("keep:"):
                v["location"] = merged_map[cur_id][v["location"]]
            for l in v:
                visit(v[l], cur_id)
        if isinstance(v, list):
            for l in v:
                visit(l, cur_id)
    visit(packed, None)
    return packed

def tag_git_version(packed):
    if tool.tool["id"].startswith("file://"):
        path = os.path.dirname(tool.tool["id"][7:])
        try:
            githash = subprocess.check_output(['git', 'log', '--first-parent', '--max-count=1', '--format=%H'], stderr=subprocess.STDOUT, cwd=path).strip()
        except (OSError, subprocess.CalledProcessError):
            pass
        else:
            packed["http://schema.org/version"] = githash


def discover_secondary_files(inputs, job_order):
    for t in inputs:
        def setSecondary(fileobj):
            if isinstance(fileobj, dict) and fileobj.get("class") == "File":
                if "secondaryFiles" not in fileobj:
                    fileobj["secondaryFiles"] = [{"location": substitute(fileobj["location"], sf), "class": "File"} for sf in t["secondaryFiles"]]

            if isinstance(fileobj, list):
                for e in fileobj:
                    setSecondary(e)

        if shortname(t["id"]) in job_order and t.get("secondaryFiles"):
            setSecondary(job_order[shortname(t["id"])])

def upload_job_order(arvrunner, name, tool, job_order):
    """Upload local files referenced in the input object and return updated input
    object with 'location' updated to the proper keep references.
    """

    discover_secondary_files(tool.tool["inputs"], job_order)

    jobmapper = upload_dependencies(arvrunner,
                                    name,
                                    tool.doc_loader,
                                    job_order,
                                    job_order.get("id", "#"),
                                    False)

    if "id" in job_order:
        del job_order["id"]

    # Need to filter this out, gets added by cwltool when providing
    # parameters on the command line.
    if "job_order" in job_order:
        del job_order["job_order"]

    return job_order

def upload_workflow_deps(arvrunner, tool):
    # Ensure that Docker images needed by this workflow are available

    upload_docker(arvrunner, tool)

    document_loader = tool.doc_loader

    merged_map = {}

    def upload_tool_deps(deptool):
        if "id" in deptool:
            pm = upload_dependencies(arvrunner,
                                "%s dependencies" % (shortname(deptool["id"])),
                                document_loader,
                                deptool,
                                deptool["id"],
                                False,
                                include_primary=False)
            document_loader.idx[deptool["id"]] = deptool
            toolmap = {}
            for k,v in pm.items():
                toolmap[k] = v.resolved
            merged_map[deptool["id"]] = toolmap

    tool.visit(upload_tool_deps)

    return merged_map

def arvados_jobs_image(arvrunner, img):
    """Determine if the right arvados/jobs image version is available.  If not, try to pull and upload it."""

    try:
        arv_docker_get_image(arvrunner.api, {"dockerPull": img}, True, arvrunner.project_uuid)
    except Exception as e:
        raise Exception("Docker image %s is not available\n%s" % (img, e) )
    return img

def upload_workflow_collection(arvrunner, name, packed):
    collection = arvados.collection.Collection(api_client=arvrunner.api,
                                               keep_client=arvrunner.keep_client,
                                               num_retries=arvrunner.num_retries)
    with collection.open("workflow.cwl", "w") as f:
        f.write(json.dumps(packed, indent=2, sort_keys=True, separators=(',',': ')))

    filters = [["portable_data_hash", "=", collection.portable_data_hash()],
               ["name", "like", name+"%"]]
    if arvrunner.project_uuid:
        filters.append(["owner_uuid", "=", arvrunner.project_uuid])
    exists = arvrunner.api.collections().list(filters=filters).execute(num_retries=arvrunner.num_retries)

    if exists["items"]:
        logger.info("Using collection %s", exists["items"][0]["uuid"])
    else:
        collection.save_new(name=name,
                            owner_uuid=arvrunner.project_uuid,
                            ensure_unique_name=True,
                            num_retries=arvrunner.num_retries)
        logger.info("Uploaded to %s", collection.manifest_locator())

    return collection.portable_data_hash()


class Runner(object):
    """Base class for runner processes, which submit an instance of
    arvados-cwl-runner and wait for the final result."""

    def __init__(self, runner, tool, job_order, enable_reuse,
                 output_name, output_tags, submit_runner_ram=0,
                 name=None, on_error=None, submit_runner_image=None,
                 intermediate_output_ttl=0, merged_map=None):
        self.arvrunner = runner
        self.tool = tool
        self.job_order = job_order
        self.running = False
        if enable_reuse:
            # If reuse is permitted by command line arguments but
            # disabled by the workflow itself, disable it.
            reuse_req, _ = get_feature(self.tool, "http://arvados.org/cwl#ReuseRequirement")
            if reuse_req:
                enable_reuse = reuse_req["enableReuse"]
        self.enable_reuse = enable_reuse
        self.uuid = None
        self.final_output = None
        self.output_name = output_name
        self.output_tags = output_tags
        self.name = name
        self.on_error = on_error
        self.jobs_image = submit_runner_image or "arvados/jobs:"+__version__
        self.intermediate_output_ttl = intermediate_output_ttl

        if submit_runner_ram:
            self.submit_runner_ram = submit_runner_ram
        else:
            self.submit_runner_ram = 3000

        if self.submit_runner_ram <= 0:
            raise Exception("Value of --submit-runner-ram must be greater than zero")

        self.merged_map = merged_map or {}

    def update_pipeline_component(self, record):
        pass

    def done(self, record):
        """Base method for handling a completed runner."""

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
