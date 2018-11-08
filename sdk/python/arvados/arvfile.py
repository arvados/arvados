# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import absolute_import
from __future__ import division
from future import standard_library
from future.utils import listitems, listvalues
standard_library.install_aliases()
from builtins import range
from builtins import object
import bz2
import collections
import copy
import errno
import functools
import hashlib
import logging
import os
import queue
import re
import sys
import threading
import uuid
import zlib

from . import config
from .errors import KeepWriteError, AssertionError, ArgumentError
from .keep import KeepLocator
from ._normalize_stream import normalize_stream
from ._ranges import locators_and_ranges, replace_range, Range, LocatorAndRange
from .retry import retry_method

MOD = "mod"
WRITE = "write"

_logger = logging.getLogger('arvados.arvfile')

def split(path):
    """split(path) -> streamname, filename

    Separate the stream name and file name in a /-separated stream path and
    return a tuple (stream_name, file_name).  If no stream name is available,
    assume '.'.

    """
    try:
        stream_name, file_name = path.rsplit('/', 1)
    except ValueError:  # No / in string
        stream_name, file_name = '.', path
    return stream_name, file_name


class UnownedBlockError(Exception):
    """Raised when there's an writable block without an owner on the BlockManager."""
    pass


class _FileLikeObjectBase(object):
    def __init__(self, name, mode):
        self.name = name
        self.mode = mode
        self.closed = False

    @staticmethod
    def _before_close(orig_func):
        @functools.wraps(orig_func)
        def before_close_wrapper(self, *args, **kwargs):
            if self.closed:
                raise ValueError("I/O operation on closed stream file")
            return orig_func(self, *args, **kwargs)
        return before_close_wrapper

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        try:
            self.close()
        except Exception:
            if exc_type is None:
                raise

    def close(self):
        self.closed = True


class ArvadosFileReaderBase(_FileLikeObjectBase):
    def __init__(self, name, mode, num_retries=None):
        super(ArvadosFileReaderBase, self).__init__(name, mode)
        self._binary = 'b' in mode
        if sys.version_info >= (3, 0) and not self._binary:
            raise NotImplementedError("text mode {!r} is not implemented".format(mode))
        self._filepos = 0
        self.num_retries = num_retries
        self._readline_cache = (None, None)

    def __iter__(self):
        while True:
            data = self.readline()
            if not data:
                break
            yield data

    def decompressed_name(self):
        return re.sub('\.(bz2|gz)$', '', self.name)

    @_FileLikeObjectBase._before_close
    def seek(self, pos, whence=os.SEEK_SET):
        if whence == os.SEEK_CUR:
            pos += self._filepos
        elif whence == os.SEEK_END:
            pos += self.size()
        if pos < 0:
            raise IOError(errno.EINVAL, "Tried to seek to negative file offset.")
        self._filepos = pos
        return self._filepos

    def tell(self):
        return self._filepos

    def readable(self):
        return True

    def writable(self):
        return False

    def seekable(self):
        return True

    @_FileLikeObjectBase._before_close
    @retry_method
    def readall(self, size=2**20, num_retries=None):
        while True:
            data = self.read(size, num_retries=num_retries)
            if len(data) == 0:
                break
            yield data

    @_FileLikeObjectBase._before_close
    @retry_method
    def readline(self, size=float('inf'), num_retries=None):
        cache_pos, cache_data = self._readline_cache
        if self.tell() == cache_pos:
            data = [cache_data]
            self._filepos += len(cache_data)
        else:
            data = [b'']
        data_size = len(data[-1])
        while (data_size < size) and (b'\n' not in data[-1]):
            next_read = self.read(2 ** 20, num_retries=num_retries)
            if not next_read:
                break
            data.append(next_read)
            data_size += len(next_read)
        data = b''.join(data)
        try:
            nextline_index = data.index(b'\n') + 1
        except ValueError:
            nextline_index = len(data)
        nextline_index = min(nextline_index, size)
        self._filepos -= len(data) - nextline_index
        self._readline_cache = (self.tell(), data[nextline_index:])
        return data[:nextline_index].decode()

    @_FileLikeObjectBase._before_close
    @retry_method
    def decompress(self, decompress, size, num_retries=None):
        for segment in self.readall(size, num_retries=num_retries):
            data = decompress(segment)
            if data:
                yield data

    @_FileLikeObjectBase._before_close
    @retry_method
    def readall_decompressed(self, size=2**20, num_retries=None):
        self.seek(0)
        if self.name.endswith('.bz2'):
            dc = bz2.BZ2Decompressor()
            return self.decompress(dc.decompress, size,
                                   num_retries=num_retries)
        elif self.name.endswith('.gz'):
            dc = zlib.decompressobj(16+zlib.MAX_WBITS)
            return self.decompress(lambda segment: dc.decompress(dc.unconsumed_tail + segment),
                                   size, num_retries=num_retries)
        else:
            return self.readall(size, num_retries=num_retries)

    @_FileLikeObjectBase._before_close
    @retry_method
    def readlines(self, sizehint=float('inf'), num_retries=None):
        data = []
        data_size = 0
        for s in self.readall(num_retries=num_retries):
            data.append(s)
            data_size += len(s)
            if data_size >= sizehint:
                break
        return b''.join(data).decode().splitlines(True)

    def size(self):
        raise IOError(errno.ENOSYS, "Not implemented")

    def read(self, size, num_retries=None):
        raise IOError(errno.ENOSYS, "Not implemented")

    def readfrom(self, start, size, num_retries=None):
        raise IOError(errno.ENOSYS, "Not implemented")


