# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import logging
import sys
import threading
import copy

from schema_salad.sourceline import SourceLine

import cwltool.docker
from cwltool.errors import WorkflowException
import arvados.commands.keepdocker

logger = logging.getLogger('arvados.cwl-runner')

cached_lookups = {}
cached_lookups_lock = threading.Lock()

def arv_docker_get_image(api_client, dockerRequirement, pull_image, project_uuid):
    """Check if a Docker image is available in Keep, if not, upload it using arv-keepdocker."""

    if "http://arvados.org/cwl#dockerCollectionPDH" in dockerRequirement:
        return dockerRequirement["http://arvados.org/cwl#dockerCollectionPDH"]

    if "dockerImageId" not in dockerRequirement and "dockerPull" in dockerRequirement:
        dockerRequirement = copy.deepcopy(dockerRequirement)
        dockerRequirement["dockerImageId"] = dockerRequirement["dockerPull"]
        if hasattr(dockerRequirement, 'lc'):
            dockerRequirement.lc.data["dockerImageId"] = dockerRequirement.lc.data["dockerPull"]

    global cached_lookups
    global cached_lookups_lock
    with cached_lookups_lock:
        if dockerRequirement["dockerImageId"] in cached_lookups:
            return cached_lookups[dockerRequirement["dockerImageId"]]

    with SourceLine(dockerRequirement, "dockerImageId", WorkflowException, logger.isEnabledFor(logging.DEBUG)):
        sp = dockerRequirement["dockerImageId"].split(":")
        image_name = sp[0]
        image_tag = sp[1] if len(sp) > 1 else "latest"

        images = arvados.commands.keepdocker.list_images_in_arv(api_client, 3,
                                                                image_name=image_name,
                                                                image_tag=image_tag)

        if not images:
            # Fetch Docker image if necessary.
            try:
                cwltool.docker.DockerCommandLineJob.get_image(dockerRequirement, pull_image)
            except OSError as e:
                raise WorkflowException("While trying to get Docker image '%s', failed to execute 'docker': %s" % (dockerRequirement["dockerImageId"], e))

            # Upload image to Arvados
            args = []
            if project_uuid:
                args.append("--project-uuid="+project_uuid)
            args.append(image_name)
            args.append(image_tag)
            logger.info("Uploading Docker image %s:%s", image_name, image_tag)
            try:
                arvados.commands.put.api_client = api_client
                arvados.commands.keepdocker.main(args, stdout=sys.stderr, install_sig_handlers=False, api=api_client)
            except SystemExit as e:
                if e.code:
                    raise WorkflowException("keepdocker exited with code %s" % e.code)

            images = arvados.commands.keepdocker.list_images_in_arv(api_client, 3,
                                                                    image_name=image_name,
                                                                    image_tag=image_tag)

        if not images:
            raise WorkflowException("Could not find Docker image %s:%s" % (image_name, image_tag))

        pdh = api_client.collections().get(uuid=images[0][0]).execute()["portable_data_hash"]

        with cached_lookups_lock:
            cached_lookups[dockerRequirement["dockerImageId"]] = pdh

    return pdh

def arv_docker_clear_cache():
    global cached_lookups
    global cached_lookups_lock
    with cached_lookups_lock:
        cached_lookups = {}
