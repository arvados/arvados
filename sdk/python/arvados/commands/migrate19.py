# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import print_function
from __future__ import division
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
    migrate19_parser.add_argument(
        '--verbose', action="store_true", help="Print stdout/stderr even on success")
    migrate19_parser.add_argument(
        '--force', action="store_true", help="Try to migrate even if there isn't enough space")

    migrate19_parser.add_argument(
        '--storage-driver', type=str, default="overlay",
        help="Docker storage driver, e.g. aufs, overlay, vfs")

    exgroup = migrate19_parser.add_mutually_exclusive_group()
    exgroup.add_argument(
        '--dry-run', action='store_true', help="Print number of pending migrations.")
    exgroup.add_argument(
        '--print-unmigrated', action='store_true',
        default=False, help="Print list of images needing migration.")

    migrate19_parser.add_argument('--tempdir', help="Set temporary directory")

    migrate19_parser.add_argument('infile', nargs='?', type=argparse.FileType('r'),
                                  default=None, help="List of images to be migrated")

    args = migrate19_parser.parse_args(arguments)

    if args.tempdir:
        tempfile.tempdir = args.tempdir

    if args.verbose:
        logger.setLevel(logging.DEBUG)

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
                                  select=["uuid", "portable_data_hash", "manifest_text", "owner_uuid"])
    uuid_to_collection = {i["uuid"]: i for i in items}

    need_migrate = {}
    totalbytes = 0
    biggest = 0
    biggest_pdh = None
    for img in old_images:
        i = uuid_to_collection[img["collection"]]
        pdh = i["portable_data_hash"]
        if pdh not in already_migrated and pdh not in need_migrate and (only_migrate is None or pdh in only_migrate):
            need_migrate[pdh] = img
            with CollectionReader(i["manifest_text"]) as c:
                size = list(c.values())[0].size()
                if size > biggest:
                    biggest = size
                    biggest_pdh = pdh
                totalbytes += size


    if args.storage_driver == "vfs":
        will_need = (biggest*20)
    else:
        will_need = (biggest*2.5)

    if args.print_unmigrated:
        only_migrate = set()
        for pdh in need_migrate:
            print(pdh)
        return

    logger.info("Already migrated %i images", len(already_migrated))
    logger.info("Need to migrate %i images", len(need_migrate))
    logger.info("Using tempdir %s", tempfile.gettempdir())
    logger.info("Biggest image %s is about %i MiB", biggest_pdh, biggest>>20)
    logger.info("Total data to migrate about %i MiB", totalbytes>>20)

    df_out = subprocess.check_output(["df", "-B1", tempfile.gettempdir()])
    ln = df_out.splitlines()[1]
    filesystem, blocks, used, available, use_pct, mounted = re.match(r"^([^ ]+) *([^ ]+) *([^ ]+) *([^ ]+) *([^ ]+) *([^ ]+)", ln).groups(1)
    if int(available) <= will_need:
        logger.warn("Temp filesystem mounted at %s does not have enough space for biggest image (has %i MiB, needs %i MiB)", mounted, int(available)>>20, int(will_need)>>20)
        if not args.force:
            exit(1)
        else:
            logger.warn("--force provided, will migrate anyway")

    if args.dry_run:
        return

    success = []
    failures = []
    count = 1
    for old_image in list(need_migrate.values()):
        if uuid_to_collection[old_image["collection"]]["portable_data_hash"] in already_migrated:
            continue

        oldcol = CollectionReader(uuid_to_collection[old_image["collection"]]["manifest_text"])
        tarfile = list(oldcol.keys())[0]

        logger.info("[%i/%i] Migrating %s:%s (%s) (%i MiB)", count, len(need_migrate), old_image["repo"],
                    old_image["tag"], old_image["collection"], list(oldcol.values())[0].size()>>20)
        count += 1
        start = time.time()

        varlibdocker = tempfile.mkdtemp()
        dockercache = tempfile.mkdtemp()
        try:
            with tempfile.NamedTemporaryFile() as envfile:
                envfile.write("ARVADOS_API_HOST=%s\n" % (arvados.config.get("ARVADOS_API_HOST")))
                envfile.write("ARVADOS_API_TOKEN=%s\n" % (arvados.config.get("ARVADOS_API_TOKEN")))
                if arvados.config.get("ARVADOS_API_HOST_INSECURE"):
                    envfile.write("ARVADOS_API_HOST_INSECURE=%s\n" % (arvados.config.get("ARVADOS_API_HOST_INSECURE")))
                envfile.flush()

                dockercmd = ["docker", "run",
                             "--privileged",
                             "--rm",
                             "--env-file", envfile.name,
                             "--volume", "%s:/var/lib/docker" % varlibdocker,
                             "--volume", "%s:/root/.cache/arvados/docker" % dockercache,
                             "arvados/migrate-docker19:1.0",
                             "/root/migrate.sh",
                             "%s/%s" % (old_image["collection"], tarfile),
                             tarfile[0:40],
                             old_image["repo"],
                             old_image["tag"],
                             uuid_to_collection[old_image["collection"]]["owner_uuid"],
                             args.storage_driver]

                proc = subprocess.Popen(dockercmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
                out, err = proc.communicate()

                initial_space = re.search(r"Initial available space is (\d+)", out)
                imgload_space = re.search(r"Available space after image load is (\d+)", out)
                imgupgrade_space = re.search(r"Available space after image upgrade is (\d+)", out)
                keepdocker_space = re.search(r"Available space after arv-keepdocker is (\d+)", out)
                cleanup_space = re.search(r"Available space after cleanup is (\d+)", out)

                if initial_space:
                    isp = int(initial_space.group(1))
                    logger.info("Available space initially: %i MiB", (isp)/(2**20))
                    if imgload_space:
                        sp = int(imgload_space.group(1))
                        logger.debug("Used after load: %i MiB", (isp-sp)/(2**20))
                    if imgupgrade_space:
                        sp = int(imgupgrade_space.group(1))
                        logger.debug("Used after upgrade: %i MiB", (isp-sp)/(2**20))
                    if keepdocker_space:
                        sp = int(keepdocker_space.group(1))
                        logger.info("Used after upload: %i MiB", (isp-sp)/(2**20))

                if cleanup_space:
                    sp = int(cleanup_space.group(1))
                    logger.debug("Available after cleanup: %i MiB", (sp)/(2**20))

                if proc.returncode != 0:
                    logger.error("Failed with return code %i", proc.returncode)
                    logger.error("--- Stdout ---\n%s", out)
                    logger.error("--- Stderr ---\n%s", err)
                    raise MigrationFailed()

                if args.verbose:
                    logger.info("--- Stdout ---\n%s", out)
                    logger.info("--- Stderr ---\n%s", err)

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
            shutil.rmtree(dockercache)

    logger.info("Successfully migrated %i images", len(success))
    if failures:
        logger.error("Failed to migrate %i images", len(failures))
