from __future__ import print_function

import arvados
import Queue
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
    EOF = None

    def __init__(self, job_uuid):
        logger.debug('load stderr events for job %s', job_uuid)
        self._filters = [
            ['object_uuid', '=', job_uuid],
            ['event_type', '=', 'stderr']]
        self._label = job_uuid

    def _get_all_pages(self):
        got = 0
        last_id = 0
        while True:
            page = arvados.api().logs().index(
                limit=1000,
                order=['id asc'],
                filters=self._filters + [['id','>',str(last_id)]],
            ).execute(num_retries=2)
            got += len(page['items'])
            logger.debug(
                '%s: received %d of %d log events',
                self._label, got,
                got + page['items_available'] - len(page['items']))
            for i in page['items']:
                for line in i['properties']['text'].split('\n'):
                    self._queue.put(line+'\n')
                last_id = i['id']
            if (len(page['items']) == 0 or
                len(page['items']) >= page['items_available']):
                break
        self._queue.put(self.EOF)

    def __iter__(self):
        self._queue = Queue.Queue()
        self._thread = threading.Thread(target=self._get_all_pages)
        self._thread.daemon = True
        self._thread.start()
        return self

    def next(self):
        line = self._queue.get()
        if line is self.EOF:
            raise StopIteration
        return line
