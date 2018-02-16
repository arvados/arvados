# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import logging
import json
import os
import urllib
import time
import datetime
import ciso8601

import ruamel.yaml as yaml

from cwltool.errors import WorkflowException
from cwltool.process import get_feature, UnsupportedRequirement, shortname
from cwltool.pathmapper import adjustFileObjs, adjustDirObjs, visit_class
from cwltool.utils import aslist

import arvados.collection

from .arvdocker import arv_docker_get_image
from . import done
from .runner import Runner, arvados_jobs_image, packed_workflow, trim_anonymous_location, remove_redundant_fields
from .fsaccess import CollectionFetcher
from .pathmapper import NoFollowPathMapper, trim_listing
from .perf import Perf

logger = logging.getLogger('arvados.cwl-runner')
metrics = logging.getLogger('arvados.cwl-runner.metrics')

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
            "state": "Committed",
            "properties": {},
        }
        runtime_constraints = {}

        resources = self.builder.resources
        if resources is not None:
            runtime_constraints["vcpus"] = resources.get("cores", 1)
            runtime_constraints["ram"] = resources.get("ram") * 2**20

        mounts = {
            self.outdir: {
                "kind": "tmp",
                "capacity": resources.get("outdirSize", 0) * 2**20
            },
            self.tmpdir: {
                "kind": "tmp",
                "capacity": resources.get("tmpdirSize", 0) * 2**20
            }
        }
        scheduling_parameters = {}

        rf = [self.pathmapper.mapper(f) for f in self.pathmapper.referenced_files]
        rf.sort(key=lambda k: k.resolved)
        prevdir = None
        for resolved, target, tp, stg in rf:
            if not stg:
                continue
            if prevdir and target.startswith(prevdir):
                continue
            if tp == "Directory":
                targetdir = target
            else:
                targetdir = os.path.dirname(target)
            sp = resolved.split("/", 1)
            pdh = sp[0][5:]   # remove "keep:"
            mounts[targetdir] = {
                "kind": "collection",
                "portable_data_hash": pdh
            }
            if len(sp) == 2:
                if tp == "Directory":
                    path = sp[1]
                else:
                    path = os.path.dirname(sp[1])
                if path and path != "/":
                    mounts[targetdir]["path"] = path
            prevdir = targetdir + "/"

        with Perf(metrics, "generatefiles %s" % self.name):
            if self.generatefiles["listing"]:
                vwd = arvados.collection.Collection(api_client=self.arvrunner.api,
                                                    keep_client=self.arvrunner.keep_client,
                                                    num_retries=self.arvrunner.num_retries)
                generatemapper = NoFollowPathMapper([self.generatefiles], "", "",
                                                    separateDirs=False)

                logger.debug("generatemapper is %s", generatemapper._pathmap)

                with Perf(metrics, "createfiles %s" % self.name):
                    for f, p in generatemapper.items():
                        if not p.target:
                            pass
                        elif p.type in ("File", "Directory", "WritableFile", "WritableDirectory"):
                            if p.resolved.startswith("_:"):
                                vwd.mkdirs(p.target)
                            else:
                                source, path = self.arvrunner.fs_access.get_collection(p.resolved)
                                vwd.copy(path, p.target, source_collection=source)
                        elif p.type == "CreateFile":
                            with vwd.open(p.target, "w") as n:
                                n.write(p.resolved.encode("utf-8"))

                def keepemptydirs(p):
                    if isinstance(p, arvados.collection.RichCollectionBase):
                        if len(p) == 0:
                            p.open(".keep", "w").close()
                        else:
                            for c in p:
                                keepemptydirs(p[c])

                keepemptydirs(vwd)

                with Perf(metrics, "generatefiles.save_new %s" % self.name):
                    vwd.save_new()

                for f, p in generatemapper.items():
                    if not p.target:
                        continue
                    mountpoint = "%s/%s" % (self.outdir, p.target)
                    mounts[mountpoint] = {"kind": "collection",
                                          "portable_data_hash": vwd.portable_data_hash(),
                                          "path": p.target}
                    if p.type.startswith("Writable"):
                        mounts[mountpoint]["writable"] = True

        container_request["environment"] = {"TMPDIR": self.tmpdir, "HOME": self.outdir}
        if self.environment:
            container_request["environment"].update(self.environment)

        if self.stdin:
            sp = self.stdin[6:].split("/", 1)
            mounts["stdin"] = {"kind": "collection",
                                "portable_data_hash": sp[0],
                                "path": sp[1]}

        if self.stderr:
            mounts["stderr"] = {"kind": "file",
                                "path": "%s/%s" % (self.outdir, self.stderr)}

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

        api_req, _ = get_feature(self, "http://arvados.org/cwl#APIRequirement")
        if api_req:
            runtime_constraints["API"] = True

        runtime_req, _ = get_feature(self, "http://arvados.org/cwl#RuntimeConstraints")
        if runtime_req:
            if "keep_cache" in runtime_req:
                runtime_constraints["keep_cache_ram"] = runtime_req["keep_cache"] * 2**20
            if "outputDirType" in runtime_req:
                if runtime_req["outputDirType"] == "local_output_dir":
                    # Currently the default behavior.
                    pass
                elif runtime_req["outputDirType"] == "keep_output_dir":
                    mounts[self.outdir]= {
                        "kind": "collection",
                        "writable": True
                    }

        partition_req, _ = get_feature(self, "http://arvados.org/cwl#PartitionRequirement")
        if partition_req:
            scheduling_parameters["partitions"] = aslist(partition_req["partition"])

        intermediate_output_req, _ = get_feature(self, "http://arvados.org/cwl#IntermediateOutput")
        if intermediate_output_req:
            self.output_ttl = intermediate_output_req["outputTTL"]
        else:
            self.output_ttl = self.arvrunner.intermediate_output_ttl

        if self.output_ttl < 0:
            raise WorkflowError("Invalid value %d for output_ttl, cannot be less than zero" % container_request["output_ttl"])

        container_request["output_ttl"] = self.output_ttl
        container_request["mounts"] = mounts
        container_request["runtime_constraints"] = runtime_constraints
        container_request["scheduling_parameters"] = scheduling_parameters

        enable_reuse = kwargs.get("enable_reuse", True)
        if enable_reuse:
            reuse_req, _ = get_feature(self, "http://arvados.org/cwl#ReuseRequirement")
            if reuse_req:
                enable_reuse = reuse_req["enableReuse"]
        container_request["use_existing"] = enable_reuse

        if kwargs.get("runnerjob", "").startswith("arvwf:"):
            wfuuid = kwargs["runnerjob"][6:kwargs["runnerjob"].index("#")]
            wfrecord = self.arvrunner.api.workflows().get(uuid=wfuuid).execute(num_retries=self.arvrunner.num_retries)
            if container_request["name"] == "main":
                container_request["name"] = wfrecord["name"]
            container_request["properties"]["template_uuid"] = wfuuid

        try:
            response = self.arvrunner.api.container_requests().create(
                body=container_request
            ).execute(num_retries=self.arvrunner.num_retries)

            self.uuid = response["uuid"]
            self.arvrunner.processes[self.uuid] = self

            if response["state"] == "Final":
                logger.info("%s reused container %s", self.arvrunner.label(self), response["container_uuid"])
                self.done(response)
            else:
                logger.info("%s %s state is %s", self.arvrunner.label(self), response["uuid"], response["state"])
        except Exception as e:
            logger.error("%s got error %s" % (self.arvrunner.label(self), str(e)))
            self.output_callback({}, "permanentFail")

    def done(self, record):
        outputs = {}
        try:
            container = self.arvrunner.api.containers().get(
                uuid=record["container_uuid"]
            ).execute(num_retries=self.arvrunner.num_retries)
            if container["state"] == "Complete":
                rcode = container["exit_code"]
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

            if processStatus == "permanentFail":
                logc = arvados.collection.CollectionReader(container["log"],
                                                           api_client=self.arvrunner.api,
                                                           keep_client=self.arvrunner.keep_client,
                                                           num_retries=self.arvrunner.num_retries)
                done.logtail(logc, logger, "%s error log:" % self.arvrunner.label(self))

            if record["output_uuid"]:
                if self.arvrunner.trash_intermediate or self.arvrunner.intermediate_output_ttl:
                    # Compute the trash time to avoid requesting the collection record.
                    trash_at = ciso8601.parse_datetime_unaware(record["modified_at"]) + datetime.timedelta(0, self.arvrunner.intermediate_output_ttl)
                    aftertime = " at %s" % trash_at.strftime("%Y-%m-%d %H:%M:%S UTC") if self.arvrunner.intermediate_output_ttl else ""
                    orpart = ", or" if self.arvrunner.trash_intermediate and self.arvrunner.intermediate_output_ttl else ""
                    oncomplete = " upon successful completion of the workflow" if self.arvrunner.trash_intermediate else ""
                    logger.info("%s Intermediate output %s (%s) will be trashed%s%s%s." % (
                        self.arvrunner.label(self), record["output_uuid"], container["output"], aftertime, orpart, oncomplete))
                self.arvrunner.add_intermediate_output(record["output_uuid"])

            if container["output"]:
                outputs = done.done_outputs(self, container, "/tmp", self.outdir, "/keep")
        except WorkflowException as e:
            logger.error("%s unable to collect output from %s:\n%s",
                         self.arvrunner.label(self), container["output"], e, exc_info=(e if self.arvrunner.debug else False))
            processStatus = "permanentFail"
        except Exception as e:
            logger.exception("%s while getting output object: %s", self.arvrunner.label(self), e)
            processStatus = "permanentFail"
        finally:
            self.output_callback(outputs, processStatus)
            if record["uuid"] in self.arvrunner.processes:
                del self.arvrunner.processes[record["uuid"]]


