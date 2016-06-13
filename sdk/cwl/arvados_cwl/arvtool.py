from cwltool.draft2tool import CommandLineTool
from .arvjob import ArvadosJob
from .arvcontainer import ArvadosContainer
from .pathmapper import ArvPathMapper

class ArvadosCommandTool(CommandLineTool):
    """Wrap cwltool CommandLineTool to override selected methods."""

    def __init__(self, arvrunner, toolpath_object, crunch2, **kwargs):
        super(ArvadosCommandTool, self).__init__(toolpath_object, **kwargs)
        self.arvrunner = arvrunner
        self.crunch2 = crunch2

    def makeJobRunner(self):
        if self.crunch2:
            return ArvadosContainer(self.arvrunner)
        else:
            return ArvadosJob(self.arvrunner)

    def makePathMapper(self, reffiles, **kwargs):
        if self.crunch2:
            return ArvPathMapper(self.arvrunner, reffiles, kwargs["basedir"],
                                 "/keep/%s",
                                 "/keep/%s/%s",
                                 **kwargs)
        else:
            return ArvPathMapper(self.arvrunner, reffiles, kwargs["basedir"],
                                 "$(task.keep)/%s",
                                 "$(task.keep)/%s/%s",
                                 **kwargs)