class StreamFileReader(ArvadosFileReaderBase):
    class _NameAttribute(str):
        # The Python file API provides a plain .name attribute.
        # Older SDK provided a name() method.
        # This class provides both, for maximum compatibility.
        def __call__(self):
            return self

    def __init__(self, stream, segments, name):
        super(StreamFileReader, self).__init__(self._NameAttribute(name), 'rb', num_retries=stream.num_retries)
        self._stream = stream
        self.segments = segments

    def stream_name(self):
        return self._stream.name()

    def size(self):
        n = self.segments[-1]
        return n.range_start + n.range_size

    @_FileLikeObjectBase._before_close
    @retry_method
    def read(self, size, num_retries=None):
        """Read up to 'size' bytes from the stream, starting at the current file position"""
        if size == 0:
            return b''

        data = b''
        available_chunks = locators_and_ranges(self.segments, self._filepos, size)
        if available_chunks:
            lr = available_chunks[0]
            data = self._stream.readfrom(lr.locator+lr.segment_offset,
                                         lr.segment_size,
                                         num_retries=num_retries)

        self._filepos += len(data)
        return data

    @_FileLikeObjectBase._before_close
    @retry_method
    def readfrom(self, start, size, num_retries=None):
        """Read up to 'size' bytes from the stream, starting at 'start'"""
        if size == 0:
            return b''

        data = []
        for lr in locators_and_ranges(self.segments, start, size):
            data.append(self._stream.readfrom(lr.locator+lr.segment_offset, lr.segment_size,
                                              num_retries=num_retries))
        return b''.join(data)

    def as_manifest(self):
        segs = []
        for r in self.segments:
            segs.extend(self._stream.locators_and_ranges(r.locator, r.range_size))
        return " ".join(normalize_stream(".", {self.name: segs})) + "\n"


def synchronized(orig_func):
    @functools.wraps(orig_func)
    def synchronized_wrapper(self, *args, **kwargs):
        with self.lock:
            return orig_func(self, *args, **kwargs)
    return synchronized_wrapper


class StateChangeError(Exception):
    def __init__(self, message, state, nextstate):
        super(StateChangeError, self).__init__(message)
        self.state = state
        self.nextstate = nextstate

