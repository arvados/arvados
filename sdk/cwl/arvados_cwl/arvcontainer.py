import logging
import json
import os

from cwltool.errors import WorkflowException
from cwltool.process import get_feature, UnsupportedRequirement, shortname
from cwltool.pathmapper import adjustFiles
from cwltool.utils import aslist

import arvados.collection

from .arvdocker import arv_docker_get_image
from . import done
from .runner import Runner

logger = logging.getLogger('arvados.cwl-runner')

class ArvadosContainer(object):
    """Submit and manage a Crunch container request for executing a CWL CommandLineTool."""

    def __init__(self, runner):
        self.arvrunner = runner
        self.running = False
        self.uuid = None

    def update_pipeline_component(self, r):
        pass

    def run(self, dry_run=False, pull_image=True, **kwargs):
        container_request = {
            "command": self.command_line,
            "owner_uuid": self.arvrunner.project_uuid,
            "name": self.name,
            "output_path": self.outdir,
            "cwd": self.outdir,
            "priority": 1,
            "state": "Committed"
        }
        runtime_constraints = {}
        mounts = {
            self.outdir: {
                "kind": "tmp"
            }
        }

        dirs = set()
        for f in self.pathmapper.files():
            _, p, tp = self.pathmapper.mapper(f)
            if tp == "Directory" and '/' not in p[6:]:
                mounts[p] = {
                    "kind": "collection",
                    "portable_data_hash": p[6:]
                }
                dirs.add(p[6:])
        for f in self.pathmapper.files():
            _, p, tp = self.pathmapper.mapper(f)
            if p[6:].split("/")[0] not in dirs:
                mounts[p] = {
                    "kind": "collection",
                    "portable_data_hash": p[6:]
                }

        if self.generatefiles["listing"]:
            raise UnsupportedRequirement("Generate files not supported")

        container_request["environment"] = {"TMPDIR": self.tmpdir, "HOME": self.outdir}
        if self.environment:
            container_request["environment"].update(self.environment)

        if self.stdin:
            raise UnsupportedRequirement("Stdin redirection currently not suppported")

        if self.stderr:
            raise UnsupportedRequirement("Stderr redirection currently not suppported")

        if self.stdout:
            mounts["stdout"] = {"kind": "file",
                                "path": "%s/%s" % (self.outdir, self.stdout)}

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

        api_req, _ = get_feature(self, "http://arvados.org/cwl#APIRequirement")
        if api_req:
            runtime_constraints["API"] = True

        runtime_req, _ = get_feature(self, "http://arvados.org/cwl#RuntimeConstraints")
        if runtime_req:
            logger.warn("RuntimeConstraints not yet supported by container API")

        partition_req, _ = get_feature(self, "http://arvados.org/cwl#PartitionRequirement")
        if partition_req:
            runtime_constraints["partition"] = aslist(partition_req["partition"])

        container_request["mounts"] = mounts
        container_request["runtime_constraints"] = runtime_constraints

        try:
            response = self.arvrunner.api.container_requests().create(
                body=container_request
            ).execute(num_retries=self.arvrunner.num_retries)

            self.arvrunner.processes[response["container_uuid"]] = self

            logger.info("Container %s (%s) request state is %s", self.name, response["uuid"], response["state"])

            if response["state"] == "Final":
                self.done(response)
        except Exception as e:
            logger.error("Got error %s" % str(e))
            self.output_callback({}, "permanentFail")

    def done(self, record):
        try:
            if record["state"] == "Complete":
                rcode = record["exit_code"]
                if self.successCodes and rcode in self.successCodes:
                    processStatus = "success"
                elif self.temporaryFailCodes and rcode in self.temporaryFailCodes:
                    processStatus = "temporaryFail"
                elif self.permanentFailCodes and rcode in self.permanentFailCodes:
                    processStatus = "permanentFail"
                elif rcode == 0:
                    processStatus = "success"
                else:
                    processStatus = "permanentFail"
            else:
                processStatus = "permanentFail"

            try:
                outputs = {}
                if record["output"]:
                    outputs = done.done(self, record, "/tmp", self.outdir, "/keep")
            except WorkflowException as e:
                logger.error("Error while collecting container outputs:\n%s", e, exc_info=(e if self.arvrunner.debug else False))
                processStatus = "permanentFail"
            except Exception as e:
                logger.exception("Got unknown exception while collecting job outputs:")
                processStatus = "permanentFail"

            self.output_callback(outputs, processStatus)
        finally:
            del self.arvrunner.processes[record["uuid"]]


class RunnerContainer(Runner):
    """Submit and manage a container that runs arvados-cwl-runner."""

    def arvados_job_spec(self, dry_run=False, pull_image=True, **kwargs):
        """Create an Arvados container request for this workflow.

        The returned dict can be used to create a container passed as
        the +body+ argument to container_requests().create().
        """

        workflowmapper = super(RunnerContainer, self).arvados_job_spec(dry_run=dry_run, pull_image=pull_image, **kwargs)

        with arvados.collection.Collection(api_client=self.arvrunner.api,
                                           keep_client=self.arvrunner.keep_client,
                                           num_retries=self.arvrunner.num_retries) as jobobj:
            with jobobj.open("cwl.input.json", "w") as f:
                json.dump(self.job_order, f, sort_keys=True, indent=4)
            jobobj.save_new(owner_uuid=self.arvrunner.project_uuid)

        workflowname = os.path.basename(self.tool.tool["id"])
        workflowpath = "/var/lib/cwl/workflow/%s" % workflowname
        workflowcollection = workflowmapper.mapper(self.tool.tool["id"])[1]
        workflowcollection = workflowcollection[5:workflowcollection.index('/')]
        jobpath = "/var/lib/cwl/job/cwl.input.json"

        container_image = arv_docker_get_image(self.arvrunner.api,
                                               {"dockerImageId": "arvados/jobs"},
                                               pull_image,
                                               self.arvrunner.project_uuid)

        command = ["arvados-cwl-runner", "--local", "--api=containers"]
        if self.output_name:
            command.append("--output-name=" + self.output_name)
        command.extend([workflowpath, jobpath])

        return {
            "command": command,
            "owner_uuid": self.arvrunner.project_uuid,
            "name": self.name,
            "output_path": "/var/spool/cwl",
            "cwd": "/var/spool/cwl",
            "priority": 1,
            "state": "Committed",
            "container_image": container_image,
            "mounts": {
                "/var/lib/cwl/workflow": {
                    "kind": "collection",
                    "portable_data_hash": "%s" % workflowcollection
                },
                jobpath: {
                    "kind": "collection",
                    "portable_data_hash": "%s/cwl.input.json" % jobobj.portable_data_hash()
                },
                "stdout": {
                    "kind": "file",
                    "path": "/var/spool/cwl/cwl.output.json"
                },
                "/var/spool/cwl": {
                    "kind": "collection",
                    "writable": True
                }
            },
            "runtime_constraints": {
                "vcpus": 1,
                "ram": 1024*1024*256,
                "API": True
            }
        }

    def run(self, *args, **kwargs):
        kwargs["keepprefix"] = "keep:"
        job_spec = self.arvados_job_spec(*args, **kwargs)
        job_spec.setdefault("owner_uuid", self.arvrunner.project_uuid)

        response = self.arvrunner.api.container_requests().create(
            body=job_spec
        ).execute(num_retries=self.arvrunner.num_retries)

        self.uuid = response["uuid"]
        self.arvrunner.processes[response["container_uuid"]] = self

        logger.info("Submitted container %s", response["uuid"])

        if response["state"] in ("Complete", "Failed", "Cancelled"):
            self.done(response)
