#!/usr/bin/env python

# Implement cwl-runner interface for submitting and running jobs on Arvados.

import argparse
import arvados
import arvados.collection
import arvados.commands.keepdocker
import arvados.commands.run
import arvados.events
import arvados.util
import copy
import cwltool.docker
from cwltool.draft2tool import revmap_file, remove_hostfs, CommandLineTool
from cwltool.errors import WorkflowException
import cwltool.main
import cwltool.workflow
import fnmatch
from functools import partial
import json
import logging
import os
import pkg_resources  # part of setuptools
import re
import sys
import threading
from cwltool.load_tool import fetch_document
from cwltool.builder import Builder
import urlparse

from cwltool.process import shortname, get_feature, adjustFiles, adjustFileObjs, scandeps
from arvados.api import OrderedJsonModel

logger = logging.getLogger('arvados.cwl-runner')
logger.setLevel(logging.INFO)

tmpdirre = re.compile(r"^\S+ \S+ \d+ \d+ stderr \S+ \S+ crunchrunner: \$\(task\.tmpdir\)=(.*)")
outdirre = re.compile(r"^\S+ \S+ \d+ \d+ stderr \S+ \S+ crunchrunner: \$\(task\.outdir\)=(.*)")
keepre = re.compile(r"^\S+ \S+ \d+ \d+ stderr \S+ \S+ crunchrunner: \$\(task\.keep\)=(.*)")


def arv_docker_get_image(api_client, dockerRequirement, pull_image, project_uuid):
    """Check if a Docker image is available in Keep, if not, upload it using arv-keepdocker."""

    if "dockerImageId" not in dockerRequirement and "dockerPull" in dockerRequirement:
        dockerRequirement["dockerImageId"] = dockerRequirement["dockerPull"]

    sp = dockerRequirement["dockerImageId"].split(":")
    image_name = sp[0]
    image_tag = sp[1] if len(sp) > 1 else None

    images = arvados.commands.keepdocker.list_images_in_arv(api_client, 3,
                                                            image_name=image_name,
                                                            image_tag=image_tag)

    if not images:
        imageId = cwltool.docker.get_image(dockerRequirement, pull_image)
        args = ["--project-uuid="+project_uuid, image_name]
        if image_tag:
            args.append(image_tag)
        logger.info("Uploading Docker image %s", ":".join(args[1:]))
        arvados.commands.keepdocker.main(args, stdout=sys.stderr)

    return dockerRequirement["dockerImageId"]


class CollectionFsAccess(cwltool.process.StdFsAccess):
    """Implement the cwltool FsAccess interface for Arvados Collections."""

    def __init__(self, basedir):
        super(CollectionFsAccess, self).__init__(basedir)
        self.collections = {}

    def get_collection(self, path):
        p = path.split("/")
        if p[0].startswith("keep:") and arvados.util.keep_locator_pattern.match(p[0][5:]):
            pdh = p[0][5:]
            if pdh not in self.collections:
                self.collections[pdh] = arvados.collection.CollectionReader(pdh)
            return (self.collections[pdh], "/".join(p[1:]))
        else:
            return (None, path)

    def _match(self, collection, patternsegments, parent):
        if not patternsegments:
            return []

        if not isinstance(collection, arvados.collection.RichCollectionBase):
            return []

        ret = []
        # iterate over the files and subcollections in 'collection'
        for filename in collection:
            if patternsegments[0] == '.':
                # Pattern contains something like "./foo" so just shift
                # past the "./"
                ret.extend(self._match(collection, patternsegments[1:], parent))
            elif fnmatch.fnmatch(filename, patternsegments[0]):
                cur = os.path.join(parent, filename)
                if len(patternsegments) == 1:
                    ret.append(cur)
                else:
                    ret.extend(self._match(collection[filename], patternsegments[1:], cur))
        return ret

    def glob(self, pattern):
        collection, rest = self.get_collection(pattern)
        patternsegments = rest.split("/")
        return self._match(collection, patternsegments, "keep:" + collection.manifest_locator())

    def open(self, fn, mode):
        collection, rest = self.get_collection(fn)
        if collection:
            return collection.open(rest, mode)
        else:
            return open(self._abs(fn), mode)

    def exists(self, fn):
        collection, rest = self.get_collection(fn)
        if collection:
            return collection.exists(rest)
        else:
            return os.path.exists(self._abs(fn))

