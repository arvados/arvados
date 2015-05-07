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
import run_test_server

logger = logging.getLogger('arvados.arv-mount')

class MountTestBase(unittest.TestCase):
    def setUp(self):
        self.keeptmp = tempfile.mkdtemp()
        os.environ['KEEP_LOCAL_STORE'] = self.keeptmp
        self.mounttmp = tempfile.mkdtemp()
        run_test_server.run()
        run_test_server.authorize_with("admin")
        self.api = arvados.safeapi.ThreadSafeApiCache(arvados.config.settings())

    def make_mount(self, root_class, **root_kwargs):
        self.operations = fuse.Operations(os.getuid(), os.getgid())
        self.operations.inodes.add_entry(root_class(
            llfuse.ROOT_INODE, self.operations.inodes, self.api, 0, **root_kwargs))
        llfuse.init(self.operations, self.mounttmp, [])
        threading.Thread(None, llfuse.main).start()
        # wait until the driver is finished initializing
        self.operations.initlock.wait()
        return self.operations.inodes[llfuse.ROOT_INODE]

    def tearDown(self):
        # llfuse.close is buggy, so use fusermount instead.
        #llfuse.close(unmount=True)
        count = 0
        success = 1
        while (count < 9 and success != 0):
          success = subprocess.call(["fusermount", "-u", self.mounttmp])
          time.sleep(0.5)
          count += 1

        os.rmdir(self.mounttmp)
        shutil.rmtree(self.keeptmp)
        run_test_server.reset()

    def assertDirContents(self, subdir, expect_content):
        path = self.mounttmp
        if subdir:
            path = os.path.join(path, subdir)
        self.assertEqual(sorted(expect_content), sorted(llfuse.listdir(path)))


class FuseMountTest(MountTestBase):
    def setUp(self):
        super(FuseMountTest, self).setUp()

        cw = arvados.CollectionWriter()

        cw.start_new_file('thing1.txt')
        cw.write("data 1")
        cw.start_new_file('thing2.txt')
        cw.write("data 2")
        cw.start_new_stream('dir1')

        cw.start_new_file('thing3.txt')
        cw.write("data 3")
        cw.start_new_file('thing4.txt')
        cw.write("data 4")

        cw.start_new_stream('dir2')
        cw.start_new_file('thing5.txt')
        cw.write("data 5")
        cw.start_new_file('thing6.txt')
        cw.write("data 6")

        cw.start_new_stream('dir2/dir3')
        cw.start_new_file('thing7.txt')
        cw.write("data 7")

        cw.start_new_file('thing8.txt')
        cw.write("data 8")

        cw.start_new_stream('edgecases')
        for f in ":/./../.../-/*/\x01\\/ ".split("/"):
            cw.start_new_file(f)
            cw.write('x')

        for f in ":/../.../-/*/\x01\\/ ".split("/"):
            cw.start_new_stream('edgecases/dirs/' + f)
            cw.start_new_file('x/x')
            cw.write('x')

        self.testcollection = cw.finish()
        self.api.collections().create(body={"manifest_text":cw.manifest_text()}).execute()

    def runTest(self):
        self.make_mount(fuse.CollectionDirectory, collection_record=self.testcollection)

        self.assertDirContents(None, ['thing1.txt', 'thing2.txt',
                                      'edgecases', 'dir1', 'dir2'])
        self.assertDirContents('dir1', ['thing3.txt', 'thing4.txt'])
        self.assertDirContents('dir2', ['thing5.txt', 'thing6.txt', 'dir3'])
        self.assertDirContents('dir2/dir3', ['thing7.txt', 'thing8.txt'])
        self.assertDirContents('edgecases',
                               "dirs/:/_/__/.../-/*/\x01\\/ ".split("/"))
        self.assertDirContents('edgecases/dirs',
                               ":/__/.../-/*/\x01\\/ ".split("/"))

        files = {'thing1.txt': 'data 1',
                 'thing2.txt': 'data 2',
                 'dir1/thing3.txt': 'data 3',
                 'dir1/thing4.txt': 'data 4',
                 'dir2/thing5.txt': 'data 5',
                 'dir2/thing6.txt': 'data 6',
                 'dir2/dir3/thing7.txt': 'data 7',
                 'dir2/dir3/thing8.txt': 'data 8'}

        for k, v in files.items():
            with open(os.path.join(self.mounttmp, k)) as f:
                self.assertEqual(v, f.read())


