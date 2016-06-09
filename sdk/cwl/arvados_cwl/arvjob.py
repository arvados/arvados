import logging
import re
from . import done
from .arvdocker import arv_docker_get_image
from cwltool.process import get_feature
from cwltool.errors import WorkflowException
import arvados.collection

logger = logging.getLogger('arvados.cwl-runner')

tmpdirre = re.compile(r"^\S+ \S+ \d+ \d+ stderr \S+ \S+ crunchrunner: \$\(task\.tmpdir\)=(.*)")
outdirre = re.compile(r"^\S+ \S+ \d+ \d+ stderr \S+ \S+ crunchrunner: \$\(task\.outdir\)=(.*)")
keepre = re.compile(r"^\S+ \S+ \d+ \d+ stderr \S+ \S+ crunchrunner: \$\(task\.keep\)=(.*)")

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
            del self.arvrunner.jobs[record["uuid"]]
