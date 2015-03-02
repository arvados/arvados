import functools
import os
import zlib
import bz2
from ._ranges import locators_and_ranges, replace_range, Range
from arvados.retry import retry_method
import config
import hashlib
import threading
import Queue
import copy
import errno
from .errors import KeepWriteError, AssertionError
from .keep import KeepLocator
from _normalize_stream import normalize_stream

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
        self._filepos = 0L
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
    def seek(self, pos, whence=os.SEEK_CUR):
        if whence == os.SEEK_CUR:
            pos += self._filepos
        elif whence == os.SEEK_END:
            pos += self.size()
        self._filepos = min(max(pos, 0L), self.size())

    def tell(self):
        return self._filepos

    @_FileLikeObjectBase._before_close
    @retry_method
    def readall(self, size=2**20, num_retries=None):
        while True:
            data = self.read(size, num_retries=num_retries)
            if data == '':
                break
            yield data

    @_FileLikeObjectBase._before_close
    @retry_method
    def readline(self, size=float('inf'), num_retries=None):
        cache_pos, cache_data = self._readline_cache
        if self.tell() == cache_pos:
            data = [cache_data]
        else:
            data = ['']
        data_size = len(data[-1])
        while (data_size < size) and ('\n' not in data[-1]):
            next_read = self.read(2 ** 20, num_retries=num_retries)
            if not next_read:
                break
            data.append(next_read)
            data_size += len(next_read)
        data = ''.join(data)
        try:
            nextline_index = data.index('\n') + 1
        except ValueError:
            nextline_index = len(data)
        nextline_index = min(nextline_index, size)
        self._readline_cache = (self.tell(), data[nextline_index:])
        return data[:nextline_index]

    @_FileLikeObjectBase._before_close
    @retry_method
    def decompress(self, decompress, size, num_retries=None):
        for segment in self.readall(size, num_retries):
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
        return ''.join(data).splitlines(True)

    def size(self):
        raise NotImplementedError()

    def read(self, size, num_retries=None):
        raise NotImplementedError()

    def readfrom(self, start, size, num_retries=None):
        raise NotImplementedError()


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
            return ''

        data = ''
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
            return ''

        data = []
        for lr in locators_and_ranges(self.segments, start, size):
            data.append(self._stream.readfrom(lr.locator+lr.segment_offset, lr.segment_size,
                                              num_retries=num_retries))
        return ''.join(data)

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

    @synchronized
    def append(self, data):
        """Append some data to the buffer.

        Only valid if the block is in WRITABLE state.  Implements an expanding
        buffer, doubling capacity as needed to accomdate all the data.

        """
        if self._state == _BufferBlock.WRITABLE:
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

    @synchronized
    def set_state(self, nextstate, loc=None):
        if ((self._state == _BufferBlock.WRITABLE and nextstate == _BufferBlock.PENDING) or
            (self._state == _BufferBlock.PENDING and nextstate == _BufferBlock.COMMITTED)):
            self._state = nextstate
            if self._state == _BufferBlock.COMMITTED:
                self._locator = loc
                self.buffer_view = None
                self.buffer_block = None
        else:
            raise AssertionError("Invalid state change from %s to %s" % (self.state, state))

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
            raise AssertionError("Can only duplicate a writable or pending buffer block")
        bufferblock = _BufferBlock(new_blockid, self.size(), owner)
        bufferblock.append(self.buffer_view[0:self.size()])
        return bufferblock


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
            raise IOError((errno.EROFS, "Collection must be writable."))
        return orig_func(self, *args, **kwargs)
    return must_be_writable_wrapper


