# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from cwltool.command_line_tool import CommandLineTool
from .arvjob import ArvadosJob
from .arvcontainer import ArvadosContainer
from .pathmapper import ArvPathMapper

class ArvadosCommandTool(CommandLineTool):
    """Wrap cwltool CommandLineTool to override selected methods."""

    def __init__(self, arvrunner, toolpath_object, runtimeContext):
        super(ArvadosCommandTool, self).__init__(toolpath_object, runtimeContext)
        self.arvrunner = arvrunner
        self.work_api = runtimeContext.work_api

    def makeJobRunner(self, runtimeContext):
        if self.work_api == "containers":
            return ArvadosContainer(self.arvrunner)
        elif self.work_api == "jobs":
            return ArvadosJob(self.arvrunner)

    def makePathMapper(self, reffiles, stagedir, runtimeContext):
        # type: (List[Any], unicode, **Any) -> PathMapper
        if self.work_api == "containers":
            return ArvPathMapper(self.arvrunner, reffiles+runtimeContext.extra_reffiles, runtimeContext.basedir,
                                 "/keep/%s",
                                 "/keep/%s/%s")
        elif self.work_api == "jobs":
            return ArvPathMapper(self.arvrunner, reffiles, runtimeContext.basedir,
                                 "$(task.keep)/%s",
                                 "$(task.keep)/%s/%s")

    def job(self, joborder, output_callback, runtimeContext):

        # Workaround for #13365
        builderargs = runtimeContext.copy()
        builderargs.toplevel = True
        builderargs.tmp_outdir_prefix = ""
        builder = self._init_job(joborder, builderargs)
        joborder = builder.job

        runtimeContext = runtimeContext.copy()

        if self.work_api == "containers":
            dockerReq, is_req = self.get_requirement("DockerRequirement")
            if dockerReq and dockerReq.get("dockerOutputDirectory"):
                runtimeContext.outdir = dockerReq.get("dockerOutputDirectory")
                runtimeContext.docker_outdir = dockerReq.get("dockerOutputDirectory")
            else:
                runtimeContext.outdir = "/var/spool/cwl"
                runtimeContext.docker_outdir = "/var/spool/cwl"
        elif self.work_api == "jobs":
            runtimeContext.outdir = "$(task.outdir)"
            runtimeContext.docker_outdir = "$(task.outdir)"
            runtimeContext.tmpdir = "$(task.tmpdir)"
            runtimeContext.docker_tmpdir = "$(task.tmpdir)"
        return super(ArvadosCommandTool, self).job(joborder, output_callback, runtimeContext)
