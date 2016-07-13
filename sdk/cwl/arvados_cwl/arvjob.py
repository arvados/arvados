import logging
import re
import copy

from cwltool.process import get_feature, shortname
from cwltool.errors import WorkflowException
from cwltool.draft2tool import revmap_file, CommandLineTool
from cwltool.load_tool import fetch_document
from cwltool.builder import Builder
from cwltool.pathmapper import PathMapper

import arvados.collection

from .arvdocker import arv_docker_get_image
from .runner import Runner
from . import done

logger = logging.getLogger('arvados.cwl-runner')

tmpdirre = re.compile(r"^\S+ \S+ \d+ \d+ stderr \S+ \S+ crunchrunner: \$\(task\.tmpdir\)=(.*)")
outdirre = re.compile(r"^\S+ \S+ \d+ \d+ stderr \S+ \S+ crunchrunner: \$\(task\.outdir\)=(.*)")
keepre = re.compile(r"^\S+ \S+ \d+ \d+ stderr \S+ \S+ crunchrunner: \$\(task\.keep\)=(.*)")

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

        if self.generatefiles["listing"]:
            vwd = arvados.collection.Collection()
            script_parameters["task.vwd"] = {}
            generatemapper = PathMapper([self.generatefiles], self.outdir,
                                        ".", separateDirs=False)
            for f, p in generatemapper.items():
                if p.type == "CreateFile":
                    with vwd.open(p.target, "w") as n:
                        n.write(p.resolved.encode("utf-8"))
            vwd.save_new()
            for f, p in generatemapper.items():
                if p.type == "File":
                    script_parameters["task.vwd"][p.target] = self.pathmapper.mapper(f).target
                if p.type == "CreateFile":
                    script_parameters["task.vwd"][p.target] = "$(task.keep)/%s/%s" % (vwd.portable_data_hash(), p.target)

        script_parameters["task.env"] = {"TMPDIR": "$(task.tmpdir)"}
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

            self.arvrunner.processes[response["uuid"]] = self

            self.update_pipeline_component(response)

            logger.info("Job %s (%s) is %s", self.name, response["uuid"], response["state"])

            if response["state"] in ("Complete", "Failed", "Cancelled"):
                self.done(response)
        except Exception as e:
            logger.error("Got error %s" % str(e))
            self.output_callback({}, "permanentFail")

    def update_pipeline_component(self, record):
        if self.arvrunner.pipeline:
            self.arvrunner.pipeline["components"][self.name] = {"job": record}
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

                    outputs = done.done(self, record, tmpdir, outdir, keepdir)
            except WorkflowException as e:
                logger.error("Error while collecting job outputs:\n%s", e, exc_info=(e if self.arvrunner.debug else False))
                processStatus = "permanentFail"
            except Exception as e:
                logger.exception("Got unknown exception while collecting job outputs:")
                processStatus = "permanentFail"

            self.output_callback(outputs, processStatus)
        finally:
            del self.arvrunner.processes[record["uuid"]]


class RunnerJob(Runner):
    """Submit and manage a Crunch job that runs crunch_scripts/cwl-runner."""

    def arvados_job_spec(self, dry_run=False, pull_image=True, **kwargs):
        """Create an Arvados job specification for this workflow.

        The returned dict can be used to create a job (i.e., passed as
        the +body+ argument to jobs().create()), or as a component in
        a pipeline template or pipeline instance.
        """

        workflowmapper = super(RunnerJob, self).arvados_job_spec(dry_run=dry_run, pull_image=pull_image, **kwargs)

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
        self.arvrunner.processes[self.uuid] = self

        logger.info("Submitted job %s", response["uuid"])

        if kwargs.get("submit"):
            self.pipeline = self.arvrunner.api.pipeline_instances().create(
                body={
                    "owner_uuid": self.arvrunner.project_uuid,
                    "name": shortname(self.tool.tool["id"]),
                    "components": {"cwl-runner": {"job": {"uuid": self.uuid, "state": response["state"]} } },
                    "state": "RunningOnClient"}).execute(num_retries=self.arvrunner.num_retries)

        if response["state"] in ("Complete", "Failed", "Cancelled"):
            self.done(response)


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
            elif param.get('dataclass') == 'File' and value.get('location'):
                param['value'] = value['location']

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