class _BufferBlock(object):
    """A stand-in for a Keep block that is in the process of being written.

    Writers can append to it, get the size, and compute the Keep locator.
    There are three valid states:

    WRITABLE
      Can append to block.

    PENDING
      Block is in the process of being uploaded to Keep, append is an error.

    COMMITTED
      The block has been written to Keep, its internal buffer has been
      released, fetching the block will fetch it via keep client (since we
      discarded the internal copy), and identifiers referring to the BufferBlock
      can be replaced with the block locator.

    """

    WRITABLE = 0
    PENDING = 1
    COMMITTED = 2
    ERROR = 3
    DELETED = 4

    def __init__(self, blockid, starting_capacity, owner):
        """
        :blockid:
          the identifier for this block

        :starting_capacity:
          the initial buffer capacity

        :owner:
          ArvadosFile that owns this block

        """
        self.blockid = blockid
        self.buffer_block = bytearray(starting_capacity)
        self.buffer_view = memoryview(self.buffer_block)
        self.write_pointer = 0
        self._state = _BufferBlock.WRITABLE
        self._locator = None
        self.owner = owner
        self.lock = threading.Lock()
        self.wait_for_commit = threading.Event()
        self.error = None

    @synchronized
    def append(self, data):
        """Append some data to the buffer.

        Only valid if the block is in WRITABLE state.  Implements an expanding
        buffer, doubling capacity as needed to accomdate all the data.

        """
        if self._state == _BufferBlock.WRITABLE:
            if not isinstance(data, bytes) and not isinstance(data, memoryview):
                data = data.encode()
            while (self.write_pointer+len(data)) > len(self.buffer_block):
                new_buffer_block = bytearray(len(self.buffer_block) * 2)
                new_buffer_block[0:self.write_pointer] = self.buffer_block[0:self.write_pointer]
                self.buffer_block = new_buffer_block
                self.buffer_view = memoryview(self.buffer_block)
            self.buffer_view[self.write_pointer:self.write_pointer+len(data)] = data
            self.write_pointer += len(data)
            self._locator = None
        else:
            raise AssertionError("Buffer block is not writable")

    STATE_TRANSITIONS = frozenset([
            (WRITABLE, PENDING),
            (PENDING, COMMITTED),
            (PENDING, ERROR),
            (ERROR, PENDING)])

    @synchronized
    def set_state(self, nextstate, val=None):
        if (self._state, nextstate) not in self.STATE_TRANSITIONS:
            raise StateChangeError("Invalid state change from %s to %s" % (self._state, nextstate), self._state, nextstate)
        self._state = nextstate

        if self._state == _BufferBlock.PENDING:
            self.wait_for_commit.clear()

        if self._state == _BufferBlock.COMMITTED:
            self._locator = val
            self.buffer_view = None
            self.buffer_block = None
            self.wait_for_commit.set()

        if self._state == _BufferBlock.ERROR:
            self.error = val
            self.wait_for_commit.set()

    @synchronized
    def state(self):
        return self._state

    def size(self):
        """The amount of data written to the buffer."""
        return self.write_pointer

    @synchronized
    def locator(self):
        """The Keep locator for this buffer's contents."""
        if self._locator is None:
            self._locator = "%s+%i" % (hashlib.md5(self.buffer_view[0:self.write_pointer]).hexdigest(), self.size())
        return self._locator

    @synchronized
    def clone(self, new_blockid, owner):
        if self._state == _BufferBlock.COMMITTED:
            raise AssertionError("Cannot duplicate committed buffer block")
        bufferblock = _BufferBlock(new_blockid, self.size(), owner)
        bufferblock.append(self.buffer_view[0:self.size()])
        return bufferblock

    @synchronized
    def clear(self):
        self._state = _BufferBlock.DELETED
        self.owner = None
        self.buffer_block = None
        self.buffer_view = None

    @synchronized
    def repack_writes(self):
        """Optimize buffer block by repacking segments in file sequence.

        When the client makes random writes, they appear in the buffer block in
        the sequence they were written rather than the sequence they appear in
        the file.  This makes for inefficient, fragmented manifests.  Attempt
        to optimize by repacking writes in file sequence.

        """
        if self._state != _BufferBlock.WRITABLE:
            raise AssertionError("Cannot repack non-writable block")

        segs = self.owner.segments()

        # Collect the segments that reference the buffer block.
        bufferblock_segs = [s for s in segs if s.locator == self.blockid]

        # Collect total data referenced by segments (could be smaller than
        # bufferblock size if a portion of the file was written and
        # then overwritten).
        write_total = sum([s.range_size for s in bufferblock_segs])

        if write_total < self.size() or len(bufferblock_segs) > 1:
            # If there's more than one segment referencing this block, it is
            # due to out-of-order writes and will produce a fragmented
            # manifest, so try to optimize by re-packing into a new buffer.
            contents = self.buffer_view[0:self.write_pointer].tobytes()
            new_bb = _BufferBlock(None, write_total, None)
            for t in bufferblock_segs:
                new_bb.append(contents[t.segment_offset:t.segment_offset+t.range_size])
                t.segment_offset = new_bb.size() - t.range_size

            self.buffer_block = new_bb.buffer_block
            self.buffer_view = new_bb.buffer_view
            self.write_pointer = new_bb.write_pointer
            self._locator = None
            new_bb.clear()
            self.owner.set_segments(segs)

    def __repr__(self):
        return "<BufferBlock %s>" % (self.blockid)


class NoopLock(object):
    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        pass

    def acquire(self, blocking=False):
        pass

    def release(self):
        pass


def must_be_writable(orig_func):
    @functools.wraps(orig_func)
    def must_be_writable_wrapper(self, *args, **kwargs):
        if not self.writable():
            raise IOError(errno.EROFS, "Collection is read-only.")
        return orig_func(self, *args, **kwargs)
    return must_be_writable_wrapper


