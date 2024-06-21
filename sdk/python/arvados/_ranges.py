# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import logging

_logger = logging.getLogger('arvados.ranges')

# Log level below 'debug' !
RANGES_SPAM = 9

class Range(object):
    __slots__ = ("locator", "range_start", "range_size", "segment_offset")

    def __init__(self, locator, range_start, range_size, segment_offset=0):
        self.locator = locator
        self.range_start = range_start
        self.range_size = range_size
        self.segment_offset = segment_offset

    def __repr__(self):
        return "Range(%r, %r, %r, %r)" % (self.locator, self.range_start, self.range_size, self.segment_offset)

    def __eq__(self, other):
        return (self.locator == other.locator and
                self.range_start == other.range_start and
                self.range_size == other.range_size and
                self.segment_offset == other.segment_offset)

def first_block(data_locators, range_start):
    block_start = 0

    # range_start/block_start is the inclusive lower bound
    # range_end/block_end is the exclusive upper bound

    hi = len(data_locators)
    lo = 0
    i = (hi + lo) // 2
    block_size = data_locators[i].range_size
    block_start = data_locators[i].range_start
    block_end = block_start + block_size

    # perform a binary search for the first block
    # assumes that all of the blocks are contiguous, so range_start is guaranteed
    # to either fall into the range of a block or be outside the block range entirely
    while not (range_start >= block_start and range_start < block_end):
        if lo == i:
            # must be out of range, fail
            return None
        if range_start > block_start:
            lo = i
        else:
            hi = i
        i = (hi + lo) // 2
        block_size = data_locators[i].range_size
        block_start = data_locators[i].range_start
        block_end = block_start + block_size

    return i

class LocatorAndRange(object):
    __slots__ = ("locator", "block_size", "segment_offset", "segment_size")

    def __init__(self, locator, block_size, segment_offset, segment_size):
        self.locator = locator
        self.block_size = block_size
        self.segment_offset = segment_offset
        self.segment_size = segment_size

    def __eq__(self, other):
        return  (self.locator == other.locator and
                 self.block_size == other.block_size and
                 self.segment_offset == other.segment_offset and
                 self.segment_size == other.segment_size)

    def __repr__(self):
        return "LocatorAndRange(%r, %r, %r, %r)" % (self.locator, self.block_size, self.segment_offset, self.segment_size)

def locators_and_ranges(data_locators, range_start, range_size, limit=None):
    """Get blocks that are covered by a range.

    Returns a list of LocatorAndRange objects.

    :data_locators:
      list of Range objects, assumes that blocks are in order and contiguous

    :range_start:
      start of range

    :range_size:
      size of range

    :limit:
      Maximum segments to return, default None (unlimited).  Will truncate the
      result if there are more segments needed to cover the range than the
      limit.

    """
    if range_size == 0:
        return []
    resp = []
    range_end = range_start + range_size

    i = first_block(data_locators, range_start)
    if i is None:
        return []

    # We should always start at the first segment due to the binary
    # search.
    while i < len(data_locators) and len(resp) != limit:
        dl = data_locators[i]
        block_start = dl.range_start
        block_size = dl.range_size
        block_end = block_start + block_size
        _logger.log(RANGES_SPAM,
            "L&R %s range_start %s block_start %s range_end %s block_end %s",
            dl.locator, range_start, block_start, range_end, block_end)
        if range_end <= block_start:
            # range ends before this block starts, so don't look at any more locators
            break

        if range_start >= block_start and range_end <= block_end:
            # range starts and ends in this block
            resp.append(LocatorAndRange(dl.locator, block_size, dl.segment_offset + (range_start - block_start), range_size))
        elif range_start >= block_start and range_end > block_end:
            # range starts in this block
            resp.append(LocatorAndRange(dl.locator, block_size, dl.segment_offset + (range_start - block_start), block_end - range_start))
        elif range_start < block_start and range_end > block_end:
            # range starts in a previous block and extends to further blocks
            resp.append(LocatorAndRange(dl.locator, block_size, dl.segment_offset, block_size))
        elif range_start < block_start and range_end <= block_end:
            # range starts in a previous block and ends in this block
            resp.append(LocatorAndRange(dl.locator, block_size, dl.segment_offset, range_end - block_start))
        block_start = block_end
        i += 1
    return resp

def replace_range(data_locators, new_range_start, new_range_size, new_locator, new_segment_offset):
    """
    Replace a file segment range with a new segment.

    NOTE::
    data_locators will be updated in place

    :data_locators:
      list of Range objects, assumes that segments are in order and contiguous

    :new_range_start:
      start of range to replace in data_locators

    :new_range_size:
      size of range to replace in data_locators

    :new_locator:
      locator for new segment to be inserted

    :new_segment_offset:
      segment offset within the locator

    """
    if new_range_size == 0:
        return

    new_range_end = new_range_start + new_range_size

    if len(data_locators) == 0:
        data_locators.append(Range(new_locator, new_range_start, new_range_size, new_segment_offset))
        return

    last = data_locators[-1]
    if (last.range_start+last.range_size) == new_range_start:
        if last.locator == new_locator and (last.segment_offset+last.range_size) == new_segment_offset:
            # extend last segment
            last.range_size += new_range_size
        else:
            data_locators.append(Range(new_locator, new_range_start, new_range_size, new_segment_offset))
        return

    i = first_block(data_locators, new_range_start)
    if i is None:
        return

    # We should always start at the first segment due to the binary
    # search.
    while i < len(data_locators):
        dl = data_locators[i]
        old_segment_start = dl.range_start
        old_segment_end = old_segment_start + dl.range_size
        _logger.log(RANGES_SPAM,
            "RR %s range_start %s segment_start %s range_end %s segment_end %s",
            dl, new_range_start, old_segment_start, new_range_end,
            old_segment_end)
        if new_range_end <= old_segment_start:
            # range ends before this segment starts, so don't look at any more locators
            break

        if old_segment_start <= new_range_start and new_range_end <= old_segment_end:
            # new range starts and ends in old segment
            # split segment into up to 3 pieces
            if (new_range_start-old_segment_start) > 0:
                data_locators[i] = Range(dl.locator, old_segment_start, (new_range_start-old_segment_start), dl.segment_offset)
                data_locators.insert(i+1, Range(new_locator, new_range_start, new_range_size, new_segment_offset))
            else:
                data_locators[i] = Range(new_locator, new_range_start, new_range_size, new_segment_offset)
                i -= 1
            if (old_segment_end-new_range_end) > 0:
                data_locators.insert(i+2, Range(dl.locator, new_range_end, (old_segment_end-new_range_end), dl.segment_offset + (new_range_start-old_segment_start) + new_range_size))
            return
        elif old_segment_start <= new_range_start and new_range_end > old_segment_end:
            # range starts in this segment
            # split segment into 2 pieces
            data_locators[i] = Range(dl.locator, old_segment_start, (new_range_start-old_segment_start), dl.segment_offset)
            data_locators.insert(i+1, Range(new_locator, new_range_start, new_range_size, new_segment_offset))
            i += 1
        elif new_range_start < old_segment_start and new_range_end >= old_segment_end:
            # range starts in a previous segment and extends to further segments
            # delete this segment
            del data_locators[i]
            i -= 1
        elif new_range_start < old_segment_start and new_range_end < old_segment_end:
            # range starts in a previous segment and ends in this segment
            # move the starting point of this segment up, and shrink it.
            data_locators[i] = Range(dl.locator, new_range_end, (old_segment_end-new_range_end), dl.segment_offset + (new_range_end-old_segment_start))
            return
        i += 1