class ArvadosJob(object):
    """Submit and manage a Crunch job for executing a CWL CommandLineTool."""

    def __init__(self, runner):
        self.arvrunner = runner
        self.running = False

    def run(self, dry_run=False, pull_image=True, **kwargs):
        script_parameters = {
            "command": self.command_line
        }
        runtime_constraints = {}

        if self.generatefiles:
            vwd = arvados.collection.Collection()
            script_parameters["task.vwd"] = {}
            for t in self.generatefiles:
                if isinstance(self.generatefiles[t], dict):
                    src, rest = self.arvrunner.fs_access.get_collection(self.generatefiles[t]["path"].replace("$(task.keep)/", "keep:"))
                    vwd.copy(rest, t, source_collection=src)
                else:
                    with vwd.open(t, "w") as f:
                        f.write(self.generatefiles[t])
            vwd.save_new()
            for t in self.generatefiles:
                script_parameters["task.vwd"][t] = "$(task.keep)/%s/%s" % (vwd.portable_data_hash(), t)

        script_parameters["task.env"] = {"TMPDIR": "$(task.tmpdir)"}
        if self.environment:
            script_parameters["task.env"].update(self.environment)

        if self.stdin:
            script_parameters["task.stdin"] = self.pathmapper.mapper(self.stdin)[1]

        if self.stdout:
            script_parameters["task.stdout"] = self.stdout

        (docker_req, docker_is_req) = get_feature(self, "DockerRequirement")
        if docker_req and kwargs.get("use_container") is not False:
            runtime_constraints["docker_image"] = arv_docker_get_image(self.arvrunner.api, docker_req, pull_image, self.arvrunner.project_uuid)
        else:
            runtime_constraints["docker_image"] = "arvados/jobs"

        resources = self.builder.resources
        if resources is not None:
            runtime_constraints["min_cores_per_node"] = resources.get("cores", 1)
            runtime_constraints["min_ram_mb_per_node"] = resources.get("ram")
            runtime_constraints["min_scratch_mb_per_node"] = resources.get("tmpdirSize", 0) + resources.get("outdirSize", 0)

        filters = [["repository", "=", "arvados"],
                   ["script", "=", "crunchrunner"],
                   ["script_version", "in git", "9e5b98e8f5f4727856b53447191f9c06e3da2ba6"]]
        if not self.arvrunner.ignore_docker_for_reuse:
            filters.append(["docker_image_locator", "in docker", runtime_constraints["docker_image"]])

        try:
            response = self.arvrunner.api.jobs().create(
                body={
                    "owner_uuid": self.arvrunner.project_uuid,
                    "script": "crunchrunner",
                    "repository": "arvados",
                    "script_version": "master",
                    "minimum_script_version": "9e5b98e8f5f4727856b53447191f9c06e3da2ba6",
                    "script_parameters": {"tasks": [script_parameters]},
                    "runtime_constraints": runtime_constraints
                },
                filters=filters,
                find_or_create=kwargs.get("enable_reuse", True)
            ).execute(num_retries=self.arvrunner.num_retries)

            self.arvrunner.jobs[response["uuid"]] = self

            self.arvrunner.pipeline["components"][self.name] = {"job": response}
            self.arvrunner.pipeline = self.arvrunner.api.pipeline_instances().update(uuid=self.arvrunner.pipeline["uuid"],
                                                                                     body={
                                                                                         "components": self.arvrunner.pipeline["components"]
                                                                                     }).execute(num_retries=self.arvrunner.num_retries)

            logger.info("Job %s (%s) is %s", self.name, response["uuid"], response["state"])

            if response["state"] in ("Complete", "Failed", "Cancelled"):
                self.done(response)
        except Exception as e:
            logger.error("Got error %s" % str(e))
            self.output_callback({}, "permanentFail")

    def update_pipeline_component(self, record):
        self.arvrunner.pipeline["components"][self.name] = {"job": record}
        self.arvrunner.pipeline = self.arvrunner.api.pipeline_instances().update(uuid=self.arvrunner.pipeline["uuid"],
                                                                                 body={
                                                                                    "components": self.arvrunner.pipeline["components"]
                                                                                 }).execute(num_retries=self.arvrunner.num_retries)

    def done(self, record):
        try:
            self.update_pipeline_component(record)
        except:
            pass

        try:
            if record["state"] == "Complete":
                processStatus = "success"
            else:
                processStatus = "permanentFail"

            try:
                outputs = {}
                if record["output"]:
                    logc = arvados.collection.Collection(record["log"])
                    log = logc.open(logc.keys()[0])
                    tmpdir = None
                    outdir = None
                    keepdir = None
                    for l in log:
                        # Determine the tmpdir, outdir and keepdir paths from
                        # the job run.  Unfortunately, we can't take the first
                        # values we find (which are expected to be near the
                        # top) and stop scanning because if the node fails and
                        # the job restarts on a different node these values
                        # will different runs, and we need to know about the
                        # final run that actually produced output.

                        g = tmpdirre.match(l)
                        if g:
                            tmpdir = g.group(1)
                        g = outdirre.match(l)
                        if g:
                            outdir = g.group(1)
                        g = keepre.match(l)
                        if g:
                            keepdir = g.group(1)

                    colname = "Output %s of %s" % (record["output"][0:7], self.name)

                    # check if collection already exists with same owner, name and content
                    collection_exists = self.arvrunner.api.collections().list(
                        filters=[["owner_uuid", "=", self.arvrunner.project_uuid],
                                 ['portable_data_hash', '=', record["output"]],
                                 ["name", "=", colname]]
                    ).execute(num_retries=self.arvrunner.num_retries)

                    if not collection_exists["items"]:
                        # Create a collection located in the same project as the
                        # pipeline with the contents of the output.
                        # First, get output record.
                        collections = self.arvrunner.api.collections().list(
                            limit=1,
                            filters=[['portable_data_hash', '=', record["output"]]],
                            select=["manifest_text"]
                        ).execute(num_retries=self.arvrunner.num_retries)

                        if not collections["items"]:
                            raise WorkflowException(
                                "Job output '%s' cannot be found on API server" % (
                                    record["output"]))

                        # Create new collection in the parent project
                        # with the output contents.
                        self.arvrunner.api.collections().create(body={
                            "owner_uuid": self.arvrunner.project_uuid,
                            "name": colname,
                            "portable_data_hash": record["output"],
                            "manifest_text": collections["items"][0]["manifest_text"]
                        }, ensure_unique_name=True).execute(
                            num_retries=self.arvrunner.num_retries)

                    self.builder.outdir = outdir
                    self.builder.pathmapper.keepdir = keepdir
                    outputs = self.collect_outputs("keep:" + record["output"])
            except WorkflowException as e:
                logger.error("Error while collecting job outputs:\n%s", e, exc_info=(e if self.arvrunner.debug else False))
                processStatus = "permanentFail"
            except Exception as e:
                logger.exception("Got unknown exception while collecting job outputs:")
                processStatus = "permanentFail"

            self.output_callback(outputs, processStatus)
        finally:
            del self.arvrunner.jobs[record["uuid"]]


