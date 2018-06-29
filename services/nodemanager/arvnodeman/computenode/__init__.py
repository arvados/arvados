#!/usr/bin/env python
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import, print_function

import calendar
import functools
import itertools
import re
import time

from ..config import CLOUD_ERRORS
from ..status import tracker
from libcloud.common.exceptions import BaseHTTPError, RateLimitReachedError

ARVADOS_TIMEFMT = '%Y-%m-%dT%H:%M:%SZ'
ARVADOS_TIMESUBSEC_RE = re.compile(r'(\.\d+)Z$')

def arvados_node_fqdn(arvados_node, default_hostname='dynamic.compute'):
    hostname = arvados_node.get('hostname') or default_hostname
    return '{}.{}'.format(hostname, arvados_node['domain'])

def arvados_node_mtime(node):
    return arvados_timestamp(node['modified_at'])

def arvados_timestamp(timestr):
    subsec_match = ARVADOS_TIMESUBSEC_RE.search(timestr)
    if subsec_match is None:
        subsecs = .0
    else:
        subsecs = float(subsec_match.group(1))
        timestr = timestr[:subsec_match.start()] + 'Z'
    return calendar.timegm(time.strptime(timestr + 'UTC',
                                         ARVADOS_TIMEFMT + '%Z')) + subsecs

def timestamp_fresh(timestamp, fresh_time):
    return (time.time() - timestamp) < fresh_time

def arvados_node_missing(arvados_node, fresh_time):
    """Indicate if cloud node corresponding to the arvados
    node is "missing".

    If True, this means the node has not pinged the API server within the timeout
    period.  If False, the ping is up to date.  If the node has never pinged,
    returns None.
    """
    if arvados_node["last_ping_at"] is None:
        return None
    else:
        return not timestamp_fresh(arvados_timestamp(arvados_node["last_ping_at"]), fresh_time)

class RetryMixin(object):
    """Retry decorator for an method that makes remote requests.

    Use this function to decorate method, and pass in a tuple of exceptions to
    catch.  If the original method raises a known cloud driver error, or any of
    the given exception types, this decorator will either go into a
    sleep-and-retry loop with exponential backoff either by sleeping (if
    self._timer is None) or by scheduling retries of the method (if self._timer
    is a timer actor.)

    """
    def __init__(self, retry_wait, max_retry_wait, logger, cloud, timer=None):
        self.min_retry_wait = max(1, retry_wait)
        self.max_retry_wait = max(self.min_retry_wait, max_retry_wait)
        self.retry_wait = retry_wait
        self._logger = logger
        self._cloud = cloud
        self._timer = timer

    @staticmethod
    def _retry(errors=()):
        def decorator(orig_func):
            @functools.wraps(orig_func)
            def retry_wrapper(self, *args, **kwargs):
                while True:
                    should_retry = False
                    try:
                        ret = orig_func(self, *args, **kwargs)
                    except RateLimitReachedError as error:
                        # If retry-after is zero, continue with exponential
                        # backoff.
                        if error.retry_after != 0:
                            self.retry_wait = error.retry_after
                        should_retry = True
                    except BaseHTTPError as error:
                        if error.headers and error.headers.get("retry-after"):
                            try:
                                retry_after = int(error.headers["retry-after"])
                                # If retry-after is zero, continue with
                                # exponential backoff.
                                if retry_after != 0:
                                    self.retry_wait = retry_after
                                should_retry = True
                            except ValueError:
                                self._logger.warning(
                                    "Unrecognizable Retry-After header: %r",
                                    error.headers["retry-after"],
                                    exc_info=error)
                        if error.code == 429 or error.code >= 500:
                            should_retry = True
                    except CLOUD_ERRORS as error:
                        tracker.counter_add('cloud_errors')
                        should_retry = True
                    except errors as error:
                        should_retry = True
                    except Exception as error:
                        # As a libcloud workaround for drivers that don't use
                        # typed exceptions, consider bare Exception() objects
                        # retryable.
                        if type(error) is Exception:
                            tracker.counter_add('cloud_errors')
                            should_retry = True
                    else:
                        # No exception
                        self.retry_wait = self.min_retry_wait
                        return ret

                    # Only got here if an exception was caught.  Now determine what to do about it.
                    if not should_retry:
                        self.retry_wait = self.min_retry_wait
                        self._logger.warning(
                            "Re-raising error (no retry): %s",
                            error, exc_info=error)
                        raise

                    # Retry wait out of bounds?
                    if self.retry_wait < self.min_retry_wait:
                        self.retry_wait = self.min_retry_wait
                    elif self.retry_wait > self.max_retry_wait:
                        self.retry_wait = self.max_retry_wait

                    self._logger.warning(
                        "Client error: %s - %s %s seconds",
                        error,
                        "scheduling retry in" if self._timer else "sleeping",
                        self.retry_wait,
                        exc_info=error)

                    if self._timer:
                        start_time = time.time()
                        # reschedule to be called again
                        self._timer.schedule(start_time + self.retry_wait,
                                             getattr(self._later,
                                                     orig_func.__name__),
                                             *args, **kwargs)
                    else:
                        # sleep on it.
                        time.sleep(self.retry_wait)

                    self.retry_wait = min(self.retry_wait * 2,
                                          self.max_retry_wait)
                    if self._timer:
                        # expect to be called again by timer so don't loop
                        return

            return retry_wrapper
        return decorator

class ShutdownTimer(object):
    """Keep track of a cloud node's shutdown windows.

    Instantiate this class with a timestamp of when a cloud node started,
    and a list of durations (in minutes) of when the node must not and may
    be shut down, alternating.  The class will tell you when a shutdown
    window is open, and when the next open window will start.
    """
    def __init__(self, start_time, shutdown_windows):
        # The implementation is easiest if we have an even number of windows,
        # because then windows always alternate between open and closed.
        # Rig that up: calculate the first shutdown window based on what's
        # passed in.  Then, if we were given an odd number of windows, merge
        # that first window into the last one, since they both# represent
        # closed state.
        first_window = shutdown_windows[0]
        shutdown_windows = list(shutdown_windows[1:])
        self._next_opening = start_time + (60 * first_window)
        if len(shutdown_windows) % 2:
            shutdown_windows.append(first_window)
        else:
            shutdown_windows[-1] += first_window
        self.shutdown_windows = itertools.cycle([60 * n
                                                 for n in shutdown_windows])
        self._open_start = self._next_opening
        self._open_for = next(self.shutdown_windows)

    def _advance_opening(self):
        while self._next_opening < time.time():
            self._open_start = self._next_opening
            self._next_opening += self._open_for + next(self.shutdown_windows)
            self._open_for = next(self.shutdown_windows)

    def next_opening(self):
        self._advance_opening()
        return self._next_opening

    def window_open(self):
        self._advance_opening()
        return 0 < (time.time() - self._open_start) < self._open_for
