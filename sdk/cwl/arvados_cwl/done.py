from cwltool.errors import WorkflowException

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
                "Job output '%s' cannot be found on API server" % (
                    record["output"]))

        # Create new collection in the parent project
        # with the output contents.
        self.arvrunner.api.collections().create(body={
            "owner_uuid": self.arvrunner.project_uuid,
            "name": colname,
            "portable_data_hash": record["output"],
            "manifest_text": collections["items"][0]["manifest_text"]
        }, ensure_unique_name=True).execute(
            num_retries=self.arvrunner.num_retries)

    self.builder.outdir = outdir
    self.builder.pathmapper.keepdir = keepdir
    return self.collect_outputs("keep:" + record["output"])
