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

logging.basicConfig(level=logging.INFO)

parser = argparse.ArgumentParser()
parser.add_argument('--project', type=str, required=True, help="Project to watch")
parser.add_argument('--port', type=int, default=8080, help="Local bind port")
parser.add_argument('--image', type=str, required=True, help="Docker image to run")

args = parser.parse_args()

api = SafeApi(arvados.config)
project = args.project
docker_image = args.image
port = args.port
evqueue = Queue.Queue()

def run_fuse_mount(collection):
    global api

    mountdir = tempfile.mkdtemp()

    operations = Operations(os.getuid(), os.getgid(), "utf-8")
    operations.inodes.add_entry(CollectionDirectory(llfuse.ROOT_INODE, operations.inodes, api, 2, collection))

    # Initialize the fuse connection
    llfuse.init(operations, mountdir, ['allow_other'])

    t = threading.Thread(None, lambda: llfuse.main())
    t.start()

    # wait until the driver is finished initializing
    operations.initlock.wait()

    return mountdir

def on_message(ev):
    global project
    global evqueue

    if 'event_type' in ev:
        old_attr = None
        if 'old_attributes' in ev['properties'] and ev['properties']['old_attributes']:
            old_attr = ev['properties']['old_attributes']
        if project not in (ev['properties']['new_attributes']['owner_uuid'],
                                old_attr['owner_uuid'] if old_attr else None):
            return

        et = ev['event_type']
        if ev['event_type'] == 'update' and ev['properties']['new_attributes']['owner_uuid'] != ev['properties']['old_attributes']['owner_uuid']:
            if args.project == ev['properties']['new_attributes']['owner_uuid']:
                et = 'add'
            else:
                et = 'remove'

        evqueue.put((project, et, ev['object_uuid']))

collection = api.collections().list(filters=[["owner_uuid", "=", project]],
                    limit=1,
                    order='modified_at desc').execute()['items'][0]['uuid']

ws = arvados.events.subscribe(api, [["object_uuid", "is_a", "arvados#collection"]], on_message)

signal.signal(signal.SIGTERM, lambda signal, frame: sys.exit(0))

loop = True
cid = None
while loop:
    logging.info("Mounting %s" % collection)
    mountdir = run_fuse_mount(collection)
    try:
        logging.info("Starting docker container")
        cid = subprocess.check_output(["docker", "run",
                                       "--detach=true",
                                       "--publish=%i:80" % (port),
                                       "--volume=%s:/mnt:ro" % mountdir,
                                       docker_image])
        cid = cid.rstrip()
        logging.info("Container id is %s" % cid)

        logging.info("Waiting for events")
        running = True
        while running:
            try:
                eq = evqueue.get(True, 1)
                logging.info("%s %s" % (eq[1], eq[2]))
                newcollection = collection
                if eq[1] in ('add', 'update', 'create'):
                    newcollection = eq[2]
                elif eq[1] == 'remove':
                    newcollection = api.collections().list(filters=[["owner_uuid", "=", project]],
                                                        limit=1,
                                                        order='modified_at desc').execute()['items'][0]['uuid']
                if newcollection != collection:
                    logging.info("restarting web service")
                    collection = newcollection
                    running = False
            except Queue.Empty:
                pass
    except (KeyboardInterrupt):
        logging.info("Got keyboard interrupt")
        ws.close()
        loop = False
    except Exception as e:
        logging.exception(str(e))
        ws.close()
        loop = False
    finally:
        if cid:
            logging.info("Stopping docker container")
            cid = subprocess.call(["docker", "stop", cid])

        logging.info("Unmounting")
        subprocess.call(["fusermount", "-u", "-z", mountdir])
        os.rmdir(mountdir)
