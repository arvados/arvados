import functools
import os
import zlib
import bz2
from .ranges import *
from arvados.retry import retry_method
import config
import hashlib
import hashlib
import threading
import Queue

def split(path):
    """split(path) -> streamname, filename

    Separate the stream name and file name in a /-separated stream path.
    If no stream name is available, assume '.'.
    """
    try:
        stream_name, file_name = path.rsplit('/', 1)
    except ValueError:  # No / in string
        stream_name, file_name = '.', path
    return stream_name, file_name

class ArvadosFileBase(object):
    def __init__(self, name, mode):
        self.name = name
        self.mode = mode
        self.closed = False

    @staticmethod
    def _before_close(orig_func):
        @functools.wraps(orig_func)
        def wrapper(self, *args, **kwargs):
            if self.closed:
                raise ValueError("I/O operation on closed stream file")
            return orig_func(self, *args, **kwargs)
        return wrapper

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


class ArvadosFileReaderBase(ArvadosFileBase):
    class _NameAttribute(str):
        # The Python file API provides a plain .name attribute.
        # Older SDK provided a name() method.
        # This class provides both, for maximum compatibility.
        def __call__(self):
            return self

    def __init__(self, name, mode, num_retries=None):
        super(ArvadosFileReaderBase, self).__init__(self._NameAttribute(name), mode)
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

    @ArvadosFileBase._before_close
    def seek(self, pos, whence=os.SEEK_CUR):
        if whence == os.SEEK_CUR:
            pos += self._filepos
        elif whence == os.SEEK_END:
            pos += self.size()
        self._filepos = min(max(pos, 0L), self.size())

    def tell(self):
        return self._filepos

    @ArvadosFileBase._before_close
    @retry_method
    def readall(self, size=2**20, num_retries=None):
        while True:
            data = self.read(size, num_retries=num_retries)
            if data == '':
                break
            yield data

    @ArvadosFileBase._before_close
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

    @ArvadosFileBase._before_close
    @retry_method
    def decompress(self, decompress, size, num_retries=None):
        for segment in self.readall(size, num_retries):
            data = decompress(segment)
            if data:
                yield data

    @ArvadosFileBase._before_close
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

    @ArvadosFileBase._before_close
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
    def __init__(self, stream, segments, name):
        super(StreamFileReader, self).__init__(name, 'rb', num_retries=stream.num_retries)
        self._stream = stream
        self.segments = segments

    def stream_name(self):
        return self._stream.name()

    def size(self):
        n = self.segments[-1]
        return n.range_start + n.range_size

    @ArvadosFileBase._before_close
    @retry_method
    def read(self, size, num_retries=None):
        """Read up to 'size' bytes from the stream, starting at the current file position"""
        if size == 0:
            return ''

        data = ''
        available_chunks = locators_and_ranges(self.segments, self._filepos, size)
        if available_chunks:
            lr = available_chunks[0]
            data = self._stream._readfrom(lr.locator+lr.segment_offset,
                                          lr.segment_size,
                                          num_retries=num_retries)

        self._filepos += len(data)
        return data

    @ArvadosFileBase._before_close
    @retry_method
    def readfrom(self, start, size, num_retries=None):
        """Read up to 'size' bytes from the stream, starting at 'start'"""
        if size == 0:
            return ''

        data = []
        for lr in locators_and_ranges(self.segments, start, size):
            data.append(self._stream._readfrom(lr.locator+lr.segment_offset, lr.segment_size,
                                              num_retries=num_retries))
        return ''.join(data)

    def as_manifest(self):
        from stream import normalize_stream
        segs = []
        for r in self.segments:
            segs.extend(self._stream.locators_and_ranges(r.locator, r.range_size))
        return " ".join(normalize_stream(".", {self.name: segs})) + "\n"