class FuseNoAPITest(MountTestBase):
    def setUp(self):
        super(FuseNoAPITest, self).setUp()
        keep = arvados.keep.KeepClient(local_store=self.keeptmp)
        self.file_data = "API-free text\n"
        self.file_loc = keep.put(self.file_data)
        self.coll_loc = keep.put(". {} 0:{}:api-free.txt\n".format(
                self.file_loc, len(self.file_data)))

    def runTest(self):
        self.make_mount(fuse.MagicDirectory)
        self.assertDirContents(self.coll_loc, ['api-free.txt'])
        with open(os.path.join(
                self.mounttmp, self.coll_loc, 'api-free.txt')) as keep_file:
            actual = keep_file.read(-1)
        self.assertEqual(self.file_data, actual)


class FuseMagicTest(MountTestBase):
    def setUp(self):
        super(FuseMagicTest, self).setUp()

        cw = arvados.CollectionWriter()

        cw.start_new_file('thing1.txt')
        cw.write("data 1")

        self.testcollection = cw.finish()
        self.api.collections().create(body={"manifest_text":cw.manifest_text()}).execute()

    def runTest(self):
        self.make_mount(fuse.MagicDirectory)

        mount_ls = llfuse.listdir(self.mounttmp)
        self.assertIn('README', mount_ls)
        self.assertFalse(any(arvados.util.keep_locator_pattern.match(fn) or
                             arvados.util.uuid_pattern.match(fn)
                             for fn in mount_ls),
                         "new FUSE MagicDirectory lists Collection")
        self.assertDirContents(self.testcollection, ['thing1.txt'])
        self.assertDirContents(os.path.join('by_id', self.testcollection),
                               ['thing1.txt'])
        mount_ls = llfuse.listdir(self.mounttmp)
        self.assertIn('README', mount_ls)
        self.assertIn(self.testcollection, mount_ls)
        self.assertIn(self.testcollection,
                      llfuse.listdir(os.path.join(self.mounttmp, 'by_id')))

        files = {}
        files[os.path.join(self.mounttmp, self.testcollection, 'thing1.txt')] = 'data 1'

        for k, v in files.items():
            with open(os.path.join(self.mounttmp, k)) as f:
                self.assertEqual(v, f.read())


class FuseTagsTest(MountTestBase):
    def runTest(self):
        self.make_mount(fuse.TagsDirectory)

        d1 = llfuse.listdir(self.mounttmp)
        d1.sort()
        self.assertEqual(['foo_tag'], d1)

        d2 = llfuse.listdir(os.path.join(self.mounttmp, 'foo_tag'))
        d2.sort()
        self.assertEqual(['zzzzz-4zz18-fy296fx3hot09f7'], d2)

        d3 = llfuse.listdir(os.path.join(self.mounttmp, 'foo_tag', 'zzzzz-4zz18-fy296fx3hot09f7'))
        d3.sort()
        self.assertEqual(['foo'], d3)


class FuseTagsUpdateTest(MountTestBase):
    def tag_collection(self, coll_uuid, tag_name):
        return self.api.links().create(
            body={'link': {'head_uuid': coll_uuid,
                           'link_class': 'tag',
                           'name': tag_name,
        }}).execute()

    def runTest(self):
        self.make_mount(fuse.TagsDirectory, poll_time=1)

        self.assertIn('foo_tag', llfuse.listdir(self.mounttmp))

        bar_uuid = run_test_server.fixture('collections')['bar_file']['uuid']
        self.tag_collection(bar_uuid, 'fuse_test_tag')
        time.sleep(1)
        self.assertIn('fuse_test_tag', llfuse.listdir(self.mounttmp))
        self.assertDirContents('fuse_test_tag', [bar_uuid])

        baz_uuid = run_test_server.fixture('collections')['baz_file']['uuid']
        l = self.tag_collection(baz_uuid, 'fuse_test_tag')
        time.sleep(1)
        self.assertDirContents('fuse_test_tag', [bar_uuid, baz_uuid])

        self.api.links().delete(uuid=l['uuid']).execute()
        time.sleep(1)
        self.assertDirContents('fuse_test_tag', [bar_uuid])


