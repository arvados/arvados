from __future__ import print_function

import arvados
import collections
import threading

from crunchstat_summary import logger


class CollectionReader(object):
    def __init__(self, collection_id):
        logger.debug('load collection %s', collection_id)
        collection = arvados.collection.CollectionReader(collection_id)
        filenames = [filename for filename in collection]
        if len(filenames) != 1:
            raise ValueError(
                "collection {} has {} files; need exactly one".format(
                    collection_id, len(filenames)))
        self._reader = collection.open(filenames[0])

    def __iter__(self):
        return iter(self._reader)


class LiveLogReader(object):
    def __init__(self, job_uuid):
        logger.debug('load stderr events for job %s', job_uuid)
        self._filters = [
            ['object_uuid', '=', job_uuid],
            ['event_type', '=', 'stderr']]
        self._buffer = collections.deque()
        self._got = 0
        self._label = job_uuid
        self._last_id = 0
        self._start_getting_next_page()

    def _start_getting_next_page(self):
        self._thread = threading.Thread(target=self._get_next_page)
        self._thread.daemon = True
        self._thread.start()

    def _get_next_page(self):
        page = arvados.api().logs().index(
            limit=1000,
            order=['id asc'],
            filters=self._filters + [['id','>',str(self._last_id)]],
        ).execute()
        self._got += len(page['items'])
        logger.debug(
            '%s: received %d of %d log events',
            self._label, self._got,
            self._got + page['items_available'] - len(page['items']))
        self._page = page

    def _buffer_page(self):
        """Wait for current worker, copy results to _buffer, start next worker.

        Return True if anything was added to the buffer."""
        if self._thread is None:
            return False
        self._thread.join()
        self._thread = None
        page = self._page
        if len(page['items']) == 0:
            return False
        if len(page['items']) < page['items_available']:
            self._start_getting_next_page()
        for i in page['items']:
            for line in i['properties']['text'].split('\n'):
                self._buffer.append(line)
            self._last_id = i['id']
        return True

    def __iter__(self):
        return self

    def next(self):
        if len(self._buffer) == 0:
            if not self._buffer_page():
                raise StopIteration
        return self._buffer.popleft() + '\n'
