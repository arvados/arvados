import collections
import hashlib
import os
import re
import threading
import functools
import copy

from .ranges import *
from .arvfile import ArvadosFileBase, StreamFileReader
from arvados.retry import retry_method
from keep import *
import config
import errors

def locator_block_size(loc):
    s = re.match(r'[0-9a-f]{32}\+(\d+)(\+\S+)*', loc)
    return long(s.group(1))

def normalize_stream(s, stream):
    '''
    s is the stream name
    stream is a dict mapping each filename to a list in the form [block locator, block size, segment offset (from beginning of block), segment size]
    returns the stream as a list of tokens
    '''
    stream_tokens = [s]
    sortedfiles = list(stream.keys())
    sortedfiles.sort()

    blocks = {}
    streamoffset = 0L
    # Go through each file and add each referenced block exactly once.
    for f in sortedfiles:
        for b in stream[f]:
            if b.locator not in blocks:
                stream_tokens.append(b.locator)
                blocks[b.locator] = streamoffset
                streamoffset += locator_block_size(b.locator)

    # Add the empty block if the stream is otherwise empty.
    if len(stream_tokens) == 1:
        stream_tokens.append(config.EMPTY_BLOCK_LOCATOR)

    for f in sortedfiles:
        # Add in file segments
        current_span = None
        fout = f.replace(' ', '\\040')
        for segment in stream[f]:
            # Collapse adjacent segments
            streamoffset = blocks[segment.locator] + segment.segment_offset
            if current_span is None:
                current_span = [streamoffset, streamoffset + segment.segment_size]
            else:
                if streamoffset == current_span[1]:
                    current_span[1] += segment.segment_size
                else:
                    stream_tokens.append("{0}:{1}:{2}".format(current_span[0], current_span[1] - current_span[0], fout))
                    current_span = [streamoffset, streamoffset + segment.segment_size]

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
                self._data_locators.append(Range(tok, streamoffset, blocksize))
                streamoffset += blocksize
                continue

            s = re.search(r'^(\d+):(\d+):(\S+)', tok)
            if s:
                pos = long(s.group(1))
                size = long(s.group(2))
                name = s.group(3).replace('\\040', ' ')
                if name not in self._files:
                    self._files[name] = StreamFileReader(self, [Range(pos, 0, size)], name)
                else:
                    filereader = self._files[name]
                    filereader.segments.append(Range(pos, filereader.size(), size))
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
        return n.range_start + n.range_size

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
        for lr in locators_and_ranges(self._data_locators, start, size):
            data.append(self._keepget(lr.locator, num_retries=num_retries)[lr.segment_offset:lr.segment_offset+lr.segment_size])
        return ''.join(data)

    def manifest_text(self, strip=False):
        manifest_text = [self.name().replace(' ', '\\040')]
        if strip:
            for d in self._data_locators:
                m = re.match(r'^[0-9a-f]{32}\+\d+', d.locator)
                manifest_text.append(m.group(0))
        else:
            manifest_text.extend([d.locator for d in self._data_locators])
        manifest_text.extend([' '.join(["{}:{}:{}".format(seg.locator, seg.range_size, f.name.replace(' ', '\\040'))
                                        for seg in f.segments])
                              for f in self._files.values()])
        return ' '.join(manifest_text) + '\n'