class FuseSharedTest(MountTestBase):
    def runTest(self):
        self.make_mount(fuse.SharedDirectory,
                        exclude=self.api.users().current().execute()['uuid'])

        # shared_dirs is a list of the directories exposed
        # by fuse.SharedDirectory (i.e. any object visible
        # to the current user)
        shared_dirs = llfuse.listdir(self.mounttmp)
        shared_dirs.sort()
        self.assertIn('FUSE User', shared_dirs)

        # fuse_user_objs is a list of the objects owned by the FUSE
        # test user (which present as files in the 'FUSE User'
        # directory)
        fuse_user_objs = llfuse.listdir(os.path.join(self.mounttmp, 'FUSE User'))
        fuse_user_objs.sort()
        self.assertEqual(['FUSE Test Project',                    # project owned by user
                          'collection #1 owned by FUSE',          # collection owned by user
                          'collection #2 owned by FUSE',          # collection owned by user
                          'pipeline instance owned by FUSE.pipelineInstance',  # pipeline instance owned by user
                      ], fuse_user_objs)

        # test_proj_files is a list of the files in the FUSE Test Project.
        test_proj_files = llfuse.listdir(os.path.join(self.mounttmp, 'FUSE User', 'FUSE Test Project'))
        test_proj_files.sort()
        self.assertEqual(['collection in FUSE project',
                          'pipeline instance in FUSE project.pipelineInstance',
                          'pipeline template in FUSE project.pipelineTemplate'
                      ], test_proj_files)

        # Double check that we can open and read objects in this folder as a file,
        # and that its contents are what we expect.
        pipeline_template_path = os.path.join(
                self.mounttmp,
                'FUSE User',
                'FUSE Test Project',
                'pipeline template in FUSE project.pipelineTemplate')
        with open(pipeline_template_path) as f:
            j = json.load(f)
            self.assertEqual("pipeline template in FUSE project", j['name'])

        # check mtime on template
        st = os.stat(pipeline_template_path)
        self.assertEqual(st.st_mtime, 1397493304)

        # check mtime on collection
        st = os.stat(os.path.join(
                self.mounttmp,
                'FUSE User',
                'collection #1 owned by FUSE'))
        self.assertEqual(st.st_mtime, 1391448174)


class FuseHomeTest(MountTestBase):
    def runTest(self):
        self.make_mount(fuse.ProjectDirectory,
                        project_object=self.api.users().current().execute())

        d1 = llfuse.listdir(self.mounttmp)
        self.assertIn('Unrestricted public data', d1)

        d2 = llfuse.listdir(os.path.join(self.mounttmp, 'Unrestricted public data'))
        public_project = run_test_server.fixture('groups')[
            'anonymously_accessible_project']
        found_in = 0
        found_not_in = 0
        for name, item in run_test_server.fixture('collections').iteritems():
            if 'name' not in item:
                pass
            elif item['owner_uuid'] == public_project['uuid']:
                self.assertIn(item['name'], d2)
                found_in += 1
            else:
                # Artificial assumption here: there is no public
                # collection fixture with the same name as a
                # non-public collection.
                self.assertNotIn(item['name'], d2)
                found_not_in += 1
        self.assertNotEqual(0, found_in)
        self.assertNotEqual(0, found_not_in)

        d3 = llfuse.listdir(os.path.join(self.mounttmp, 'Unrestricted public data', 'GNU General Public License, version 3'))
        self.assertEqual(["GNU_General_Public_License,_version_3.pdf"], d3)

class FuseUpdateFileTest(MountTestBase):
    def runTest(self):
        collection = arvados.collection.Collection(api_client=self.api)
        with collection.open("file1.txt", "w") as f:
            f.write("blub")

        collection.save_new()

        m = self.make_mount(fuse.CollectionDirectory)
        with llfuse.lock:
            m.new_collection(collection.api_response(), collection)

        d1 = llfuse.listdir(self.mounttmp)
        self.assertEqual(["file1.txt"], d1)
        with open(os.path.join(self.mounttmp, "file1.txt")) as f:
            self.assertEqual("blub", f.read())

        with collection.open("file1.txt", "w") as f:
            f.write("plnp")

        d1 = llfuse.listdir(self.mounttmp)
        self.assertEqual(["file1.txt"], d1)
        with open(os.path.join(self.mounttmp, "file1.txt")) as f:
            self.assertEqual("plnp", f.read())

