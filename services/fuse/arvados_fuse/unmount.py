import errno
import os
import subprocess
import time

def unmount(path, timeout=10):
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

    was_mounted = False
    t0 = time.time()
    delay = 0
    while True:
        if timeout and t0 + timeout < time.time():
            raise Exception("timed out")

        mounted = False
        with open('/proc/self/mountinfo') as mi:
            for m in mi.readlines():
                mntid, pmntid, dev, root, mnt, extra = m.split(" ", 5)
                mnttype = extra.split(" - ")[1].split(" ")[0]
                if not (mnttype == "fuse" or mnttype.startswith("fuse.")):
                    continue
                try:
                    if os.path.realpath(mnt) == path:
                        was_mounted = True
                        mounted = True
                        break
                except OSError:
                    continue
        if not mounted:
            return was_mounted

        major, minor = dev.split(":")
        try:
            with open('/sys/fs/fuse/connections/'+str(minor)+'/abort', 'w') as f:
                f.write("1")
        except OSError as e:
            if e.errno != errno.ENOENT:
                raise
        try:
            subprocess.check_call(["fusermount", "-u", "-z", path])
        except subprocess.CalledProcessError:
            pass

        time.sleep(delay)
        if delay == 0:
            delay = 1
