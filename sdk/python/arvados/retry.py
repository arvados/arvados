"""Utilities to retry operations.

The core of this module is `RetryLoop`, a utility class to retry operations
that might fail. It can distinguish between temporary and permanent failures;
provide exponential backoff; and save a series of results.

It also provides utility functions for common operations with `RetryLoop`:

* `check_http_response_success` can be used as a `RetryLoop` `success_check`
  for HTTP response codes from the Arvados API server.
* `retry_method` can decorate methods to provide a default `num_retries`
  keyword argument.
"""
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from builtins import range
from builtins import object
import functools
import inspect
import pycurl
import time

from collections import deque

import arvados.errors

_HTTP_SUCCESSES = set(range(200, 300))
_HTTP_CAN_RETRY = set([408, 409, 422, 423, 500, 502, 503, 504])

class RetryLoop(object):
    """Coordinate limited retries of code.

    `RetryLoop` coordinates a loop that runs until it records a
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

    Arguments:

    num_retries: int
    : The maximum number of times to retry the loop if it
      doesn't succeed.  This means the loop body could run at most
      `num_retries + 1` times.

    success_check: Callable
    : This is a function that will be called each
      time the loop saves a result.  The function should return
      `True` if the result indicates the code succeeded, `False` if it
      represents a permanent failure, and `None` if it represents a
      temporary failure.  If no function is provided, the loop will
      end after any result is saved.

    backoff_start: float
    : The number of seconds that must pass before the loop's second
      iteration.  Default 0, which disables all waiting.

    backoff_growth: float
    : The wait time multiplier after each iteration.
      Default 2 (i.e., double the wait time each time).

    save_results: int
    : Specify a number to store that many saved results from the loop.
      These are available through the `results` attribute, oldest first.
      Default 1.

    max_wait: float
    : Maximum number of seconds to wait between retries. Default 60.
    """
    def __init__(self, num_retries, success_check=lambda r: True,
                 backoff_start=0, backoff_growth=2, save_results=1,
                 max_wait=60):
        self.tries_left = num_retries + 1
        self.check_result = success_check
        self.backoff_wait = backoff_start
        self.backoff_growth = backoff_growth
        self.max_wait = max_wait
        self.next_start_time = 0
        self.results = deque(maxlen=save_results)
        self._attempts = 0
        self._running = None
        self._success = None

    def __iter__(self):
        """Return an iterator of retries."""
        return self

    def running(self):
        """Return whether this loop is running.

        Returns `None` if the loop has never run, `True` if it is still running,
        or `False` if it has stopped—whether that's because it has saved a
        successful result, a permanent failure, or has run out of retries.
        """
        return self._running and (self._success is None)

    def __next__(self):
        """Record a loop attempt.

        If the loop is still running, decrements the number of tries left and
        returns it. Otherwise, raises `StopIteration`.
        """
        if self._running is None:
            self._running = True
        if (self.tries_left < 1) or not self.running():
            self._running = False
            raise StopIteration
        else:
            wait_time = max(0, self.next_start_time - time.time())
            time.sleep(wait_time)
            self.backoff_wait *= self.backoff_growth
            if self.backoff_wait > self.max_wait:
                self.backoff_wait = self.max_wait
        self.next_start_time = time.time() + self.backoff_wait
        self.tries_left -= 1
        return self.tries_left

    def save_result(self, result):
        """Record a loop result.

        Save the given result, and end the loop if it indicates
        success or permanent failure. See documentation for the `__init__`
        `success_check` argument to learn how that's indicated.

        Raises `arvados.errors.AssertionError` if called after the loop has
        already ended.

        Arguments:

        result: Any
        : The result from this loop attempt to check and save.
        """
        if not self.running():
            raise arvados.errors.AssertionError(
                "recorded a loop result after the loop finished")
        self.results.append(result)
        self._success = self.check_result(result)
        self._attempts += 1

    def success(self):
        """Return the loop's end state.

        Returns `True` if the loop recorded a successful result, `False` if it
        recorded permanent failure, or else `None`.
        """
        return self._success

    def last_result(self):
        """Return the most recent result the loop saved.

        Raises `arvados.errors.AssertionError` if called before any result has
        been saved.
        """
        try:
            return self.results[-1]
        except IndexError:
            raise arvados.errors.AssertionError(
                "queried loop results before any were recorded")

    def attempts(self):
        """Return the number of results that have been saved.

        This count includes all kinds of results: success, permanent failure,
        and temporary failure.
        """
        return self._attempts

    def attempts_str(self):
        """Return a human-friendly string counting saved results.

        This method returns '1 attempt' or 'N attempts', where the number
        in the string is the number of saved results.
        """
        if self._attempts == 1:
            return '1 attempt'
        else:
            return '{} attempts'.format(self._attempts)


def check_http_response_success(status_code):
    """Convert a numeric HTTP status code to a loop control flag.

    This method takes a numeric HTTP status code and returns `True` if
    the code indicates success, `None` if it indicates temporary
    failure, and `False` otherwise.  You can use this as the
    `success_check` for a `RetryLoop` that queries the Arvados API server.
    Specifically:

    * Any 2xx result returns `True`.

    * A select few status codes, or any malformed responses, return `None`.
      422 Unprocessable Entity is in this category.  This may not meet the
      letter of the HTTP specification, but the Arvados API server will
      use it for various server-side problems like database connection
      errors.

    * Everything else returns `False`.  Note that this includes 1xx and
      3xx status codes.  They don't indicate success, and you can't
      retry those requests verbatim.

    Arguments:

    status_code: int
    : A numeric HTTP response code
    """
    if status_code in _HTTP_SUCCESSES:
        return True
    elif status_code in _HTTP_CAN_RETRY:
        return None
    elif 100 <= status_code < 600:
        return False
    else:
        return None  # Get well soon, server.

def retry_method(orig_func):
    """Provide a default value for a method's num_retries argument.

    This is a decorator for instance and class methods that accept a
    `num_retries` keyword argument, with a `None` default.  When the method
    is called without a value for `num_retries`, this decorator will set it
    from the `num_retries` attribute of the underlying instance or class.

    Arguments:

    orig_func: Callable
    : A class or instance method that accepts a `num_retries` keyword argument
    """
    @functools.wraps(orig_func)
    def num_retries_setter(self, *args, **kwargs):
        if kwargs.get('num_retries') is None:
            kwargs['num_retries'] = self.num_retries
        return orig_func(self, *args, **kwargs)
    return num_retries_setter