class FuseAddFileToCollectionTest(MountTestBase):
    def runTest(self):
        collection = arvados.collection.Collection(api_client=self.api)
        with collection.open("file1.txt", "w") as f:
            f.write("blub")

        collection.save_new()

        m = self.make_mount(fuse.CollectionDirectory)
        with llfuse.lock:
            m.new_collection(collection.api_response(), collection)

        d1 = llfuse.listdir(self.mounttmp)
        self.assertEqual(["file1.txt"], d1)

        with collection.open("file2.txt", "w") as f:
            f.write("plnp")

        d1 = llfuse.listdir(self.mounttmp)
        self.assertEqual(["file1.txt", "file2.txt"], sorted(d1))

class FuseRemoveFileFromCollectionTest(MountTestBase):
    def runTest(self):
        collection = arvados.collection.Collection(api_client=self.api)
        with collection.open("file1.txt", "w") as f:
            f.write("blub")

        with collection.open("file2.txt", "w") as f:
            f.write("plnp")

        collection.save_new()

        m = self.make_mount(fuse.CollectionDirectory)
        with llfuse.lock:
            m.new_collection(collection.api_response(), collection)

        d1 = llfuse.listdir(self.mounttmp)
        self.assertEqual(["file1.txt", "file2.txt"], sorted(d1))

        collection.remove("file2.txt")

        d1 = llfuse.listdir(self.mounttmp)
        self.assertEqual(["file1.txt"], d1)

class FuseCreateFileTest(MountTestBase):
    def runTest(self):
        collection = arvados.collection.Collection(api_client=self.api)
        collection.save_new()

        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()
        self.assertEqual(collection2["manifest_text"], "")

        collection.save_new()

        m = self.make_mount(fuse.CollectionDirectory)
        with llfuse.lock:
            m.new_collection(collection.api_response(), collection)
        self.assertTrue(m.writable())

        self.assertNotIn("file1.txt", collection)

        with open(os.path.join(self.mounttmp, "file1.txt"), "w") as f:
            pass

        self.assertIn("file1.txt", collection)

        d1 = llfuse.listdir(self.mounttmp)
        self.assertEqual(["file1.txt"], d1)

        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()
        self.assertRegexpMatches(collection2["manifest_text"],
            r'\. d41d8cd98f00b204e9800998ecf8427e\+0\+A[a-f0-9]{40}@[a-f0-9]{8} 0:0:file1\.txt$')

def fuseWriteFileTestHelper(mounttmp):
    with open(os.path.join(mounttmp, "file1.txt"), "r") as f:
        return f.read() == "Hello world!"

class FuseWriteFileTest(MountTestBase):
    def runTest(self):
        arvados.logger.setLevel(logging.DEBUG)

        collection = arvados.collection.Collection(api_client=self.api)
        collection.save_new()

        m = self.make_mount(fuse.CollectionDirectory)
        with llfuse.lock:
            m.new_collection(collection.api_response(), collection)
        self.assertTrue(m.writable())

        self.assertNotIn("file1.txt", collection)

        with open(os.path.join(self.mounttmp, "file1.txt"), "w") as f:
            f.write("Hello world!")

        with collection.open("file1.txt") as f:
            self.assertEqual(f.read(), "Hello world!")

        # We can't just open the collection for reading because the underlying
        # C implementation of open() makes a fstat() syscall with the GIL still
        # held.  When the GETATTR message comes back to llfuse (which in these
        # tests is in the same interpreter process) it can't acquire the GIL,
        # so it can't service the fstat() call, so it deadlocks.  The
        # workaround is to run some of our test code in a separate process.
        # Forturnately the multiprocessing module makes this relatively easy.
        pool = multiprocessing.Pool(1)
        self.assertTrue(pool.apply(fuseWriteFileTestHelper, (self.mounttmp,)))
        pool.close()

        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()
        self.assertRegexpMatches(collection2["manifest_text"],
            r'\. 86fb269d190d2c85f6e0468ceca42a20\+12\+A[a-f0-9]{40}@[a-f0-9]{8} 0:12:file1\.txt$')

def fuseUpdateFileTestHelper1(mounttmp):
    with open(os.path.join(mounttmp, "file1.txt"), "r+") as f:
        fr = f.read()
        if fr != "Hello world!":
            raise Exception("Got %s expected 'Hello world!'" % fr)
        f.seek(0)
        f.write("Hola mundo!")
        f.seek(0)
        fr = f.read()
        if fr != "Hola mundo!!":
            raise Exception("Got %s expected 'Hola mundo!!'" % fr)
        return True

