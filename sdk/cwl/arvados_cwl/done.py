import re
from cwltool.errors import WorkflowException
from collections import deque

def done(self, record, tmpdir, outdir, keepdir):
    cols = [
        ("output", "Output %s of %s" % (record["output"][0:7], self.name), record["output"]),
        ("log", "Log of %s" % (record["uuid"]), record["log"])
    ]

    for coltype, colname, colpdh in cols:
        # check if collection already exists with same owner, name and content
        collection_exists = self.arvrunner.api.collections().list(
            filters=[["owner_uuid", "=", self.arvrunner.project_uuid],
                     ['portable_data_hash', '=', colpdh],
                     ["name", "=", colname]]
        ).execute(num_retries=self.arvrunner.num_retries)

        if not collection_exists["items"]:
            # Create a collection located in the same project as the
            # pipeline with the contents of the output/log.
            # First, get output/log record.
            collections = self.arvrunner.api.collections().list(
                limit=1,
                filters=[['portable_data_hash', '=', colpdh]],
                select=["manifest_text"]
            ).execute(num_retries=self.arvrunner.num_retries)

            if not collections["items"]:
                raise WorkflowException(
                    "[job %s] %s '%s' cannot be found on API server" % (
                        self.name, coltype, colpdh))

            # Create new collection in the parent project
            # with the output/log contents.
            self.arvrunner.api.collections().create(body={
                "owner_uuid": self.arvrunner.project_uuid,
                "name": colname,
                "portable_data_hash": colpdh,
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
    if len(logcollection) == 0:
        logger.info(header)
        logger.info("  ** log is empty **")
        return

    containersapi = ("crunch-run.txt" in logcollection)
    mergelogs = {}

    for log in logcollection.keys():
        if not containersapi or log in ("crunch-run.txt", "stdout.txt", "stderr.txt"):
            logname = log[:-4]
            logt = deque([], maxlen)
            mergelogs[logname] = logt
            with logcollection.open(log) as f:
                for l in f:
                    if containersapi:
                        g = timestamp_re.match(l)
                        logt.append((g.group(1), g.group(2)))
                    elif not crunchstat_re.match(l):
                        logt.append(l)

    if containersapi:
        keys = mergelogs.keys()
        loglines = []
        while True:
            earliest = None
            for k in keys:
                if mergelogs[k]:
                    if earliest is None or mergelogs[k][0][0] < mergelogs[earliest][0][0]:
                        earliest = k
            if earliest is None:
                break
            ts, msg = mergelogs[earliest].popleft()
            loglines.append("%s %s %s" % (ts, earliest, msg))
        loglines = loglines[-maxlen:]
    else:
        loglines = mergelogs.values()[0]

    logtxt = "\n  ".join(l.strip() for l in loglines)
    logger.info(header)
    logger.info("\n  %s", logtxt)
