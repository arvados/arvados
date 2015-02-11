#!/usr/bin/env python

# arv-web enables you to run a custom web service from the contents of an Arvados collection.
#
# See http://doc.arvados.org/user/topics/arv-web.html

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

logger = logging.getLogger('arvados.arv-web')
logger.setLevel(logging.INFO)

class ArvWeb(object):
    def __init__(self, project, docker_image, port):
        self.project = project
        self.loop = True
        self.cid = None
        self.prev_docker_image = None
        self.mountdir = None
        self.collection = None
        self.override_docker_image = docker_image
        self.port = port
        self.evqueue = Queue.Queue()
        self.api = SafeApi(arvados.config)

        if arvados.util.group_uuid_pattern.match(project) is None:
            raise arvados.errors.ArgumentError("Project uuid is not valid")

        collections = self.api.collections().list(filters=[["owner_uuid", "=", project]],
                        limit=1,
                        order='modified_at desc').execute()['items']
        self.newcollection = collections[0]['uuid'] if collections else None

        self.ws = arvados.events.subscribe(self.api, [["object_uuid", "is_a", "arvados#collection"]], self.on_message)

    def check_docker_running(self):
        # It would be less hacky to use "docker events" than poll "docker ps"
        # but that would require writing a bigger pile of code.
        if self.cid:
            ps = subprocess.check_output(["docker", "ps", "--no-trunc=true", "--filter=status=running"])
            for l in ps.splitlines():
                if l.startswith(self.cid):
                    return True
        return False

    # Handle messages from Arvados event bus.
    def on_message(self, ev):
        if 'event_type' in ev:
            old_attr = None
            if 'old_attributes' in ev['properties'] and ev['properties']['old_attributes']:
                old_attr = ev['properties']['old_attributes']
            if self.project not in (ev['properties']['new_attributes']['owner_uuid'],
                                    old_attr['owner_uuid'] if old_attr else None):
                return

            et = ev['event_type']
            if ev['event_type'] == 'update':
                if ev['properties']['new_attributes']['owner_uuid'] != ev['properties']['old_attributes']['owner_uuid']:
                    if self.project == ev['properties']['new_attributes']['owner_uuid']:
                        et = 'add'
                    else:
                        et = 'remove'
                if ev['properties']['new_attributes']['expires_at'] is not None:
                    et = 'remove'

            self.evqueue.put((self.project, et, ev['object_uuid']))

    # Run an arvados_fuse mount under the control of the local process.  This lets
    # us switch out the contents of the directory without having to unmount and
    # remount.
    def run_fuse_mount(self):
        self.mountdir = tempfile.mkdtemp()

        self.operations = Operations(os.getuid(), os.getgid(), "utf-8")
        self.cdir = CollectionDirectory(llfuse.ROOT_INODE, self.operations.inodes, self.api, 2, self.collection)
        self.operations.inodes.add_entry(self.cdir)

        # Initialize the fuse connection
        llfuse.init(self.operations, self.mountdir, ['allow_other'])

        t = threading.Thread(None, llfuse.main)
        t.start()

        # wait until the driver is finished initializing
        self.operations.initlock.wait()

    def mount_collection(self):
        if self.newcollection != self.collection:
            self.collection = self.newcollection
            if not self.mountdir and self.collection:
                self.run_fuse_mount()

            if self.mountdir:
                with llfuse.lock:
                    self.cdir.clear()
                    # Switch the FUSE directory object so that it stores
                    # the newly selected collection
                    if self.collection:
                        logger.info("Mounting %s", self.collection)
                    else:
                        logger.info("Mount is empty")
                    self.cdir.change_collection(self.collection)


    def stop_docker(self):
        if self.cid:
            logger.info("Stopping Docker container")
            subprocess.call(["docker", "stop", self.cid])
            self.cid = None

    def run_docker(self):
        try:
            if self.collection is None:
                self.stop_docker()
                return

            docker_image = None
            if self.override_docker_image:
                docker_image = self.override_docker_image
            else:
                try:
                    with llfuse.lock:
                        if "docker_image" in self.cdir:
                            docker_image = self.cdir["docker_image"].readfrom(0, 1024).strip()
                except IOError as e:
                    pass

            has_reload = False
            try:
                with llfuse.lock:
                    has_reload = "reload" in self.cdir
            except IOError as e:
                pass

            if docker_image is None:
                logger.error("Collection must contain a file 'docker_image' or must specify --image on the command line.")
                self.stop_docker()
                return

            if docker_image == self.prev_docker_image and self.cid is not None and has_reload:
                logger.info("Running container reload command")
                subprocess.check_call(["docker", "exec", self.cid, "/mnt/reload"])
                return

            self.stop_docker()

            logger.info("Starting Docker container %s", docker_image)
            ciddir = tempfile.mkdtemp()
            cidfilepath = os.path.join(ciddir, "cidfile")
            self.cid = subprocess.check_output(["docker", "run",
                                                "--detach=true",
                                                "--publish=%i:80" % (self.port),
                                                "--volume=%s:/mnt:ro" % self.mountdir,
                                                docker_image]).strip()

            self.prev_docker_image = docker_image
            logger.info("Container id %s", self.cid)

        except subprocess.CalledProcessError:
            self.cid = None

    def wait_for_events(self):
        if not self.cid:
            logger.warning("No service running!  Will wait for a new collection to appear in the project.")
        else:
            logger.info("Waiting for events")

        running = True
        self.loop = True
        while running:
            # Main run loop.  Wait on project events, signals, or the
            # Docker container stopping.

            try:
                # Poll the queue with a 1 second timeout, if we have no
                # timeout the Python runtime doesn't have a chance to
                # process SIGINT or SIGTERM.
                eq = self.evqueue.get(True, 1)
                logger.info("%s %s", eq[1], eq[2])
                self.newcollection = self.collection
                if eq[1] in ('add', 'update', 'create'):
                    self.newcollection = eq[2]
                elif eq[1] == 'remove':
                    collections = self.api.collections().list(filters=[["owner_uuid", "=", self.project]],
                                                        limit=1,
                                                        order='modified_at desc').execute()['items']
                    self.newcollection = collections[0]['uuid'] if collections else None
                running = False
            except Queue.Empty:
                pass

            if self.cid and not self.check_docker_running():
                logger.warning("Service has terminated.  Will try to restart.")
                self.cid = None
                running = False


    def run(self):
        try:
            while self.loop:
                self.loop = False
                self.mount_collection()
                try:
                    self.run_docker()
                    self.wait_for_events()
                except (KeyboardInterrupt):
                    logger.info("Got keyboard interrupt")
                    self.ws.close()
                    self.loop = False
                except Exception as e:
                    logger.exception("Caught fatal exception, shutting down")
                    self.ws.close()
                    self.loop = False
        finally:
            self.stop_docker()

            if self.mountdir:
                logger.info("Unmounting")
                subprocess.call(["fusermount", "-u", self.mountdir])
                os.rmdir(self.mountdir)


def main(argv):
    parser = argparse.ArgumentParser()
    parser.add_argument('--project-uuid', type=str, required=True, help="Project uuid to watch")
    parser.add_argument('--port', type=int, default=8080, help="Host port to listen on (default 8080)")
    parser.add_argument('--image', type=str, help="Docker image to run")

    args = parser.parse_args(argv)

    signal.signal(signal.SIGTERM, lambda signal, frame: sys.exit(0))

    try:
        arvweb = ArvWeb(args.project_uuid, args.image, args.port)
        arvweb.run()
    except arvados.errors.ArgumentError as e:
        logger.error(e)

if __name__ == '__main__':
    main(sys.argv[1:])
