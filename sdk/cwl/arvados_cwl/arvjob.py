import logging
import re
import copy
import json
import time

from cwltool.process import get_feature, shortname, UnsupportedRequirement
from cwltool.errors import WorkflowException
from cwltool.draft2tool import revmap_file, CommandLineTool
from cwltool.load_tool import fetch_document
from cwltool.builder import Builder
from cwltool.pathmapper import adjustFileObjs, adjustDirObjs

from schema_salad.sourceline import SourceLine

import ruamel.yaml as yaml

import arvados.collection

from .arvdocker import arv_docker_get_image
from .runner import Runner, arvados_jobs_image, packed_workflow, upload_workflow_collection, trim_anonymous_location
from .pathmapper import VwdPathMapper, trim_listing
from .perf import Perf
from . import done
from ._version import __version__

logger = logging.getLogger('arvados.cwl-runner')
metrics = logging.getLogger('arvados.cwl-runner.metrics')

crunchrunner_re = re.compile(r"^\S+ \S+ \d+ \d+ stderr \S+ \S+ crunchrunner: \$\(task\.(tmpdir|outdir|keep)\)=(.*)")

crunchrunner_git_commit = 'a3f2cb186e437bfce0031b024b2157b73ed2717d'

class ArvadosJob(object):
    """Submit and manage a Crunch job for executing a CWL CommandLineTool."""

    def __init__(self, runner):
        self.arvrunner = runner
        self.running = False
        self.uuid = None

    def run(self, dry_run=False, pull_image=True, **kwargs):
        script_parameters = {
            "command": self.command_line
        }
        runtime_constraints = {}

        with Perf(metrics, "generatefiles %s" % self.name):
            if self.generatefiles["listing"]:
                vwd = arvados.collection.Collection(api_client=self.arvrunner.api,
                                                    keep_client=self.arvrunner.keep_client,
                                                    num_retries=self.arvrunner.num_retries)
                script_parameters["task.vwd"] = {}
                generatemapper = VwdPathMapper([self.generatefiles], "", "",
                                               separateDirs=False)

                with Perf(metrics, "createfiles %s" % self.name):
                    for f, p in generatemapper.items():
                        if p.type == "CreateFile":
                            with vwd.open(p.target, "w") as n:
                                n.write(p.resolved.encode("utf-8"))

                if vwd:
                    with Perf(metrics, "generatefiles.save_new %s" % self.name):
                        vwd.save_new()

                for f, p in generatemapper.items():
                    if p.type == "File":
                        script_parameters["task.vwd"][p.target] = p.resolved
                    if p.type == "CreateFile":
                        script_parameters["task.vwd"][p.target] = "$(task.keep)/%s/%s" % (vwd.portable_data_hash(), p.target)

        script_parameters["task.env"] = {"TMPDIR": self.tmpdir, "HOME": self.outdir}
        if self.environment:
            script_parameters["task.env"].update(self.environment)

        if self.stdin:
            script_parameters["task.stdin"] = self.stdin

        if self.stdout:
            script_parameters["task.stdout"] = self.stdout

        if self.stderr:
            script_parameters["task.stderr"] = self.stderr

        if self.successCodes:
            script_parameters["task.successCodes"] = self.successCodes
        if self.temporaryFailCodes:
            script_parameters["task.temporaryFailCodes"] = self.temporaryFailCodes
        if self.permanentFailCodes:
            script_parameters["task.permanentFailCodes"] = self.permanentFailCodes

        with Perf(metrics, "arv_docker_get_image %s" % self.name):
            (docker_req, docker_is_req) = get_feature(self, "DockerRequirement")
            if docker_req and kwargs.get("use_container") is not False:
                if docker_req.get("dockerOutputDirectory"):
                    raise SourceLine(docker_req, "dockerOutputDirectory", UnsupportedRequirement).makeError(
                        "Option 'dockerOutputDirectory' of DockerRequirement not supported.")
                runtime_constraints["docker_image"] = arv_docker_get_image(self.arvrunner.api, docker_req, pull_image, self.arvrunner.project_uuid)
            else:
                runtime_constraints["docker_image"] = "arvados/jobs"

        resources = self.builder.resources
        if resources is not None:
            runtime_constraints["min_cores_per_node"] = resources.get("cores", 1)
            runtime_constraints["min_ram_mb_per_node"] = resources.get("ram")
            runtime_constraints["min_scratch_mb_per_node"] = resources.get("tmpdirSize", 0) + resources.get("outdirSize", 0)

        runtime_req, _ = get_feature(self, "http://arvados.org/cwl#RuntimeConstraints")
        if runtime_req:
            if "keep_cache" in runtime_req:
                runtime_constraints["keep_cache_mb_per_task"] = runtime_req["keep_cache"]
                runtime_constraints["min_ram_mb_per_node"] += runtime_req["keep_cache"]
            if "outputDirType" in runtime_req:
                if runtime_req["outputDirType"] == "local_output_dir":
                    script_parameters["task.keepTmpOutput"] = False
                elif runtime_req["outputDirType"] == "keep_output_dir":
                    script_parameters["task.keepTmpOutput"] = True

        filters = [["repository", "=", "arvados"],
                   ["script", "=", "crunchrunner"],
                   ["script_version", "in git", crunchrunner_git_commit]]
        if not self.arvrunner.ignore_docker_for_reuse:
            filters.append(["docker_image_locator", "in docker", runtime_constraints["docker_image"]])

        try:
            with Perf(metrics, "create %s" % self.name):
                response = self.arvrunner.api.jobs().create(
                    body={
                        "owner_uuid": self.arvrunner.project_uuid,
                        "script": "crunchrunner",
                        "repository": "arvados",
                        "script_version": "master",
                        "minimum_script_version": crunchrunner_git_commit,
                        "script_parameters": {"tasks": [script_parameters]},
                        "runtime_constraints": runtime_constraints
                    },
                    filters=filters,
                    find_or_create=kwargs.get("enable_reuse", True)
                ).execute(num_retries=self.arvrunner.num_retries)

            self.arvrunner.processes[response["uuid"]] = self

            self.update_pipeline_component(response)

            if response["state"] == "Complete":
                logger.info("%s reused job %s", self.arvrunner.label(self), response["uuid"])
                # Give read permission to the desired project on reused jobs
                if response["owner_uuid"] != self.arvrunner.project_uuid:
                    self.arvrunner.api.links().create(body={
                        'link_class': 'permission',
                        'name': 'can_read',
                        'tail_uuid': self.arvrunner.project_uuid,
                        'head_uuid': response["uuid"],
                        }).execute(num_retries=self.arvrunner.num_retries)

                with Perf(metrics, "done %s" % self.name):
                    self.done(response)
            else:
                logger.info("%s %s is %s", self.arvrunner.label(self), response["uuid"], response["state"])
        except Exception as e:
            logger.exception("%s error" % (self.arvrunner.label(self)))
            self.output_callback({}, "permanentFail")

    def update_pipeline_component(self, record):
        if self.arvrunner.pipeline:
            self.arvrunner.pipeline["components"][self.name] = {"job": record}
            with Perf(metrics, "update_pipeline_component %s" % self.name):
                self.arvrunner.pipeline = self.arvrunner.api.pipeline_instances().update(uuid=self.arvrunner.pipeline["uuid"],
                                                                                 body={
                                                                                    "components": self.arvrunner.pipeline["components"]
                                                                                 }).execute(num_retries=self.arvrunner.num_retries)
        if self.arvrunner.uuid:
            try:
                job = self.arvrunner.api.jobs().get(uuid=self.arvrunner.uuid).execute()
                if job:
                    components = job["components"]
                    components[self.name] = record["uuid"]
                    self.arvrunner.api.jobs().update(uuid=self.arvrunner.uuid,
                        body={
                            "components": components
                        }).execute(num_retries=self.arvrunner.num_retries)
            except Exception as e:
                logger.info("Error adding to components: %s", e)

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

            outputs = {}
            try:
                if record["output"]:
                    with Perf(metrics, "inspect log %s" % self.name):
                        logc = arvados.collection.CollectionReader(record["log"],
                                                                   api_client=self.arvrunner.api,
                                                                   keep_client=self.arvrunner.keep_client,
                                                                   num_retries=self.arvrunner.num_retries)
                        log = logc.open(logc.keys()[0])
                        dirs = {}
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
                            g = crunchrunner_re.match(l)
                            if g:
                                dirs[g.group(1)] = g.group(2)

                    if processStatus == "permanentFail":
                        done.logtail(logc, logger, "%s error log:" % self.arvrunner.label(self))

                    with Perf(metrics, "output collection %s" % self.name):
                        outputs = done.done(self, record, dirs["tmpdir"],
                                            dirs["outdir"], dirs["keep"])
            except WorkflowException as e:
                logger.error("%s unable to collect output from %s:\n%s",
                             self.arvrunner.label(self), record["output"], e, exc_info=(e if self.arvrunner.debug else False))
                processStatus = "permanentFail"
            except Exception as e:
                logger.exception("Got unknown exception while collecting output for job %s:", self.name)
                processStatus = "permanentFail"

            # Note: Currently, on error output_callback is expecting an empty dict,
            # anything else will fail.
            if not isinstance(outputs, dict):
                logger.error("Unexpected output type %s '%s'", type(outputs), outputs)
                outputs = {}
                processStatus = "permanentFail"
        finally:
            self.output_callback(outputs, processStatus)
            if record["uuid"] in self.arvrunner.processes:
                del self.arvrunner.processes[record["uuid"]]