class RunnerJob(object):
    """Submit and manage a Crunch job that runs crunch_scripts/cwl-runner."""

    def __init__(self, runner, tool, job_order, enable_reuse):
        self.arvrunner = runner
        self.tool = tool
        self.job_order = job_order
        self.running = False
        self.enable_reuse = enable_reuse

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

    def arvados_job_spec(self, dry_run=False, pull_image=True, **kwargs):
        """Create an Arvados job specification for this workflow.

        The returned dict can be used to create a job (i.e., passed as
        the +body+ argument to jobs().create()), or as a component in
        a pipeline template or pipeline instance.
        """
        self.upload_docker(self.tool)

        workflowfiles = set()
        jobfiles = set()
        workflowfiles.add(self.tool.tool["id"])

        self.name = os.path.basename(self.tool.tool["id"])

        def visitFiles(files, path):
            files.add(path)
            return path

        document_loader, workflowobj, uri = fetch_document(self.tool.tool["id"])
        def loadref(b, u):
            return document_loader.fetch(urlparse.urljoin(b, u))

        sc = scandeps(uri, workflowobj,
                      set(("$import", "run")),
                      set(("$include", "$schemas", "path")),
                      loadref)
        adjustFiles(sc, partial(visitFiles, workflowfiles))
        adjustFiles(self.job_order, partial(visitFiles, jobfiles))

        workflowmapper = ArvPathMapper(self.arvrunner, workflowfiles, "",
                                       "%s",
                                       "%s/%s",
                                       name=self.name,
                                       **kwargs)

        jobmapper = ArvPathMapper(self.arvrunner, jobfiles, "",
                                  "%s",
                                  "%s/%s",
                                  name=os.path.basename(self.job_order.get("id", "#")),
                                  **kwargs)

        adjustFiles(self.job_order, lambda p: jobmapper.mapper(p)[1])

        if "id" in self.job_order:
            del self.job_order["id"]

        self.job_order["cwl:tool"] = workflowmapper.mapper(self.tool.tool["id"])[1]
        return {
            "script": "cwl-runner",
            "script_version": "master",
            "repository": "arvados",
            "script_parameters": self.job_order,
            "runtime_constraints": {
                "docker_image": "arvados/jobs"
            }
        }

    def run(self, *args, **kwargs):
        job_spec = self.arvados_job_spec(*args, **kwargs)
        job_spec.setdefault("owner_uuid", self.arvrunner.project_uuid)

        response = self.arvrunner.api.jobs().create(
            body=job_spec,
            find_or_create=self.enable_reuse
        ).execute(num_retries=self.arvrunner.num_retries)

        self.uuid = response["uuid"]
        self.arvrunner.jobs[self.uuid] = self

        logger.info("Submitted job %s", response["uuid"])

        if response["state"] in ("Complete", "Failed", "Cancelled"):
            self.done(response)

    def done(self, record):
        if record["state"] == "Complete":
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
                adjustFiles(outputs, keepify)
            except Exception as e:
                logger.error("While getting final output object: %s", e)
            self.arvrunner.output_callback(outputs, processStatus)
        finally:
            del self.arvrunner.jobs[record["uuid"]]