class BufferBlock(object):
'''
A BufferBlock is a stand-in for a Keep block that is in the process of being
written.  Writers can append to it, get the size, and compute the Keep locator.

There are three valid states:

WRITABLE - can append

PENDING - is in the process of being uploaded to Keep, append is an error

COMMITTED - the block has been written to Keep, its internal buffer has been
released, and the BufferBlock should be discarded in favor of fetching the
block through normal Keep means.
'''
    WRITABLE = 0
    PENDING = 1
    COMMITTED = 2

    def __init__(self, blockid, starting_capacity):
        '''
        blockid: the identifier for this block
        starting_capacity: the initial buffer capacity
        '''
        self.blockid = blockid
        self.buffer_block = bytearray(starting_capacity)
        self.buffer_view = memoryview(self.buffer_block)
        self.write_pointer = 0
        self.state = BufferBlock.WRITABLE
        self._locator = None

    def append(self, data):
        '''
        Append some data to the buffer.  Only valid if the block is in WRITABLE
        state.  Implements an expanding buffer, doubling capacity as needed to
        accomdate all the data.
        '''
        if self.state == BufferBlock.WRITABLE:
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

    def size(self):
        '''Amount of data written to the buffer'''
        return self.write_pointer

    def locator(self):
        '''The Keep locator for this buffer's contents.'''
        if self._locator is None:
            self._locator = "%s+%i" % (hashlib.md5(self.buffer_view[0:self.write_pointer]).hexdigest(), self.size())
        return self._locator


class AsyncKeepWriteErrors(Exception):
    '''
    Roll up one or more Keep write exceptions (generated by background
    threads) into a single one.
    '''
    def __init__(self, errors):
        self.errors = errors

    def __repr__(self):
        return "\n".join(self.errors)

class BlockManager(object):
    '''
    BlockManager handles buffer blocks, background block uploads, and
    background block prefetch for a Collection of ArvadosFiles.
    '''
    def __init__(self, keep):
        '''keep: KeepClient object to use'''
        self._keep = keep
        self._bufferblocks = {}
        self._put_queue = None
        self._put_errors = None
        self._put_threads = None
        self._prefetch_queue = None
        self._prefetch_threads = None

    def alloc_bufferblock(self, blockid=None, starting_capacity=2**14):
        '''
        Allocate a new, empty bufferblock in WRITABLE state and return it.
        blockid: optional block identifier, otherwise one will be automatically assigned
        starting_capacity: optional capacity, otherwise will use default capacity
        '''
        if blockid is None:
            blockid = "bufferblock%i" % len(self._bufferblocks)
        bb = BufferBlock(blockid, starting_capacity=starting_capacity)
        self._bufferblocks[bb.blockid] = bb
        return bb

    def stop_threads(self):
        '''
        Shut down and wait for background upload and download threads to finish.
        '''
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
        '''
        Initiate a background upload of a bufferblock.  This will block if the
        upload queue is at capacity, otherwise it will return immediately.
        '''

        def worker(self):
            '''
            Background uploader thread.
            '''
            while True:
                try:
                    b = self._put_queue.get()
                    if b is None:
                        return
                    b._locator = self._keep.put(b.buffer_view[0:b.write_pointer].tobytes())
                    b.state = BufferBlock.COMMITTED
                    b.buffer_view = None
                    b.buffer_block = None
                except Exception as e:
                    print e
                    self._put_errors.put(e)
                finally:
                    if self._put_queue is not None:
                        self._put_queue.task_done()

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
            self._put_threads = [threading.Thread(target=worker, args=(self,)),
                                 threading.Thread(target=worker, args=(self,))]
            for t in self._put_threads:
                t.daemon = True
                t.start()

        # Mark the block as PENDING so to disallow any more appends.
        block.state = BufferBlock.PENDING
        self._put_queue.put(block)

    def get_block(self, locator, num_retries, cache_only=False):
        '''
        Fetch a block.  First checks to see if the locator is a BufferBlock and
        return that, if not, passes the request through to KeepClient.get().
        '''
        if locator in self._bufferblocks:
            bb = self._bufferblocks[locator]
            if bb.state != BufferBlock.COMMITTED:
                return bb.buffer_view[0:bb.write_pointer].tobytes()
            else:
                locator = bb._locator
        return self._keep.get(locator, num_retries=num_retries, cache_only=cache_only)

    def commit_all(self):
        '''
        Commit all outstanding buffer blocks.  Unlike commit_bufferblock(), this
        is a synchronous call, and will not return until all buffer blocks are
        uploaded.  Raises AsyncKeepWriteErrors() if any blocks failed to
        upload.
        '''
        for k,v in self._bufferblocks.items():
            if v.state == BufferBlock.WRITABLE:
                self.commit_bufferblock(v)
        if self._put_queue is not None:
            self._put_queue.join()
            if not self._put_errors.empty():
                e = []
                try:
                    while True:
                        e.append(self._put_errors.get(False))
                except Queue.Empty:
                    pass
                raise AsyncKeepWriteErrors(e)

    def block_prefetch(self, locator):
        '''
        Initiate a background download of a block.  This assumes that the
        underlying KeepClient implements a block cache, so repeated requests
        for the same block will not result in repeated downloads (unless the
        block is evicted from the cache.)  This method does not block.
        '''
        def worker(self):
            '''Background downloader thread.'''
            while True:
                try:
                    b = self._prefetch_queue.get()
                    if b is None:
                        return
                    self._keep.get(b)
                except:
                    pass

        if locator in self._bufferblocks:
            return
        if self._prefetch_threads is None:
            self._prefetch_queue = Queue.Queue()
            self._prefetch_threads = [threading.Thread(target=worker, args=(self,)),
                                      threading.Thread(target=worker, args=(self,))]
            for t in self._prefetch_threads:
                t.daemon = True
                t.start()
        self._prefetch_queue.put(locator)


