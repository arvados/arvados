import logging
import multiprocessing
import multiprocessing.pool
import os
import sys
import threading
try:
    from . import stacktracer
except:
    pass

"""Creating new pool workers (e.g., to run a test) after starting
threads elsewhere in the program (e.g., in llfuse) might cause
deadlock.

(fork() can be called while some mutexes are locked in other threads:
in the child process, those mutexes will never be released.)

To avoid this, we create a static pool of multiprocessing workers up
front, before we create any threads in our main process.  Also, if any
of our pool's workers exit, we don't replace them with new workers.

POOL_SIZE is our guess about how many workers will be enough.  This is
1 + the number of workers that die during the test suite: i.e., if all
tests run normally, POOL_SIZE=1 is enough: we use pool.apply(), which
runs one task at a time.  If we guess too low, the test suite will
fail with a suggestion to increase POOL_SIZE.
"""
POOL_SIZE = 1

logger = logging.getLogger('arvados.arv-mount')
_pool = None


class _Pool(multiprocessing.pool.Pool):
    """A Pool that doesn't replenish its worker pool."""

    def _maintain_pool(self):
        self._join_exited_workers()

    def apply_async(self, *args, **kwargs):
        if not self._pool:
            sys.stderr.write("\n\nmultiprocessing pool is empty! "
                             "increase POOL_SIZE.\n\n")
            os._exit(1)
        return super(_Pool, self).apply_async(*args, **kwargs)


def Pool():
    global _pool
    if not _pool:
        nthreads = threading.active_count()
        if nthreads > 1:
            logger = logging.getLogger('arvados.arv-mount')
            logger.error("threading.active_count() is {} when "
                         "creating multiprocessing.Pool.  Danger ahead!"
                         "".format(nthreads))
        _pool = _Pool(POOL_SIZE)
        try:
            stacktracer.trace_start("/tmp/trace.html", interval=1, auto=True)
            pass
        except:
            pass
    return _pool
