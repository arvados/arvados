# errors.py - Arvados-specific exceptions.

class SyntaxError(Exception):
    pass
class AssertionError(Exception):
    pass
class NotFoundError(Exception):
    pass
class CommandFailedError(Exception):
    pass
class KeepWriteError(Exception):
    pass
class NotImplementedError(Exception):
    pass
