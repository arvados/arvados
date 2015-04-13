def convertTime(t):
    """Parse Arvados timestamp to unix time."""
    if not t:
        return 0
    try:
        return calendar.timegm(ciso8601.parse_datetime_unaware(t).timetuple())
    except (TypeError, ValueError):
        return 0

class FreshBase(object):
    '''Base class for maintaining fresh/stale state to determine when to update.'''
    def __init__(self):
        self._stale = True
        self._poll = False
        self._last_update = time.time()
        self._atime = time.time()
        self._poll_time = 60

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
