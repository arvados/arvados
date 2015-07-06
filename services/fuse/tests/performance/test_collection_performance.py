import arvados
import arvados_fuse as fuse
import llfuse
import logging
import os
import sys
import unittest
from .. import run_test_server
from ..mount_test_base import MountTestBase

logger = logging.getLogger('arvados.arv-mount')

from performance_profiler import profiled

def fuse_createCollectionWithMultipleBlocks(mounttmp, streams=1, files_per_stream=1, data='x'):
    class Test(unittest.TestCase):
        def runTest(self):
            self.createCollectionWithMultipleBlocks()

        @profiled
        def createCollectionWithMultipleBlocks(self):
            for i in range(0, streams):
                os.mkdir(os.path.join(mounttmp, "./stream" + str(i)))

                # Create files
                for j in range(0, files_per_stream):
                    with open(os.path.join(mounttmp, "./stream" + str(i), "file" + str(j) +".txt"), "w") as f:
                        f.write(data)

    Test().runTest()

def fuse_readContentsFromCollectionWithMultipleBlocks(mounttmp, streams=1, files_per_stream=1, data='x'):
    class Test(unittest.TestCase):
        def runTest(self):
            self.readContentsFromCollectionWithMultipleBlocks()

        @profiled
        def readContentsFromCollectionWithMultipleBlocks(self):
            for i in range(0, streams):
                d1 = llfuse.listdir(os.path.join(mounttmp, 'stream'+str(i)))
                for j in range(0, files_per_stream):
                    with open(os.path.join(mounttmp, 'stream'+str(i), 'file'+str(i)+'.txt')) as f:
                        self.assertEqual(data, f.read())

    Test().runTest()

def fuse_moveFileFromCollectionWithMultipleBlocks(mounttmp, stream, filename):
    class Test(unittest.TestCase):
        def runTest(self):
            self.moveFileFromCollectionWithMultipleBlocks()

        @profiled
        def moveFileFromCollectionWithMultipleBlocks(self):
            d1 = llfuse.listdir(os.path.join(mounttmp, stream))
            self.assertIn(filename, d1)

            os.rename(os.path.join(mounttmp, stream, filename), os.path.join(mounttmp, 'moved_from_'+stream+'_'+filename))

            d1 = llfuse.listdir(os.path.join(mounttmp))
            self.assertIn('moved_from_'+stream+'_'+filename, d1)

            d1 = llfuse.listdir(os.path.join(mounttmp, stream))
            self.assertNotIn(filename, d1)

    Test().runTest()

def fuse_deleteFileFromCollectionWithMultipleBlocks(mounttmp, stream, filename):
    class Test(unittest.TestCase):
        def runTest(self):
            self.deleteFileFromCollectionWithMultipleBlocks()

        @profiled
        def deleteFileFromCollectionWithMultipleBlocks(self):
            os.remove(os.path.join(mounttmp, stream, filename))

    Test().runTest()

