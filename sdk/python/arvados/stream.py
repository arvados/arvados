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

class StreamFileReader(object):
    def __init__(self, stream, pos, size, name):
        self._stream = stream
        self._pos = pos
        self._size = size
        self._name = name
        self._filepos = 0

    def name(self):
        return self._name

    def decompressed_name(self):
        return re.sub('\.(bz2|gz)$', '', self._name)

    def size(self):
        return self._size

    def stream_name(self):
        return self._stream.name()

    def read(self, size, **kwargs):
        self._stream.seek(self._pos + self._filepos)
        data = self._stream.read(min(size, self._size - self._filepos))
        self._filepos += len(data)
        return data

    def readall(self, size=2**20, **kwargs):
        while True:
            data = self.read(size, **kwargs)
            if data == '':
                break
            yield data

    def seek(self, pos):
        self._filepos = pos

    def bunzip2(self, size):
        decompressor = bz2.BZ2Decompressor()
        for chunk in self.readall(size):
            data = decompressor.decompress(chunk)
            if data and data != '':
                yield data

    def gunzip(self, size):
        decompressor = zlib.decompressobj(16+zlib.MAX_WBITS)
        for chunk in self.readall(size):
            data = decompressor.decompress(decompressor.unconsumed_tail + chunk)
            if data and data != '':
                yield data

    def readall_decompressed(self, size=2**20):
        self._stream.seek(self._pos + self._filepos)
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

    def as_manifest(self):
        if self.size() == 0:
            return ("%s %s 0:0:%s\n"
                    % (self._stream.name(), config.EMPTY_BLOCK_LOCATOR, self.name()))
        return string.join(self._stream.tokens_for_range(self._pos, self._size),
                           " ") + "\n"

class StreamReader(object):
    def __init__(self, tokens):
        self._tokens = tokens
        self._current_datablock_data = None
        self._current_datablock_pos = 0
        self._current_datablock_index = -1
        self._pos = 0

        self._stream_name = None
        self.data_locators = []
        self.files = []

        for tok in self._tokens:
            if self._stream_name == None:
                self._stream_name = tok.replace('\\040', ' ')
            elif re.search(r'^[0-9a-f]{32}(\+\S+)*$', tok):
                self.data_locators += [tok]
            elif re.search(r'^\d+:\d+:\S+', tok):
                pos, size, name = tok.split(':',2)
                self.files += [[int(pos), int(size), name.replace('\\040', ' ')]]
            else:
                raise errors.SyntaxError("Invalid manifest format")

    def tokens(self):
        return self._tokens

    def tokens_for_range(self, range_start, range_size):
        resp = [self._stream_name]
        return_all_tokens = False
        block_start = 0
        token_bytes_skipped = 0
        for locator in self.data_locators:
            sizehint = re.search(r'\+(\d+)', locator)
            if not sizehint:
                return_all_tokens = True
            if return_all_tokens:
                resp += [locator]
                next
            blocksize = int(sizehint.group(0))
            if range_start + range_size <= block_start:
                break
            if range_start < block_start + blocksize:
                resp += [locator]
            else:
                token_bytes_skipped += blocksize
            block_start += blocksize
        for f in self.files:
            if ((f[0] < range_start + range_size)
                and
                (f[0] + f[1] > range_start)
                and
                f[1] > 0):
                resp += ["%d:%d:%s" % (f[0] - token_bytes_skipped, f[1], f[2])]
        return resp

    def name(self):
        return self._stream_name

    def all_files(self):
        for f in self.files:
            pos, size, name = f
            yield StreamFileReader(self, pos, size, name)

    def nextdatablock(self):
        if self._current_datablock_index < 0:
            self._current_datablock_pos = 0
            self._current_datablock_index = 0
        else:
            self._current_datablock_pos += self.current_datablock_size()
            self._current_datablock_index += 1
        self._current_datablock_data = None

    def current_datablock_data(self):
        if self._current_datablock_data == None:
            self._current_datablock_data = Keep.get(self.data_locators[self._current_datablock_index])
        return self._current_datablock_data

    def current_datablock_size(self):
        if self._current_datablock_index < 0:
            self.nextdatablock()
        sizehint = re.search('\+(\d+)', self.data_locators[self._current_datablock_index])
        if sizehint:
            return int(sizehint.group(0))
        return len(self.current_datablock_data())

    def seek(self, pos):
        """Set the position of the next read operation."""
        self._pos = pos

    def really_seek(self):
        """Find and load the appropriate data block, so the byte at
        _pos is in memory.
        """
        if self._pos == self._current_datablock_pos:
            return True
        if (self._current_datablock_pos != None and
            self._pos >= self._current_datablock_pos and
            self._pos <= self._current_datablock_pos + self.current_datablock_size()):
            return True
        if self._pos < self._current_datablock_pos:
            self._current_datablock_index = -1
            self.nextdatablock()
        while (self._pos > self._current_datablock_pos and
               self._pos > self._current_datablock_pos + self.current_datablock_size()):
            self.nextdatablock()

    def read(self, size):
        """Read no more than size bytes -- but at least one byte,
        unless _pos is already at the end of the stream.
        """
        if size == 0:
            return ''
        self.really_seek()
        while self._pos >= self._current_datablock_pos + self.current_datablock_size():
            self.nextdatablock()
            if self._current_datablock_index >= len(self.data_locators):
                return None
        data = self.current_datablock_data()[self._pos - self._current_datablock_pos : self._pos - self._current_datablock_pos + size]
        self._pos += len(data)
        return data
