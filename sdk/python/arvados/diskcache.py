# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import threading
import mmap
import os
import traceback
import stat
import tempfile
import fcntl
import time
import errno
import logging

_logger = logging.getLogger('arvados.keep')

cacheblock_suffix = ".keepcacheblock"

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
        tmpfile = None
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

            final = os.path.join(blockdir, self.locator) + cacheblock_suffix

            f = tempfile.NamedTemporaryFile(dir=blockdir, delete=False, prefix="tmp", suffix=cacheblock_suffix)
            tmpfile = f.name
            os.chmod(tmpfile, stat.S_IRUSR | stat.S_IWUSR)

            # aquire a shared lock, this tells other processes that
            # we're using this block and to please not delete it.
            fcntl.flock(f, fcntl.LOCK_SH)

            f.write(value)
            f.flush()
            os.rename(tmpfile, final)
            tmpfile = None

            self.content = mmap.mmap(f.fileno(), 0, access=mmap.ACCESS_READ)
        except OSError as e:
            if e.errno == errno.ENODEV:
                _logger.error("Unable to use disk cache: The underlying filesystem does not support memory mapping.")
            elif e.errno == errno.ENOMEM:
                _logger.error("Unable to use disk cache: The process's maximum number of mappings would have been exceeded.")
            elif e.errno == errno.ENOSPC:
                _logger.error("Unable to use disk cache: Out of disk space.")
            else:
                traceback.print_exc()
        except Exception as e:
            traceback.print_exc()
        finally:
            if tmpfile is not None:
                # If the tempfile hasn't been renamed on disk yet, try to delete it.
                try:
                    os.remove(tmpfile)
                except:
                    pass
            if self.content is None:
                # Something went wrong with the disk cache, fall back
                # to RAM cache behavior (the alternative is to cache
                # nothing and return a read error).
                self.content = value
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
            final = os.path.join(blockdir, self.locator) + cacheblock_suffix
            try:
                with open(final, "rb") as f:
                    # unlock
                    fcntl.flock(f, fcntl.LOCK_UN)
                    self.content = None

                    # try to get an exclusive lock, this ensures other
                    # processes are not using the block.  It is
                    # nonblocking and will throw an exception if we
                    # can't get it, which is fine because that means
                    # we just won't try to delete it.
                    #
                    # I should note here, the file locking is not
                    # strictly necessary, we could just remove it and
                    # the kernel would ensure that the underlying
                    # inode remains available as long as other
                    # processes still have the file open.  However, if
                    # you have multiple processes sharing the cache
                    # and deleting each other's files, you'll end up
                    # with a bunch of ghost files that don't show up
                    # in the file system but are still taking up
                    # space, which isn't particularly user friendly.
                    # The locking strategy ensures that cache blocks
                    # in use remain visible.
                    #
                    fcntl.flock(f, fcntl.LOCK_EX | fcntl.LOCK_NB)

                    os.remove(final)
                    return True
            except OSError:
                pass
            return False

    @staticmethod
    def get_from_disk(locator, cachedir):
        blockdir = os.path.join(cachedir, locator[0:3])
        final = os.path.join(blockdir, locator) + cacheblock_suffix

        try:
            filehandle = open(final, "rb")

            # aquire a shared lock, this tells other processes that
            # we're using this block and to please not delete it.
            fcntl.flock(filehandle, fcntl.LOCK_SH)

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
                if not name.endswith(cacheblock_suffix):
                    continue

                blockpath = os.path.join(root, name)
                res = os.stat(blockpath)

                if len(name) == (32+len(cacheblock_suffix)) and not name.startswith("tmp"):
                    blocks.append((name[0:32], res.st_atime))
                elif name.startswith("tmp") and ((time.time() - res.st_mtime) > 60):
                    # found a temporary file more than 1 minute old,
                    # try to delete it.
                    try:
                        os.remove(blockpath)
                    except:
                        pass

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