class _BlockManager(object):
    """BlockManager handles buffer blocks.

    Also handles background block uploads, and background block prefetch for a
    Collection of ArvadosFiles.

    """

    DEFAULT_PUT_THREADS = 2
    DEFAULT_GET_THREADS = 2

    def __init__(self, keep, copies=None, put_threads=None):
        """keep: KeepClient object to use"""
        self._keep = keep
        self._bufferblocks = collections.OrderedDict()
        self._put_queue = None
        self._put_threads = None
        self._prefetch_queue = None
        self._prefetch_threads = None
        self.lock = threading.Lock()
        self.prefetch_enabled = True
        if put_threads:
            self.num_put_threads = put_threads
        else:
            self.num_put_threads = _BlockManager.DEFAULT_PUT_THREADS
        self.num_get_threads = _BlockManager.DEFAULT_GET_THREADS
        self.copies = copies
        self._pending_write_size = 0
        self.threads_lock = threading.Lock()
        self.padding_block = None

    @synchronized
    def alloc_bufferblock(self, blockid=None, starting_capacity=2**14, owner=None):
        """Allocate a new, empty bufferblock in WRITABLE state and return it.

        :blockid:
          optional block identifier, otherwise one will be automatically assigned

        :starting_capacity:
          optional capacity, otherwise will use default capacity

        :owner:
          ArvadosFile that owns this block

        """
        return self._alloc_bufferblock(blockid, starting_capacity, owner)

    def _alloc_bufferblock(self, blockid=None, starting_capacity=2**14, owner=None):
        if blockid is None:
            blockid = str(uuid.uuid4())
        bufferblock = _BufferBlock(blockid, starting_capacity=starting_capacity, owner=owner)
        self._bufferblocks[bufferblock.blockid] = bufferblock
        return bufferblock

    @synchronized
    def dup_block(self, block, owner):
        """Create a new bufferblock initialized with the content of an existing bufferblock.

        :block:
          the buffer block to copy.

        :owner:
          ArvadosFile that owns the new block

        """
        new_blockid = str(uuid.uuid4())
        bufferblock = block.clone(new_blockid, owner)
        self._bufferblocks[bufferblock.blockid] = bufferblock
        return bufferblock

    @synchronized
    def is_bufferblock(self, locator):
        return locator in self._bufferblocks

    def _commit_bufferblock_worker(self):
        """Background uploader thread."""

        while True:
            try:
                bufferblock = self._put_queue.get()
                if bufferblock is None:
                    return

                if self.copies is None:
                    loc = self._keep.put(bufferblock.buffer_view[0:bufferblock.write_pointer].tobytes())
                else:
                    loc = self._keep.put(bufferblock.buffer_view[0:bufferblock.write_pointer].tobytes(), copies=self.copies)
                bufferblock.set_state(_BufferBlock.COMMITTED, loc)
            except Exception as e:
                bufferblock.set_state(_BufferBlock.ERROR, e)
            finally:
                if self._put_queue is not None:
                    self._put_queue.task_done()

    def start_put_threads(self):
        with self.threads_lock:
            if self._put_threads is None:
                # Start uploader threads.

                # If we don't limit the Queue size, the upload queue can quickly
                # grow to take up gigabytes of RAM if the writing process is
                # generating data more quickly than it can be send to the Keep
                # servers.
                #
                # With two upload threads and a queue size of 2, this means up to 4
                # blocks pending.  If they are full 64 MiB blocks, that means up to
                # 256 MiB of internal buffering, which is the same size as the
                # default download block cache in KeepClient.
                self._put_queue = queue.Queue(maxsize=2)

                self._put_threads = []
                for i in range(0, self.num_put_threads):
                    thread = threading.Thread(target=self._commit_bufferblock_worker)
                    self._put_threads.append(thread)
                    thread.daemon = True
                    thread.start()

    def _block_prefetch_worker(self):
        """The background downloader thread."""
        while True:
            try:
                b = self._prefetch_queue.get()
                if b is None:
                    return
                self._keep.get(b)
            except Exception:
                _logger.exception("Exception doing block prefetch")

    @synchronized
    def start_get_threads(self):
        if self._prefetch_threads is None:
            self._prefetch_queue = queue.Queue()
            self._prefetch_threads = []
            for i in range(0, self.num_get_threads):
                thread = threading.Thread(target=self._block_prefetch_worker)
                self._prefetch_threads.append(thread)
                thread.daemon = True
                thread.start()


    @synchronized
    def stop_threads(self):
        """Shut down and wait for background upload and download threads to finish."""

        if self._put_threads is not None:
            for t in self._put_threads:
                self._put_queue.put(None)
            for t in self._put_threads:
                t.join()
        self._put_threads = None
        self._put_queue = None

        if self._prefetch_threads is not None:
            for t in self._prefetch_threads:
                self._prefetch_queue.put(None)
            for t in self._prefetch_threads:
                t.join()
        self._prefetch_threads = None
        self._prefetch_queue = None

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        self.stop_threads()

    @synchronized
    def repack_small_blocks(self, force=False, sync=False, closed_file_size=0):
        """Packs small blocks together before uploading"""

        self._pending_write_size += closed_file_size

        # Check if there are enough small blocks for filling up one in full
        if not (force or (self._pending_write_size >= config.KEEP_BLOCK_SIZE)):
            return

        # Search blocks ready for getting packed together before being
        # committed to Keep.
        # A WRITABLE block always has an owner.
        # A WRITABLE block with its owner.closed() implies that its
        # size is <= KEEP_BLOCK_SIZE/2.
        try:
            small_blocks = [b for b in listvalues(self._bufferblocks)
                            if b.state() == _BufferBlock.WRITABLE and b.owner.closed()]
        except AttributeError:
            # Writable blocks without owner shouldn't exist.
            raise UnownedBlockError()

        if len(small_blocks) <= 1:
            # Not enough small blocks for repacking
            return

        for bb in small_blocks:
            bb.repack_writes()

        # Update the pending write size count with its true value, just in case
        # some small file was opened, written and closed several times.
        self._pending_write_size = sum([b.size() for b in small_blocks])

        if self._pending_write_size < config.KEEP_BLOCK_SIZE and not force:
            return

        new_bb = self._alloc_bufferblock()
        new_bb.owner = []
        files = []
        while len(small_blocks) > 0 and (new_bb.write_pointer + small_blocks[0].size()) <= config.KEEP_BLOCK_SIZE:
            bb = small_blocks.pop(0)
            new_bb.owner.append(bb.owner)
            self._pending_write_size -= bb.size()
            new_bb.append(bb.buffer_view[0:bb.write_pointer].tobytes())
            files.append((bb, new_bb.write_pointer - bb.size()))

        self.commit_bufferblock(new_bb, sync=sync)

        for bb, new_bb_segment_offset in files:
            newsegs = bb.owner.segments()
            for s in newsegs:
                if s.locator == bb.blockid:
                    s.locator = new_bb.blockid
                    s.segment_offset = new_bb_segment_offset+s.segment_offset
            bb.owner.set_segments(newsegs)
            self._delete_bufferblock(bb.blockid)

    def commit_bufferblock(self, block, sync):
        """Initiate a background upload of a bufferblock.

        :block:
          The block object to upload

        :sync:
          If `sync` is True, upload the block synchronously.
          If `sync` is False, upload the block asynchronously.  This will
          return immediately unless the upload queue is at capacity, in
          which case it will wait on an upload queue slot.

        """
        try:
            # Mark the block as PENDING so to disallow any more appends.
            block.set_state(_BufferBlock.PENDING)
        except StateChangeError as e:
            if e.state == _BufferBlock.PENDING:
                if sync:
                    block.wait_for_commit.wait()
                else:
                    return
            if block.state() == _BufferBlock.COMMITTED:
                return
            elif block.state() == _BufferBlock.ERROR:
                raise block.error
            else:
                raise

        if sync:
            try:
                if self.copies is None:
                    loc = self._keep.put(block.buffer_view[0:block.write_pointer].tobytes())
                else:
                    loc = self._keep.put(block.buffer_view[0:block.write_pointer].tobytes(), copies=self.copies)
                block.set_state(_BufferBlock.COMMITTED, loc)
            except Exception as e:
                block.set_state(_BufferBlock.ERROR, e)
                raise
        else:
            self.start_put_threads()
            self._put_queue.put(block)

    @synchronized
    def get_bufferblock(self, locator):
        return self._bufferblocks.get(locator)

    @synchronized
    def get_padding_block(self):
        """Get a bufferblock 64 MB in size consisting of all zeros, used as padding
        when using truncate() to extend the size of a file.

        For reference (and possible future optimization), the md5sum of the
        padding block is: 7f614da9329cd3aebf59b91aadc30bf0+67108864

        """

        if self.padding_block is None:
            self.padding_block = self._alloc_bufferblock(starting_capacity=config.KEEP_BLOCK_SIZE)
            self.padding_block.write_pointer = config.KEEP_BLOCK_SIZE
            self.commit_bufferblock(self.padding_block, False)
        return self.padding_block

    @synchronized
    def delete_bufferblock(self, locator):
        self._delete_bufferblock(locator)

    def _delete_bufferblock(self, locator):
        bb = self._bufferblocks[locator]
        bb.clear()
        del self._bufferblocks[locator]

    def get_block_contents(self, locator, num_retries, cache_only=False):
        """Fetch a block.

        First checks to see if the locator is a BufferBlock and return that, if
        not, passes the request through to KeepClient.get().

        """
        with self.lock:
            if locator in self._bufferblocks:
                bufferblock = self._bufferblocks[locator]
                if bufferblock.state() != _BufferBlock.COMMITTED:
                    return bufferblock.buffer_view[0:bufferblock.write_pointer].tobytes()
                else:
                    locator = bufferblock._locator
        if cache_only:
            return self._keep.get_from_cache(locator)
        else:
            return self._keep.get(locator, num_retries=num_retries)

    def commit_all(self):
        """Commit all outstanding buffer blocks.

        This is a synchronous call, and will not return until all buffer blocks
        are uploaded.  Raises KeepWriteError() if any blocks failed to upload.

        """
        self.repack_small_blocks(force=True, sync=True)

        with self.lock:
            items = listitems(self._bufferblocks)

        for k,v in items:
            if v.state() != _BufferBlock.COMMITTED and v.owner:
                # Ignore blocks with a list of owners, as if they're not in COMMITTED
                # state, they're already being committed asynchronously.
                if isinstance(v.owner, ArvadosFile):
                    v.owner.flush(sync=False)

        with self.lock:
            if self._put_queue is not None:
                self._put_queue.join()

                err = []
                for k,v in items:
                    if v.state() == _BufferBlock.ERROR:
                        err.append((v.locator(), v.error))
                if err:
                    raise KeepWriteError("Error writing some blocks", err, label="block")

        for k,v in items:
            # flush again with sync=True to remove committed bufferblocks from
            # the segments.
            if v.owner:
                if isinstance(v.owner, ArvadosFile):
                    v.owner.flush(sync=True)
                elif isinstance(v.owner, list) and len(v.owner) > 0:
                    # This bufferblock is referenced by many files as a result
                    # of repacking small blocks, so don't delete it when flushing
                    # its owners, just do it after flushing them all.
                    for owner in v.owner:
                        owner.flush(sync=True)
                    self.delete_bufferblock(k)

    def block_prefetch(self, locator):
        """Initiate a background download of a block.

        This assumes that the underlying KeepClient implements a block cache,
        so repeated requests for the same block will not result in repeated
        downloads (unless the block is evicted from the cache.)  This method
        does not block.

        """

        if not self.prefetch_enabled:
            return

        if self._keep.get_from_cache(locator) is not None:
            return

        with self.lock:
            if locator in self._bufferblocks:
                return

        self.start_get_threads()
        self._prefetch_queue.put(locator)