# Create a collection with 2 streams, 3 files_per_stream, 2 blocks_per_file, 2**26 bytes_per_block
class CreateCollectionWithMultipleBlocksAndMoveAndDeleteFile(MountTestBase):
    def setUp(self):
        super(CreateCollectionWithMultipleBlocksAndMoveAndDeleteFile, self).setUp()

    def test_CreateCollectionWithManyBlocksAndMoveAndDeleteFile(self):
        collection = arvados.collection.Collection(api_client=self.api)
        collection.save_new()

        m = self.make_mount(fuse.CollectionDirectory)
        with llfuse.lock:
            m.new_collection(collection.api_response(), collection)
        self.assertTrue(m.writable())

        streams = 2
        files_per_stream = 3
        blocks_per_file = 2
        bytes_per_block = 2**26

        data = 'x' * blocks_per_file * bytes_per_block

        self.pool.apply(fuse_createCollectionWithMultipleBlocks, (self.mounttmp, streams, files_per_stream, data,))

        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()

        for i in range(0, streams):
            self.assertIn('./stream' + str(i), collection2["manifest_text"])

        for i in range(0, files_per_stream):
            self.assertIn('file' + str(i) + '.txt', collection2["manifest_text"])

        # Read file contents
        self.pool.apply(fuse_readContentsFromCollectionWithMultipleBlocks, (self.mounttmp, streams, files_per_stream, data,))

        # Move file0.txt out of the streams into .
        for i in range(0, streams):
            self.pool.apply(fuse_moveFileFromCollectionWithMultipleBlocks, (self.mounttmp, 'stream'+str(i), 'file0.txt',))

        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()

        manifest_streams = collection2['manifest_text'].split('\n')
        self.assertEqual(4, len(manifest_streams))

        for i in range(0, streams):
            self.assertIn('file0.txt', manifest_streams[0])

        for i in range(0, streams):
            self.assertNotIn('file0.txt', manifest_streams[i+1])

        for i in range(0, streams):
            for j in range(1, files_per_stream):
                self.assertIn('file' + str(j) + '.txt', manifest_streams[i+1])

        # Delete 'file1.txt' from all the streams
        for i in range(0, streams):
            self.pool.apply(fuse_deleteFileFromCollectionWithMultipleBlocks, (self.mounttmp, 'stream'+str(i), 'file1.txt'))

        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()

        manifest_streams = collection2['manifest_text'].split('\n')
        self.assertEqual(4, len(manifest_streams))

        for i in range(0, streams):
            self.assertIn('file0.txt', manifest_streams[0])

        self.assertNotIn('file1.txt', collection2['manifest_text'])

        for i in range(0, streams):
            for j in range(2, files_per_stream):
                self.assertIn('file' + str(j) + '.txt', manifest_streams[i+1])


def fuse_createCollectionWithManyFiles(mounttmp, streams=1, files_per_stream=1, data='x'):
    class Test(unittest.TestCase):
        def runTest(self):
            self.createCollectionWithManyFiles()

        @profiled
        def createCollectionWithManyFiles(self):
            for i in range(0, streams):
                os.mkdir(os.path.join(mounttmp, "./stream" + str(i)))

                # Create files
                for j in range(0, files_per_stream):
                    with open(os.path.join(mounttmp, "./stream" + str(i), "file" + str(j) +".txt"), "w") as f:
                        f.write(data)

    Test().runTest()

def fuse_readContentsFromCollectionWithManyFiles(mounttmp, streams=1, files_per_stream=1, data='x'):
    class Test(unittest.TestCase):
        def runTest(self):
            self.readContentsFromCollectionWithManyFiles()

        @profiled
        def readContentsFromCollectionWithManyFiles(self):
            for i in range(0, streams):
                d1 = llfuse.listdir(os.path.join(mounttmp, 'stream'+str(i)))
                for j in range(0, files_per_stream):
                    with open(os.path.join(mounttmp, 'stream'+str(i), 'file'+str(i)+'.txt')) as f:
                        self.assertEqual(data, f.read())

    Test().runTest()

def fuse_moveFileFromCollectionWithManyFiles(mounttmp, stream, filename):
    class Test(unittest.TestCase):
        def runTest(self):
            self.moveFileFromCollectionWithManyFiles()

        @profiled
        def moveFileFromCollectionWithManyFiles(self):
            d1 = llfuse.listdir(os.path.join(mounttmp, stream))
            self.assertIn(filename, d1)

            os.rename(os.path.join(mounttmp, stream, filename), os.path.join(mounttmp, 'moved_from_'+stream+'_'+filename))

            d1 = llfuse.listdir(os.path.join(mounttmp))
            self.assertIn('moved_from_'+stream+'_'+filename, d1)

            d1 = llfuse.listdir(os.path.join(mounttmp, stream))
            self.assertNotIn(filename, d1)

    Test().runTest()

