# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import collections
import errno
import os
import subprocess
import time


MountInfo = collections.namedtuple(
    'MountInfo', ['is_fuse', 'major', 'minor', 'mnttype', 'path'])


def mountinfo():
    mi = []
    with open('/proc/self/mountinfo') as f:
        for m in f.readlines():
            mntid, pmntid, dev, root, path, extra = m.split(" ", 5)
            mnttype = extra.split(" - ")[1].split(" ", 1)[0]
            major, minor = dev.split(":")
            mi.append(MountInfo(
                is_fuse=(mnttype == "fuse" or mnttype.startswith("fuse.")),
                major=major,
                minor=minor,
                mnttype=mnttype,
                path=path,
            ))
    return mi


def unmount(path, subtype=None, timeout=10, recursive=False):
    """Unmount the fuse mount at path.

    Unmounting is done by writing 1 to the "abort" control file in
    sysfs to kill the fuse driver process, then executing "fusermount
    -u -z" to detach the mount point, and repeating these steps until
    the mount is no longer listed in /proc/self/mountinfo.

    This procedure should enable a non-root user to reliably unmount
    their own fuse filesystem without risk of deadlock.

    Returns True if unmounting was successful, False if it wasn't a
    fuse mount at all. Raises an exception if it cannot be unmounted.
    """

    path = os.path.realpath(path)

    if subtype is None:
        mnttype = None
    elif subtype == '':
        mnttype = 'fuse'
    else:
        mnttype = 'fuse.' + subtype

    if recursive:
        paths = []
        for m in mountinfo():
            if m.path == path or m.path.startswith(path+"/"):
                paths.append(m.path)
                if not (m.is_fuse and (mnttype is None or
                                       mnttype == m.mnttype)):
                    raise Exception(
                        "cannot unmount {}: mount type is {}".format(
                            path, m.mnttype))
        for path in sorted(paths, key=len, reverse=True):
            unmount(path, timeout=timeout, recursive=False)
        return len(paths) > 0

    was_mounted = False
    attempted = False
    if timeout is None:
        deadline = None
    else:
        deadline = time.time() + timeout

    while True:
        mounted = False
        for m in mountinfo():
            if m.is_fuse and (mnttype is None or mnttype == m.mnttype):
                try:
                    if os.path.realpath(m.path) == path:
                        was_mounted = True
                        mounted = True
                        break
                except OSError:
                    continue
        if not mounted:
            return was_mounted

        if attempted:
            delay = 1
            if deadline:
                delay = min(delay, deadline - time.time())
                if delay <= 0:
                    raise Exception("timed out")
            time.sleep(delay)

        try:
            with open('/sys/fs/fuse/connections/{}/abort'.format(m.minor),
                      'w') as f:
                f.write("1")
        except OSError as e:
            if e.errno != errno.ENOENT:
                raise

        attempted = True
        try:
            subprocess.check_call(["fusermount", "-u", "-z", path])
        except subprocess.CalledProcessError:
            pass
