import argparse
import time
import sys
import logging
import shutil
import tempfile
import os
import subprocess
import re

import arvados
import arvados.commands.keepdocker
from arvados._version import __version__
from arvados.collection import CollectionReader

logger = logging.getLogger('arvados.migrate-docker19')
logger.setLevel(logging.DEBUG if arvados.config.get('ARVADOS_DEBUG')
                else logging.INFO)

_migration_link_class = 'docker_image_migration'
_migration_link_name = 'migrate_1.9_1.10'

class MigrationFailed(Exception):
    pass

def main(arguments=None):
    """Docker image format migration tool for Arvados.

    This converts Docker images stored in Arvados from image format v1
    (Docker <= 1.9) to image format v2 (Docker >= 1.10).

    Requires Docker running on the local host.

    Usage:

    1) Run arvados/docker/migrate-docker19/build.sh to create
    arvados/migrate-docker19 Docker image.

    2) Set ARVADOS_API_HOST and ARVADOS_API_TOKEN to the cluster you want to migrate.

    3) Run arv-migrate-docker19 from the Arvados Python SDK on the host (not in a container).

    This will query Arvados for v1 format Docker images.  For each image that
    does not already have a corresponding v2 format image (as indicated by a
    docker_image_migration tag) it will perform the following process:

    i) download the image from Arvados
    ii) load it into Docker
    iii) update the Docker version, which updates the image
    iv) save the v2 format image and upload to Arvados
    v) create a migration link

    """

    migrate19_parser = argparse.ArgumentParser()
    migrate19_parser.add_argument(
        '--version', action='version', version="%s %s" % (sys.argv[0], __version__),
        help='Print version and exit.')

    exgroup = migrate19_parser.add_mutually_exclusive_group()
    exgroup.add_argument(
        '--dry-run', action='store_true', help="Print number of pending migrations.")
    exgroup.add_argument(
        '--print-unmigrated', action='store_true',
        default=False, help="Print list of images needing migration.")

    migrate19_parser.add_argument('infile', nargs='?', type=argparse.FileType('r'),
                                  default=None, help="List of images to be migrated")

    args = migrate19_parser.parse_args(arguments)

    only_migrate = None
    if args.infile:
        only_migrate = set()
        for l in args.infile:
            only_migrate.add(l.strip())

    api_client  = arvados.api()

    user = api_client.users().current().execute()
    if not user['is_admin']:
        raise Exception("This command requires an admin token")
    sys_uuid = user['uuid'][:12] + '000000000000000'

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
        ['link_class', '=', _migration_link_class],
        ['name', '=', _migration_link_name],
    ])

    already_migrated = set()
    for m in migration_links:
        already_migrated.add(m["tail_uuid"])

    items = arvados.util.list_all(api_client.collections().list,
                                  filters=[["uuid", "in", [img["collection"] for img in old_images]]],
                                  select=["uuid", "portable_data_hash"])
    uuid_to_pdh = {i["uuid"]: i["portable_data_hash"] for i in items}

    need_migrate = {}
    for img in old_images:
        pdh = uuid_to_pdh[img["collection"]]
        if pdh not in already_migrated and (only_migrate is None or pdh in only_migrate):
            need_migrate[pdh] = img

    if args.print_unmigrated:
        only_migrate = set()
        for pdh in need_migrate:
            print pdh
        return

    logger.info("Already migrated %i images", len(already_migrated))
    logger.info("Need to migrate %i images", len(need_migrate))

    if args.dry_run:
        return

    success = []
    failures = []
    count = 1
    for old_image in need_migrate.values():
        if uuid_to_pdh[old_image["collection"]] in already_migrated:
            continue

        logger.info("[%i/%i] Migrating %s:%s (%s)", count, len(need_migrate), old_image["repo"], old_image["tag"], old_image["collection"])
        count += 1
        start = time.time()

        oldcol = CollectionReader(old_image["collection"])
        tarfile = oldcol.keys()[0]

        varlibdocker = tempfile.mkdtemp()
        try:
            with tempfile.NamedTemporaryFile() as envfile:
                envfile.write("ARVADOS_API_HOST=%s\n" % (os.environ["ARVADOS_API_HOST"]))
                envfile.write("ARVADOS_API_TOKEN=%s\n" % (os.environ["ARVADOS_API_TOKEN"]))
                if "ARVADOS_API_HOST_INSECURE" in os.environ:
                    envfile.write("ARVADOS_API_HOST_INSECURE=%s\n" % (os.environ["ARVADOS_API_HOST_INSECURE"]))
                envfile.flush()

                dockercmd = ["docker", "run",
                             "--privileged",
                             "--rm",
                             "--env-file", envfile.name,
                             "--volume", "%s:/var/lib/docker" % varlibdocker,
                             "arvados/migrate-docker19",
                             "/root/migrate.sh",
                             "%s/%s" % (old_image["collection"], tarfile),
                             tarfile[0:40],
                             old_image["repo"],
                             old_image["tag"],
                             oldcol.api_response()["owner_uuid"]]

                proc = subprocess.Popen(dockercmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
                out, err = proc.communicate()

                if proc.returncode != 0:
                    logger.error("Failed with return code %i", proc.returncode)
                    logger.error("--- Stdout ---\n%s", out)
                    logger.error("--- Stderr ---\n%s", err)
                    raise MigrationFailed()

            migrated = re.search(r"Migrated uuid is ([a-z0-9]{5}-[a-z0-9]{5}-[a-z0-9]{15})", out)
            if migrated:
                newcol = CollectionReader(migrated.group(1))

                api_client.links().create(body={"link": {
                    'owner_uuid': sys_uuid,
                    'link_class': _migration_link_class,
                    'name': _migration_link_name,
                    'tail_uuid': oldcol.portable_data_hash(),
                    'head_uuid': newcol.portable_data_hash()
                    }}).execute(num_retries=3)

                logger.info("Migrated '%s' (%s) to '%s' (%s) in %is",
                            oldcol.portable_data_hash(), old_image["collection"],
                            newcol.portable_data_hash(), migrated.group(1),
                            time.time() - start)
                already_migrated.add(oldcol.portable_data_hash())
                success.append(old_image["collection"])
            else:
                logger.error("Error migrating '%s'", old_image["collection"])
                failures.append(old_image["collection"])
        except Exception as e:
            logger.error("Failed to migrate %s in %is", old_image["collection"], time.time() - start,
                         exc_info=(not isinstance(e, MigrationFailed)))
            failures.append(old_image["collection"])
        finally:
            shutil.rmtree(varlibdocker)

    logger.info("Successfully migrated %i images", len(success))
    if failures:
        logger.error("Failed to migrate %i images", len(failures))
