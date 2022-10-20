# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import threading
import mmap
import os
import traceback
import stat
import tempfile

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
                os.remove(final)
            except OSError:
                pass

    @staticmethod
    def get_from_disk(locator, cachedir):
        blockdir = os.path.join(cachedir, locator[0:3])
        final = os.path.join(blockdir, locator)

        try:
            filehandle = open(final, "rb")

            content = mmap.mmap(filehandle.fileno(), 0, access=mmap.ACCESS_READ)
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
    def init_cache(cachedir, maxslots):
        # map in all the files in the cache directory, up to max slots.
        # after max slots, try to delete the excess blocks.
        #
        # this gives the calling process ownership of all the blocks

        blocks = []
        for root, dirs, files in os.walk(cachedir):
            for name in files:
                blockpath = os.path.join(root, name)
                res = os.stat(blockpath)
                blocks.append((name, res.st_atime))

        # sort by access time (atime), going from most recently
        # accessed (highest timestamp) to least recently accessed
        # (lowest timestamp).
        blocks.sort(key=lambda x: x[1], reverse=True)

        # Map in all the files we found, up to maxslots, if we exceed
        # maxslots, start throwing things out.
        cachelist = []
        for b in blocks:
            got = DiskCacheSlot.get_from_disk(b[0], cachedir)
            if got is None:
                continue
            if len(cachelist) < maxslots:
                cachelist.append(got)
            else:
                # we found more blocks than maxslots, try to
                # throw it out of the cache.
                got.evict()

        return cachelist