class RunnerTemplate(object):
    """An Arvados pipeline template that invokes a CWL workflow."""

    type_to_dataclass = {
        'boolean': 'boolean',
        'File': 'File',
        'float': 'number',
        'int': 'number',
        'string': 'text',
    }

    def __init__(self, runner, tool, job_order, enable_reuse):
        self.runner = runner
        self.tool = tool
        self.job = RunnerJob(
            runner=runner,
            tool=tool,
            job_order=job_order,
            enable_reuse=enable_reuse)

    def pipeline_component_spec(self):
        """Return a component that Workbench and a-r-p-i will understand.

        Specifically, translate CWL input specs to Arvados pipeline
        format, like {"dataclass":"File","value":"xyz"}.
        """
        spec = self.job.arvados_job_spec()

        # Most of the component spec is exactly the same as the job
        # spec (script, script_version, etc.).
        # spec['script_parameters'] isn't right, though. A component
        # spec's script_parameters hash is a translation of
        # self.tool.tool['inputs'] with defaults/overrides taken from
        # the job order. So we move the job parameters out of the way
        # and build a new spec['script_parameters'].
        job_params = spec['script_parameters']
        spec['script_parameters'] = {}

        for param in self.tool.tool['inputs']:
            param = copy.deepcopy(param)

            # Data type and "required" flag...
            types = param['type']
            if not isinstance(types, list):
                types = [types]
            param['required'] = 'null' not in types
            non_null_types = set(types) - set(['null'])
            if len(non_null_types) == 1:
                the_type = [c for c in non_null_types][0]
                dataclass = self.type_to_dataclass.get(the_type)
                if dataclass:
                    param['dataclass'] = dataclass
            # Note: If we didn't figure out a single appropriate
            # dataclass, we just left that attribute out.  We leave
            # the "type" attribute there in any case, which might help
            # downstream.

            # Title and description...
            title = param.pop('label', '')
            descr = param.pop('description', '').rstrip('\n')
            if title:
                param['title'] = title
            if descr:
                param['description'] = descr

            # Fill in the value from the current job order, if any.
            param_id = shortname(param.pop('id'))
            value = job_params.get(param_id)
            if value is None:
                pass
            elif not isinstance(value, dict):
                param['value'] = value
            elif param.get('dataclass') == 'File' and value.get('path'):
                param['value'] = value['path']

            spec['script_parameters'][param_id] = param
        spec['script_parameters']['cwl:tool'] = job_params['cwl:tool']
        return spec

    def save(self):
        job_spec = self.pipeline_component_spec()
        response = self.runner.api.pipeline_templates().create(body={
            "components": {
                self.job.name: job_spec,
            },
            "name": self.job.name,
            "owner_uuid": self.runner.project_uuid,
        }, ensure_unique_name=True).execute(num_retries=self.runner.num_retries)
        self.uuid = response["uuid"]
        logger.info("Created template %s", self.uuid)


