import gflags
import httplib
import httplib2
import logging
import os
import pprint
import sys
import types
import subprocess
import json
import UserDict
import re
import hashlib
import string
import bz2
import zlib
import fcntl
import time
import threading

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

    hi = len(data_locators)
    lo = 0
    i = int((hi + lo) / 2)
    block_size = data_locators[i][BLOCKSIZE]
    block_start = data_locators[i][OFFSET]
    block_end = block_start + block_size
    if debug: print '---'
    while not (range_start >= block_start and range_start <= block_end):
        if lo == i:
            break
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
        if range_start >= block_end:
            # range starts after this block ends, so go to next block
            next
        elif range_start >= block_start and range_end <= block_end:
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


class StreamFileReader(object):
    def __init__(self, stream, segments, name):
        self._stream = stream
        self.segments = segments
        self._name = name
        self._filepos = 0L

    def name(self):
        return self._name

    def decompressed_name(self):
        return re.sub('\.(bz2|gz)$', '', self._name)

    def stream_name(self):
        return self._stream.name()

    def seek(self, pos):
        self._filepos = min(max(pos, 0L), self.size())

    def tell(self, pos):
        return self._filepos

    def size(self):
        n = self.segments[-1]
        return n[OFFSET] + n[BLOCKSIZE]

    def read(self, size):
        """Read up to 'size' bytes from the stream, starting at the current file position"""
        if size == 0:
            return ''

        data = ''
        for locator, blocksize, segmentoffset, segmentsize in locators_and_ranges(self.segments, self._filepos, size):
            self._stream.seek(locator+segmentoffset)
            data += self._stream.read(segmentsize)
            self._filepos += len(data)
        return data

    def readfrom(self, start, size):
        """Read up to 'size' bytes from the stream, starting at 'start'"""
        if size == 0:
            return ''

        data = []
        for locator, blocksize, segmentoffset, segmentsize in locators_and_ranges(self.segments, start, size):
            data += self._stream.readfrom(locator+segmentoffset, segmentsize)
        return data.join()

    def readall(self, size=2**20):
        while True:
            data = self.read(size)
            if data == '':
                break
            yield data

    def bunzip2(self, size):
        decompressor = bz2.BZ2Decompressor()
        for segment in self.readall(size):
            data = decompressor.decompress(segment)
            if data and data != '':
                yield data

    def gunzip(self, size):
        decompressor = zlib.decompressobj(16+zlib.MAX_WBITS)
        for segment in self.readall(size):
            data = decompressor.decompress(decompressor.unconsumed_tail + segment)
            if data and data != '':
                yield data

    def readall_decompressed(self, size=2**20):
        self.seek(0)
        if re.search('\.bz2$', self._name):
            return self.bunzip2(size)
        elif re.search('\.gz$', self._name):
            return self.gunzip(size)
        else:
            return self.readall(size)

    def readlines(self, decompress=True):
        if decompress:
            datasource = self.readall_decompressed()
        else:
            self._stream.seek(self._pos + self._filepos)
            datasource = self.readall()
        data = ''
        for newdata in datasource:
            data += newdata
            sol = 0
            while True:
                eol = string.find(data, "\n", sol)
                if eol < 0:
                    break
                yield data[sol:eol+1]
                sol = eol+1
            data = data[sol:]
        if data != '':
            yield data


class StreamReader(object):
    def __init__(self, tokens):
        self._tokens = tokens
        self._pos = 0L

        self._stream_name = None
        self.data_locators = []
        self.files = {}

        streamoffset = 0L

        for tok in self._tokens:
            if self._stream_name == None:
                self._stream_name = tok.replace('\\040', ' ')
                continue

            s = re.match(r'^[0-9a-f]{32}\+(\d+)(\+\S+)*$', tok)
            if s:
                blocksize = long(s.group(1))
                self.data_locators.append([tok, blocksize, streamoffset])
                streamoffset += blocksize
                continue

            s = re.search(r'^(\d+):(\d+):(\S+)', tok)
            if s:
                pos = long(s.group(1))
                size = long(s.group(2))
                name = s.group(3).replace('\\040', ' ')
                if name not in self.files:
                    self.files[name] = StreamFileReader(self, [[pos, size, 0]], name)
                else:
                    n = self.files[name]
                    n.segments.append([pos, size, n.size()])
                continue

            raise errors.SyntaxError("Invalid manifest format")
            
    def tokens(self):
        return self._tokens

    def name(self):
        return self._stream_name

    def all_files(self):
        return self.files.values()

    def seek(self, pos):
        """Set the position of the next read operation."""
        self._pos = pos

    def tell(self):
        return self._pos

    def size(self):
        n = self.data_locators[-1]
        return n[self.OFFSET] + n[self.BLOCKSIZE]

    def locators_and_ranges(self, range_start, range_size):
        return locators_and_ranges(self.data_locators, range_start, range_size)

    def read(self, size):
        """Read up to 'size' bytes from the stream, starting at the current file position"""
        if size == 0:
            return ''
        data = ''
        for locator, blocksize, segmentoffset, segmentsize in locators_and_ranges(self.data_locators, self._pos, size):
            data += Keep.get(locator)[segmentoffset:segmentoffset+segmentsize]
        self._pos += len(data)
        return data

    def readfrom(self, start, size):
        """Read up to 'size' bytes from the stream, starting at 'start'"""
        if size == 0:
            return ''
        data = ''
        for locator, blocksize, segmentoffset, segmentsize in locators_and_ranges(self.data_locators, start, size):
            data += Keep.get(locator)[segmentoffset:segmentoffset+segmentsize]
        return data