class ArvadosFile(object):
    '''
    ArvadosFile manages the underlying representation of a file in Keep as a sequence of
    segments spanning a set of blocks, and implements random read/write access.
    '''

    def __init__(self, parent, stream=[], segments=[]):
        '''
        stream: a list of Range objects representing a block stream
        segments: a list of Range objects representing segments
        '''
        self.parent = parent
        self._modified = True
        self.segments = []
        for s in segments:
            self.add_segment(stream, s.locator, s.range_size)
        self._current_bblock = None
        self.lock = threading.Lock()

    def clone(self):
        '''Make a copy of this file.'''
        # TODO: copy bufferblocks?
        with self.lock:
            cp = ArvadosFile()
            cp.parent = self.parent
            cp._modified = False
            cp.segments = [Range(r.locator, r.range_start, r.range_size, r.segment_offset) for r in self.segments]
            return cp

    def set_unmodified(self):
        '''Clear the modified flag'''
        self._modified = False

    def modified(self):
        '''Test the modified flag'''
        return self._modified

    def truncate(self, size):
        '''
        Adjust the size of the file.  If "size" is less than the size of the file,
        the file contents after "size" will be discarded.  If "size" is greater
        than the current size of the file, an IOError will be raised.
        '''
        if size < self.size():
            new_segs = []
            for r in self.segments:
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

            self.segments = new_segs
            self._modified = True
        elif size > self.size():
            raise IOError("truncate() does not support extending the file size")


    def readfrom(self, offset, size, num_retries):
        '''
        read upto "size" bytes from the file starting at "offset".
        '''
        if size == 0 or offset >= self.size():
            return ''
        data = []

        for lr in locators_and_ranges(self.segments, offset, size + config.KEEP_BLOCK_SIZE):
            self.parent._my_block_manager().block_prefetch(lr.locator)

        for lr in locators_and_ranges(self.segments, offset, size):
            d = self.parent._my_block_manager().get_block(lr.locator, num_retries=num_retries, cache_only=bool(data))
            if d:
                data.append(d[lr.segment_offset:lr.segment_offset+lr.segment_size])
            else:
                break
        return ''.join(data)

    def _repack_writes(self):
        '''
        Test if the buffer block has more data than is referenced by actual segments
        (this happens when a buffered write over-writes a file range written in
        a previous buffered write).  Re-pack the buffer block for efficiency
        and to avoid leaking information.
        '''
        segs = self.segments

        # Sum up the segments to get the total bytes of the file referencing
        # into the buffer block.
        bufferblock_segs = [s for s in segs if s.locator == self._current_bblock.blockid]
        write_total = sum([s.range_size for s in bufferblock_segs])

        if write_total < self._current_bblock.size():
            # There is more data in the buffer block than is actually accounted for by segments, so
            # re-pack into a new buffer by copying over to a new buffer block.
            new_bb = self.parent._my_block_manager().alloc_bufferblock(self._current_bblock.blockid, starting_size=write_total)
            for t in bufferblock_segs:
                new_bb.append(self._current_bblock.buffer_view[t.segment_offset:t.segment_offset+t.range_size].tobytes())
                t.segment_offset = new_bb.size() - t.range_size

            self._current_bblock = new_bb

    def writeto(self, offset, data, num_retries):
        '''
        Write "data" to the file starting at "offset".  This will update
        existing bytes and/or extend the size of the file as necessary.
        '''
        if len(data) == 0:
            return

        if offset > self.size():
            raise ArgumentError("Offset is past the end of the file")

        if len(data) > config.KEEP_BLOCK_SIZE:
            raise ArgumentError("Please append data in chunks smaller than %i bytes (config.KEEP_BLOCK_SIZE)" % (config.KEEP_BLOCK_SIZE))

        self._modified = True

        if self._current_bblock is None or self._current_bblock.state != BufferBlock.WRITABLE:
            self._current_bblock = self.parent._my_block_manager().alloc_bufferblock()

        if (self._current_bblock.size() + len(data)) > config.KEEP_BLOCK_SIZE:
            self._repack_writes()
            if (self._current_bblock.size() + len(data)) > config.KEEP_BLOCK_SIZE:
                self.parent._my_block_manager().commit_bufferblock(self._current_bblock)
                self._current_bblock = self.parent._my_block_manager().alloc_bufferblock()

        self._current_bblock.append(data)
        replace_range(self.segments, offset, len(data), self._current_bblock.blockid, self._current_bblock.write_pointer - len(data))

    def add_segment(self, blocks, pos, size):
        '''
        Add a segment to the end of the file, with "pos" and "offset" referencing a
        section of the stream described by "blocks" (a list of Range objects)
        '''
        self._modified = True
        for lr in locators_and_ranges(blocks, pos, size):
            last = self.segments[-1] if self.segments else Range(0, 0, 0)
            r = Range(lr.locator, last.range_start+last.range_size, lr.segment_size, lr.segment_offset)
            self.segments.append(r)

    def size(self):
        '''Get the file size'''
        if self.segments:
            n = self.segments[-1]
            return n.range_start + n.range_size
        else:
            return 0