def fuseUpdateFileTestHelper2(mounttmp):
    with open(os.path.join(mounttmp, "file1.txt"), "r") as f:
        return f.read() == "Hola mundo!!"

class FuseUpdateFileTest(MountTestBase):
    def runTest(self):
        arvados.logger.setLevel(logging.DEBUG)

        collection = arvados.collection.Collection(api_client=self.api)
        collection.save_new()

        m = self.make_mount(fuse.CollectionDirectory)
        with llfuse.lock:
            m.new_collection(collection.api_response(), collection)
        self.assertTrue(m.writable())

        with open(os.path.join(self.mounttmp, "file1.txt"), "w") as f:
            f.write("Hello world!")

        # See note in FuseWriteFileTest
        pool = multiprocessing.Pool(1)
        self.assertTrue(pool.apply(fuseUpdateFileTestHelper1, (self.mounttmp,)))
        self.assertTrue(pool.apply(fuseUpdateFileTestHelper2, (self.mounttmp,)))
        pool.close()

        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()
        self.assertRegexpMatches(collection2["manifest_text"],
            r'\. daaef200ebb921e011e3ae922dd3266b\+11\+A[a-f0-9]{40}@[a-f0-9]{8} 86fb269d190d2c85f6e0468ceca42a20\+12\+A[a-f0-9]{40}@[a-f0-9]{8} 0:11:file1\.txt 22:1:file1\.txt$')


class FuseMkdirTest(MountTestBase):
    def runTest(self):
        arvados.logger.setLevel(logging.DEBUG)

        collection = arvados.collection.Collection(api_client=self.api)
        collection.save_new()

        m = self.make_mount(fuse.CollectionDirectory)
        with llfuse.lock:
            m.new_collection(collection.api_response(), collection)
        self.assertTrue(m.writable())

        with self.assertRaises(IOError):
            with open(os.path.join(self.mounttmp, "testdir", "file1.txt"), "w") as f:
                f.write("Hello world!")

        os.mkdir(os.path.join(self.mounttmp, "testdir"))

        with self.assertRaises(OSError):
            os.mkdir(os.path.join(self.mounttmp, "testdir"))

        d1 = llfuse.listdir(self.mounttmp)
        self.assertEqual(["testdir"], d1)

        with open(os.path.join(self.mounttmp, "testdir", "file1.txt"), "w") as f:
            f.write("Hello world!")

        d1 = llfuse.listdir(os.path.join(self.mounttmp, "testdir"))
        self.assertEqual(["file1.txt"], d1)

        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()
        self.assertRegexpMatches(collection2["manifest_text"],
            r'\./testdir 86fb269d190d2c85f6e0468ceca42a20\+12\+A[a-f0-9]{40}@[a-f0-9]{8} 0:12:file1\.txt$')


class FuseRmTest(MountTestBase):
    def runTest(self):
        arvados.logger.setLevel(logging.DEBUG)

        collection = arvados.collection.Collection(api_client=self.api)
        collection.save_new()

        m = self.make_mount(fuse.CollectionDirectory)
        with llfuse.lock:
            m.new_collection(collection.api_response(), collection)
        self.assertTrue(m.writable())

        os.mkdir(os.path.join(self.mounttmp, "testdir"))

        with open(os.path.join(self.mounttmp, "testdir", "file1.txt"), "w") as f:
            f.write("Hello world!")

        # Starting manifest
        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()
        self.assertRegexpMatches(collection2["manifest_text"],
            r'\./testdir 86fb269d190d2c85f6e0468ceca42a20\+12\+A[a-f0-9]{40}@[a-f0-9]{8} 0:12:file1\.txt$')

        # Can't delete because it's not empty
        with self.assertRaises(OSError):
            os.rmdir(os.path.join(self.mounttmp, "testdir"))

        d1 = llfuse.listdir(os.path.join(self.mounttmp, "testdir"))
        self.assertEqual(["file1.txt"], d1)

        # Delete file
        os.remove(os.path.join(self.mounttmp, "testdir", "file1.txt"))

        # Make sure it's empty
        d1 = llfuse.listdir(os.path.join(self.mounttmp, "testdir"))
        self.assertEqual([], d1)

        # Try to delete it again
        with self.assertRaises(OSError):
            os.remove(os.path.join(self.mounttmp, "testdir", "file1.txt"))

        # Can't have empty directories :-( so manifest will be empty.
        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()
        self.assertEqual(collection2["manifest_text"], "")

        # Should be able to delete now that it is empty
        os.rmdir(os.path.join(self.mounttmp, "testdir"))

        # Make sure it's empty
        d1 = llfuse.listdir(os.path.join(self.mounttmp))
        self.assertEqual([], d1)

        # Try to delete it again
        with self.assertRaises(OSError):
            os.rmdir(os.path.join(self.mounttmp, "testdir"))

        # manifest should be empty now.
        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()
        self.assertEqual(collection2["manifest_text"], "")