def fuse_deleteFileFromCollectionWithManyFiles(mounttmp, stream, filename):
    class Test(unittest.TestCase):
        def runTest(self):
            self.deleteFileFromCollectionWithManyFiles()

        @profiled
        def deleteFileFromCollectionWithManyFiles(self):
            os.remove(os.path.join(mounttmp, stream, filename))

    Test().runTest()

# Create a collection with two streams, each with 200 files
class CreateCollectionWithManyFilesAndMoveAndDeleteFile(MountTestBase):
    def setUp(self):
        super(CreateCollectionWithManyFilesAndMoveAndDeleteFile, self).setUp()

    def test_CreateCollectionWithManyFilesAndMoveAndDeleteFile(self):
        collection = arvados.collection.Collection(api_client=self.api)
        collection.save_new()

        m = self.make_mount(fuse.CollectionDirectory)
        with llfuse.lock:
            m.new_collection(collection.api_response(), collection)
        self.assertTrue(m.writable())

        streams = 2
        files_per_stream = 200
        data = 'x'

        self.pool.apply(fuse_createCollectionWithManyFiles, (self.mounttmp, streams, files_per_stream, data,))

        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()

        for i in range(0, streams):
            self.assertIn('./stream' + str(i), collection2["manifest_text"])

        for i in range(0, files_per_stream):
            self.assertIn('file' + str(i) + '.txt', collection2["manifest_text"])

        # Read file contents
        self.pool.apply(fuse_readContentsFromCollectionWithManyFiles, (self.mounttmp, streams, files_per_stream, data,))

        # Move file0.txt out of the streams into .
        for i in range(0, streams):
            self.pool.apply(fuse_moveFileFromCollectionWithManyFiles, (self.mounttmp, 'stream'+str(i), 'file0.txt',))

        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()

        manifest_streams = collection2['manifest_text'].split('\n')
        self.assertEqual(4, len(manifest_streams))

        for i in range(0, streams):
            self.assertIn('file0.txt', manifest_streams[0])

        for i in range(0, streams):
            self.assertNotIn('file0.txt', manifest_streams[i+1])

        for i in range(0, streams):
            for j in range(1, files_per_stream):
                self.assertIn('file' + str(j) + '.txt', manifest_streams[i+1])

        # Delete 'file1.txt' from all the streams
        for i in range(0, streams):
            self.pool.apply(fuse_deleteFileFromCollectionWithManyFiles, (self.mounttmp, 'stream'+str(i), 'file1.txt'))

        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()

        manifest_streams = collection2['manifest_text'].split('\n')
        self.assertEqual(4, len(manifest_streams))

        for i in range(0, streams):
            self.assertIn('file0.txt', manifest_streams[0])

        self.assertNotIn('file1.txt', collection2['manifest_text'])

        for i in range(0, streams):
            for j in range(2, files_per_stream):
                self.assertIn('file' + str(j) + '.txt', manifest_streams[i+1])


def magicDirTest_MoveFileFromCollection(mounttmp, collection1, collection2, stream, filename):
    class Test(unittest.TestCase):
        def runTest(self):
            self.magicDirTest_moveFileFromCollection()

        @profiled
        def magicDirTest_moveFileFromCollection(self):
            os.rename(os.path.join(mounttmp, collection1, filename), os.path.join(mounttmp, collection2, filename))

    Test().runTest()

def magicDirTest_RemoveFileFromCollection(mounttmp, collection1, stream, filename):
    class Test(unittest.TestCase):
        def runTest(self):
            self.magicDirTest_removeFileFromCollection()

        @profiled
        def magicDirTest_removeFileFromCollection(self):
            os.remove(os.path.join(mounttmp, collection1, filename))

    Test().runTest()

