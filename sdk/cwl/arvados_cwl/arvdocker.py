import logging
import sys

import cwltool.docker
from cwltool.errors import WorkflowException
import arvados.commands.keepdocker


logger = logging.getLogger('arvados.cwl-runner')

def arv_docker_get_image(api_client, dockerRequirement, pull_image, project_uuid):
    """Check if a Docker image is available in Keep, if not, upload it using arv-keepdocker."""

    if "dockerImageId" not in dockerRequirement and "dockerPull" in dockerRequirement:
        dockerRequirement["dockerImageId"] = dockerRequirement["dockerPull"]

    sp = dockerRequirement["dockerImageId"].split(":")
    image_name = sp[0]
    image_tag = sp[1] if len(sp) > 1 else None

    images = arvados.commands.keepdocker.list_images_in_arv(api_client, 3,
                                                            image_name=image_name,
                                                            image_tag=image_tag)

    if not images:
        imageId = cwltool.docker.get_image(dockerRequirement, pull_image)
        args = ["--project-uuid="+project_uuid, image_name]
        if image_tag:
            args.append(image_tag)
        logger.info("Uploading Docker image %s", ":".join(args[1:]))
        try:
            arvados.commands.keepdocker.main(args, stdout=sys.stderr)
        except SystemExit:
            raise WorkflowException()

    images = arvados.commands.keepdocker.list_images_in_arv(api_client, 3,
                                                            image_name=image_name,
                                                            image_tag=image_tag)

    #return dockerRequirement["dockerImageId"]

    pdh = api_client.collections().get(uuid=images[0][0]).execute()["portable_data_hash"]
    return pdh
