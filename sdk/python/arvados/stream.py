import collections
import hashlib
import os
import re
import threading
import functools

from .ranges import *
from .arvfile import ArvadosFileBase, StreamFileReader, StreamFileWriter
from arvados.retry import retry_method
from keep import *
import config
import errors

def normalize_stream(s, stream):
    '''
    s is the stream name
    stream is a StreamReader object
    '''
    stream_tokens = [s]
    sortedfiles = list(stream.keys())
    sortedfiles.sort()

    blocks = {}
    streamoffset = 0L
    for f in sortedfiles:
        for b in stream[f]:
            if b[arvados.LOCATOR] not in blocks:
                stream_tokens.append(b[arvados.LOCATOR])
                blocks[b[arvados.LOCATOR]] = streamoffset
                streamoffset += b[arvados.BLOCKSIZE]

    if len(stream_tokens) == 1:
        stream_tokens.append(config.EMPTY_BLOCK_LOCATOR)

    for f in sortedfiles:
        current_span = None
        fout = f.replace(' ', '\\040')
        for segment in stream[f]:
            segmentoffset = blocks[segment[arvados.LOCATOR]] + segment[arvados.OFFSET]
            if current_span is None:
                current_span = [segmentoffset, segmentoffset + segment[arvados.SEGMENTSIZE]]
            else:
                if segmentoffset == current_span[1]:
                    current_span[1] += segment[arvados.SEGMENTSIZE]
                else:
                    stream_tokens.append("{0}:{1}:{2}".format(current_span[0], current_span[1] - current_span[0], fout))
                    current_span = [segmentoffset, segmentoffset + segment[arvados.SEGMENTSIZE]]

        if current_span is not None:
            stream_tokens.append("{0}:{1}:{2}".format(current_span[0], current_span[1] - current_span[0], fout))

        if not stream[f]:
            stream_tokens.append("0:0:{0}".format(fout))

    return stream_tokens


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
                    filereader = self._files[name]
                    filereader.segments.append([pos, size, filereader.size()])
                continue

            raise errors.SyntaxError("Invalid manifest format")

    def name(self):
        return self._stream_name

    def files(self):
        return self._files

    def all_files(self):
        return self._files.values()

    def _size(self):
        n = self._data_locators[-1]
        return n[OFFSET] + n[BLOCKSIZE]

    def size(self):
        return self._size()

    def locators_and_ranges(self, range_start, range_size):
        return locators_and_ranges(self._data_locators, range_start, range_size)

    @retry_method
    def _keepget(self, locator, num_retries=None):
        return self._keep.get(locator, num_retries=num_retries)

    @retry_method
    def readfrom(self, start, size, num_retries=None):
        return self._readfrom(start, size, num_retries=num_retries)

    @retry_method
    def _readfrom(self, start, size, num_retries=None):
        """Read up to 'size' bytes from the stream, starting at 'start'"""
        if size == 0:
            return ''
        if self._keep is None:
            self._keep = KeepClient(num_retries=self.num_retries)
        data = []
        for locator, blocksize, segmentoffset, segmentsize in locators_and_ranges(self._data_locators, start, size):
            data.append(self._keepget(locator, num_retries=num_retries)[segmentoffset:segmentoffset+segmentsize])
        return ''.join(data)

    def manifest_text(self, strip=False):
        manifest_text = [self.name().replace(' ', '\\040')]
        if strip:
            for d in self._data_locators:
                m = re.match(r'^[0-9a-f]{32}\+\d+', d[LOCATOR])
                manifest_text.append(m.group(0))
        else:
            manifest_text.extend([d[LOCATOR] for d in self._data_locators])
        manifest_text.extend([' '.join(["{}:{}:{}".format(seg[LOCATOR], seg[BLOCKSIZE], f.name.replace(' ', '\\040'))
                                        for seg in f.segments])
                              for f in self._files.values()])
        return ' '.join(manifest_text) + '\n'


class BufferBlock(object):
    def __init__(self, locator, streamoffset):
        self.locator = locator
        self.buffer_block = bytearray(config.KEEP_BLOCK_SIZE)
        self.buffer_view = memoryview(self.buffer_block)
        self.write_pointer = 0
        self.locator_list_entry = [locator, 0, streamoffset]

    def append(self, data):
        self.buffer_view[self.write_pointer:self.write_pointer+len(data)] = data
        self.write_pointer += len(data)
        self.locator_list_entry[1] = self.write_pointer


class StreamWriter(StreamReader):
    def __init__(self, tokens, keep=None, debug=False, _empty=False,
                 num_retries=0):
        super(StreamWriter, self).__init__(tokens, keep, debug, _empty, num_retries)

        if len(self._files) != 1:
            raise AssertionError("StreamWriter can only have one file at a time")
        sr = self._files.popitem()[1]
        self._files[sr.name] = StreamFileWriter(self, sr.segments, sr.name)

        self.mutex = threading.Lock()
        self.current_bblock = None
        self.bufferblocks = {}

    # wrap superclass methods in mutex
    def _proxy_method(name):
        method = getattr(StreamReader, name)
        @functools.wraps(method, ('__name__', '__doc__'))
        def wrapper(self, *args, **kwargs):
            with self.mutex:
                return method(self, *args, **kwargs)
        return wrapper

    for _method_name in ['files', 'all_files', 'size', 'locators_and_ranges', 'readfrom', 'manifest_text']:
        locals()[_method_name] = _proxy_method(_method_name)

    @retry_method
    def _keepget(self, locator, num_retries=None):
        if locator in self.bufferblocks:
            bb = self.bufferblocks[locator]
            return str(bb.buffer_block[0:bb.write_pointer])
        else:
            return self._keep.get(locator, num_retries=num_retries)

    def _append(self, data):
        if self.current_bblock is None:
            last = self._data_locators[-1]
            streamoffset = last[OFFSET] + last[BLOCKSIZE]
            self.current_bblock = BufferBlock("bufferblock%i" % len(self.bufferblocks), streamoffset)
            self.bufferblocks[self.current_bblock.locator] = self.current_bblock
            self._data_locators.append(self.current_bblock.locator_list_entry)
        self.current_bblock.append(data)

    def append(self, data):
        with self.mutex:
            self._append(data)