class ArvPathMapper(cwltool.pathmapper.PathMapper):
    """Convert container-local paths to and from Keep collection ids."""

    def __init__(self, arvrunner, referenced_files, input_basedir,
                 collection_pattern, file_pattern, name=None, **kwargs):
        self._pathmap = arvrunner.get_uploaded()
        uploadfiles = set()

        pdh_path = re.compile(r'^keep:[0-9a-f]{32}\+\d+/.+')

        for src in referenced_files:
            if isinstance(src, basestring) and pdh_path.match(src):
                self._pathmap[src] = (src, collection_pattern % src[5:])
            if "#" in src:
                src = src[:src.index("#")]
            if src not in self._pathmap:
                ab = cwltool.pathmapper.abspath(src, input_basedir)
                st = arvados.commands.run.statfile("", ab, fnPattern=file_pattern)
                if kwargs.get("conformance_test"):
                    self._pathmap[src] = (src, ab)
                elif isinstance(st, arvados.commands.run.UploadFile):
                    uploadfiles.add((src, ab, st))
                elif isinstance(st, arvados.commands.run.ArvFile):
                    self._pathmap[src] = (ab, st.fn)
                else:
                    raise cwltool.workflow.WorkflowException("Input file path '%s' is invalid" % st)

        if uploadfiles:
            arvados.commands.run.uploadfiles([u[2] for u in uploadfiles],
                                             arvrunner.api,
                                             dry_run=kwargs.get("dry_run"),
                                             num_retries=3,
                                             fnPattern=file_pattern,
                                             name=name,
                                             project=arvrunner.project_uuid)

        for src, ab, st in uploadfiles:
            arvrunner.add_uploaded(src, (ab, st.fn))
            self._pathmap[src] = (ab, st.fn)

        self.keepdir = None

    def reversemap(self, target):
        if target.startswith("keep:"):
            return (target, target)
        elif self.keepdir and target.startswith(self.keepdir):
            return (target, "keep:" + target[len(self.keepdir)+1:])
        else:
            return super(ArvPathMapper, self).reversemap(target)


class ArvadosCommandTool(CommandLineTool):
    """Wrap cwltool CommandLineTool to override selected methods."""

    def __init__(self, arvrunner, toolpath_object, **kwargs):
        super(ArvadosCommandTool, self).__init__(toolpath_object, **kwargs)
        self.arvrunner = arvrunner

    def makeJobRunner(self):
        return ArvadosJob(self.arvrunner)

    def makePathMapper(self, reffiles, **kwargs):
        return ArvPathMapper(self.arvrunner, reffiles, kwargs["basedir"],
                             "$(task.keep)/%s",
                             "$(task.keep)/%s/%s",
                             **kwargs)