class ArvadosFile(object):
    """Represent a file in a Collection.

    ArvadosFile manages the underlying representation of a file in Keep as a
    sequence of segments spanning a set of blocks, and implements random
    read/write access.

    This object may be accessed from multiple threads.

    """

    __slots__ = ('parent', 'name', '_writers', '_committed',
                 '_segments', 'lock', '_current_bblock', 'fuse_entry')

    def __init__(self, parent, name, stream=[], segments=[]):
        """
        ArvadosFile constructor.

        :stream:
          a list of Range objects representing a block stream

        :segments:
          a list of Range objects representing segments
        """
        self.parent = parent
        self.name = name
        self._writers = set()
        self._committed = False
        self._segments = []
        self.lock = parent.root_collection().lock
        for s in segments:
            self._add_segment(stream, s.locator, s.range_size)
        self._current_bblock = None

    def writable(self):
        return self.parent.writable()

    @synchronized
    def permission_expired(self, as_of_dt=None):
        """Returns True if any of the segment's locators is expired"""
        for r in self._segments:
            if KeepLocator(r.locator).permission_expired(as_of_dt):
                return True
        return False

    @synchronized
    def has_remote_blocks(self):
        """Returns True if any of the segment's locators has a +R signature"""

        for s in self._segments:
            if '+R' in s.locator:
                return True
        return False

    @synchronized
    def _copy_remote_blocks(self, remote_blocks={}):
        """Ask Keep to copy remote blocks and point to their local copies.

        This is called from the parent Collection.

        :remote_blocks:
            Shared cache of remote to local block mappings. This is used to avoid
            doing extra work when blocks are shared by more than one file in
            different subdirectories.
        """

        for s in self._segments:
            if '+R' in s.locator:
                try:
                    loc = remote_blocks[s.locator]
                except KeyError:
                    loc = self.parent._my_keep().refresh_signature(s.locator)
                    remote_blocks[s.locator] = loc
                s.locator = loc
                self.parent.set_committed(False)
        return remote_blocks

    @synchronized
    def segments(self):
        return copy.copy(self._segments)

    @synchronized
    def clone(self, new_parent, new_name):
        """Make a copy of this file."""
        cp = ArvadosFile(new_parent, new_name)
        cp.replace_contents(self)
        return cp

    @must_be_writable
    @synchronized
    def replace_contents(self, other):
        """Replace segments of this file with segments from another `ArvadosFile` object."""

        map_loc = {}
        self._segments = []
        for other_segment in other.segments():
            new_loc = other_segment.locator
            if other.parent._my_block_manager().is_bufferblock(other_segment.locator):
                if other_segment.locator not in map_loc:
                    bufferblock = other.parent._my_block_manager().get_bufferblock(other_segment.locator)
                    if bufferblock.state() != _BufferBlock.WRITABLE:
                        map_loc[other_segment.locator] = bufferblock.locator()
                    else:
                        map_loc[other_segment.locator] = self.parent._my_block_manager().dup_block(bufferblock, self).blockid
                new_loc = map_loc[other_segment.locator]

            self._segments.append(Range(new_loc, other_segment.range_start, other_segment.range_size, other_segment.segment_offset))

        self.set_committed(False)

    def __eq__(self, other):
        if other is self:
            return True
        if not isinstance(other, ArvadosFile):
            return False

        othersegs = other.segments()
        with self.lock:
            if len(self._segments) != len(othersegs):
                return False
            for i in range(0, len(othersegs)):
                seg1 = self._segments[i]
                seg2 = othersegs[i]
                loc1 = seg1.locator
                loc2 = seg2.locator

                if self.parent._my_block_manager().is_bufferblock(loc1):
                    loc1 = self.parent._my_block_manager().get_bufferblock(loc1).locator()

                if other.parent._my_block_manager().is_bufferblock(loc2):
                    loc2 = other.parent._my_block_manager().get_bufferblock(loc2).locator()

                if (KeepLocator(loc1).stripped() != KeepLocator(loc2).stripped() or
                    seg1.range_start != seg2.range_start or
                    seg1.range_size != seg2.range_size or
                    seg1.segment_offset != seg2.segment_offset):
                    return False

        return True

    def __ne__(self, other):
        return not self.__eq__(other)

    @synchronized
    def set_segments(self, segs):
        self._segments = segs

    @synchronized
    def set_committed(self, value=True):
        """Set committed flag.

        If value is True, set committed to be True.

        If value is False, set committed to be False for this and all parents.
        """
        if value == self._committed:
            return
        self._committed = value
        if self._committed is False and self.parent is not None:
            self.parent.set_committed(False)

    @synchronized
    def committed(self):
        """Get whether this is committed or not."""
        return self._committed

    @synchronized
    def add_writer(self, writer):
        """Add an ArvadosFileWriter reference to the list of writers"""
        if isinstance(writer, ArvadosFileWriter):
            self._writers.add(writer)

    @synchronized
    def remove_writer(self, writer, flush):
        """
        Called from ArvadosFileWriter.close(). Remove a writer reference from the list
        and do some block maintenance tasks.
        """
        self._writers.remove(writer)

        if flush or self.size() > config.KEEP_BLOCK_SIZE // 2:
            # File writer closed, not small enough for repacking
            self.flush()
        elif self.closed():
            # All writers closed and size is adequate for repacking
            self.parent._my_block_manager().repack_small_blocks(closed_file_size=self.size())

    def closed(self):
        """
        Get whether this is closed or not. When the writers list is empty, the file
        is supposed to be closed.
        """
        return len(self._writers) == 0

    @must_be_writable
    @synchronized
    def truncate(self, size):
        """Shrink or expand the size of the file.

        If `size` is less than the size of the file, the file contents after
        `size` will be discarded.  If `size` is greater than the current size
        of the file, it will be filled with zero bytes.

        """
        if size < self.size():
            new_segs = []
            for r in self._segments:
                range_end = r.range_start+r.range_size
                if r.range_start >= size:
                    # segment is past the trucate size, all done
                    break
                elif size < range_end:
                    nr = Range(r.locator, r.range_start, size - r.range_start, 0)
                    nr.segment_offset = r.segment_offset
                    new_segs.append(nr)
                    break
                else:
                    new_segs.append(r)

            self._segments = new_segs
            self.set_committed(False)
        elif size > self.size():
            padding = self.parent._my_block_manager().get_padding_block()
            diff = size - self.size()
            while diff > config.KEEP_BLOCK_SIZE:
                self._segments.append(Range(padding.blockid, self.size(), config.KEEP_BLOCK_SIZE, 0))
                diff -= config.KEEP_BLOCK_SIZE
            if diff > 0:
                self._segments.append(Range(padding.blockid, self.size(), diff, 0))
            self.set_committed(False)
        else:
            # size == self.size()
            pass

    def readfrom(self, offset, size, num_retries, exact=False):
        """Read up to `size` bytes from the file starting at `offset`.

        :exact:
         If False (default), return less data than requested if the read
         crosses a block boundary and the next block isn't cached.  If True,
         only return less data than requested when hitting EOF.
        """

        with self.lock:
            if size == 0 or offset >= self.size():
                return b''
            readsegs = locators_and_ranges(self._segments, offset, size)
            prefetch = locators_and_ranges(self._segments, offset + size, config.KEEP_BLOCK_SIZE, limit=32)

        locs = set()
        data = []
        for lr in readsegs:
            block = self.parent._my_block_manager().get_block_contents(lr.locator, num_retries=num_retries, cache_only=(bool(data) and not exact))
            if block:
                blockview = memoryview(block)
                data.append(blockview[lr.segment_offset:lr.segment_offset+lr.segment_size].tobytes())
                locs.add(lr.locator)
            else:
                break

        for lr in prefetch:
            if lr.locator not in locs:
                self.parent._my_block_manager().block_prefetch(lr.locator)
                locs.add(lr.locator)

        return b''.join(data)

    @must_be_writable
    @synchronized
    def writeto(self, offset, data, num_retries):
        """Write `data` to the file starting at `offset`.

        This will update existing bytes and/or extend the size of the file as
        necessary.

        """
        if not isinstance(data, bytes) and not isinstance(data, memoryview):
            data = data.encode()
        if len(data) == 0:
            return

        if offset > self.size():
            self.truncate(offset)

        if len(data) > config.KEEP_BLOCK_SIZE:
            # Chunk it up into smaller writes
            n = 0
            dataview = memoryview(data)
            while n < len(data):
                self.writeto(offset+n, dataview[n:n + config.KEEP_BLOCK_SIZE].tobytes(), num_retries)
                n += config.KEEP_BLOCK_SIZE
            return

        self.set_committed(False)

        if self._current_bblock is None or self._current_bblock.state() != _BufferBlock.WRITABLE:
            self._current_bblock = self.parent._my_block_manager().alloc_bufferblock(owner=self)

        if (self._current_bblock.size() + len(data)) > config.KEEP_BLOCK_SIZE:
            self._current_bblock.repack_writes()
            if (self._current_bblock.size() + len(data)) > config.KEEP_BLOCK_SIZE:
                self.parent._my_block_manager().commit_bufferblock(self._current_bblock, sync=False)
                self._current_bblock = self.parent._my_block_manager().alloc_bufferblock(owner=self)

        self._current_bblock.append(data)

        replace_range(self._segments, offset, len(data), self._current_bblock.blockid, self._current_bblock.write_pointer - len(data))

        self.parent.notify(WRITE, self.parent, self.name, (self, self))

        return len(data)

    @synchronized
    def flush(self, sync=True, num_retries=0):
        """Flush the current bufferblock to Keep.

        :sync:
          If True, commit block synchronously, wait until buffer block has been written.
          If False, commit block asynchronously, return immediately after putting block into
          the keep put queue.
        """
        if self.committed():
            return

        if self._current_bblock and self._current_bblock.state() != _BufferBlock.COMMITTED:
            if self._current_bblock.state() == _BufferBlock.WRITABLE:
                self._current_bblock.repack_writes()
            if self._current_bblock.state() != _BufferBlock.DELETED:
                self.parent._my_block_manager().commit_bufferblock(self._current_bblock, sync=sync)

        if sync:
            to_delete = set()
            for s in self._segments:
                bb = self.parent._my_block_manager().get_bufferblock(s.locator)
                if bb:
                    if bb.state() != _BufferBlock.COMMITTED:
                        self.parent._my_block_manager().commit_bufferblock(bb, sync=True)
                    to_delete.add(s.locator)
                    s.locator = bb.locator()
            for s in to_delete:
                # Don't delete the bufferblock if it's owned by many files. It'll be
                # deleted after all of its owners are flush()ed.
                if self.parent._my_block_manager().get_bufferblock(s).owner is self:
                    self.parent._my_block_manager().delete_bufferblock(s)

        self.parent.notify(MOD, self.parent, self.name, (self, self))

    @must_be_writable
    @synchronized
    def add_segment(self, blocks, pos, size):
        """Add a segment to the end of the file.

        `pos` and `offset` reference a section of the stream described by
        `blocks` (a list of Range objects)

        """
        self._add_segment(blocks, pos, size)

    def _add_segment(self, blocks, pos, size):
        """Internal implementation of add_segment."""
        self.set_committed(False)
        for lr in locators_and_ranges(blocks, pos, size):
            last = self._segments[-1] if self._segments else Range(0, 0, 0, 0)
            r = Range(lr.locator, last.range_start+last.range_size, lr.segment_size, lr.segment_offset)
            self._segments.append(r)

    @synchronized
    def size(self):
        """Get the file size."""
        if self._segments:
            n = self._segments[-1]
            return n.range_start + n.range_size
        else:
            return 0

    @synchronized
    def manifest_text(self, stream_name=".", portable_locators=False,
                      normalize=False, only_committed=False):
        buf = ""
        filestream = []
        for segment in self._segments:
            loc = segment.locator
            if self.parent._my_block_manager().is_bufferblock(loc):
                if only_committed:
                    continue
                loc = self.parent._my_block_manager().get_bufferblock(loc).locator()
            if portable_locators:
                loc = KeepLocator(loc).stripped()
            filestream.append(LocatorAndRange(loc, KeepLocator(loc).size,
                                 segment.segment_offset, segment.range_size))
        buf += ' '.join(normalize_stream(stream_name, {self.name: filestream}))
        buf += "\n"
        return buf

    @must_be_writable
    @synchronized
    def _reparent(self, newparent, newname):
        self.set_committed(False)
        self.flush(sync=True)
        self.parent.remove(self.name)
        self.parent = newparent
        self.name = newname
        self.lock = self.parent.root_collection().lock


