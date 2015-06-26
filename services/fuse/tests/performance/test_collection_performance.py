import arvados
import arvados.safeapi
import arvados_fuse as fuse
import glob
import json
import llfuse
import os
import shutil
import subprocess
import sys
import tempfile
import threading
import time
import unittest
import logging
import multiprocessing
from .. import run_test_server
from ..mount_test_base import MountTestBase

logger = logging.getLogger('arvados.arv-mount')

from performance_profiler import profiled

@profiled
def fuseCreateCollectionWithManyFiles(mounttmp, streams=1, files_per_stream=1, blocks_per_file=1, bytes_per_block=1, data='x'):
    class Test(unittest.TestCase):
        def runTest(self):
            names = 'file0.txt'
            for i in range(1, files_per_stream):
                names += ',file' + str(i) + '.txt'
            file_names = names.split(',')

            for i in range(0, streams):
                with self.assertRaises(IOError):
                    with open(os.path.join(mounttmp, "./stream", "file0.txt"), "w") as f:
                        f.write(data)

                os.mkdir(os.path.join(mounttmp, "./stream" + str(i)))

                with self.assertRaises(OSError):
                    os.mkdir(os.path.join(mounttmp, "./stream" + str(i)))

                # Create files
                for j in range(0, files_per_stream):
                    with open(os.path.join(mounttmp, "./stream" + str(i), "file" + str(j) +".txt"), "w") as f:
                        f.write(data)

                d1 = llfuse.listdir(os.path.join(mounttmp, "./stream" + str(i)))
                self.assertEqual(sorted(file_names), sorted(d1))

    Test().runTest()

@profiled
def fuseReadContentsFromCollectionWithManyFiles(mounttmp, streams, files_per_stream, content):
    class Test(unittest.TestCase):
        def runTest(self):
            for i in range(0, streams):
                d1 = llfuse.listdir(os.path.join(mounttmp, 'stream'+str(i)))
                for j in range(0, files_per_stream):
                    with open(os.path.join(mounttmp, 'stream'+str(i), 'file'+str(i)+'.txt')) as f:
                        self.assertEqual(content, f.read())

    Test().runTest()

@profiled
def fuseMoveFileFromCollectionWithManyFiles(mounttmp, stream, filename):
    class Test(unittest.TestCase):
        def runTest(self):
            d1 = llfuse.listdir(os.path.join(mounttmp, stream))
            self.assertIn(filename, d1)

            os.rename(os.path.join(mounttmp, stream, filename), os.path.join(mounttmp, 'moved-from-'+stream+'-'+filename))

            d1 = llfuse.listdir(os.path.join(mounttmp))
            self.assertIn('moved-from-'+stream+'-'+filename, d1)

            d1 = llfuse.listdir(os.path.join(mounttmp, stream))
            self.assertNotIn(filename, d1)

    Test().runTest()

@profiled
def fuseDeleteFileFromCollectionWithManyFiles(mounttmp, stream, filename):
    class Test(unittest.TestCase):
        def runTest(self):
            d1 = llfuse.listdir(os.path.join(mounttmp, stream))

            # Delete file
            os.remove(os.path.join(mounttmp, stream, filename))

            # Try to delete it again
            with self.assertRaises(OSError):
                os.remove(os.path.join(mounttmp, "testdir", "file1.txt"))

    Test().runTest()

# Create a collection with two streams, each with 200 files
class CreateCollectionWithManyFilesAndRenameMoveAndDeleteFile(MountTestBase):
    def runTest(self):
        collection = arvados.collection.Collection(api_client=self.api)
        collection.save_new()

        m = self.make_mount(fuse.CollectionDirectory)
        with llfuse.lock:
            m.new_collection(collection.api_response(), collection)
        self.assertTrue(m.writable())

        streams = 2
        files_per_stream = 200
        blocks_per_file = 1
        bytes_per_block = 1

        data = 'x' * blocks_per_file * bytes_per_block

        self.pool.apply(fuseCreateCollectionWithManyFiles, (self.mounttmp, streams, files_per_stream, blocks_per_file, bytes_per_block, data))

        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()

        for i in range(0, streams):
            self.assertIn('./stream' + str(i), collection2["manifest_text"])

        for i in range(0, files_per_stream):
            self.assertIn('file' + str(i) + '.txt', collection2["manifest_text"])

        # Read file contents
        self.pool.apply(fuseReadContentsFromCollectionWithManyFiles, (self.mounttmp, streams, files_per_stream, data,))

        # Move file0.txt out of the streams into .
        for i in range(0, streams):
            self.pool.apply(fuseMoveFileFromCollectionWithManyFiles, (self.mounttmp, 'stream'+str(i), 'file0.txt',))

        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()

        manifest_streams = collection2['manifest_text'].split('\n')
        self.assertEqual(4, len(manifest_streams))

        for i in range(0, streams):
            self.assertIn('moved-from-stream'+str(i)+'-file0.txt', manifest_streams[0])

        for i in range(0, streams):
            self.assertNotIn('file0.txt', manifest_streams[i+1])

        for i in range(0, streams):
            for j in range(1, files_per_stream):
                self.assertIn('file' + str(j) + '.txt', manifest_streams[i+1])

        # Delete 'file1.txt' from all the streams
        for i in range(0, streams):
            self.pool.apply(fuseDeleteFileFromCollectionWithManyFiles, (self.mounttmp, 'stream'+str(i), 'file1.txt'))

        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()

        manifest_streams = collection2['manifest_text'].split('\n')
        self.assertEqual(4, len(manifest_streams))

        for i in range(0, streams):
            self.assertIn('moved-from-stream'+str(i)+'-file0.txt', manifest_streams[0])

        self.assertNotIn('file1.txt', collection2['manifest_text'])

        for i in range(0, streams):
            for j in range(2, files_per_stream):
                self.assertIn('file' + str(j) + '.txt', manifest_streams[i+1])