class RunnerContainer(Runner):
    """Submit and manage a container that runs arvados-cwl-runner."""

    def arvados_job_spec(self, dry_run=False, pull_image=True, **kwargs):
        """Create an Arvados container request for this workflow.

        The returned dict can be used to create a container passed as
        the +body+ argument to container_requests().create().
        """

        adjustDirObjs(self.job_order, trim_listing)
        visit_class(self.job_order, ("File", "Directory"), trim_anonymous_location)
        visit_class(self.job_order, ("File", "Directory"), remove_redundant_fields)

        container_req = {
            "owner_uuid": self.arvrunner.project_uuid,
            "name": self.name,
            "output_path": "/var/spool/cwl",
            "cwd": "/var/spool/cwl",
            "priority": 1,
            "state": "Committed",
            "container_image": arvados_jobs_image(self.arvrunner, self.jobs_image),
            "mounts": {
                "/var/lib/cwl/cwl.input.json": {
                    "kind": "json",
                    "content": self.job_order
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
                "ram": 1024*1024 * self.submit_runner_ram,
                "API": True
            },
            "use_existing": self.enable_reuse,
            "properties": {}
        }

        if self.tool.tool.get("id", "").startswith("keep:"):
            sp = self.tool.tool["id"].split('/')
            workflowcollection = sp[0][5:]
            workflowname = "/".join(sp[1:])
            workflowpath = "/var/lib/cwl/workflow/%s" % workflowname
            container_req["mounts"]["/var/lib/cwl/workflow"] = {
                "kind": "collection",
                "portable_data_hash": "%s" % workflowcollection
            }
        else:
            packed = packed_workflow(self.arvrunner, self.tool, self.merged_map)
            workflowpath = "/var/lib/cwl/workflow.json#main"
            container_req["mounts"]["/var/lib/cwl/workflow.json"] = {
                "kind": "json",
                "content": packed
            }
            if self.tool.tool.get("id", "").startswith("arvwf:"):
                container_req["properties"]["template_uuid"] = self.tool.tool["id"][6:33]


        command = ["arvados-cwl-runner", "--local", "--api=containers", "--no-log-timestamps"]
        if self.output_name:
            command.append("--output-name=" + self.output_name)
            container_req["output_name"] = self.output_name

        if self.output_tags:
            command.append("--output-tags=" + self.output_tags)

        if kwargs.get("debug"):
            command.append("--debug")

        if self.enable_reuse:
            command.append("--enable-reuse")
        else:
            command.append("--disable-reuse")

        if self.on_error:
            command.append("--on-error=" + self.on_error)

        if self.intermediate_output_ttl:
            command.append("--intermediate-output-ttl=%d" % self.intermediate_output_ttl)

        if self.arvrunner.trash_intermediate:
            command.append("--trash-intermediate")

        if self.arvrunner.project_uuid:
            command.append("--project-uuid="+self.arvrunner.project_uuid)

        command.append("--eval-timeout=%s" % self.arvrunner.eval_timeout)

        command.extend([workflowpath, "/var/lib/cwl/cwl.input.json"])

        container_req["command"] = command

        return container_req


    def run(self, *args, **kwargs):
        kwargs["keepprefix"] = "keep:"
        job_spec = self.arvados_job_spec(*args, **kwargs)
        job_spec.setdefault("owner_uuid", self.arvrunner.project_uuid)

        response = self.arvrunner.api.container_requests().create(
            body=job_spec
        ).execute(num_retries=self.arvrunner.num_retries)

        self.uuid = response["uuid"]
        self.arvrunner.processes[self.uuid] = self

        logger.info("%s submitted container %s", self.arvrunner.label(self), response["uuid"])

        if response["state"] == "Final":
            self.done(response)

    def done(self, record):
        try:
            container = self.arvrunner.api.containers().get(
                uuid=record["container_uuid"]
            ).execute(num_retries=self.arvrunner.num_retries)
        except Exception as e:
            logger.exception("%s while getting runner container: %s", self.arvrunner.label(self), e)
            self.arvrunner.output_callback({}, "permanentFail")
        else:
            super(RunnerContainer, self).done(container)
        finally:
            if record["uuid"] in self.arvrunner.processes:
                del self.arvrunner.processes[record["uuid"]]
