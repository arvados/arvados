#!/usr/bin/env python

import arvados
import arvados.util
from arvados.collection import CollectionReader
import arvados.commands.keepdocker
import re
import subprocess
import os
import tempfile
import shutil

from pprint import pprint

def main():
    api_client  = arvados.api()

    images = arvados.commands.keepdocker.list_images_in_arv(api_client, 3)

    is_new = lambda img: img['dockerhash'].startswith('sha256:')

    count_new = 0
    old_images = []
    for uuid, img in images:
        if img["dockerhash"].startswith("sha256:"):
            continue
        key = (img["repo"], img["tag"], img["timestamp"])
        old_images.append(img)

    migration_links = arvados.util.list_all(api_client.links().list, filters=[
        ['link_class', '=', arvados.commands.keepdocker._migration_link_class],
        ['name', '=', arvados.commands.keepdocker._migration_link_name],
    ])

    already_migrated = set()
    for m in migration_links:
        already_migrated.add(m["tail_uuid"])

    for old_image in old_images:
        if old_image["collection"] in already_migrated:
            continue
        col = CollectionReader(old_image["collection"])
        tarfile = col.keys()[0]

        varlibdocker = tempfile.mkdtemp()

        try:
            dockercmd = ["docker", "run",
                         "--privileged",
                         "--rm",
                         "--env", "ARVADOS_API_HOST=%s" % (os.environ["ARVADOS_API_HOST"]),
                         "--env", "ARVADOS_API_TOKEN=%s" % (os.environ["ARVADOS_API_TOKEN"]),
                         "--env", "ARVADOS_API_HOST_INSECURE=%s" % (os.environ["ARVADOS_API_HOST_INSECURE"]),
                         "--volume", "%s:/var/lib/docker" % varlibdocker,
                         "arvados/docker19-migrate",
                         "/root/migrate.sh",
                         "%s/%s" % (old_image["collection"], tarfile),
                         tarfile[0:40],
                         old_image["repo"],
                         old_image["tag"],
                         col.api_response()["owner_uuid"]]

            out = subprocess.check_output(dockercmd)

            new_collection = re.search(r"Migrated uuid is ([a-z0-9]{5}-[a-z0-9]{5}-[a-z0-9]{15})", out)
            api_client.links().create(body={"link": {
                'owner_uuid': col.api_response()["owner_uuid"],
                'link_class': arvados.commands.keepdocker._migration_link_class,
                'name': arvados.commands.keepdocker._migration_link_name,
                'tail_uuid': old_image["collection"],
                'head_uuid': new_collection.group(1)
                }}).execute(num_retries=3)

            print "Migrated '%s' to '%s'" % (old_image["collection"], new_collection.group(1))
        finally:
            shutil.rmtree(varlibdocker)


main()
