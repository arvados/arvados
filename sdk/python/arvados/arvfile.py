import functools
import os
import zlib
import bz2
from .ranges import *
from arvados.retry import retry_method
import config
import hashlib

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
    def __init__(self, locator, starting_size=2**14):
        self.locator = locator
        self.buffer_block = bytearray(starting_size)
        self.buffer_view = memoryview(self.buffer_block)
        self.write_pointer = 0

    def append(self, data):
        while (self.write_pointer+len(data)) > len(self.buffer_block):
            new_buffer_block = bytearray(len(self.buffer_block) * 2)
            new_buffer_block[0:self.write_pointer] = self.buffer_block[0:self.write_pointer]
            self.buffer_block = new_buffer_block
            self.buffer_view = memoryview(self.buffer_block)
        self.buffer_view[self.write_pointer:self.write_pointer+len(data)] = data
        self.write_pointer += len(data)

    def size(self):
        return self.write_pointer

    def calculate_locator(self):
        return "%s+%i" % (hashlib.md5(self.buffer_view[0:self.write_pointer]).hexdigest(), self.size())


class ArvadosFile(object):
    def __init__(self, stream=[], segments=[], keep=None):
        '''
        stream: a list of Range objects representing a block stream
        segments: a list of Range objects representing segments
        '''
        self._modified = True
        self._segments = []
        for s in segments:
            self.add_segment(stream, s.range_start, s.range_size)
        self._current_bblock = None
        self._bufferblocks = None
        self._keep = keep

    def set_unmodified(self):
        self._modified = False

    def modified(self):
        return self._modified

    def truncate(self, size):
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

    def _keepget(self, locator, num_retries):
        if self._bufferblocks and locator in self._bufferblocks:
            bb = self._bufferblocks[locator]
            return bb.buffer_view[0:bb.write_pointer].tobytes()
        else:
            return self._keep.get(locator, num_retries=num_retries)

    def readfrom(self, offset, size, num_retries):
        if size == 0 or offset >= self.size():
            return ''
        if self._keep is None:
            self._keep = KeepClient(num_retries=num_retries)
        data = []
        # TODO: initiate prefetch on all blocks in the range (offset, offset + size + config.KEEP_BLOCK_SIZE)

        for lr in locators_and_ranges(self._segments, offset, size):
            # TODO: if data is empty, wait on block get, otherwise only
            # get more data if the block is already in the cache.
            data.append(self._keepget(lr.locator, num_retries=num_retries)[lr.segment_offset:lr.segment_offset+lr.segment_size])
        return ''.join(data)

    def _init_bufferblock(self):
        if self._bufferblocks is None:
            self._bufferblocks = {}
        self._current_bblock = BufferBlock("bufferblock%i" % len(self._bufferblocks))
        self._bufferblocks[self._current_bblock.locator] = self._current_bblock

    def _repack_writes(self):
        pass
         # TODO: fixme
        '''Test if the buffer block has more data than is referenced by actual segments
        (this happens when a buffered write over-writes a file range written in
        a previous buffered write).  Re-pack the buffer block for efficiency
        and to avoid leaking information.
        '''
        segs = self._segments

        # Sum up the segments to get the total bytes of the file referencing
        # into the buffer block.
        bufferblock_segs = [s for s in segs if s.locator == self._current_bblock.locator]
        write_total = sum([s.range_size for s in bufferblock_segs])

        if write_total < self._current_bblock.size():
            # There is more data in the buffer block than is actually accounted for by segments, so
            # re-pack into a new buffer by copying over to a new buffer block.
            new_bb = BufferBlock(self._current_bblock.locator, starting_size=write_total)
            for t in bufferblock_segs:
                new_bb.append(self._current_bblock.buffer_view[t.segment_offset:t.segment_offset+t.range_size].tobytes())
                t.segment_offset = new_bb.size() - t.range_size

            self._current_bblock = new_bb
            self._bufferblocks[self._current_bblock.locator] = self._current_bblock


    def writeto(self, offset, data, num_retries):
        if len(data) == 0:
            return

        if offset > self.size():
            raise ArgumentError("Offset is past the end of the file")

        if len(data) > config.KEEP_BLOCK_SIZE:
            raise ArgumentError("Please append data in chunks smaller than %i bytes (config.KEEP_BLOCK_SIZE)" % (config.KEEP_BLOCK_SIZE))

        self._modified = True

        if self._current_bblock is None:
            self._init_bufferblock()

        if (self._current_bblock.write_pointer + len(data)) > config.KEEP_BLOCK_SIZE:
            self._repack_writes()
            if (self._current_bblock.write_pointer + len(data)) > config.KEEP_BLOCK_SIZE:
                self._init_bufferblock()

        self._current_bblock.append(data)
        replace_range(self._segments, offset, len(data), self._current_bblock.locator, self._current_bblock.write_pointer - len(data))

    def add_segment(self, blocks, pos, size):
        self._modified = True
        for lr in locators_and_ranges(blocks, pos, size):
            last = self._segments[-1] if self._segments else Range(0, 0, 0)
            r = Range(lr.locator, last.range_start+last.range_size, lr.segment_size, lr.segment_offset)
            self._segments.append(r)

    def size(self):
        if self._segments:
            n = self._segments[-1]
            return n.range_start + n.range_size
        else:
            return 0


class ArvadosFileReader(ArvadosFileReaderBase):
    def __init__(self, arvadosfile, name, mode="r", num_retries=None):
        super(ArvadosFileReader, self).__init__(name, mode, num_retries=num_retries)
        self.arvadosfile = arvadosfile

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

class ArvadosFileWriter(ArvadosFileReader):
    def __init__(self, arvadosfile, name, mode, num_retries=None):
        super(ArvadosFileWriter, self).__init__(arvadosfile, name, mode, num_retries=num_retries)

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