class UsingMagicDir_CreateCollectionWithManyFilesAndMoveAndDeleteFile(MountTestBase):
    def setUp(self):
        super(UsingMagicDir_CreateCollectionWithManyFilesAndMoveAndDeleteFile, self).setUp()

    @profiled
    def magicDirTest_createCollectionWithManyFiles(self, streams=0, files_per_stream=0, data='x'):
        # Create collection
        collection = arvados.collection.Collection(api_client=self.api)
        for j in range(0, files_per_stream):
            with collection.open("file"+str(j)+".txt", "w") as f:
                f.write(data)
        collection.save_new()
        return collection

    @profiled
    def magicDirTest_readCollectionContents(self, collection, streams=1, files_per_stream=1, data='x'):
        mount_ls = os.listdir(os.path.join(self.mounttmp, collection))

        files = {}
        for j in range(0, files_per_stream):
            files[os.path.join(self.mounttmp, collection, 'file'+str(j)+'.txt')] = data

        for k, v in files.items():
            with open(os.path.join(self.mounttmp, collection, k)) as f:
                self.assertEqual(v, f.read())

    def test_UsingMagicDirCreateCollectionWithManyFilesAndMoveAndDeleteFile(self):
        streams = 2
        files_per_stream = 200
        data = 'x'

        collection1 = self.magicDirTest_createCollectionWithManyFiles()
        # Create collection with multiple files
        collection2 = self.magicDirTest_createCollectionWithManyFiles(streams, files_per_stream, data)

        # Mount FuseMagicDir
        self.make_mount(fuse.MagicDirectory)

        self.magicDirTest_readCollectionContents(collection2.manifest_locator(), streams, files_per_stream, data)

        # Move file0.txt out of the collection2 into collection1
        self.pool.apply(magicDirTest_MoveFileFromCollection, (self.mounttmp, collection2.manifest_locator(),
              collection1.manifest_locator(), 'stream0', 'file0.txt',))
        updated_collection = self.api.collections().get(uuid=collection2.manifest_locator()).execute()
        self.assertFalse('file0.txt' in updated_collection['manifest_text'])
        self.assertTrue('file1.txt' in updated_collection['manifest_text'])

        # Delete file1.txt from collection2
        self.pool.apply(magicDirTest_RemoveFileFromCollection, (self.mounttmp, collection2.manifest_locator(), 'stream0', 'file1.txt',))
        updated_collection = self.api.collections().get(uuid=collection2.manifest_locator()).execute()
        self.assertFalse('file1.txt' in updated_collection['manifest_text'])
        self.assertTrue('file2.txt' in updated_collection['manifest_text'])


def magicDirTest_MoveAllFilesFromCollection(mounttmp, from_collection, to_collection, stream, files_per_stream):
    class Test(unittest.TestCase):
        def runTest(self):
            self.magicDirTest_moveAllFilesFromCollection()

        @profiled
        def magicDirTest_moveAllFilesFromCollection(self):
            for j in range(0, files_per_stream):
                os.rename(os.path.join(mounttmp, from_collection, 'file'+str(j)+'.txt'), os.path.join(mounttmp, to_collection, 'file'+str(j)+'.txt'))

    Test().runTest()

class UsingMagicDir_CreateCollectionWithManyFilesAndMoveAllFilesIntoAnother(MountTestBase):
    def setUp(self):
        super(UsingMagicDir_CreateCollectionWithManyFilesAndMoveAllFilesIntoAnother, self).setUp()

    @profiled
    def magicDirTestMoveAllFiles_createCollectionWithManyFiles(self, streams=0, files_per_stream=0,
            blocks_per_file=0, bytes_per_block=0, data='x'):
        # Create collection
        collection = arvados.collection.Collection(api_client=self.api)
        for j in range(0, files_per_stream):
            with collection.open("file"+str(j)+".txt", "w") as f:
                f.write(data)
        collection.save_new()
        return collection

    @profiled
    def test_UsingMagicDirCreateCollectionWithManyFilesAndMoveAllFilesIntoAnother(self):
        streams = 2
        files_per_stream = 200
        data = 'x'

        collection1 = self.magicDirTestMoveAllFiles_createCollectionWithManyFiles()
        # Create collection with multiple files
        collection2 = self.magicDirTestMoveAllFiles_createCollectionWithManyFiles(streams, files_per_stream, data)

        # Mount FuseMagicDir
        self.make_mount(fuse.MagicDirectory)

        # Move all files from collection2 into collection1
        self.pool.apply(magicDirTest_MoveAllFilesFromCollection, (self.mounttmp, collection2.manifest_locator(),
                  collection1.manifest_locator(), 'stream0', files_per_stream,))

        updated_collection = self.api.collections().get(uuid=collection2.manifest_locator()).execute()
        file_names = ["file%i.txt" % i for i in range(0, files_per_stream)]
        for name in file_names:
            self.assertFalse(name in updated_collection['manifest_text'])

        updated_collection = self.api.collections().get(uuid=collection1.manifest_locator()).execute()
        for name in file_names:
            self.assertTrue(name in updated_collection['manifest_text'])


