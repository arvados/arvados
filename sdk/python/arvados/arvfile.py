import functools
import os
import zlib
import bz2
from .ranges import *
from arvados.retry import retry_method

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
        self.need_lock = False
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
        self._filepos = min(max(pos, 0L), self._size())

    def tell(self):
        return self._filepos

    def size(self):
        return self._size()

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
        super(StreamFileReader, self).__init__(name, 'rb')
        self._stream = stream
        self.segments = segments
        self.num_retries = stream.num_retries
        self._filepos = 0L
        self.num_retries = stream.num_retries
        self._readline_cache = (None, None)

    def stream_name(self):
        return self._stream.name()

    def _size(self):
        n = self.segments[-1]
        return n[OFFSET] + n[BLOCKSIZE]

    @ArvadosFileBase._before_close
    @retry_method
    def read(self, size, num_retries=None):
        """Read up to 'size' bytes from the stream, starting at the current file position"""
        if size == 0:
            return ''

        data = ''
        available_chunks = locators_and_ranges(self.segments, self._filepos, size)
        if available_chunks:
            locator, blocksize, segmentoffset, segmentsize = available_chunks[0]
            data = self._stream._readfrom(locator+segmentoffset, segmentsize,
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
        for locator, blocksize, segmentoffset, segmentsize in locators_and_ranges(self.segments, start, size):
            data.append(self._stream._readfrom(locator+segmentoffset, segmentsize,
                                              num_retries=num_retries))
        return ''.join(data)

    def as_manifest(self):
        manifest_text = ['.']
        manifest_text.extend([d[LOCATOR] for d in self._stream._data_locators])
        manifest_text.extend(["{}:{}:{}".format(seg[LOCATOR], seg[BLOCKSIZE], self.name().replace(' ', '\\040')) for seg in self.segments])
        return CollectionReader(' '.join(manifest_text) + '\n').manifest_text(normalize=True)


class ArvadosFile(ArvadosFileReaderBase):
    def __init__(self, name, mode, stream, segments):
        super(ArvadosFile, self).__init__(name, mode)
        self.segments = []

    def truncate(self, size=None):
        if size is None:
            size = self._filepos

        segs = locators_and_ranges(self.segments, 0, size)

        newstream = []
        self.segments = []
        streamoffset = 0L
        fileoffset = 0L

        for seg in segs:
            for locator, blocksize, segmentoffset, segmentsize in locators_and_ranges(self._stream._data_locators, seg[LOCATOR]+seg[OFFSET], seg[SEGMENTSIZE]):
                newstream.append([locator, blocksize, streamoffset])
                self.segments.append([streamoffset+segmentoffset, segmentsize, fileoffset])
                streamoffset += blocksize
                fileoffset += segmentsize
        if len(newstream) == 0:
            newstream.append(config.EMPTY_BLOCK_LOCATOR)
            self.segments.append([0, 0, 0])
        self._stream._data_locators = newstream
        if self._filepos > fileoffset:
            self._filepos = fileoffset

    def _writeto(self, offset, data):
        if offset > self._size():
            raise ArgumentError("Offset is past the end of the file")
        self._stream._append(data)
        replace_range(self.segments, self._filepos, len(data), self._stream._size()-len(data))

    def writeto(self, offset, data):
        self._writeto(offset, data)

    def write(self, data):
        self._writeto(self._filepos, data)
        self._filepos += len(data)

    def writelines(self, seq):
        for s in seq:
            self._writeto(self._filepos, s)
            self._filepos += len(s)

    def flush(self):
        pass

    def add_segment(self, blocks, pos, size):
        for locator, blocksize, segmentoffset, segmentsize in locators_and_ranges(blocks, pos, size):
            last = self.segments[-1] if self.segments else [0, 0, 0]
            self.segments.append([locator, segmentsize, last[OFFSET]+last[BLOCKSIZE], segmentoffset])
