# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from cwltool.command_line_tool import CommandLineTool
from cwltool.builder import Builder
from .arvjob import ArvadosJob
from .arvcontainer import ArvadosContainer
from .pathmapper import ArvPathMapper
from functools import partial
from schema_salad.sourceline import SourceLine
from cwltool.errors import WorkflowException

def validate_cluster_target(arvrunner, runtimeContext):
    if (runtimeContext.submit_runner_cluster and
        runtimeContext.submit_runner_cluster not in arvrunner.api._rootDesc["remoteHosts"] and
        runtimeContext.submit_runner_cluster != arvrunner.api._rootDesc["uuidPrefix"]):
        raise WorkflowException("Unknown or invalid cluster id '%s' known remote clusters are %s" % (runtimeContext.submit_runner_cluster,
                                                                                                  ", ".join(arvrunner.api._rootDesc["remoteHosts"].keys())))
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

def make_builder(joborder, hints, requirements, runtimeContext):
    return Builder(
                 job=joborder,
                 files=[],               # type: List[Dict[Text, Text]]
                 bindings=[],            # type: List[Dict[Text, Any]]
                 schemaDefs={},          # type: Dict[Text, Dict[Text, Any]]
                 names=None,               # type: Names
                 requirements=requirements,        # type: List[Dict[Text, Any]]
                 hints=hints,               # type: List[Dict[Text, Any]]
                 resources={},           # type: Dict[str, int]
                 mutation_manager=None,    # type: Optional[MutationManager]
                 formatgraph=None,         # type: Optional[Graph]
                 make_fs_access=None,      # type: Type[StdFsAccess]
                 fs_access=None,           # type: StdFsAccess
                 job_script_provider=runtimeContext.job_script_provider, # type: Optional[Any]
                 timeout=runtimeContext.eval_timeout,             # type: float
                 debug=runtimeContext.debug,               # type: bool
                 js_console=runtimeContext.js_console,          # type: bool
                 force_docker_pull=runtimeContext.force_docker_pull,   # type: bool
                 loadListing="",         # type: Text
                 outdir="",              # type: Text
                 tmpdir="",              # type: Text
                 stagedir="",            # type: Text
                )

class ArvadosCommandTool(CommandLineTool):
    """Wrap cwltool CommandLineTool to override selected methods."""

    def __init__(self, arvrunner, toolpath_object, loadingContext):
        super(ArvadosCommandTool, self).__init__(toolpath_object, loadingContext)
        self.arvrunner = arvrunner

    def make_job_runner(self, runtimeContext):
        if runtimeContext.work_api == "containers":
            return partial(ArvadosContainer, self.arvrunner, runtimeContext)
        elif runtimeContext.work_api == "jobs":
            return partial(ArvadosJob, self.arvrunner)
        else:
            raise Exception("Unsupported work_api %s", runtimeContext.work_api)

    def make_path_mapper(self, reffiles, stagedir, runtimeContext, separateDirs):
        if runtimeContext.work_api == "containers":
            return ArvPathMapper(self.arvrunner, reffiles+runtimeContext.extra_reffiles, runtimeContext.basedir,
                                 "/keep/%s",
                                 "/keep/%s/%s")
        elif runtimeContext.work_api == "jobs":
            return ArvPathMapper(self.arvrunner, reffiles, runtimeContext.basedir,
                                 "$(task.keep)/%s",
                                 "$(task.keep)/%s/%s")

    def job(self, joborder, output_callback, runtimeContext):
        builder = make_builder(joborder, self.hints, self.requirements, runtimeContext)
        runtimeContext = set_cluster_target(self.tool, self.arvrunner, builder, runtimeContext)

        if runtimeContext.work_api == "containers":
            dockerReq, is_req = self.get_requirement("DockerRequirement")
            if dockerReq and dockerReq.get("dockerOutputDirectory"):
                runtimeContext.outdir = dockerReq.get("dockerOutputDirectory")
                runtimeContext.docker_outdir = dockerReq.get("dockerOutputDirectory")
            else:
                runtimeContext.outdir = "/var/spool/cwl"
                runtimeContext.docker_outdir = "/var/spool/cwl"
        elif runtimeContext.work_api == "jobs":
            runtimeContext.outdir = "$(task.outdir)"
            runtimeContext.docker_outdir = "$(task.outdir)"
            runtimeContext.tmpdir = "$(task.tmpdir)"
            runtimeContext.docker_tmpdir = "$(task.tmpdir)"
        return super(ArvadosCommandTool, self).job(joborder, output_callback, runtimeContext)