class ArvadosFileReader(ArvadosFileReaderBase):
    def __init__(self, arvadosfile, name, mode="r", num_retries=None):
        super(ArvadosFileReader, self).__init__(name, mode, num_retries=num_retries)
        self.arvadosfile = arvadosfile.clone()

    def size(self):
        return self.arvadosfile.size()

    @ArvadosFileBase._before_close
    @retry_method
    def read(self, size, num_retries=None):
        """Read up to 'size' bytes from the stream, starting at the current file position"""
        data = self.arvadosfile.readfrom(self._filepos, size, num_retries=num_retries)
        self._filepos += len(data)
        return data

    @ArvadosFileBase._before_close
    @retry_method
    def readfrom(self, offset, size, num_retries=None):
        """Read up to 'size' bytes from the stream, starting at the current file position"""
        return self.arvadosfile.readfrom(offset, size, num_retries)

    def flush(self):
        pass


class SynchronizedArvadosFile(object):
    def __init__(self, arvadosfile):
        self.arvadosfile = arvadosfile

    def clone(self):
        return self

    def __getattr__(self, name):
        with self.arvadosfile.lock:
            return getattr(self.arvadosfile, name)


class ArvadosFileWriter(ArvadosFileReader):
    def __init__(self, arvadosfile, name, mode, num_retries=None):
        self.arvadosfile = SynchronizedArvadosFile(arvadosfile)
        super(ArvadosFileWriter, self).__init__(self.arvadosfile, name, mode, num_retries=num_retries)

    @ArvadosFileBase._before_close
    @retry_method
    def write(self, data, num_retries=None):
        if self.mode[0] == "a":
            self.arvadosfile.writeto(self.size(), data)
        else:
            self.arvadosfile.writeto(self._filepos, data, num_retries)
            self._filepos += len(data)

    @ArvadosFileBase._before_close
    @retry_method
    def writelines(self, seq, num_retries=None):
        for s in seq:
            self.write(s)

    def truncate(self, size=None):
        if size is None:
            size = self._filepos
        self.arvadosfile.truncate(size)
        if self._filepos > self.size():
            self._filepos = self.size()
