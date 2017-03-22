import errno
import md5
import os
import tempfile

class SafeHTTPCache(object):
    def __init__(self, path=None):
        self._dir = path

    def __str__(self):
        return self._dir

    def _filename(self, url):
        return os.path.join(self._dir, md5.new(url).hexdigest()+'.tmp')

    def get(self, url):
        filename = self._filename(url)
        try:
            with open(filename, 'rb') as f:
                return f.read()
        except IOError, OSError:
            return None

    def set(self, url, content):
        try:
            fd, tempname = tempfile.mkstemp(dir=self._dir)
        except:
            return None
        try:
            try:
                f = os.fdopen(fd, 'w')
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
