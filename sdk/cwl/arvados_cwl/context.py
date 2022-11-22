# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from cwltool.context import LoadingContext, RuntimeContext
from collections import namedtuple

class ArvLoadingContext(LoadingContext):
    def __init__(self, kwargs=None):
        super(ArvLoadingContext, self).__init__(kwargs)

class ArvRuntimeContext(RuntimeContext):
    def __init__(self, kwargs=None):
        self.work_api = None
        self.extra_reffiles = []
        self.priority = 500
        self.enable_reuse = True
        self.runnerjob = ""
        self.submit_request_uuid = None
        self.project_uuid = None
        self.trash_intermediate = False
        self.intermediate_output_ttl = 0
        self.update_workflow = ""
        self.create_workflow = False
        self.submit_runner_ram = 0
        self.ignore_docker_for_reuse = False
        self.submit = True
        self.submit_runner_image = None
        self.wait = True
        self.cwl_runner_job = None
        self.storage_classes = "default"
        self.intermediate_storage_classes = "default"
        self.current_container = None
        self.http_timeout = 300
        self.submit_runner_cluster = None
        self.cluster_target_id = 0
        self.always_submit_runner = False
        self.collection_cache_size = 256
        self.match_local_docker = False
        self.enable_preemptible = None
        self.copy_deps = None
        self.defer_downloads = False
        self.varying_url_params = ""
        self.prefer_cached_downloads = False

        super(ArvRuntimeContext, self).__init__(kwargs)

        if self.submit_request_uuid:
            self.submit_runner_cluster = self.submit_request_uuid[0:5]

    def get_outdir(self) -> str:
        """Return self.outdir or create one with self.tmp_outdir_prefix."""
        return self.outdir

    def get_tmpdir(self) -> str:
        """Return self.tmpdir or create one with self.tmpdir_prefix."""
        return self.tmpdir

    def create_tmpdir(self) -> str:
        """Return self.tmpdir or create one with self.tmpdir_prefix."""
        return self.tmpdir
