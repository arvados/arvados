# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from builtins import object
import errno
import hashlib
import os
import tempfile
import time

class SafeHTTPCache(object):
    """Thread-safe replacement for httplib2.FileCache"""

    def __init__(self, path=None, max_age=None):
        self._dir = path
        if max_age is not None:
            try:
                self._clean(threshold=time.time() - max_age)
            except:
                pass

    def _clean(self, threshold=0):
        for ent in os.listdir(self._dir):
            fnm = os.path.join(self._dir, ent)
            if os.path.isdir(fnm) or not fnm.endswith('.tmp'):
                continue
            stat = os.lstat(fnm)
            if stat.st_mtime < threshold:
                try:
                    os.unlink(fnm)
                except OSError as err:
                    if err.errno != errno.ENOENT:
                        raise

    def __str__(self):
        return self._dir

    def _filename(self, url):
        return os.path.join(self._dir, hashlib.md5(url.encode('utf-8')).hexdigest()+'.tmp')

    def get(self, url):
        filename = self._filename(url)
        try:
            with open(filename, 'rb') as f:
                return f.read()
        except (IOError, OSError):
            return None

    def set(self, url, content):
        try:
            fd, tempname = tempfile.mkstemp(dir=self._dir)
        except:
            return None
        try:
            try:
                f = os.fdopen(fd, 'wb')
            except:
                os.close(fd)
                raise
            try:
                f.write(content)
            finally:
                f.close()
            os.rename(tempname, self._filename(url))
            tempname = None
        finally:
            if tempname:
                os.unlink(tempname)

    def delete(self, url):
        try:
            os.unlink(self._filename(url))
        except OSError as err:
            if err.errno != errno.ENOENT:
                raise
