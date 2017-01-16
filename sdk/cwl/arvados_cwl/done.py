import re
from cwltool.errors import WorkflowException
from collections import deque

def done(self, record, tmpdir, outdir, keepdir):
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
                "[job %s] output '%s' cannot be found on API server" % (
                    self.name, record["output"]))

        # Create new collection in the parent project
        # with the output contents.
        self.arvrunner.api.collections().create(body={
            "owner_uuid": self.arvrunner.project_uuid,
            "name": colname,
            "portable_data_hash": record["output"],
            "manifest_text": collections["items"][0]["manifest_text"]
        }, ensure_unique_name=True).execute(
            num_retries=self.arvrunner.num_retries)

    return done_outputs(self, record, tmpdir, outdir, keepdir)

def done_outputs(self, record, tmpdir, outdir, keepdir):
    self.builder.outdir = outdir
    self.builder.pathmapper.keepdir = keepdir
    return self.collect_outputs("keep:" + record["output"])

crunchstat_re = re.compile(r"^\d{4}-\d\d-\d\d_\d\d:\d\d:\d\d [a-z0-9]{5}-8i9sb-[a-z0-9]{15} \d+ \d+ stderr crunchstat:")
timestamp_re = re.compile(r"^(\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d\.\d+Z) (.*)")

def logtail(logcollection, logger, header, maxlen=25):
    logtail = deque([], maxlen*len(logcollection))
    containersapi = ("crunch-run.txt" in logcollection)

    for log in logcollection.keys():
        if not containersapi or log in ("crunch-run.txt", "stdout.txt", "stderr.txt"):
            logname = log[:-4]
            with logcollection.open(log) as f:
                for l in f:
                    if containersapi:
                        g = timestamp_re.match(l)
                        logtail.append("%s %s %s" % (g.group(1), logname, g.group(2)))
                    elif not crunchstat_re.match(l):
                        logtail.append(l)
    if len(logcollection) > 1:
        logtail = sorted(logtail)[-maxlen:]
    logtxt = "\n  ".join(l.strip() for l in logtail)
    logger.info(header)
    logger.info("\n  %s", logtxt)
