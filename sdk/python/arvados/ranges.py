class Range(object):
    def __init__(self, locator, range_start, range_size, segment_offset=0):
        self.locator = locator
        self.range_start = range_start
        self.range_size = range_size
        self.segment_offset = segment_offset

    def __repr__(self):
        return "Range(\"%s\", %i, %i, %i)" % (self.locator, self.range_start, self.range_size, self.segment_offset)

def first_block(data_locators, range_start, range_size, debug=False):
    block_start = 0L

    # range_start/block_start is the inclusive lower bound
    # range_end/block_end is the exclusive upper bound

    hi = len(data_locators)
    lo = 0
    i = int((hi + lo) / 2)
    block_size = data_locators[i].range_size
    block_start = data_locators[i].range_start
    block_end = block_start + block_size
    if debug: print '---'

    # perform a binary search for the first block
    # assumes that all of the blocks are contigious, so range_start is guaranteed
    # to either fall into the range of a block or be outside the block range entirely
    while not (range_start >= block_start and range_start < block_end):
        if lo == i:
            # must be out of range, fail
            return None
        if range_start > block_start:
            lo = i
        else:
            hi = i
        i = int((hi + lo) / 2)
        if debug: print lo, i, hi
        block_size = data_locators[i].range_size
        block_start = data_locators[i].range_start
        block_end = block_start + block_size

    return i

class LocatorAndRange(object):
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
        return "LocatorAndRange(\"%s\", %i, %i, %i)" % (self.locator, self.block_size, self.segment_offset, self.segment_size)

def locators_and_ranges(data_locators, range_start, range_size, debug=False):
    '''
    Get blocks that are covered by the range
    data_locators: list of Range objects, assumes that blocks are in order and contigous
    range_start: start of range
    range_size: size of range
    returns list of LocatorAndRange objects
    '''
    if range_size == 0:
        return []
    resp = []
    range_start = long(range_start)
    range_size = long(range_size)
    range_end = range_start + range_size

    i = first_block(data_locators, range_start, range_size, debug)
    if i is None:
        return []

    while i < len(data_locators):
        dl = data_locators[i]
        block_start = dl.range_start
        block_size = dl.range_size
        block_end = block_start + block_size
        if debug:
            print dl.locator, "range_start", range_start, "block_start", block_start, "range_end", range_end, "block_end", block_end
        if range_end <= block_start:
            # range ends before this block starts, so don't look at any more locators
            break

        #if range_start >= block_end:
            # range starts after this block ends, so go to next block
            # we should always start at the first block due to the binary above, so this test is redundant
            #next

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

def replace_range(data_locators, new_range_start, new_range_size, new_locator, new_segment_offset, debug=False):
    '''
    Replace a file segment range with a new segment.
    data_locators: list of Range objects, assumes that segments are in order and contigous
    range_start: start of range
    range_size: size of range
    new_locator: locator for new segment to be inserted
    !!! data_locators will be updated in place !!!
    '''
    if new_range_size == 0:
        return

    new_range_start = long(new_range_start)
    new_range_size = long(new_range_size)
    new_range_end = new_range_start + new_range_size

    if len(data_locators) == 0:
        data_locators.append(Range(new_locator, new_range_start, new_range_size, new_segment_offset))
        return

    last = data_locators[-1]
    if (last.range_start+last.range_size) == new_range_start:
        if last.locator == new_locator:
            # extend last segment
            last.range_size += new_range_size
        else:
            data_locators.append(Range(new_locator, new_range_start, new_range_size, new_segment_offset))
        return

    i = first_block(data_locators, new_range_start, new_range_size, debug)
    if i is None:
        return

    while i < len(data_locators):
        dl = data_locators[i]
        old_segment_start = dl.range_start
        old_segment_end = old_segment_start + dl.range_size
        if debug:
            print locator, "range_start", new_range_start, "segment_start", old_segment_start, "range_end", new_range_end, "segment_end", old_segment_end
        if new_range_end <= old_segment_start:
            # range ends before this segment starts, so don't look at any more locators
            break

        #if range_start >= old_segment_end:
            # range starts after this segment ends, so go to next segment
            # we should always start at the first segment due to the binary above, so this test is redundant
            #next

        if  old_segment_start <= new_range_start and new_range_end <= old_segment_end:
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
