import arvados
import subprocess
from arvados_fuse import Operations, SafeApi, CollectionDirectory
import tempfile
import os
import llfuse
import threading
import Queue
import argparse

parser = argparse.ArgumentParser()
parser.add_argument('--project', type=str, required=True, help="Project to watch")
parser.add_argument('--image', type=str, required=True, help="Docker image to run")

args = parser.parse_args()

api = SafeApi(arvados.config)
project = args.project
docker_image = args.image
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

    import pprint
    pprint.pprint(ev)

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

filters = [['owner_uuid', '=', project],
           ['uuid', 'is_a', 'arvados#collection']]

collection = api.collections().list(filters=filters,
                    limit=1,
                    order='modified_at desc').execute()['items'][0]['uuid']

ws = arvados.events.subscribe(api, filters, on_message)

while True:
    mountdir = run_fuse_mount(collection)
    try:
        cid = subprocess.check_output(["docker", "run",
                                       "--detach=true",
                                       "--volume=%s:/mnt:ro" % mountdir,
                                       docker_image])
        running = True
        while running:
            eq = evqueue.get()
            if eq[1] == 'add' or eq[1] == 'update':
                collection = eq[2]
                running = False

        cid = subprocess.call(["docker", "stop", cid.rstrip()])
    finally:
        subprocess.call(["fusermount", "-u", "-z", mountdir])
        os.rmdir(mountdir)
