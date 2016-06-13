import re

import arvados.commands.run
import arvados.collection
import cwltool.pathmapper

class ArvPathMapper(cwltool.pathmapper.PathMapper):
    """Convert container-local paths to and from Keep collection ids."""

    def __init__(self, arvrunner, referenced_files, input_basedir,
                 collection_pattern, file_pattern, name=None, **kwargs):
        self._pathmap = arvrunner.get_uploaded()
        uploadfiles = set()

        pdh_path = re.compile(r'^keep:[0-9a-f]{32}\+\d+/.+')

        for src in referenced_files:
            if isinstance(src, basestring) and pdh_path.match(src):
                self._pathmap[src] = (src, collection_pattern % src[5:])
            if "#" in src:
                src = src[:src.index("#")]
            if src not in self._pathmap:
                ab = cwltool.pathmapper.abspath(src, input_basedir)
                st = arvados.commands.run.statfile("", ab, fnPattern=file_pattern)
                if kwargs.get("conformance_test"):
                    self._pathmap[src] = (src, ab)
                elif isinstance(st, arvados.commands.run.UploadFile):
                    uploadfiles.add((src, ab, st))
                elif isinstance(st, arvados.commands.run.ArvFile):
                    self._pathmap[src] = (ab, st.fn)
                else:
                    raise cwltool.workflow.WorkflowException("Input file path '%s' is invalid" % st)

        if uploadfiles:
            arvados.commands.run.uploadfiles([u[2] for u in uploadfiles],
                                             arvrunner.api,
                                             dry_run=kwargs.get("dry_run"),
                                             num_retries=3,
                                             fnPattern=file_pattern,
                                             name=name,
                                             project=arvrunner.project_uuid)

        for src, ab, st in uploadfiles:
            arvrunner.add_uploaded(src, (ab, st.fn))
            self._pathmap[src] = (ab, st.fn)

        self.keepdir = None

    def reversemap(self, target):
        if target.startswith("keep:"):
            return (target, target)
        elif self.keepdir and target.startswith(self.keepdir):
            return (target, "keep:" + target[len(self.keepdir)+1:])
        else:
            return super(ArvPathMapper, self).reversemap(target)
