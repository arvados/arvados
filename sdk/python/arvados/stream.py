import bz2
import collections
import hashlib
import os
import re
import zlib

from .arvfile import ArvadosFileBase
from arvados.retry import retry_method
from keep import *
import config
import errors

LOCATOR = 0
BLOCKSIZE = 1
OFFSET = 2
SEGMENTSIZE = 3

def locators_and_ranges(data_locators, range_start, range_size, debug=False):
    '''
    Get blocks that are covered by the range
    data_locators: list of [locator, block_size, block_start], assumes that blocks are in order and contigous
    range_start: start of range
    range_size: size of range
    returns list of [block locator, blocksize, segment offset, segment size] that satisfies the range
    '''
    if range_size == 0:
        return []
    resp = []
    range_start = long(range_start)
    range_size = long(range_size)
    range_end = range_start + range_size
    block_start = 0L

    # range_start/block_start is the inclusive lower bound
    # range_end/block_end is the exclusive upper bound

    hi = len(data_locators)
    lo = 0
    i = int((hi + lo) / 2)
    block_size = data_locators[i][BLOCKSIZE]
    block_start = data_locators[i][OFFSET]
    block_end = block_start + block_size
    if debug: print '---'

    # perform a binary search for the first block
    # assumes that all of the blocks are contigious, so range_start is guaranteed
    # to either fall into the range of a block or be outside the block range entirely
    while not (range_start >= block_start and range_start < block_end):
        if lo == i:
            # must be out of range, fail
            return []
        if range_start > block_start:
            lo = i
        else:
            hi = i
        i = int((hi + lo) / 2)
        if debug: print lo, i, hi
        block_size = data_locators[i][BLOCKSIZE]
        block_start = data_locators[i][OFFSET]
        block_end = block_start + block_size

    while i < len(data_locators):
        locator, block_size, block_start = data_locators[i]
        block_end = block_start + block_size
        if debug:
            print locator, "range_start", range_start, "block_start", block_start, "range_end", range_end, "block_end", block_end
        if range_end <= block_start:
            # range ends before this block starts, so don't look at any more locators
            break

        #if range_start >= block_end:
            # range starts after this block ends, so go to next block
            # we should always start at the first block due to the binary above, so this test is redundant
            #next

        if range_start >= block_start and range_end <= block_end:
            # range starts and ends in this block
            resp.append([locator, block_size, range_start - block_start, range_size])
        elif range_start >= block_start and range_end > block_end:
            # range starts in this block
            resp.append([locator, block_size, range_start - block_start, block_end - range_start])
        elif range_start < block_start and range_end > block_end:
            # range starts in a previous block and extends to further blocks
            resp.append([locator, block_size, 0L, block_size])
        elif range_start < block_start and range_end <= block_end:
            # range starts in a previous block and ends in this block
            resp.append([locator, block_size, 0L, range_end - block_start])
        block_start = block_end
        i += 1
    return resp

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

class StreamFileReader(ArvadosFileBase):
    class _NameAttribute(str):
        # The Python file API provides a plain .name attribute.
        # Older SDK provided a name() method.
        # This class provides both, for maximum compatibility.
        def __call__(self):
            return self


    def __init__(self, stream, segments, name):
        super(StreamFileReader, self).__init__(self._NameAttribute(name), 'rb')
        self._stream = stream
        self.segments = segments
        self._filepos = 0L
        self.num_retries = stream.num_retries
        self._readline_cache = (None, None)

    def __iter__(self):
        while True:
            data = self.readline()
            if not data:
                break
            yield data

    def decompressed_name(self):
        return re.sub('\.(bz2|gz)$', '', self.name)

    def stream_name(self):
        return self._stream.name()

    @ArvadosFileBase._before_close
    def seek(self, pos, whence=os.SEEK_CUR):
        if whence == os.SEEK_CUR:
            pos += self._filepos
        elif whence == os.SEEK_END:
            pos += self.size()
        self._filepos = min(max(pos, 0L), self.size())

    def tell(self):
        return self._filepos

    def size(self):
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
            data = self._stream.readfrom(locator+segmentoffset, segmentsize,
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
            data.append(self._stream.readfrom(locator+segmentoffset, segmentsize,
                                              num_retries=num_retries))
        return ''.join(data)

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

    def as_manifest(self):
        manifest_text = ['.']
        manifest_text.extend([d[LOCATOR] for d in self._stream._data_locators])
        manifest_text.extend(["{}:{}:{}".format(seg[LOCATOR], seg[BLOCKSIZE], self.name().replace(' ', '\\040')) for seg in self.segments])
        return arvados.CollectionReader(' '.join(manifest_text) + '\n').manifest_text(normalize=True)


class StreamReader(object):
    def __init__(self, tokens, keep=None, debug=False, _empty=False,
                 num_retries=0):
        self._stream_name = None
        self._data_locators = []
        self._files = collections.OrderedDict()
        self._keep = keep
        self.num_retries = num_retries

        streamoffset = 0L

        # parse stream
        for tok in tokens:
            if debug: print 'tok', tok
            if self._stream_name is None:
                self._stream_name = tok.replace('\\040', ' ')
                continue

            s = re.match(r'^[0-9a-f]{32}\+(\d+)(\+\S+)*$', tok)
            if s:
                blocksize = long(s.group(1))
                self._data_locators.append([tok, blocksize, streamoffset])
                streamoffset += blocksize
                continue

            s = re.search(r'^(\d+):(\d+):(\S+)', tok)
            if s:
                pos = long(s.group(1))
                size = long(s.group(2))
                name = s.group(3).replace('\\040', ' ')
                if name not in self._files:
                    self._files[name] = StreamFileReader(self, [[pos, size, 0]], name)
                else:
                    n = self._files[name]
                    n.segments.append([pos, size, n.size()])
                continue

            raise errors.SyntaxError("Invalid manifest format")

    def name(self):
        return self._stream_name

    def files(self):
        return self._files

    def all_files(self):
        return self._files.values()

    def size(self):
        n = self._data_locators[-1]
        return n[OFFSET] + n[BLOCKSIZE]

    def locators_and_ranges(self, range_start, range_size):
        return locators_and_ranges(self._data_locators, range_start, range_size)

    @retry_method
    def readfrom(self, start, size, num_retries=None):
        """Read up to 'size' bytes from the stream, starting at 'start'"""
        if size == 0:
            return ''
        if self._keep is None:
            self._keep = KeepClient(num_retries=self.num_retries)
        data = []
        for locator, blocksize, segmentoffset, segmentsize in locators_and_ranges(self._data_locators, start, size):
            data.append(self._keep.get(locator, num_retries=num_retries)[segmentoffset:segmentoffset+segmentsize])
        return ''.join(data)

    def manifest_text(self, strip=False):
        manifest_text = [self.name().replace(' ', '\\040')]
        if strip:
            for d in self._data_locators:
                m = re.match(r'^[0-9a-f]{32}\+\d+', d[LOCATOR])
                manifest_text.append(m.group(0))
        else:
            manifest_text.extend([d[LOCATOR] for d in self._data_locators])
        manifest_text.extend([' '.join(["{}:{}:{}".format(seg[LOCATOR], seg[BLOCKSIZE], f.name().replace(' ', '\\040'))
                                        for seg in f.segments])
                              for f in self._files.values()])
        return ' '.join(manifest_text) + '\n'
