# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import absolute_import
from . import config

import re

def escape(path):
    path = re.sub('\\\\', lambda m: '\\134', path)
    path = re.sub('[:\000-\040]', lambda m: "\\%03o" % ord(m.group(0)), path)
    return path

def normalize_stream(stream_name, stream):
    """Take manifest stream and return a list of tokens in normalized format.

    :stream_name:
      The name of the stream.

    :stream:
      A dict mapping each filename to a list of `_range.LocatorAndRange` objects.

    """

    stream_name = escape(stream_name)
    stream_tokens = [stream_name]
    sortedfiles = list(stream.keys())
    sortedfiles.sort()

    blocks = {}
    streamoffset = 0
    # Go through each file and add each referenced block exactly once.
    for streamfile in sortedfiles:
        for segment in stream[streamfile]:
            if segment.locator not in blocks:
                stream_tokens.append(segment.locator)
                blocks[segment.locator] = streamoffset
                streamoffset += segment.block_size

    # Add the empty block if the stream is otherwise empty.
    if len(stream_tokens) == 1:
        stream_tokens.append(config.EMPTY_BLOCK_LOCATOR)

    for streamfile in sortedfiles:
        # Add in file segments
        current_span = None
        fout = escape(streamfile)
        for segment in stream[streamfile]:
            # Collapse adjacent segments
            streamoffset = blocks[segment.locator] + segment.segment_offset
            if current_span is None:
                current_span = [streamoffset, streamoffset + segment.segment_size]
            else:
                if streamoffset == current_span[1]:
                    current_span[1] += segment.segment_size
                else:
                    stream_tokens.append(u"{0}:{1}:{2}".format(current_span[0], current_span[1] - current_span[0], fout))
                    current_span = [streamoffset, streamoffset + segment.segment_size]

        if current_span is not None:
            stream_tokens.append(u"{0}:{1}:{2}".format(current_span[0], current_span[1] - current_span[0], fout))

        if not stream[streamfile]:
            stream_tokens.append(u"0:0:{0}".format(fout))

    return stream_tokens
