# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from cwltool.command_line_tool import CommandLineTool, ExpressionTool
from .arvcontainer import ArvadosContainer
from .pathmapper import ArvPathMapper
from .runner import make_builder
from ._version import __version__
from functools import partial
from schema_salad.sourceline import SourceLine
from cwltool.errors import WorkflowException

def validate_cluster_target(arvrunner, runtimeContext):
    if (runtimeContext.submit_runner_cluster and
        runtimeContext.submit_runner_cluster not in arvrunner.api._rootDesc["remoteHosts"] and
        runtimeContext.submit_runner_cluster != arvrunner.api._rootDesc["uuidPrefix"]):
        raise WorkflowException("Unknown or invalid cluster id '%s' known remote clusters are %s" % (runtimeContext.submit_runner_cluster,
                                                                                                     ", ".join(list(arvrunner.api._rootDesc["remoteHosts"].keys()))))
    if runtimeContext.project_uuid:
        cluster_target = runtimeContext.submit_runner_cluster or arvrunner.api._rootDesc["uuidPrefix"]
        if not runtimeContext.project_uuid.startswith(cluster_target):
            raise WorkflowException("Project uuid '%s' should start with id of target cluster '%s'" % (runtimeContext.project_uuid, cluster_target))

        try:
            if runtimeContext.project_uuid[5:12] == '-tpzed-':
                arvrunner.api.users().get(uuid=runtimeContext.project_uuid).execute()
            else:
                proj = arvrunner.api.groups().get(uuid=runtimeContext.project_uuid).execute()
                if proj["group_class"] != "project":
                    raise Exception("not a project, group_class is '%s'" % (proj["group_class"]))
        except Exception as e:
            raise WorkflowException("Invalid project uuid '%s': %s" % (runtimeContext.project_uuid, e))

def set_cluster_target(tool, arvrunner, builder, runtimeContext):
    cluster_target_req = None
    for field in ("hints", "requirements"):
        if field not in tool:
            continue
        for item in tool[field]:
            if item["class"] == "http://arvados.org/cwl#ClusterTarget":
                cluster_target_req = item

    if cluster_target_req is None:
        return runtimeContext

    with SourceLine(cluster_target_req, None, WorkflowException, runtimeContext.debug):
        runtimeContext = runtimeContext.copy()
        runtimeContext.submit_runner_cluster = builder.do_eval(cluster_target_req.get("cluster_id")) or runtimeContext.submit_runner_cluster
        runtimeContext.project_uuid = builder.do_eval(cluster_target_req.get("project_uuid")) or runtimeContext.project_uuid
        validate_cluster_target(arvrunner, runtimeContext)

    return runtimeContext


class ArvadosCommandTool(CommandLineTool):
    """Wrap cwltool CommandLineTool to override selected methods."""

    def __init__(self, arvrunner, toolpath_object, loadingContext):
        super(ArvadosCommandTool, self).__init__(toolpath_object, loadingContext)

        (docker_req, docker_is_req) = self.get_requirement("DockerRequirement")
        if not docker_req:
            self.hints.append({"class": "DockerRequirement",
                               "dockerPull": "arvados/jobs:"+__version__})

        self.arvrunner = arvrunner

    def make_job_runner(self, runtimeContext):
        if runtimeContext.work_api == "containers":
            return partial(ArvadosContainer, self.arvrunner, runtimeContext)
        else:
            raise Exception("Unsupported work_api %s", runtimeContext.work_api)

    def make_path_mapper(self, reffiles, stagedir, runtimeContext, separateDirs):
        if runtimeContext.work_api == "containers":
            return ArvPathMapper(self.arvrunner, reffiles+runtimeContext.extra_reffiles, runtimeContext.basedir,
                                 "/keep/%s",
                                 "/keep/%s/%s")

    def job(self, joborder, output_callback, runtimeContext):
        builder = make_builder(joborder, self.hints, self.requirements, runtimeContext, self.metadata)
        runtimeContext = set_cluster_target(self.tool, self.arvrunner, builder, runtimeContext)

        if runtimeContext.work_api == "containers":
            dockerReq, is_req = self.get_requirement("DockerRequirement")
            if dockerReq and dockerReq.get("dockerOutputDirectory"):
                runtimeContext.outdir = dockerReq.get("dockerOutputDirectory")
                runtimeContext.docker_outdir = dockerReq.get("dockerOutputDirectory")
            else:
                runtimeContext.outdir = "/var/spool/cwl"
                runtimeContext.docker_outdir = "/var/spool/cwl"
        return super(ArvadosCommandTool, self).job(joborder, output_callback, runtimeContext)

class ArvadosExpressionTool(ExpressionTool):
    def __init__(self, arvrunner, toolpath_object, loadingContext):
        super(ArvadosExpressionTool, self).__init__(toolpath_object, loadingContext)
        self.arvrunner = arvrunner

    def job(self,
            job_order,         # type: Mapping[Text, Text]
            output_callback,  # type: Callable[[Any, Any], Any]
            runtimeContext     # type: RuntimeContext
           ):
        return super(ArvadosExpressionTool, self).job(job_order, self.arvrunner.get_wrapped_callback(output_callback), runtimeContext)
