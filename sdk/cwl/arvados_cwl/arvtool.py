# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from cwltool.command_line_tool import CommandLineTool
from .arvjob import ArvadosJob
from .arvcontainer import ArvadosContainer
from .pathmapper import ArvPathMapper

class ArvadosCommandTool(CommandLineTool):
    """Wrap cwltool CommandLineTool to override selected methods."""

    def __init__(self, arvrunner, toolpath_object, **kwargs):
        super(ArvadosCommandTool, self).__init__(toolpath_object, **kwargs)
        self.arvrunner = arvrunner
        self.work_api = kwargs["work_api"]

    def makeJobRunner(self, **kwargs):
        if self.work_api == "containers":
            return ArvadosContainer(self.arvrunner)
        elif self.work_api == "jobs":
            return ArvadosJob(self.arvrunner)

    def makePathMapper(self, reffiles, stagedir, **kwargs):
        # type: (List[Any], unicode, **Any) -> PathMapper
        if self.work_api == "containers":
            return ArvPathMapper(self.arvrunner, reffiles, kwargs["basedir"],
                                 "/keep/%s",
                                 "/keep/%s/%s",
                                 **kwargs)
        elif self.work_api == "jobs":
            return ArvPathMapper(self.arvrunner, reffiles, kwargs["basedir"],
                                 "$(task.keep)/%s",
                                 "$(task.keep)/%s/%s",
                                 **kwargs)

    def job(self, joborder, output_callback, **kwargs):
        if self.work_api == "containers":
            dockerReq, is_req = self.get_requirement("DockerRequirement")
            if dockerReq and dockerReq.get("dockerOutputDirectory"):
                kwargs["outdir"] = dockerReq.get("dockerOutputDirectory")
                kwargs["docker_outdir"] = dockerReq.get("dockerOutputDirectory")
            else:
                kwargs["outdir"] = "/var/spool/cwl"
                kwargs["docker_outdir"] = "/var/spool/cwl"
        elif self.work_api == "jobs":
            kwargs["outdir"] = "$(task.outdir)"
            kwargs["docker_outdir"] = "$(task.outdir)"
            kwargs["tmpdir"] = "$(task.tmpdir)"
            kwargs["docker_tmpdir"] = "$(task.tmpdir)"
        return super(ArvadosCommandTool, self).job(joborder, output_callback, **kwargs)