class RunnerJob(Runner):
    """Submit and manage a Crunch job that runs crunch_scripts/cwl-runner."""

    def arvados_job_spec(self, dry_run=False, pull_image=True, **kwargs):
        """Create an Arvados job specification for this workflow.

        The returned dict can be used to create a job (i.e., passed as
        the +body+ argument to jobs().create()), or as a component in
        a pipeline template or pipeline instance.
        """

        if self.tool.tool["id"].startswith("keep:"):
            self.job_order["cwl:tool"] = self.tool.tool["id"][5:]
        else:
            packed = packed_workflow(self.arvrunner, self.tool)
            wf_pdh = upload_workflow_collection(self.arvrunner, self.name, packed)
            self.job_order["cwl:tool"] = "%s/workflow.cwl#main" % wf_pdh

        adjustDirObjs(self.job_order, trim_listing)
        adjustFileObjs(self.job_order, trim_anonymous_location)
        adjustDirObjs(self.job_order, trim_anonymous_location)

        if self.output_name:
            self.job_order["arv:output_name"] = self.output_name

        if self.output_tags:
            self.job_order["arv:output_tags"] = self.output_tags

        self.job_order["arv:enable_reuse"] = self.enable_reuse

        if self.on_error:
            self.job_order["arv:on_error"] = self.on_error

        return {
            "script": "cwl-runner",
            "script_version": "master",
            "minimum_script_version": "570509ab4d2ef93d870fd2b1f2eab178afb1bad9",
            "repository": "arvados",
            "script_parameters": self.job_order,
            "runtime_constraints": {
                "docker_image": arvados_jobs_image(self.arvrunner, self.jobs_image),
                "min_ram_mb_per_node": self.submit_runner_ram
            }
        }

    def run(self, *args, **kwargs):
        job_spec = self.arvados_job_spec(*args, **kwargs)

        job_spec.setdefault("owner_uuid", self.arvrunner.project_uuid)

        job = self.arvrunner.api.jobs().create(
            body=job_spec,
            find_or_create=self.enable_reuse
        ).execute(num_retries=self.arvrunner.num_retries)

        for k,v in job_spec["script_parameters"].items():
            if v is False or v is None or isinstance(v, dict):
                job_spec["script_parameters"][k] = {"value": v}

        del job_spec["owner_uuid"]
        job_spec["job"] = job
        self.arvrunner.pipeline = self.arvrunner.api.pipeline_instances().create(
            body={
                "owner_uuid": self.arvrunner.project_uuid,
                "name": self.name,
                "components": {"cwl-runner": job_spec },
                "state": "RunningOnServer"}).execute(num_retries=self.arvrunner.num_retries)
        logger.info("Created pipeline %s", self.arvrunner.pipeline["uuid"])

        if kwargs.get("wait") is False:
            self.uuid = self.arvrunner.pipeline["uuid"]
            return

        self.uuid = job["uuid"]
        self.arvrunner.processes[self.uuid] = self

        if job["state"] in ("Complete", "Failed", "Cancelled"):
            self.done(job)


