import re

import arvados.commands.run
import arvados.collection
from cwltool.pathmapper import PathMapper, MapperEnt, abspath

class ArvPathMapper(PathMapper):
    """Convert container-local paths to and from Keep collection ids."""

    pdh_path = re.compile(r'^keep:[0-9a-f]{32}\+\d+/.+')

    def __init__(self, arvrunner, referenced_files, input_basedir,
                 collection_pattern, file_pattern, name=None, **kwargs):
        self.arvrunner = arvrunner
        self.input_basedir = input_basedir
        self.collection_pattern = collection_pattern
        self.file_pattern = file_pattern
        self.name = name
        super(ArvPathMapper, self).__init__(referenced_files, input_basedir, None)

    def setup(self, referenced_files, basedir):
        # type: (List[Any], unicode) -> None
        self._pathmap = self.arvrunner.get_uploaded()
        uploadfiles = set()

        for srcobj in referenced_files:
            if srcobj["class"] == "File":
                src = srcobj["location"]
                if "#" in src:
                    src = src[:src.index("#")]
                if isinstance(src, basestring) and ArvPathMapper.pdh_path.match(src):
                    self._pathmap[src] = MapperEnt(src, self.collection_pattern % src[5:], "File")
                if src not in self._pathmap:
                    # Local FS ref, may need to be uploaded or may be on keep
                    # mount.
                    ab = abspath(src, self.input_basedir)
                    st = arvados.commands.run.statfile("", ab, fnPattern=self.file_pattern)
                    if isinstance(st, arvados.commands.run.UploadFile):
                        uploadfiles.add((src, ab, st))
                    elif isinstance(st, arvados.commands.run.ArvFile):
                        self._pathmap[src] = MapperEnt(ab, st.fn, "File")
                    else:
                        raise cwltool.workflow.WorkflowException("Input file path '%s' is invalid" % st)

        if uploadfiles:
            arvados.commands.run.uploadfiles([u[2] for u in uploadfiles],
                                             self.arvrunner.api,
                                             dry_run=False,
                                             num_retries=3,
                                             fnPattern=self.file_pattern,
                                             name=self.name,
                                             project=self.arvrunner.project_uuid)

        for src, ab, st in uploadfiles:
            self._pathmap[src] = MapperEnt(ab, st.fn, "File")
            self.arvrunner.add_uploaded(src, self._pathmap[src])

        self.keepdir = None

    def reversemap(self, target):
        if target.startswith("keep:"):
            return (target, target)
        elif self.keepdir and target.startswith(self.keepdir):
            return (target, "keep:" + target[len(self.keepdir)+1:])
        else:
            return super(ArvPathMapper, self).reversemap(target)
