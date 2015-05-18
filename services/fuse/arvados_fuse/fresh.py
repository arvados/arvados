import time
import ciso8601
import calendar
import functools

def convertTime(t):
    """Parse Arvados timestamp to unix time."""
    if not t:
        return 0
    try:
        return calendar.timegm(ciso8601.parse_datetime_unaware(t).timetuple())
    except (TypeError, ValueError):
        return 0

def use_counter(orig_func):
    @functools.wraps(orig_func)
    def use_counter_wrapper(self, *args, **kwargs):
        try:
            self.inc_use()
            return orig_func(self, *args, **kwargs)
        finally:
            self.dec_use()
    return use_counter_wrapper

def check_update(orig_func):
    @functools.wraps(orig_func)
    def check_update_wrapper(self, *args, **kwargs):
        self.checkupdate()
        return orig_func(self, *args, **kwargs)
    return check_update_wrapper

class FreshBase(object):
    """Base class for maintaining object lifecycle.

    Functions include:

    * Indicate if an object is up to date (stale() == false) or needs to be
      updated sets stale() == True).  Use invalidate() to mark the object as
      stale.  An object is also automatically stale if it has not been updated
      in `_poll_time` seconds.

    * Record access time (atime) timestamp

    * Manage internal use count used by the inode cache ("inc_use" and
      "dec_use").  An object which is in use cannot be cleared by the inode
      cache.

    * Manage the kernel reference count ("inc_ref" and "dec_ref").  An object
      which is referenced by the kernel cannot have its inode entry deleted.

    * Record cache footprint, cache priority

    * Record Arvados uuid at the time the object is placed in the cache

    * Clear the object contents (invalidates the object)

    """
    def __init__(self):
        self._stale = True
        self._poll = False
        self._last_update = time.time()
        self._atime = time.time()
        self._poll_time = 60
        self.use_count = 0
        self.ref_count = 0
        self.dead = False
        self.cache_priority = None
        self.cache_size = 0
        self.cache_uuid = None

    # Mark the value as stale
    def invalidate(self):
        self._stale = True

    # Test if the entries dict is stale.
    def stale(self):
        if self._stale:
            return True
        if self._poll:
            return (self._last_update + self._poll_time) < self._atime
        return False

    def fresh(self):
        self._stale = False
        self._last_update = time.time()

    def atime(self):
        return self._atime

    def persisted(self):
        return False

    def clear(self, force=False):
        pass

    def in_use(self):
        return self.use_count > 0

    def inc_use(self):
        self.use_count += 1

    def dec_use(self):
        self.use_count -= 1

    def inc_ref(self):
        self.ref_count += 1
        return self.ref_count

    def dec_ref(self, n):
        self.ref_count -= n
        return self.ref_count

    def objsize(self):
        return 0

    def uuid(self):
        return None