class RunnerTemplate(object):
    """An Arvados pipeline template that invokes a CWL workflow."""

    type_to_dataclass = {
        'boolean': 'boolean',
        'File': 'File',
        'Directory': 'Collection',
        'float': 'number',
        'int': 'number',
        'string': 'text',
    }

    def __init__(self, runner, tool, job_order, enable_reuse, uuid,
                 submit_runner_ram=0, name=None):
        self.runner = runner
        self.tool = tool
        self.job = RunnerJob(
            runner=runner,
            tool=tool,
            job_order=job_order,
            enable_reuse=enable_reuse,
            output_name=None,
            output_tags=None,
            submit_runner_ram=submit_runner_ram,
            name=name)
        self.uuid = uuid

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
            non_null_types = [t for t in types if t != "null"]
            if len(non_null_types) == 1:
                the_type = [c for c in non_null_types][0]
                dataclass = None
                if isinstance(the_type, basestring):
                    dataclass = self.type_to_dataclass.get(the_type)
                if dataclass:
                    param['dataclass'] = dataclass
            # Note: If we didn't figure out a single appropriate
            # dataclass, we just left that attribute out.  We leave
            # the "type" attribute there in any case, which might help
            # downstream.

            # Title and description...
            title = param.pop('label', '')
            descr = param.pop('doc', '').rstrip('\n')
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
            elif param.get('dataclass') in ('File', 'Collection') and value.get('location'):
                param['value'] = value['location'][5:]

            spec['script_parameters'][param_id] = param
        spec['script_parameters']['cwl:tool'] = job_params['cwl:tool']
        return spec

    def save(self):
        body = {
            "components": {
                self.job.name: self.pipeline_component_spec(),
            },
            "name": self.job.name,
        }
        if self.runner.project_uuid:
            body["owner_uuid"] = self.runner.project_uuid
        if self.uuid:
            self.runner.api.pipeline_templates().update(
                uuid=self.uuid, body=body).execute(
                    num_retries=self.runner.num_retries)
            logger.info("Updated template %s", self.uuid)
        else:
            self.uuid = self.runner.api.pipeline_templates().create(
                body=body, ensure_unique_name=True).execute(
                    num_retries=self.runner.num_retries)['uuid']
            logger.info("Created template %s", self.uuid)