class FuseMvTest(MountTestBase):
    def runTest(self):
        arvados.logger.setLevel(logging.DEBUG)

        collection = arvados.collection.Collection(api_client=self.api)
        collection.save_new()

        m = self.make_mount(fuse.CollectionDirectory)
        with llfuse.lock:
            m.new_collection(collection.api_response(), collection)
        self.assertTrue(m.writable())

        os.mkdir(os.path.join(self.mounttmp, "testdir"))

        with open(os.path.join(self.mounttmp, "testdir", "file1.txt"), "w") as f:
            f.write("Hello world!")

        # Starting manifest
        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()
        self.assertRegexpMatches(collection2["manifest_text"],
            r'\./testdir 86fb269d190d2c85f6e0468ceca42a20\+12\+A[a-f0-9]{40}@[a-f0-9]{8} 0:12:file1\.txt$')

        d1 = llfuse.listdir(os.path.join(self.mounttmp))
        self.assertEqual(["testdir"], d1)
        d1 = llfuse.listdir(os.path.join(self.mounttmp, "testdir"))
        self.assertEqual(["file1.txt"], d1)

        os.rename(os.path.join(self.mounttmp, "testdir", "file1.txt"), os.path.join(self.mounttmp, "file1.txt"))

        d1 = llfuse.listdir(os.path.join(self.mounttmp))
        self.assertEqual(["file1.txt", "testdir"], sorted(d1))
        d1 = llfuse.listdir(os.path.join(self.mounttmp, "testdir"))
        self.assertEqual([], d1)

        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()
        self.assertRegexpMatches(collection2["manifest_text"],
            r'\. 86fb269d190d2c85f6e0468ceca42a20\+12\+A[a-f0-9]{40}@[a-f0-9]{8} 0:12:file1\.txt$')


class FuseUpdateFromEventTest(MountTestBase):
    def runTest(self):
        arvados.logger.setLevel(logging.DEBUG)

        collection = arvados.collection.Collection(api_client=self.api)
        collection.save_new()

        m = self.make_mount(fuse.CollectionDirectory)
        with llfuse.lock:
            m.new_collection(collection.api_response(), collection)

        self.operations.listen_for_events(self.api)

        d1 = llfuse.listdir(os.path.join(self.mounttmp))
        self.assertEqual([], sorted(d1))

        with arvados.collection.Collection(collection.manifest_locator(), api_client=self.api) as collection2:
            with collection2.open("file1.txt", "w") as f:
                f.write("foo")

        time.sleep(1)

        # should show up via event bus notify

        d1 = llfuse.listdir(os.path.join(self.mounttmp))
        self.assertEqual(["file1.txt"], sorted(d1))


class FuseUnitTest(unittest.TestCase):
    def test_sanitize_filename(self):
        acceptable = [
            "foo.txt",
            ".foo",
            "..foo",
            "...",
            "foo...",
            "foo..",
            "foo.",
            "-",
            "\x01\x02\x03",
            ]
        unacceptable = [
            "f\00",
            "\00\00",
            "/foo",
            "foo/",
            "//",
            ]
        for f in acceptable:
            self.assertEqual(f, fuse.sanitize_filename(f))
        for f in unacceptable:
            self.assertNotEqual(f, fuse.sanitize_filename(f))
            # The sanitized filename should be the same length, though.
            self.assertEqual(len(f), len(fuse.sanitize_filename(f)))
        # Special cases
        self.assertEqual("_", fuse.sanitize_filename(""))
        self.assertEqual("_", fuse.sanitize_filename("."))
        self.assertEqual("__", fuse.sanitize_filename(".."))
