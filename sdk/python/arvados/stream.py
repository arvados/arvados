# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import print_function
from __future__ import absolute_import
from future.utils import listvalues
from builtins import object
import collections
import hashlib
import os
import re
import threading
import functools
import copy

from ._ranges import locators_and_ranges, Range
from .arvfile import StreamFileReader
from arvados.retry import retry_method
from arvados.keep import *
from . import config
from . import errors
from ._normalize_stream import normalize_stream

class StreamReader(object):
    def __init__(self, tokens, keep=None, debug=False, _empty=False,
                 num_retries=10):
        self._stream_name = None
        self._data_locators = []
        self._files = collections.OrderedDict()
        self._keep = keep
        self.num_retries = num_retries

        streamoffset = 0

        # parse stream
        for tok in tokens:
            if debug: print('tok', tok)
            if self._stream_name is None:
                self._stream_name = tok.replace('\\040', ' ')
                continue

            s = re.match(r'^[0-9a-f]{32}\+(\d+)(\+\S+)*$', tok)
            if s:
                blocksize = int(s.group(1))
                self._data_locators.append(Range(tok, streamoffset, blocksize, 0))
                streamoffset += blocksize
                continue

            s = re.search(r'^(\d+):(\d+):(\S+)', tok)
            if s:
                pos = int(s.group(1))
                size = int(s.group(2))
                name = s.group(3).replace('\\040', ' ')
                if name not in self._files:
                    self._files[name] = StreamFileReader(self, [Range(pos, 0, size, 0)], name)
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
        return listvalues(self._files)

    def size(self):
        n = self._data_locators[-1]
        return n.range_start + n.range_size

    def locators_and_ranges(self, range_start, range_size):
        return locators_and_ranges(self._data_locators, range_start, range_size)

    @retry_method
    def _keepget(self, locator, num_retries=None):
        return self._keep.get(locator, num_retries=num_retries)

    @retry_method
    def readfrom(self, start, size, num_retries=None):
        """Read up to 'size' bytes from the stream, starting at 'start'"""
        if size == 0:
            return b''
        if self._keep is None:
            self._keep = KeepClient(num_retries=self.num_retries)
        data = []
        for lr in locators_and_ranges(self._data_locators, start, size):
            data.append(self._keepget(lr.locator, num_retries=num_retries)[lr.segment_offset:lr.segment_offset+lr.segment_size])
        return b''.join(data)

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
                              for f in listvalues(self._files)])
        return ' '.join(manifest_text) + '\n'