# Move one file at a time from one collection into another
class UsingMagicDir_CreateCollectionWithManyFilesAndMoveEachFileIntoAnother(MountTestBase):
    def setUp(self):
        super(UsingMagicDir_CreateCollectionWithManyFilesAndMoveEachFileIntoAnother, self).setUp()

    @profiled
    def magicDirTestMoveFiles_createCollectionWithManyFiles(self, streams=0, files_per_stream=0, data='x'):
        # Create collection
        collection = arvados.collection.Collection(api_client=self.api)
        for j in range(0, files_per_stream):
            with collection.open("file"+str(j)+".txt", "w") as f:
                f.write(data)
        collection.save_new()
        return collection

    def magicDirTestMoveFiles_oneEachIntoAnother(self, from_collection, to_collection, files_per_stream):
        for j in range(0, files_per_stream):
            self.pool.apply(magicDirTest_MoveFileFromCollection, (self.mounttmp, from_collection.manifest_locator(),
                  to_collection.manifest_locator(), 'stream0', 'file'+str(j)+'.txt',))

    @profiled
    def test_UsingMagicDirCreateCollectionWithManyFilesAndMoveEachFileIntoAnother(self):
        streams = 2
        files_per_stream = 200
        data = 'x'

        collection1 = self.magicDirTestMoveFiles_createCollectionWithManyFiles()
        # Create collection with multiple files
        collection2 = self.magicDirTestMoveFiles_createCollectionWithManyFiles(streams, files_per_stream, data)

        # Mount FuseMagicDir
        self.make_mount(fuse.MagicDirectory)

        # Move all files from collection2 into collection1
        self.magicDirTestMoveFiles_oneEachIntoAnother(collection2, collection1, files_per_stream)

        updated_collection = self.api.collections().get(uuid=collection2.manifest_locator()).execute()
        file_names = ["file%i.txt" % i for i in range(0, files_per_stream)]
        for name in file_names:
            self.assertFalse(name in updated_collection['manifest_text'])

        updated_collection = self.api.collections().get(uuid=collection1.manifest_locator()).execute()
        for name in file_names:
            self.assertTrue(name in updated_collection['manifest_text'])

class FuseListLargeProjectContents(MountTestBase):
    @profiled
    def getProjectWithManyCollections(self):
        project_contents = llfuse.listdir(self.mounttmp)
        self.assertEqual(201, len(project_contents))
        self.assertIn('Collection_1', project_contents)
        return project_contents

    @profiled
    def listContentsInProjectWithManyCollections(self, project_contents):
        project_contents = llfuse.listdir(self.mounttmp)
        self.assertEqual(201, len(project_contents))
        self.assertIn('Collection_1', project_contents)

        for collection_name in project_contents:
            collection_contents = llfuse.listdir(os.path.join(self.mounttmp, collection_name))
            self.assertIn('baz', collection_contents)

    def test_listLargeProjectContents(self):
        self.make_mount(fuse.ProjectDirectory,
                        project_object=run_test_server.fixture('groups')['project_with_201_collections'])
        project_contents = self.getProjectWithManyCollections()
        self.listContentsInProjectWithManyCollections(project_contents)