class ArvCwlRunner(object):
    """Execute a CWL tool or workflow, submit crunch jobs, wait for them to
    complete, and report output."""

    def __init__(self, api_client):
        self.api = api_client
        self.jobs = {}
        self.lock = threading.Lock()
        self.cond = threading.Condition(self.lock)
        self.final_output = None
        self.uploaded = {}
        self.num_retries = 4

    def arvMakeTool(self, toolpath_object, **kwargs):
        if "class" in toolpath_object and toolpath_object["class"] == "CommandLineTool":
            return ArvadosCommandTool(self, toolpath_object, **kwargs)
        else:
            return cwltool.workflow.defaultMakeTool(toolpath_object, **kwargs)

    def output_callback(self, out, processStatus):
        if processStatus == "success":
            logger.info("Overall job status is %s", processStatus)
            if self.pipeline:
                self.api.pipeline_instances().update(uuid=self.pipeline["uuid"],
                                                     body={"state": "Complete"}).execute(num_retries=self.num_retries)

        else:
            logger.warn("Overall job status is %s", processStatus)
            if self.pipeline:
                self.api.pipeline_instances().update(uuid=self.pipeline["uuid"],
                                                     body={"state": "Failed"}).execute(num_retries=self.num_retries)
        self.final_output = out

    def on_message(self, event):
        if "object_uuid" in event:
            if event["object_uuid"] in self.jobs and event["event_type"] == "update":
                if event["properties"]["new_attributes"]["state"] == "Running" and self.jobs[event["object_uuid"]].running is False:
                    uuid = event["object_uuid"]
                    with self.lock:
                        j = self.jobs[uuid]
                        logger.info("Job %s (%s) is Running", j.name, uuid)
                        j.running = True
                        j.update_pipeline_component(event["properties"]["new_attributes"])
                elif event["properties"]["new_attributes"]["state"] in ("Complete", "Failed", "Cancelled"):
                    uuid = event["object_uuid"]
                    try:
                        self.cond.acquire()
                        j = self.jobs[uuid]
                        logger.info("Job %s (%s) is %s", j.name, uuid, event["properties"]["new_attributes"]["state"])
                        j.done(event["properties"]["new_attributes"])
                        self.cond.notify()
                    finally:
                        self.cond.release()

    def get_uploaded(self):
        return self.uploaded.copy()

    def add_uploaded(self, src, pair):
        self.uploaded[src] = pair

    def arvExecutor(self, tool, job_order, **kwargs):
        self.debug = kwargs.get("debug")

        if kwargs.get("quiet"):
            logger.setLevel(logging.WARN)
            logging.getLogger('arvados.arv-run').setLevel(logging.WARN)

        useruuid = self.api.users().current().execute()["uuid"]
        self.project_uuid = kwargs.get("project_uuid") if kwargs.get("project_uuid") else useruuid
        self.pipeline = None

        if kwargs.get("create_template"):
            tmpl = RunnerTemplate(self, tool, job_order, kwargs.get("enable_reuse"))
            tmpl.save()
            # cwltool.main will write our return value to stdout.
            return tmpl.uuid

        if kwargs.get("submit"):
            runnerjob = RunnerJob(self, tool, job_order, kwargs.get("enable_reuse"))
            if not kwargs.get("wait"):
                runnerjob.run()
                return runnerjob.uuid

        events = arvados.events.subscribe(arvados.api('v1'), [["object_uuid", "is_a", "arvados#job"]], self.on_message)

        self.debug = kwargs.get("debug")
        self.ignore_docker_for_reuse = kwargs.get("ignore_docker_for_reuse")
        self.fs_access = CollectionFsAccess(kwargs["basedir"])

        kwargs["fs_access"] = self.fs_access
        kwargs["enable_reuse"] = kwargs.get("enable_reuse")

        kwargs["outdir"] = "$(task.outdir)"
        kwargs["tmpdir"] = "$(task.tmpdir)"

        if kwargs.get("conformance_test"):
            return cwltool.main.single_job_executor(tool, job_order, **kwargs)
        else:
            if kwargs.get("submit"):
                jobiter = iter((runnerjob,))
            else:
                components = {}
                if "cwl_runner_job" in kwargs:
                    components[os.path.basename(tool.tool["id"])] = {"job": kwargs["cwl_runner_job"]}

                self.pipeline = self.api.pipeline_instances().create(
                    body={
                        "owner_uuid": self.project_uuid,
                        "name": shortname(tool.tool["id"]),
                        "components": components,
                        "state": "RunningOnClient"}).execute(num_retries=self.num_retries)

                logger.info("Pipeline instance %s", self.pipeline["uuid"])

                jobiter = tool.job(job_order,
                                   self.output_callback,
                                   docker_outdir="$(task.outdir)",
                                   **kwargs)

            try:
                self.cond.acquire()
                # Will continue to hold the lock for the duration of this code
                # except when in cond.wait(), at which point on_message can update
                # job state and process output callbacks.

                for runnable in jobiter:
                    if runnable:
                        runnable.run(**kwargs)
                    else:
                        if self.jobs:
                            self.cond.wait(1)
                        else:
                            logger.error("Workflow is deadlocked, no runnable jobs and not waiting on any pending jobs.")
                            break

                while self.jobs:
                    self.cond.wait(1)

                events.close()
            except:
                if sys.exc_info()[0] is KeyboardInterrupt:
                    logger.error("Interrupted, marking pipeline as failed")
                else:
                    logger.error("Caught unhandled exception, marking pipeline as failed.  Error was: %s", sys.exc_info()[0], exc_info=(sys.exc_info()[1] if self.debug else False))
                if self.pipeline:
                    self.api.pipeline_instances().update(uuid=self.pipeline["uuid"],
                                                         body={"state": "Failed"}).execute(num_retries=self.num_retries)
            finally:
                self.cond.release()

            if self.final_output is None:
                raise cwltool.workflow.WorkflowException("Workflow did not return a result.")

            return self.final_output

