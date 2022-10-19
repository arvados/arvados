# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import threading
import mmap
import os
import traceback
import stat
import tempfile
import hashlib
import fcntl

class DiskCacheSlot(object):
    __slots__ = ("locator", "ready", "content", "cachedir")

    def __init__(self, locator, cachedir):
        self.locator = locator
        self.ready = threading.Event()
        self.content = None
        self.cachedir = cachedir

    def get(self):
        self.ready.wait()
        return self.content

    def set(self, value):
        try:
            if value is None:
                self.content = None
                return

            if len(value) == 0:
                # Can't mmap a 0 length file
                self.content = b''
                return

            if self.content is not None:
                # Has been set already
                return

            blockdir = os.path.join(self.cachedir, self.locator[0:3])
            os.makedirs(blockdir, mode=0o700, exist_ok=True)

            final = os.path.join(blockdir, self.locator)

            f = tempfile.NamedTemporaryFile(dir=blockdir, delete=False)
            tmpfile = f.name
            os.chmod(tmpfile, stat.S_IRUSR | stat.S_IWUSR)

            # aquire a shared lock, this tells other processes that
            # we're using this block and to please not delete it.
            fcntl.flock(f, fcntl.LOCK_SH)

            f.write(value)
            f.flush()
            os.rename(tmpfile, final)

            self.content = mmap.mmap(f.fileno(), 0, access=mmap.ACCESS_READ)
        except Exception as e:
            traceback.print_exc()
        finally:
            self.ready.set()

    def size(self):
        if self.content is None:
            return 0
        else:
            return len(self.content)

    def evict(self):
        if self.content is not None and len(self.content) > 0:
            # The mmap region might be in use when we decided to evict
            # it.  This can happen if the cache is too small.
            #
            # If we call close() now, it'll throw an error if
            # something tries to access it.
            #
            # However, we don't need to explicitly call mmap.close()
            #
            # I confirmed in mmapmodule.c that that both close
            # and deallocate do the same thing:
            #
            # a) close the file descriptor
            # b) unmap the memory range
            #
            # So we can forget it in the cache and delete the file on
            # disk, and it will tear it down after any other
            # lingering Python references to the mapped memory are
            # gone.

            blockdir = os.path.join(self.cachedir, self.locator[0:3])
            final = os.path.join(blockdir, self.locator)
            try:
                # If we can't upgrade our shared lock to an exclusive
                # lock, it'll throw an error, that's fine and
                # desirable, it means another process has a lock and
                # we shouldn't delete the block.
                fcntl.flock(f, fcntl.LOCK_EX | fcntl.LOCK_NB)
                os.remove(final)
            except OSError:
                pass

    @staticmethod
    def get_from_disk(locator, cachedir):
        # Get it, check it, return it
        blockdir = os.path.join(cachedir, locator[0:3])
        final = os.path.join(blockdir, locator)

        try:
            filehandle = open(final, "rb")

            # aquire a shared lock, this tells other processes that
            # we're using this block and to please not delete it.
            fcntl.flock(f, fcntl.LOCK_SH)

            content = mmap.mmap(filehandle.fileno(), 0, access=mmap.ACCESS_READ)
            disk_md5 = hashlib.md5(content).hexdigest()
            if disk_md5 == locator:
                dc = DiskCacheSlot(locator, cachedir)
                dc.content = content
                dc.ready.set()
                return dc
        except FileNotFoundError:
            pass
        except Exception as e:
            traceback.print_exc()

        return None

    @staticmethod
    def cleanup_cachedir(cachedir, maxsize):
        blocks = []
        totalsize = 0
        for root, dirs, files in os.walk(cachedir):
            for name in files:
                blockpath = os.path.join(root, name)
                res = os.stat(blockpath)
                blocks.append((blockpath, res.st_size, res.st_atime))
                totalsize += res.st_size

        if totalsize <= maxsize:
            return

        # sort by atime, so the blocks accessed the longest time in
        # the past get deleted first.
        blocks.sort(key=lambda x: x[2])

        # go through the list and try deleting blocks until we're
        # below the target size and/or we run out of blocks
        i = 0
        while i < len(blocks) and totalsize > maxsize:
            try:
                with open(blocks[i][0], "rb") as f:
                    # If we can't get an exclusive lock, it'll
                    # throw an error, that's fine and desirable,
                    # it means another process has a lock and we
                    # shouldn't delete the block.
                    fcntl.flock(f, fcntl.LOCK_EX | fcntl.LOCK_NB)
                    os.remove(block)
                    totalsize -= blocks[i][1]
            except OSError:
                pass
            i += 1
