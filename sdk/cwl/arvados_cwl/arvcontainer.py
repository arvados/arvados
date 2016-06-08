import logging
import arvados.collection
from cwltool.process import get_feature
from .arvdocker import arv_docker_get_image

logger = logging.getLogger('arvados.cwl-runner')

class ArvadosContainer(object):
    """Submit and manage a Crunch job for executing a CWL CommandLineTool."""

    def __init__(self, runner):
        self.arvrunner = runner
        self.running = False

    def run(self, dry_run=False, pull_image=True, **kwargs):
        container_request = {
            "command": self.command_line,
            "owner_uuid": self.arvrunner.project_uuid,
            "name": self.name,
            "output_path": "/var/spool/cwl",
            "cwd": "/var/spool/cwl",
            "priority": 1,
            "state": "Committed"
        }
        runtime_constraints = {}
        mounts = {
            "/var/spool/cwl": {
                "kind": "tmp"
            },
            "/tmp": {
                "kind": "tmp"
            }
        }

        # TODO mount normal inputs...

        if self.generatefiles:
            vwd = arvados.collection.Collection()
            container_request["task.vwd"] = {}
            for t in self.generatefiles:
                if isinstance(self.generatefiles[t], dict):
                    src, rest = self.arvrunner.fs_access.get_collection(self.generatefiles[t]["path"].replace("$(task.keep)/", "keep:"))
                    vwd.copy(rest, t, source_collection=src)
                else:
                    with vwd.open(t, "w") as f:
                        f.write(self.generatefiles[t])
            vwd.save_new()
            # TODO
            # for t in self.generatefiles:
            #     container_request["task.vwd"][t] = "$(task.keep)/%s/%s" % (vwd.portable_data_hash(), t)

        container_request["environment"] = {"TMPDIR": "/tmp"}
        if self.environment:
            container_request["environment"].update(self.environment)

        # TODO, not supported
        #if self.stdin:
        #    container_request["task.stdin"] = self.pathmapper.mapper(self.stdin)[1]

        if self.stdout:
            mounts["stdout"] = {"kind": "file",
                                "path": self.stdout}

        (docker_req, docker_is_req) = get_feature(self, "DockerRequirement")
        if not docker_req:
            docker_req = {"dockerImageId": "arvados/jobs"}

        container_request["container_image"] = arv_docker_get_image(self.arvrunner.api,
                                                                     docker_req,
                                                                     pull_image,
                                                                     self.arvrunner.project_uuid)

        resources = self.builder.resources
        if resources is not None:
            runtime_constraints["vcpus"] = resources.get("cores", 1)
            runtime_constraints["ram"] = resources.get("ram") * 2**20
            #runtime_constraints["min_scratch_mb_per_node"] = resources.get("tmpdirSize", 0) + resources.get("outdirSize", 0)

        container_request["mounts"] = mounts
        container_request["runtime_constraints"] = runtime_constraints

        try:
            response = self.arvrunner.api.container_requests().create(
                body=container_request
            ).execute(num_retries=self.arvrunner.num_retries)

            self.arvrunner.jobs[response["uuid"]] = self

            logger.info("Container %s (%s) is %s", self.name, response["uuid"], response["state"])

            if response["state"] in ("Complete", "Cancelled"):
                self.done(response)
        except Exception as e:
            logger.error("Got error %s" % str(e))
            self.output_callback({}, "permanentFail")

    def done(self, record):
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
