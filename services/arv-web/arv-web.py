#!/usr/bin/env python

import arvados
import subprocess
from arvados_fuse import Operations, SafeApi, CollectionDirectory
import tempfile
import os
import llfuse
import threading
import Queue
import argparse
import logging
import signal
import sys
import functools

def run_fuse_mount(api, collection):
    mountdir = tempfile.mkdtemp()

    operations = Operations(os.getuid(), os.getgid(), "utf-8")
    cdir = CollectionDirectory(llfuse.ROOT_INODE, operations.inodes, api, 2, collection)
    operations.inodes.add_entry(cdir)

    # Initialize the fuse connection
    llfuse.init(operations, mountdir, ['allow_other'])

    t = threading.Thread(None, llfuse.main)
    t.start()

    # wait until the driver is finished initializing
    operations.initlock.wait()

    return (mountdir, cdir)

def on_message(project, evqueue, ev):
    if 'event_type' in ev:
        old_attr = None
        if 'old_attributes' in ev['properties'] and ev['properties']['old_attributes']:
            old_attr = ev['properties']['old_attributes']
        if project not in (ev['properties']['new_attributes']['owner_uuid'],
                                old_attr['owner_uuid'] if old_attr else None):
            return

        et = ev['event_type']
        if ev['event_type'] == 'update':
            if ev['properties']['new_attributes']['owner_uuid'] != ev['properties']['old_attributes']['owner_uuid']:
                if args.project_uuid == ev['properties']['new_attributes']['owner_uuid']:
                    et = 'add'
                else:
                    et = 'remove'
            if ev['properties']['new_attributes']['expires_at'] is not None:
                et = 'remove'

        evqueue.put((project, et, ev['object_uuid']))

def main(argv):
    logger = logging.getLogger('arvados.arv-web')
    logger.setLevel(logging.INFO)

    parser = argparse.ArgumentParser()
    parser.add_argument('--project-uuid', type=str, required=True, help="Project uuid to watch")
    parser.add_argument('--port', type=int, default=8080, help="Port to listen on (default 8080)")
    parser.add_argument('--image', type=str, help="Docker image to run")

    args = parser.parse_args(argv)

    api = SafeApi(arvados.config)
    project = args.project_uuid
    docker_image = args.image
    port = args.port
    evqueue = Queue.Queue()

    collections = api.collections().list(filters=[["owner_uuid", "=", project]],
                        limit=1,
                        order='modified_at desc').execute()['items']
    newcollection = collections[0]['uuid'] if len(collections) > 0 else None
    collection = None

    ws = arvados.events.subscribe(api, [["object_uuid", "is_a", "arvados#collection"]], functools.partial(on_message, project, evqueue))

    signal.signal(signal.SIGTERM, lambda signal, frame: sys.exit(0))

    loop = True
    cid = None
    docker_proc = None
    prev_docker_image = None
    mountdir = None

    try:
        while loop:
            loop = False
            if newcollection != collection:
                collection = newcollection
                if not mountdir:
                    (mountdir, cdir) = run_fuse_mount(api, collection)

                with llfuse.lock:
                    cdir.clear()
                    if collection:
                        logger.info("Mounting %s", collection)
                        cdir.collection_locator = collection
                        cdir.collection_object = None
                        cdir.update()

            try:
                try:
                    if collection:
                        if not args.image:
                            docker_image = None
                            while not docker_image and os.path.exists(os.path.join(mountdir, "docker_image")):
                                try:
                                    with open(os.path.join(mountdir, "docker_image")) as di:
                                        docker_image = di.read().strip()
                                except IOError as e:
                                    pass

                        if not docker_image:
                            logger.error("Collection must contain a file 'docker_image' or must specify --image on the command line.")

                        if docker_image and ((docker_image != prev_docker_image) or cid is None):
                            if cid:
                                logger.info("Stopping docker container")
                                subprocess.check_call(["docker", "stop", cid])
                                cid = None
                                docker_proc = None

                            if docker_image:
                                logger.info("Starting docker container %s", docker_image)
                                ciddir = tempfile.mkdtemp()
                                cidfilepath = os.path.join(ciddir, "cidfile")
                                docker_proc = subprocess.Popen(["docker", "run",
                                                                "--cidfile=%s" % (cidfilepath),
                                                                "--publish=%i:80" % (port),
                                                                "--volume=%s:/mnt:ro" % mountdir,
                                                                docker_image])
                                cid = None
                                while not cid and docker_proc.poll() is None:
                                    try:
                                        with open(cidfilepath) as cidfile:
                                            cid = cidfile.read().strip()
                                    except IOError:
                                        pass
                                os.unlink(cidfilepath)
                                os.rmdir(ciddir)

                                prev_docker_image = docker_image
                                logger.info("Container id %s", cid)
                        elif cid:
                            logger.info("Sending refresh signal to container")
                            subprocess.check_call(["docker", "exec", cid, "killall", "--regexp", ".*", "--signal", "HUP"])
                    elif cid:
                        logger.info("Stopping docker container")
                        subprocess.check_call(["docker", "stop", cid])
                except subprocess.CalledProcessError:
                    cid = None
                if not cid:
                    logger.warning("No service running!  Will wait for a new collection to appear in the project.")
                else:
                    logger.info("Waiting for events")
                running = True
                loop = True
                while running:
                    try:
                        eq = evqueue.get(True, 1)
                        logger.info("%s %s", eq[1], eq[2])
                        newcollection = collection
                        if eq[1] in ('add', 'update', 'create'):
                            newcollection = eq[2]
                        elif eq[1] == 'remove':
                            collections = api.collections().list(filters=[["owner_uuid", "=", project]],
                                                                limit=1,
                                                                order='modified_at desc').execute()['items']
                            newcollection = collections[0]['uuid'] if len(collections) > 0 else None
                        running = False
                    except Queue.Empty:
                        pass
                    if docker_proc and docker_proc.poll() is not None:
                        logger.warning("Service has terminated unexpectedly, restarting.")
                        cid = None
                        docker_proc = None
                        running = False

            except (KeyboardInterrupt):
                logger.info("Got keyboard interrupt")
                ws.close()
                loop = False
            except Exception as e:
                logger.exception(e)
                ws.close()
                loop = False
    finally:
        if cid:
            logger.info("Stopping docker container")
            subprocess.check_call(["docker", "stop", cid])

        if mountdir:
            logger.info("Unmounting")
            subprocess.call(["fusermount", "-u", "-z", mountdir])
            os.rmdir(mountdir)

if __name__ == '__main__':
    main(sys.argv[1:])
