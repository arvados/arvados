# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import print_function

import arvados
import Queue
import threading
import _strptime

from crunchstat_summary import logger


class CollectionReader(object):
    def __init__(self, collection_id):
        self._collection_id = collection_id
        self._label = collection_id
        self._reader = None

    def __str__(self):
        return self._label

    def __iter__(self):
        logger.debug('load collection %s', self._collection_id)
        collection = arvados.collection.CollectionReader(self._collection_id)
        filenames = [filename for filename in collection]
        if len(filenames) == 1:
            filename = filenames[0]
        else:
            filename = 'crunchstat.txt'
        self._label = "{}/{}".format(self._collection_id, filename)
        self._reader = collection.open(filename)
        return iter(self._reader)

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        if self._reader:
            self._reader.close()
            self._reader = None


class LiveLogReader(object):
    EOF = None

    def __init__(self, job_uuid):
        self.job_uuid = job_uuid
        self.event_types = (['stderr'] if '-8i9sb-' in job_uuid else ['crunchstat', 'arv-mount'])
        logger.debug('load %s events for job %s', self.event_types, self.job_uuid)

    def __str__(self):
        return self.job_uuid

    def _get_all_pages(self):
        got = 0
        last_id = 0
        filters = [
            ['object_uuid', '=', self.job_uuid],
            ['event_type', 'in', self.event_types]]
        try:
            while True:
                page = arvados.api().logs().index(
                    limit=1000,
                    order=['id asc'],
                    filters=filters + [['id','>',str(last_id)]],
                    select=['id', 'properties'],
                ).execute(num_retries=2)
                got += len(page['items'])
                logger.debug(
                    '%s: received %d of %d log events',
                    self.job_uuid, got,
                    got + page['items_available'] - len(page['items']))
                for i in page['items']:
                    for line in i['properties']['text'].split('\n'):
                        self._queue.put(line+'\n')
                    last_id = i['id']
                if (len(page['items']) == 0 or
                    len(page['items']) >= page['items_available']):
                    break
        finally:
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
            self._thread.join()
            raise StopIteration
        return line

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        pass
