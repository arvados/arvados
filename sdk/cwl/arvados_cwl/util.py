# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import datetime
from arvados.errors import ApiError

collectionUUID =  "http://arvados.org/cwl#collectionUUID"

def get_intermediate_collection_info(workflow_step_name, current_container, intermediate_output_ttl):
        if workflow_step_name:
            name = "Intermediate collection for step %s" % (workflow_step_name)
        else:
            name = "Intermediate collection"
        trash_time = None
        if intermediate_output_ttl > 0:
            trash_time = datetime.datetime.utcnow() + datetime.timedelta(seconds=intermediate_output_ttl)
        container_uuid = None
        props = {"type": "intermediate"}
        if current_container:
            props["container"] = current_container['uuid']

        return {"name" : name, "trash_at" : trash_time, "properties" : props}

def get_current_container(api, num_retries=0, logger=None):
    current_container = None
    try:
        current_container = api.containers().current().execute(num_retries=num_retries)
    except ApiError as e:
        # Status code 404 just means we're not running in a container.
        if e.resp.status != 404:
            if logger:
                logger.info("Getting current container: %s", e)
            raise e

    return current_container
