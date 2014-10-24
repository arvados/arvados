# errors.py - Arvados-specific exceptions.

import json
from apiclient import errors as apiclient_errors

class ApiError(apiclient_errors.HttpError):
    def _get_reason(self):
        try:
            return '; '.join(json.loads(self.content)['errors'])
        except (KeyError, TypeError, ValueError):
            return super(ApiError, self)._get_reason()


class ArgumentError(Exception):
    pass
class SyntaxError(Exception):
    pass
class AssertionError(Exception):
    pass
class CommandFailedError(Exception):
    pass
class KeepReadError(Exception):
    pass
class KeepWriteError(Exception):
    pass
class NotFoundError(KeepReadError):
    pass
class NotImplementedError(Exception):
    pass
class NoKeepServersError(Exception):
    pass
class StaleWriterStateError(Exception):
    pass
