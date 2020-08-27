# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import collections
import errno
import os
import subprocess
import sys
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


def paths_to_unmount(path, mnttype):
    paths = []
    for m in mountinfo():
        if m.path == path or m.path.startswith(path+"/"):
            paths.append(m.path)
            if not (m.is_fuse and (mnttype is None or
                                   mnttype == m.mnttype)):
                raise Exception(
                    "cannot unmount {}: mount type is {}".format(
                        path, m.mnttype))
    return paths


def safer_realpath(path, loop=True):
    """Similar to os.path.realpath(), but avoids calling lstat().

    Leaves some symlinks unresolved."""
    if path == '/':
        return path, True
    elif not path.startswith('/'):
        path = os.path.abspath(path)
    while True:
        path = path.rstrip('/')
        dirname, basename = os.path.split(path)
        try:
            path, resolved = safer_realpath(os.path.join(dirname, os.readlink(path)), loop=False)
        except OSError as e:
            # Path is not a symlink (EINVAL), or is unreadable, or
            # doesn't exist. If the error was EINVAL and dirname can
            # be resolved, we will have eliminated all symlinks and it
            # will be safe to call normpath().
            dirname, resolved = safer_realpath(dirname, loop=loop)
            path = os.path.join(dirname, basename)
            if resolved and e.errno == errno.EINVAL:
                return os.path.normpath(path), True
            else:
                return path, False
        except RuntimeError:
            if not loop:
                # Unwind to the point where we first started following
                # symlinks.
                raise
            # Resolving the whole path landed in a symlink cycle, but
            # we might still be able to resolve dirname.
            dirname, _ = safer_realpath(dirname, loop=loop)
            return os.path.join(dirname, basename), False


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

    path, _ = safer_realpath(path)

    if subtype is None:
        mnttype = None
    elif subtype == '':
        mnttype = 'fuse'
    else:
        mnttype = 'fuse.' + subtype

    if recursive:
        paths = paths_to_unmount(path, mnttype)
        if not paths:
            # We might not have found any mounts merely because path
            # contains symlinks, so we should resolve them and try
            # again. We didn't do this from the outset because
            # realpath() can hang (see explanation below).
            paths = paths_to_unmount(os.path.realpath(path), mnttype)
        for path in sorted(paths, key=len, reverse=True):
            unmount(path, timeout=timeout, recursive=False)
        return len(paths) > 0

    was_mounted = False
    attempted = False
    fusermount_output = b''
    if timeout is None:
        deadline = None
    else:
        deadline = time.time() + timeout

    while True:
        mounted = False
        for m in mountinfo():
            if m.is_fuse and (mnttype is None or mnttype == m.mnttype):
                try:
                    if m.path == path:
                        was_mounted = True
                        mounted = True
                        break
                except OSError:
                    continue
        if not was_mounted and path != os.path.realpath(path):
            # If the specified path contains symlinks, it won't appear
            # verbatim in mountinfo.
            #
            # It might seem like we should have called realpath() from
            # the outset. But we can't: realpath() hangs (in lstat())
            # if we call it on an unresponsive mount point, and this
            # is an important and common scenario.
            #
            # By waiting until now to try realpath(), we avoid this
            # problem in the most common cases, which are: (1) the
            # specified path has no symlinks and is a mount point, in
            # which case was_mounted==True and we can proceed without
            # calling realpath(); and (2) the specified path is not a
            # mount point (e.g., it was already unmounted by someone
            # else, or it's a typo), and realpath() can determine that
            # without hitting any other unresponsive mounts.
            path = os.path.realpath(path)
            continue
        elif not mounted:
            return was_mounted

        if attempted:
            # Report buffered stderr from previous call to fusermount,
            # now that we know it didn't succeed.
            sys.stderr.write(fusermount_output)

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
        except IOError as e:
            if e.errno != errno.ENOENT:
                raise

        attempted = True
        try:
            subprocess.check_output(
                ["fusermount", "-u", "-z", path],
                stderr=subprocess.STDOUT)
        except subprocess.CalledProcessError as e:
            fusermount_output = e.output
        else:
            fusermount_output = b''