def versionstring():
    """Print version string of key packages for provenance and debugging."""

    arvcwlpkg = pkg_resources.require("arvados-cwl-runner")
    arvpkg = pkg_resources.require("arvados-python-client")
    cwlpkg = pkg_resources.require("cwltool")

    return "%s %s, %s %s, %s %s" % (sys.argv[0], arvcwlpkg[0].version,
                                    "arvados-python-client", arvpkg[0].version,
                                    "cwltool", cwlpkg[0].version)

def arg_parser():  # type: () -> argparse.ArgumentParser
    parser = argparse.ArgumentParser(description='Arvados executor for Common Workflow Language')

    parser.add_argument("--conformance-test", action="store_true")
    parser.add_argument("--basedir", type=str,
                        help="Base directory used to resolve relative references in the input, default to directory of input object file or current directory (if inputs piped/provided on command line).")
    parser.add_argument("--outdir", type=str, default=os.path.abspath('.'),
                        help="Output directory, default current directory")

    parser.add_argument("--eval-timeout",
                        help="Time to wait for a Javascript expression to evaluate before giving an error, default 20s.",
                        type=float,
                        default=20)
    parser.add_argument("--version", action="store_true", help="Print version and exit")

    exgroup = parser.add_mutually_exclusive_group()
    exgroup.add_argument("--verbose", action="store_true", help="Default logging")
    exgroup.add_argument("--quiet", action="store_true", help="Only print warnings and errors.")
    exgroup.add_argument("--debug", action="store_true", help="Print even more logging")

    parser.add_argument("--tool-help", action="store_true", help="Print command line help for tool")

    exgroup = parser.add_mutually_exclusive_group()
    exgroup.add_argument("--enable-reuse", action="store_true",
                        default=True, dest="enable_reuse",
                        help="")
    exgroup.add_argument("--disable-reuse", action="store_false",
                        default=True, dest="enable_reuse",
                        help="")

    parser.add_argument("--project-uuid", type=str, help="Project that will own the workflow jobs, if not provided, will go to home project.")
    parser.add_argument("--ignore-docker-for-reuse", action="store_true",
                        help="Ignore Docker image version when deciding whether to reuse past jobs.",
                        default=False)

    exgroup = parser.add_mutually_exclusive_group()
    exgroup.add_argument("--submit", action="store_true", help="Submit workflow to run on Arvados.",
                        default=True, dest="submit")
    exgroup.add_argument("--local", action="store_false", help="Run workflow on local host (submits jobs to Arvados).",
                        default=True, dest="submit")
    exgroup.add_argument("--create-template", action="store_true", help="Create an Arvados pipeline template.")

    exgroup = parser.add_mutually_exclusive_group()
    exgroup.add_argument("--wait", action="store_true", help="After submitting workflow runner job, wait for completion.",
                        default=True, dest="wait")
    exgroup.add_argument("--no-wait", action="store_false", help="Submit workflow runner job and exit.",
                        default=True, dest="wait")

    parser.add_argument("workflow", type=str, nargs="?", default=None)
    parser.add_argument("job_order", nargs=argparse.REMAINDER)

    return parser

def main(args, stdout, stderr, api_client=None):
    parser = arg_parser()

    job_order_object = None
    arvargs = parser.parse_args(args)
    if arvargs.create_template and not arvargs.job_order:
        job_order_object = ({}, "")

    try:
        if api_client is None:
            api_client=arvados.api('v1', model=OrderedJsonModel())
        runner = ArvCwlRunner(api_client)
    except Exception as e:
        logger.error(e)
        return 1

    return cwltool.main.main(args=arvargs,
                             stdout=stdout,
                             stderr=stderr,
                             executor=runner.arvExecutor,
                             makeTool=runner.arvMakeTool,
                             versionfunc=versionstring,
                             job_order_object=job_order_object)
