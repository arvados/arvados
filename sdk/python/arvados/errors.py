# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

# errors.py - Arvados-specific exceptions.

import json

from apiclient import errors as apiclient_errors
from collections import OrderedDict

class ApiError(apiclient_errors.HttpError):
    def _get_reason(self):
        try:
            return '; '.join(json.loads(self.content.decode('utf-8'))['errors'])
        except (KeyError, TypeError, ValueError):
            return super(ApiError, self)._get_reason()


class KeepRequestError(Exception):
    """Base class for errors accessing Keep services."""
    def __init__(self, message='', request_errors=(), label=""):
        """KeepRequestError(message='', request_errors=(), label="")

        :message:
          A human-readable message describing what Keep operation
          failed.

        :request_errors:
          An iterable that yields 2-tuples of keys (where the key refers to
          some operation that was attempted) to the error encountered when
          talking to it--either an exception, or an HTTP response object.
          These will be packed into an OrderedDict, available through the
          request_errors() method.

        :label:
          A label indicating the type of value in the 'key' position of request_errors.

        """
        self.label = label
        self._request_errors = OrderedDict(request_errors)
        if self._request_errors:
            exc_reports = [self._format_error(*err_pair)
                           for err_pair in self._request_errors.items()]
            base_msg = "{}: {}".format(message, "; ".join(exc_reports))
        else:
            base_msg = message
        super(KeepRequestError, self).__init__(base_msg)
        self.message = message

    def _format_error(self, key, error):
        if isinstance(error, HttpError):
            err_fmt = "{} {} responded with {e.status_code} {e.reason}"
        else:
            err_fmt = "{} {} raised {e.__class__.__name__} ({e})"
        return err_fmt.format(self.label, key, e=error)

    def request_errors(self):
        """request_errors() -> OrderedDict

        The keys of the dictionary are described by `self.label`
        The corresponding value is the exception raised when sending the
        request to it."""
        return self._request_errors


class HttpError(Exception):
    def __init__(self, status_code, reason):
        self.status_code = status_code
        self.reason = reason


class ArgumentError(Exception):
    pass
class SyntaxError(Exception):
    pass
class AssertionError(Exception):
    pass
class CommandFailedError(Exception):
    pass
class KeepReadError(KeepRequestError):
    pass
class KeepWriteError(KeepRequestError):
    pass
class KeepCacheError(KeepRequestError):
    pass
class NotFoundError(KeepReadError):
    pass
class NotImplementedError(Exception):
    pass
class NoKeepServersError(Exception):
    pass
class StaleWriterStateError(Exception):
    pass
class FeatureNotEnabledError(Exception):
    pass