class _BlockManager(object):
    """BlockManager handles buffer blocks.

    Also handles background block uploads, and background block prefetch for a
    Collection of ArvadosFiles.

    """
    def __init__(self, keep):
        """keep: KeepClient object to use"""
        self._keep = keep
        self._bufferblocks = {}
        self._put_queue = None
        self._put_errors = None
        self._put_threads = None
        self._prefetch_queue = None
        self._prefetch_threads = None
        self.lock = threading.Lock()
        self.prefetch_enabled = True
        self.num_put_threads = 2
        self.num_get_threads = 2

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
        if blockid is None:
            blockid = "bufferblock%i" % len(self._bufferblocks)
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
        new_blockid = "bufferblock%i" % len(self._bufferblocks)
        bufferblock = block.clone(new_blockid, owner)
        self._bufferblocks[bufferblock.blockid] = bufferblock
        return bufferblock

    @synchronized
    def is_bufferblock(self, locator):
        return locator in self._bufferblocks

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
        self._put_errors = None

        if self._prefetch_threads is not None:
            for t in self._prefetch_threads:
                self._prefetch_queue.put(None)
            for t in self._prefetch_threads:
                t.join()
        self._prefetch_threads = None
        self._prefetch_queue = None

    def commit_bufferblock(self, block):
        """Initiate a background upload of a bufferblock.

        This will block if the upload queue is at capacity, otherwise it will
        return immediately.

        """

        def commit_bufferblock_worker(self):
            """Background uploader thread."""

            while True:
                try:
                    bufferblock = self._put_queue.get()
                    if bufferblock is None:
                        return
                    loc = self._keep.put(bufferblock.buffer_view[0:bufferblock.write_pointer].tobytes())
                    bufferblock.set_state(_BufferBlock.COMMITTED, loc)

                except Exception as e:
                    self._put_errors.put((bufferblock.locator(), e))
                finally:
                    if self._put_queue is not None:
                        self._put_queue.task_done()

        with self.lock:
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
                self._put_queue = Queue.Queue(maxsize=2)
                self._put_errors = Queue.Queue()

                self._put_threads = []
                for i in xrange(0, self.num_put_threads):
                    thread = threading.Thread(target=commit_bufferblock_worker, args=(self,))
                    self._put_threads.append(thread)
                    thread.daemon = True
                    thread.start()

        # Mark the block as PENDING so to disallow any more appends.
        block.set_state(_BufferBlock.PENDING)
        self._put_queue.put(block)

    @synchronized
    def get_bufferblock(self, locator):
        return self._bufferblocks.get(locator)

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

        Unlike commit_bufferblock(), this is a synchronous call, and will not
        return until all buffer blocks are uploaded.  Raises
        KeepWriteError() if any blocks failed to upload.

        """
        with self.lock:
            items = self._bufferblocks.items()

        for k,v in items:
            if v.state() == _BufferBlock.WRITABLE:
                self.commit_bufferblock(v)

        with self.lock:
            if self._put_queue is not None:
                self._put_queue.join()

                if not self._put_errors.empty():
                    err = []
                    try:
                        while True:
                            err.append(self._put_errors.get(False))
                    except Queue.Empty:
                        pass
                    raise KeepWriteError("Error writing some blocks", err, label="block")

    def block_prefetch(self, locator):
        """Initiate a background download of a block.

        This assumes that the underlying KeepClient implements a block cache,
        so repeated requests for the same block will not result in repeated
        downloads (unless the block is evicted from the cache.)  This method
        does not block.

        """

        if not self.prefetch_enabled:
            return

        def block_prefetch_worker(self):
            """The background downloader thread."""
            while True:
                try:
                    b = self._prefetch_queue.get()
                    if b is None:
                        return
                    self._keep.get(b)
                except Exception:
                    pass

        with self.lock:
            if locator in self._bufferblocks:
                return
            if self._prefetch_threads is None:
                self._prefetch_queue = Queue.Queue()
                self._prefetch_threads = []
                for i in xrange(0, self.num_get_threads):
                    thread = threading.Thread(target=block_prefetch_worker, args=(self,))
                    self._prefetch_threads.append(thread)
                    thread.daemon = True
                    thread.start()
        self._prefetch_queue.put(locator)


class ArvadosFile(object):
    """Represent a file in a Collection.

    ArvadosFile manages the underlying representation of a file in Keep as a
    sequence of segments spanning a set of blocks, and implements random
    read/write access.

    This object may be accessed from multiple threads.

    """

    def __init__(self, parent, stream=[], segments=[]):
        """
        ArvadosFile constructor.

        :stream:
          a list of Range objects representing a block stream

        :segments:
          a list of Range objects representing segments
        """
        self.parent = parent
        self._modified = True
        self._segments = []
        self.lock = parent.root_collection().lock
        for s in segments:
            self._add_segment(stream, s.locator, s.range_size)
        self._current_bblock = None

    def writable(self):
        return self.parent.writable()

    @synchronized
    def segments(self):
        return copy.copy(self._segments)

    @synchronized
    def clone(self, new_parent):
        """Make a copy of this file."""
        cp = ArvadosFile(new_parent)
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

        self._modified = True

    def __eq__(self, other):
        if other is self:
            return True
        if not isinstance(other, ArvadosFile):
            return False

        othersegs = other.segments()
        with self.lock:
            if len(self._segments) != len(othersegs):
                return False
            for i in xrange(0, len(othersegs)):
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
    def set_unmodified(self):
        """Clear the modified flag"""
        self._modified = False

    @synchronized
    def modified(self):
        """Test the modified flag"""
        return self._modified

    @must_be_writable
    @synchronized
    def truncate(self, size):
        """Shrink the size of the file.

        If `size` is less than the size of the file, the file contents after
        `size` will be discarded.  If `size` is greater than the current size
        of the file, an IOError will be raised.

        """
        if size < self.size():
            new_segs = []
            for r in self._segments:
                range_end = r.range_start+r.range_size
                if r.range_start >= size:
                    # segment is past the trucate size, all done
                    break
                elif size < range_end:
                    nr = Range(r.locator, r.range_start, size - r.range_start)
                    nr.segment_offset = r.segment_offset
                    new_segs.append(nr)
                    break
                else:
                    new_segs.append(r)

            self._segments = new_segs
            self._modified = True
        elif size > self.size():
            raise IOError("truncate() does not support extending the file size")

    def readfrom(self, offset, size, num_retries):
        """Read upto `size` bytes from the file starting at `offset`."""

        with self.lock:
            if size == 0 or offset >= self.size():
                return ''
            prefetch = locators_and_ranges(self._segments, offset, size + config.KEEP_BLOCK_SIZE)
            readsegs = locators_and_ranges(self._segments, offset, size)

        for lr in prefetch:
            self.parent._my_block_manager().block_prefetch(lr.locator)

        data = []
        for lr in readsegs:
            block = self.parent._my_block_manager().get_block_contents(lr.locator, num_retries=num_retries, cache_only=bool(data))
            if block:
                data.append(block[lr.segment_offset:lr.segment_offset+lr.segment_size])
            else:
                break
        return ''.join(data)

    def _repack_writes(self):
        """Test if the buffer block has more data than actual segments.

        This happens when a buffered write over-writes a file range written in
        a previous buffered write.  Re-pack the buffer block for efficiency
        and to avoid leaking information.

        """
        segs = self._segments

        # Sum up the segments to get the total bytes of the file referencing
        # into the buffer block.
        bufferblock_segs = [s for s in segs if s.locator == self._current_bblock.blockid]
        write_total = sum([s.range_size for s in bufferblock_segs])

        if write_total < self._current_bblock.size():
            # There is more data in the buffer block than is actually accounted for by segments, so
            # re-pack into a new buffer by copying over to a new buffer block.
            new_bb = self.parent._my_block_manager().alloc_bufferblock(self._current_bblock.blockid, starting_capacity=write_total, owner=self)
            for t in bufferblock_segs:
                new_bb.append(self._current_bblock.buffer_view[t.segment_offset:t.segment_offset+t.range_size].tobytes())
                t.segment_offset = new_bb.size() - t.range_size

            self._current_bblock = new_bb

    @must_be_writable
    @synchronized
    def writeto(self, offset, data, num_retries):
        """Write `data` to the file starting at `offset`.

        This will update existing bytes and/or extend the size of the file as
        necessary.

        """
        if len(data) == 0:
            return

        if offset > self.size():
            raise ArgumentError("Offset is past the end of the file")

        if len(data) > config.KEEP_BLOCK_SIZE:
            raise ArgumentError("Please append data in chunks smaller than %i bytes (config.KEEP_BLOCK_SIZE)" % (config.KEEP_BLOCK_SIZE))

        self._modified = True

        if self._current_bblock is None or self._current_bblock.state() != _BufferBlock.WRITABLE:
            self._current_bblock = self.parent._my_block_manager().alloc_bufferblock(owner=self)

        if (self._current_bblock.size() + len(data)) > config.KEEP_BLOCK_SIZE:
            self._repack_writes()
            if (self._current_bblock.size() + len(data)) > config.KEEP_BLOCK_SIZE:
                self.parent._my_block_manager().commit_bufferblock(self._current_bblock)
                self._current_bblock = self.parent._my_block_manager().alloc_bufferblock(owner=self)

        self._current_bblock.append(data)

        replace_range(self._segments, offset, len(data), self._current_bblock.blockid, self._current_bblock.write_pointer - len(data))

    @synchronized
    def flush(self):
        if self._current_bblock:
            self._repack_writes()
            self.parent._my_block_manager().commit_bufferblock(self._current_bblock)

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
        self._modified = True
        for lr in locators_and_ranges(blocks, pos, size):
            last = self._segments[-1] if self._segments else Range(0, 0, 0)
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
    def manifest_text(self, stream_name=".", portable_locators=False, normalize=False):
        buf = ""
        filestream = []
        for segment in self.segments:
            loc = segment.locator
            if loc.startswith("bufferblock"):
                loc = self._bufferblocks[loc].calculate_locator()
            if portable_locators:
                loc = KeepLocator(loc).stripped()
            filestream.append(LocatorAndRange(loc, locator_block_size(loc),
                                 segment.segment_offset, segment.range_size))
        buf += ' '.join(normalize_stream(stream_name, {stream_name: filestream}))
        buf += "\n"
        return buf


class ArvadosFileReader(ArvadosFileReaderBase):
    """Wraps ArvadosFile in a file-like object supporting reading only.

    Be aware that this class is NOT thread safe as there is no locking around
    updating file pointer.

    """

    def __init__(self, arvadosfile, name, mode="r", num_retries=None):
        super(ArvadosFileReader, self).__init__(name, mode, num_retries=num_retries)
        self.arvadosfile = arvadosfile

    def size(self):
        return self.arvadosfile.size()

    def stream_name(self):
        return self.arvadosfile.parent.stream_name()

    @_FileLikeObjectBase._before_close
    @retry_method
    def read(self, size, num_retries=None):
        """Read up to `size` bytes from the stream, starting at the current file position."""
        data = self.arvadosfile.readfrom(self._filepos, size, num_retries)
        self._filepos += len(data)
        return data

    @_FileLikeObjectBase._before_close
    @retry_method
    def readfrom(self, offset, size, num_retries=None):
        """Read up to `size` bytes from the stream, starting at the current file position."""
        return self.arvadosfile.readfrom(offset, size, num_retries)

    def flush(self):
        pass


class ArvadosFileWriter(ArvadosFileReader):
    """Wraps ArvadosFile in a file-like object supporting both reading and writing.

    Be aware that this class is NOT thread safe as there is no locking around
    updating file pointer.

    """

    def __init__(self, arvadosfile, name, mode, num_retries=None):
        super(ArvadosFileWriter, self).__init__(arvadosfile, name, mode, num_retries=num_retries)

    @_FileLikeObjectBase._before_close
    @retry_method
    def write(self, data, num_retries=None):
        if self.mode[0] == "a":
            self.arvadosfile.writeto(self.size(), data, num_retries)
        else:
            self.arvadosfile.writeto(self._filepos, data, num_retries)
            self._filepos += len(data)

    @_FileLikeObjectBase._before_close
    @retry_method
    def writelines(self, seq, num_retries=None):
        for s in seq:
            self.write(s, num_retries)

    @_FileLikeObjectBase._before_close
    def truncate(self, size=None):
        if size is None:
            size = self._filepos
        self.arvadosfile.truncate(size)
        if self._filepos > self.size():
            self._filepos = self.size()

    @_FileLikeObjectBase._before_close
    def flush(self):
        self.arvadosfile.flush()

    def close(self):
        if not self.closed:
            self.flush()
            super(ArvadosFileWriter, self).close()
