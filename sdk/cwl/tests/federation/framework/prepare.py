# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import arvados
import json

api = arvados.api()

with open("config.json") as f:
    config = json.load(f)

scrub_collections = set(config["scrub_collections"])

for cluster_id in config["arvados_cluster_ids"]:
    images = []
    for scrub_image in config["scrub_images"]:
        sp = scrub_image.split(":")
        image_name = sp[0]
        image_tag = sp[1] if len(sp) > 1 else "latest"
        images.append('{}:{}'.format(image_name, image_tag))

    search_links = api.links().list(
        filters=[['link_class', '=', 'docker_image_repo+tag'],
                 ['name', 'in', images]],
        cluster_id=cluster_id).execute()

    head_uuids = [lk["head_uuid"] for lk in search_links["items"]]
    cols = api.collections().list(filters=[["uuid", "in", head_uuids]],
                                  cluster_id=cluster_id).execute()
    for c in cols["items"]:
        scrub_collections.add(c["portable_data_hash"])
    for lk in search_links["items"]:
        api.links().delete(uuid=lk["uuid"]).execute()

for cluster_id in config["arvados_cluster_ids"]:
    matches = api.collections().list(filters=[["portable_data_hash", "in", list(scrub_collections)]],
                                     select=["uuid", "portable_data_hash"], cluster_id=cluster_id).execute()
    for m in matches["items"]:
        api.collections().delete(uuid=m["uuid"]).execute()
        print("Scrubbed %s (%s)" % (m["uuid"], m["portable_data_hash"]))