class ArvadosFileReader(ArvadosFileReaderBase):
    """Wraps ArvadosFile in a file-like object supporting reading only.

    Be aware that this class is NOT thread safe as there is no locking around
    updating file pointer.

    """

    def __init__(self, arvadosfile, mode="r", num_retries=None):
        super(ArvadosFileReader, self).__init__(arvadosfile.name, mode=mode, num_retries=num_retries)
        self.arvadosfile = arvadosfile

    def size(self):
        return self.arvadosfile.size()

    def stream_name(self):
        return self.arvadosfile.parent.stream_name()

    @_FileLikeObjectBase._before_close
    @retry_method
    def read(self, size=None, num_retries=None):
        """Read up to `size` bytes from the file and return the result.

        Starts at the current file position.  If `size` is None, read the
        entire remainder of the file.
        """
        if size is None:
            data = []
            rd = self.arvadosfile.readfrom(self._filepos, config.KEEP_BLOCK_SIZE, num_retries)
            while rd:
                data.append(rd)
                self._filepos += len(rd)
                rd = self.arvadosfile.readfrom(self._filepos, config.KEEP_BLOCK_SIZE, num_retries)
            return b''.join(data)
        else:
            data = self.arvadosfile.readfrom(self._filepos, size, num_retries, exact=True)
            self._filepos += len(data)
            return data

    @_FileLikeObjectBase._before_close
    @retry_method
    def readfrom(self, offset, size, num_retries=None):
        """Read up to `size` bytes from the stream, starting at the specified file offset.

        This method does not change the file position.
        """
        return self.arvadosfile.readfrom(offset, size, num_retries)

    def flush(self):
        pass


