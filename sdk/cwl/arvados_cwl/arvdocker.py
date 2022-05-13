# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import logging
import sys
import threading
import copy
import re
import subprocess

from schema_salad.sourceline import SourceLine

import cwltool.docker
from cwltool.errors import WorkflowException
import arvados.commands.keepdocker

logger = logging.getLogger('arvados.cwl-runner')

cached_lookups = {}
cached_lookups_lock = threading.Lock()

def determine_image_id(dockerImageId):
    for line in (
        subprocess.check_output(  # nosec
            ["docker", "images", "--no-trunc", "--all"]
        )
        .decode("utf-8")
        .splitlines()
    ):
        try:
            match = re.match(r"^([^ ]+)\s+([^ ]+)\s+([^ ]+)", line)
            split = dockerImageId.split(":")
            if len(split) == 1:
                split.append("latest")
            elif len(split) == 2:
                #  if split[1] doesn't  match valid tag names, it is a part of repository
                if not re.match(r"[\w][\w.-]{0,127}", split[1]):
                    split[0] = split[0] + ":" + split[1]
                    split[1] = "latest"
            elif len(split) == 3:
                if re.match(r"[\w][\w.-]{0,127}", split[2]):
                    split[0] = split[0] + ":" + split[1]
                    split[1] = split[2]
                    del split[2]

            # check for repository:tag match or image id match
            if match and (
                (split[0] == match.group(1) and split[1] == match.group(2))
                or dockerImageId == match.group(3)
            ):
                return match.group(3)
        except ValueError:
            pass

    return None


def arv_docker_get_image(api_client, dockerRequirement, pull_image, project_uuid,
                         force_pull, tmp_outdir_prefix, match_local_docker, copy_deps):
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

        out_of_project_images = arvados.commands.keepdocker.list_images_in_arv(api_client, 3,
                                                                image_name=image_name,
                                                                image_tag=image_tag,
                                                                project_uuid=None)

        if copy_deps:
            # Only images that are available in the destination project
            images = arvados.commands.keepdocker.list_images_in_arv(api_client, 3,
                                                                    image_name=image_name,
                                                                    image_tag=image_tag,
                                                                    project_uuid=project_uuid)
        else:
            images = out_of_project_images

        if match_local_docker:
            local_image_id = determine_image_id(dockerRequirement["dockerImageId"])
            if local_image_id:
                # find it in the list
                found = False
                for i in images:
                    if i[1]["dockerhash"] == local_image_id:
                        found = True
                        images = [i]
                        break
                if not found:
                    # force re-upload.
                    images = []

                for i in out_of_project_images:
                    if i[1]["dockerhash"] == local_image_id:
                        found = True
                        out_of_project_images = [i]
                        break
                if not found:
                    # force re-upload.
                    out_of_project_images = []

        if not images:
            if not out_of_project_images:
                # Fetch Docker image if necessary.
                try:
                    result = cwltool.docker.DockerCommandLineJob.get_image(dockerRequirement, pull_image,
                                                                  force_pull, tmp_outdir_prefix)
                    if not result:
                        raise WorkflowException("Docker image '%s' not available" % dockerRequirement["dockerImageId"])
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
                # If e.code is None or zero, then keepdocker exited normally and we can continue
                if e.code:
                    raise WorkflowException("keepdocker exited with code %s" % e.code)

            images = arvados.commands.keepdocker.list_images_in_arv(api_client, 3,
                                                                    image_name=image_name,
                                                                    image_tag=image_tag,
                                                                    project_uuid=project_uuid)

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
