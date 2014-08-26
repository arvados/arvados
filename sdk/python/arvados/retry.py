#!/usr/bin/env python

import time

from collections import deque

import arvados.errors

class RetryLoop(object):
    """Coordinate limited retries of code.

    RetryLoop coordinates a loop that runs until it records a
    successful result or tries too many times, whichever comes first.
    Typical use looks like:

        loop = RetryLoop(num_retries=2)
        for tries_left in loop:
            try:
                result = do_something()
            except TemporaryError as error:
                log("error: {} ({} tries left)".format(error, tries_left))
            else:
                loop.save_result(result)
        if loop.success():
            return loop.last_result()
    """
    def __init__(self, num_retries, success_check=lambda r: True,
                 backoff_start=0, backoff_growth=2, save_results=1):
        """Construct a new RetryLoop.

        Arguments:
        * num_retries: The maximum number of times to retry the loop if it
          doesn't succeed.  This means the loop could run at most 1+N times.
        * success_check: This is a function that will be called each
          time the loop saves a result.  The function should return
          True if the result indicates loop success, False if it
          represents a permanent failure state, and None if the loop
          should continue.  If no function is provided, the loop will
          end as soon as it records any result.
        * backoff_start: The number of seconds that must pass before the
          loop's second iteration.  Default 0, which disables all waiting.
        * backoff_growth: The wait time multiplier after each iteration.
          Default 2 (i.e., double the wait time each time).
        * save_results: Specify a number to save the last N results
          that the loop recorded.  These records are available through
          the results attribute, oldest first.  Default 1.
        """
        self.tries_left = num_retries + 1
        self.check_result = success_check
        self.backoff_wait = backoff_start
        self.backoff_growth = backoff_growth
        self.next_start_time = 0
        self.results = deque(maxlen=save_results)
        self._running = None
        self._success = None

    def __iter__(self):
        return self

    def running(self):
        return self._running and (self._success is None)

    def next(self):
        if self._running is None:
            self._running = True
        if (self.tries_left < 1) or not self.running():
            self._running = False
            raise StopIteration
        else:
            wait_time = max(0, self.next_start_time - time.time())
            time.sleep(wait_time)
            self.backoff_wait *= self.backoff_growth
        self.next_start_time = time.time() + self.backoff_wait
        self.tries_left -= 1
        return self.tries_left

    def save_result(self, result):
        """Record a loop result.

        Save the given result, and end the loop if it indicates
        success or permanent failure.  See __init__'s documentation
        about success_check to learn how to make that indication.
        """
        if not self.running():
            raise arvados.errors.AssertionError(
                "recorded a loop result after the loop finished")
        self.results.append(result)
        self._success = self.check_result(result)

    def success(self):
        """Return the loop's end state.

        Returns True if the loop obtained a successful result, False if it
        encountered permanent failure, or else None.
        """
        return self._success

    def last_result(self):
        """Return the most recent result the loop recorded."""
        try:
            return self.results[-1]
        except IndexError:
            raise arvados.errors.AssertionError(
                "queried loop results before any were recorded")
