# errors.py - Arvados-specific exceptions.

import json
import requests

from apiclient import errors as apiclient_errors
from collections import OrderedDict

class ApiError(apiclient_errors.HttpError):
    def _get_reason(self):
        try:
            return '; '.join(json.loads(self.content)['errors'])
        except (KeyError, TypeError, ValueError):
            return super(ApiError, self)._get_reason()


class KeepRequestError(Exception):
    """Base class for errors accessing Keep services."""
    def __init__(self, message='', service_errors=()):
        """KeepRequestError(message='', service_errors=())

        Arguments:
        * message: A human-readable message describing what Keep operation
          failed.
        * service_errors: An iterable that yields 2-tuples of Keep
          service URLs to the error encountered when talking to
          it--either an exception, or an HTTP response object.  These
          will be packed into an OrderedDict, available through the
          service_errors() method.
        """
        self._service_errors = OrderedDict(service_errors)
        if self._service_errors:
            exc_reports = [self._format_error(*err_pair)
                           for err_pair in self._service_errors.iteritems()]
            base_msg = "{}: {}".format(message, "; ".join(exc_reports))
        else:
            base_msg = message
        super(KeepRequestError, self).__init__(base_msg)
        self.message = message

    def _format_error(self, service_root, error):
        if isinstance(error, requests.Response):
            err_fmt = "{} responded with {e.status_code} {e.reason}"
        else:
            err_fmt = "{} raised {e.__class__.__name__} ({e})"
        return err_fmt.format(service_root, e=error)

    def service_errors(self):
        """service_errors() -> OrderedDict

        The keys of the dictionary are Keep service URLs.
        The corresponding value is the exception raised when sending the
        request to it."""
        return self._service_errors


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