class ArvadosFileWriter(ArvadosFileReader):
    """Wraps ArvadosFile in a file-like object supporting both reading and writing.

    Be aware that this class is NOT thread safe as there is no locking around
    updating file pointer.

    """

    def __init__(self, arvadosfile, mode, num_retries=None):
        super(ArvadosFileWriter, self).__init__(arvadosfile, mode=mode, num_retries=num_retries)
        self.arvadosfile.add_writer(self)

    def writable(self):
        return True

    @_FileLikeObjectBase._before_close
    @retry_method
    def write(self, data, num_retries=None):
        if self.mode[0] == "a":
            self._filepos = self.size()
        self.arvadosfile.writeto(self._filepos, data, num_retries)
        self._filepos += len(data)
        return len(data)

    @_FileLikeObjectBase._before_close
    @retry_method
    def writelines(self, seq, num_retries=None):
        for s in seq:
            self.write(s, num_retries=num_retries)

    @_FileLikeObjectBase._before_close
    def truncate(self, size=None):
        if size is None:
            size = self._filepos
        self.arvadosfile.truncate(size)

    @_FileLikeObjectBase._before_close
    def flush(self):
        self.arvadosfile.flush()

    def close(self, flush=True):
        if not self.closed:
            self.arvadosfile.remove_writer(self, flush)
            super(ArvadosFileWriter, self).close()
