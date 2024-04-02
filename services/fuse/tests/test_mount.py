# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

from __future__ import absolute_import
from future.utils import viewitems
from builtins import str
from builtins import object
from pathlib import Path
from six import assertRegex
import errno
import json
import llfuse
import logging
import mock
import os
import subprocess
import time
import unittest
import tempfile
import parameterized

import arvados
import arvados_fuse as fuse
from arvados_fuse import fusedir
from . import run_test_server

from .integration_test import IntegrationTest
from .mount_test_base import MountTestBase
from .test_tmp_collection import storage_classes_desired

logger = logging.getLogger('arvados.arv-mount')


class AssertWithTimeout(object):
    """Allow some time for an assertion to pass."""

    def __init__(self, timeout=0):
        self.timeout = timeout

    def __iter__(self):
        self.deadline = time.time() + self.timeout
        self.done = False
        return self

    def __next__(self):
        if self.done:
            raise StopIteration
        return self.attempt

    def attempt(self, fn, *args, **kwargs):
        try:
            fn(*args, **kwargs)
        except AssertionError:
            if time.time() > self.deadline:
                raise
            time.sleep(0.1)
        else:
            self.done = True

@parameterized.parameterized_class([{"disk_cache": True}, {"disk_cache": False}])
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
        for f in ":/.../-/*/ ".split("/"):
            cw.start_new_file(f)
            cw.write('x')

        for f in ":/.../-/*/ ".split("/"):
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
                               "dirs/:/.../-/*/ ".split("/"))
        self.assertDirContents('edgecases/dirs',
                               ":/.../-/*/ ".split("/"))

        files = {'thing1.txt': 'data 1',
                 'thing2.txt': 'data 2',
                 'dir1/thing3.txt': 'data 3',
                 'dir1/thing4.txt': 'data 4',
                 'dir2/thing5.txt': 'data 5',
                 'dir2/thing6.txt': 'data 6',
                 'dir2/dir3/thing7.txt': 'data 7',
                 'dir2/dir3/thing8.txt': 'data 8'}

        for k, v in viewitems(files):
            with open(os.path.join(self.mounttmp, k), 'rb') as f:
                self.assertEqual(v, f.read().decode())


@parameterized.parameterized_class([{"disk_cache": True}, {"disk_cache": False}])
class FuseMagicTest(MountTestBase):
    def setUp(self, api=None):
        super(FuseMagicTest, self).setUp(api=api)

        self.test_project = run_test_server.fixture('groups')['aproject']['uuid']
        self.non_project_group = run_test_server.fixture('groups')['public_role']['uuid']
        self.filter_group = run_test_server.fixture('groups')['afiltergroup']['uuid']
        self.collection_in_test_project = run_test_server.fixture('collections')['foo_collection_in_aproject']['name']
        self.collection_in_filter_group = run_test_server.fixture('collections')['baz_file']['name']

        cw = arvados.CollectionWriter()

        cw.start_new_file('thing1.txt')
        cw.write("data 1")

        self.testcollection = cw.finish()
        self.test_manifest = cw.manifest_text()
        coll = self.api.collections().create(body={"manifest_text":self.test_manifest}).execute()
        self.test_manifest_pdh = coll['portable_data_hash']

    def runTest(self):
        self.make_mount(fuse.MagicDirectory)

        mount_ls = llfuse.listdir(self.mounttmp)
        self.assertIn('README', mount_ls)
        self.assertFalse(any(arvados.util.keep_locator_pattern.match(fn) or
                             arvados.util.uuid_pattern.match(fn)
                             for fn in mount_ls),
                         "new FUSE MagicDirectory has no collections or projects")
        self.assertDirContents(self.testcollection, ['thing1.txt'])
        self.assertDirContents(os.path.join('by_id', self.testcollection),
                               ['thing1.txt'])
        self.assertIn(self.collection_in_test_project,
                      llfuse.listdir(os.path.join(self.mounttmp, self.test_project)))
        self.assertIn(self.collection_in_test_project,
                      llfuse.listdir(os.path.join(self.mounttmp, 'by_id', self.test_project)))
        self.assertIn(self.collection_in_filter_group,
                      llfuse.listdir(os.path.join(self.mounttmp, self.filter_group)))
        self.assertIn(self.collection_in_filter_group,
                      llfuse.listdir(os.path.join(self.mounttmp, 'by_id', self.filter_group)))


        mount_ls = llfuse.listdir(self.mounttmp)
        self.assertIn('README', mount_ls)
        self.assertIn(self.testcollection, mount_ls)
        self.assertIn(self.testcollection,
                      llfuse.listdir(os.path.join(self.mounttmp, 'by_id')))
        self.assertIn(self.test_project, mount_ls)
        self.assertIn(self.test_project,
                      llfuse.listdir(os.path.join(self.mounttmp, 'by_id')))
        self.assertIn(self.filter_group,
                      llfuse.listdir(os.path.join(self.mounttmp, 'by_id')))

        with self.assertRaises(OSError):
            llfuse.listdir(os.path.join(self.mounttmp, 'by_id', self.non_project_group))

        files = {}
        files[os.path.join(self.mounttmp, self.testcollection, 'thing1.txt')] = 'data 1'

        for k, v in viewitems(files):
            with open(os.path.join(self.mounttmp, k), 'rb') as f:
                self.assertEqual(v, f.read().decode())


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
        for attempt in AssertWithTimeout(10):
            attempt(self.assertIn, 'fuse_test_tag', llfuse.listdir(self.mounttmp))
        self.assertDirContents('fuse_test_tag', [bar_uuid])

        baz_uuid = run_test_server.fixture('collections')['baz_file']['uuid']
        l = self.tag_collection(baz_uuid, 'fuse_test_tag')
        for attempt in AssertWithTimeout(10):
            attempt(self.assertDirContents, 'fuse_test_tag', [bar_uuid, baz_uuid])

        self.api.links().delete(uuid=l['uuid']).execute()
        for attempt in AssertWithTimeout(10):
            attempt(self.assertDirContents, 'fuse_test_tag', [bar_uuid])


def fuseSharedTestHelper(mounttmp):
    class Test(unittest.TestCase):
        def runTest(self):
            # Double check that we can open and read objects in this folder as a file,
            # and that its contents are what we expect.
            baz_path = os.path.join(
                mounttmp,
                'FUSE User',
                'FUSE Test Project',
                'collection in FUSE project',
                'baz')
            with open(baz_path) as f:
                self.assertEqual("baz", f.read())

            # check mtime on collection
            st = os.stat(baz_path)
            try:
                mtime = st.st_mtime_ns // 1000000000
            except AttributeError:
                mtime = st.st_mtime
            self.assertEqual(mtime, 1391448174)

            # shared_dirs is a list of the directories exposed
            # by fuse.SharedDirectory (i.e. any object visible
            # to the current user)
            shared_dirs = llfuse.listdir(mounttmp)
            shared_dirs.sort()
            self.assertIn('FUSE User', shared_dirs)

            # fuse_user_objs is a list of the objects owned by the FUSE
            # test user (which present as files in the 'FUSE User'
            # directory)
            fuse_user_objs = llfuse.listdir(os.path.join(mounttmp, 'FUSE User'))
            fuse_user_objs.sort()
            self.assertEqual(['FUSE Test Project',                    # project owned by user
                              'collection #1 owned by FUSE',          # collection owned by user
                              'collection #2 owned by FUSE'          # collection owned by user
                          ], fuse_user_objs)

            # test_proj_files is a list of the files in the FUSE Test Project.
            test_proj_files = llfuse.listdir(os.path.join(mounttmp, 'FUSE User', 'FUSE Test Project'))
            test_proj_files.sort()
            self.assertEqual(['collection in FUSE project'
                          ], test_proj_files)


    Test().runTest()

@parameterized.parameterized_class([{"disk_cache": True}, {"disk_cache": False}])
class FuseSharedTest(MountTestBase):
    def runTest(self):
        self.make_mount(fuse.SharedDirectory,
                        exclude=self.api.users().current().execute()['uuid'])
        keep = arvados.keep.KeepClient()
        keep.put("baz".encode())

        self.pool.apply(fuseSharedTestHelper, (self.mounttmp,))


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
        for name, item in viewitems(run_test_server.fixture('collections')):
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


def fuseModifyFileTestHelperReadStartContents(mounttmp):
    class Test(unittest.TestCase):
        def runTest(self):
            d1 = llfuse.listdir(mounttmp)
            self.assertEqual(["file1.txt"], d1)
            with open(os.path.join(mounttmp, "file1.txt")) as f:
                self.assertEqual("blub", f.read())
    Test().runTest()

def fuseModifyFileTestHelperReadEndContents(mounttmp):
    class Test(unittest.TestCase):
        def runTest(self):
            d1 = llfuse.listdir(mounttmp)
            self.assertEqual(["file1.txt"], d1)
            with open(os.path.join(mounttmp, "file1.txt")) as f:
                self.assertEqual("plnp", f.read())
    Test().runTest()

@parameterized.parameterized_class([{"disk_cache": True}, {"disk_cache": False}])
class FuseModifyFileTest(MountTestBase):
    def runTest(self):
        collection = arvados.collection.Collection(api_client=self.api)
        with collection.open("file1.txt", "w") as f:
            f.write("blub")

        collection.save_new()

        m = self.make_mount(fuse.CollectionDirectory)
        with llfuse.lock:
            m.new_collection(collection.api_response(), collection)

        self.pool.apply(fuseModifyFileTestHelperReadStartContents, (self.mounttmp,))

        with collection.open("file1.txt", "w") as f:
            f.write("plnp")

        self.pool.apply(fuseModifyFileTestHelperReadEndContents, (self.mounttmp,))


@parameterized.parameterized_class([{"disk_cache": True}, {"disk_cache": False}])
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


@parameterized.parameterized_class([{"disk_cache": True}, {"disk_cache": False}])
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


def fuseCreateFileTestHelper(mounttmp):
    class Test(unittest.TestCase):
        def runTest(self):
            with open(os.path.join(mounttmp, "file1.txt"), "w") as f:
                pass
    Test().runTest()

@parameterized.parameterized_class([{"disk_cache": True}, {"disk_cache": False}])
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

        self.pool.apply(fuseCreateFileTestHelper, (self.mounttmp,))

        self.assertIn("file1.txt", collection)

        d1 = llfuse.listdir(self.mounttmp)
        self.assertEqual(["file1.txt"], d1)

        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()
        assertRegex(self, collection2["manifest_text"],
            r'\. d41d8cd98f00b204e9800998ecf8427e\+0\+A\S+ 0:0:file1\.txt$')


def fuseWriteFileTestHelperWriteFile(mounttmp):
    class Test(unittest.TestCase):
        def runTest(self):
            with open(os.path.join(mounttmp, "file1.txt"), "w") as f:
                f.write("Hello world!")
    Test().runTest()

def fuseWriteFileTestHelperReadFile(mounttmp):
    class Test(unittest.TestCase):
        def runTest(self):
            with open(os.path.join(mounttmp, "file1.txt"), "r") as f:
                self.assertEqual(f.read(), "Hello world!")
    Test().runTest()

@parameterized.parameterized_class([{"disk_cache": True}, {"disk_cache": False}])
class FuseWriteFileTest(MountTestBase):
    def runTest(self):
        collection = arvados.collection.Collection(api_client=self.api)
        collection.save_new()

        m = self.make_mount(fuse.CollectionDirectory)
        with llfuse.lock:
            m.new_collection(collection.api_response(), collection)
        self.assertTrue(m.writable())

        self.assertNotIn("file1.txt", collection)

        self.assertEqual(0, self.operations.write_counter.get())
        self.pool.apply(fuseWriteFileTestHelperWriteFile, (self.mounttmp,))
        self.assertEqual(12, self.operations.write_counter.get())

        with collection.open("file1.txt") as f:
            self.assertEqual(f.read(), "Hello world!")

        self.assertEqual(0, self.operations.read_counter.get())
        self.pool.apply(fuseWriteFileTestHelperReadFile, (self.mounttmp,))
        self.assertEqual(12, self.operations.read_counter.get())

        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()
        assertRegex(self, collection2["manifest_text"],
            r'\. 86fb269d190d2c85f6e0468ceca42a20\+12\+A\S+ 0:12:file1\.txt$')


def fuseUpdateFileTestHelper(mounttmp):
    class Test(unittest.TestCase):
        def runTest(self):
            with open(os.path.join(mounttmp, "file1.txt"), "w") as f:
                f.write("Hello world!")

            with open(os.path.join(mounttmp, "file1.txt"), "r+") as f:
                fr = f.read()
                self.assertEqual(fr, "Hello world!")
                f.seek(0)
                f.write("Hola mundo!")
                f.seek(0)
                fr = f.read()
                self.assertEqual(fr, "Hola mundo!!")

            with open(os.path.join(mounttmp, "file1.txt"), "r") as f:
                self.assertEqual(f.read(), "Hola mundo!!")

    Test().runTest()

@parameterized.parameterized_class([{"disk_cache": True}, {"disk_cache": False}])
class FuseUpdateFileTest(MountTestBase):
    def runTest(self):
        collection = arvados.collection.Collection(api_client=self.api)
        collection.save_new()

        m = self.make_mount(fuse.CollectionDirectory)
        with llfuse.lock:
            m.new_collection(collection.api_response(), collection)
        self.assertTrue(m.writable())

        # See note in MountTestBase.setUp
        self.pool.apply(fuseUpdateFileTestHelper, (self.mounttmp,))

        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()
        assertRegex(self, collection2["manifest_text"],
            r'\. daaef200ebb921e011e3ae922dd3266b\+11\+A\S+ 86fb269d190d2c85f6e0468ceca42a20\+12\+A\S+ 0:11:file1\.txt 22:1:file1\.txt$')


def fuseMkdirTestHelper(mounttmp):
    class Test(unittest.TestCase):
        def runTest(self):
            with self.assertRaises(IOError):
                with open(os.path.join(mounttmp, "testdir", "file1.txt"), "w") as f:
                    f.write("Hello world!")

            os.mkdir(os.path.join(mounttmp, "testdir"))

            with self.assertRaises(OSError):
                os.mkdir(os.path.join(mounttmp, "testdir"))

            d1 = llfuse.listdir(mounttmp)
            self.assertEqual(["testdir"], d1)

            with open(os.path.join(mounttmp, "testdir", "file1.txt"), "w") as f:
                f.write("Hello world!")

            d1 = llfuse.listdir(os.path.join(mounttmp, "testdir"))
            self.assertEqual(["file1.txt"], d1)

    Test().runTest()

class FuseMkdirTest(MountTestBase):
    def runTest(self):
        collection = arvados.collection.Collection(api_client=self.api)
        collection.save_new()

        m = self.make_mount(fuse.CollectionDirectory)
        with llfuse.lock:
            m.new_collection(collection.api_response(), collection)
        self.assertTrue(m.writable())

        self.pool.apply(fuseMkdirTestHelper, (self.mounttmp,))

        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()
        assertRegex(self, collection2["manifest_text"],
            r'\./testdir 86fb269d190d2c85f6e0468ceca42a20\+12\+A\S+ 0:12:file1\.txt$')


def fuseRmTestHelperWriteFile(mounttmp):
    class Test(unittest.TestCase):
        def runTest(self):
            os.mkdir(os.path.join(mounttmp, "testdir"))

            with open(os.path.join(mounttmp, "testdir", "file1.txt"), "w") as f:
                f.write("Hello world!")

    Test().runTest()

def fuseRmTestHelperDeleteFile(mounttmp):
    class Test(unittest.TestCase):
        def runTest(self):
            # Can't delete because it's not empty
            with self.assertRaises(OSError):
                os.rmdir(os.path.join(mounttmp, "testdir"))

            d1 = llfuse.listdir(os.path.join(mounttmp, "testdir"))
            self.assertEqual(["file1.txt"], d1)

            # Delete file
            os.remove(os.path.join(mounttmp, "testdir", "file1.txt"))

            # Make sure it's empty
            d1 = llfuse.listdir(os.path.join(mounttmp, "testdir"))
            self.assertEqual([], d1)

            # Try to delete it again
            with self.assertRaises(OSError):
                os.remove(os.path.join(mounttmp, "testdir", "file1.txt"))

    Test().runTest()

def fuseRmTestHelperRmdir(mounttmp):
    class Test(unittest.TestCase):
        def runTest(self):
            # Should be able to delete now that it is empty
            os.rmdir(os.path.join(mounttmp, "testdir"))

            # Make sure it's empty
            d1 = llfuse.listdir(os.path.join(mounttmp))
            self.assertEqual([], d1)

            # Try to delete it again
            with self.assertRaises(OSError):
                os.rmdir(os.path.join(mounttmp, "testdir"))

    Test().runTest()

class FuseRmTest(MountTestBase):
    def runTest(self):
        collection = arvados.collection.Collection(api_client=self.api)
        collection.save_new()

        m = self.make_mount(fuse.CollectionDirectory)
        with llfuse.lock:
            m.new_collection(collection.api_response(), collection)
        self.assertTrue(m.writable())

        self.pool.apply(fuseRmTestHelperWriteFile, (self.mounttmp,))

        # Starting manifest
        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()
        assertRegex(self, collection2["manifest_text"],
            r'\./testdir 86fb269d190d2c85f6e0468ceca42a20\+12\+A\S+ 0:12:file1\.txt$')
        self.pool.apply(fuseRmTestHelperDeleteFile, (self.mounttmp,))

        # Empty directories are represented by an empty file named "."
        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()
        assertRegex(self, collection2["manifest_text"],
                                 r'./testdir d41d8cd98f00b204e9800998ecf8427e\+0\+A\S+ 0:0:\\056\n')

        self.pool.apply(fuseRmTestHelperRmdir, (self.mounttmp,))

        # manifest should be empty now.
        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()
        self.assertEqual(collection2["manifest_text"], "")


def fuseMvFileTestHelperWriteFile(mounttmp):
    class Test(unittest.TestCase):
        def runTest(self):
            os.mkdir(os.path.join(mounttmp, "testdir"))

            with open(os.path.join(mounttmp, "testdir", "file1.txt"), "w") as f:
                f.write("Hello world!")

    Test().runTest()

def fuseMvFileTestHelperMoveFile(mounttmp):
    class Test(unittest.TestCase):
        def runTest(self):
            d1 = llfuse.listdir(os.path.join(mounttmp))
            self.assertEqual(["testdir"], d1)
            d1 = llfuse.listdir(os.path.join(mounttmp, "testdir"))
            self.assertEqual(["file1.txt"], d1)

            os.rename(os.path.join(mounttmp, "testdir", "file1.txt"), os.path.join(mounttmp, "file1.txt"))

            d1 = llfuse.listdir(os.path.join(mounttmp))
            self.assertEqual(["file1.txt", "testdir"], sorted(d1))
            d1 = llfuse.listdir(os.path.join(mounttmp, "testdir"))
            self.assertEqual([], d1)

    Test().runTest()

class FuseMvFileTest(MountTestBase):
    def runTest(self):
        collection = arvados.collection.Collection(api_client=self.api)
        collection.save_new()

        m = self.make_mount(fuse.CollectionDirectory)
        with llfuse.lock:
            m.new_collection(collection.api_response(), collection)
        self.assertTrue(m.writable())

        self.pool.apply(fuseMvFileTestHelperWriteFile, (self.mounttmp,))

        # Starting manifest
        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()
        assertRegex(self, collection2["manifest_text"],
            r'\./testdir 86fb269d190d2c85f6e0468ceca42a20\+12\+A\S+ 0:12:file1\.txt$')

        self.pool.apply(fuseMvFileTestHelperMoveFile, (self.mounttmp,))

        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()
        assertRegex(self, collection2["manifest_text"],
            r'\. 86fb269d190d2c85f6e0468ceca42a20\+12\+A\S+ 0:12:file1\.txt\n\./testdir d41d8cd98f00b204e9800998ecf8427e\+0\+A\S+ 0:0:\\056\n')


def fuseRenameTestHelper(mounttmp):
    class Test(unittest.TestCase):
        def runTest(self):
            os.mkdir(os.path.join(mounttmp, "testdir"))

            with open(os.path.join(mounttmp, "testdir", "file1.txt"), "w") as f:
                f.write("Hello world!")

    Test().runTest()

class FuseRenameTest(MountTestBase):
    def runTest(self):
        collection = arvados.collection.Collection(api_client=self.api)
        collection.save_new()

        m = self.make_mount(fuse.CollectionDirectory)
        with llfuse.lock:
            m.new_collection(collection.api_response(), collection)
        self.assertTrue(m.writable())

        self.pool.apply(fuseRenameTestHelper, (self.mounttmp,))

        # Starting manifest
        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()
        assertRegex(self, collection2["manifest_text"],
            r'\./testdir 86fb269d190d2c85f6e0468ceca42a20\+12\+A\S+ 0:12:file1\.txt$')

        d1 = llfuse.listdir(os.path.join(self.mounttmp))
        self.assertEqual(["testdir"], d1)
        d1 = llfuse.listdir(os.path.join(self.mounttmp, "testdir"))
        self.assertEqual(["file1.txt"], d1)

        os.rename(os.path.join(self.mounttmp, "testdir"), os.path.join(self.mounttmp, "testdir2"))

        d1 = llfuse.listdir(os.path.join(self.mounttmp))
        self.assertEqual(["testdir2"], sorted(d1))
        d1 = llfuse.listdir(os.path.join(self.mounttmp, "testdir2"))
        self.assertEqual(["file1.txt"], d1)

        collection2 = self.api.collections().get(uuid=collection.manifest_locator()).execute()
        assertRegex(self, collection2["manifest_text"],
            r'\./testdir2 86fb269d190d2c85f6e0468ceca42a20\+12\+A\S+ 0:12:file1\.txt$')


class FuseUpdateFromEventTest(MountTestBase):
    def runTest(self):
        collection = arvados.collection.Collection(api_client=self.api)
        collection.save_new()

        m = self.make_mount(fuse.CollectionDirectory)
        with llfuse.lock:
            m.new_collection(collection.api_response(), collection)

        self.operations.listen_for_events()

        d1 = llfuse.listdir(os.path.join(self.mounttmp))
        self.assertEqual([], sorted(d1))

        with arvados.collection.Collection(collection.manifest_locator(), api_client=self.api) as collection2:
            with collection2.open("file1.txt", "w") as f:
                f.write("foo")

        for attempt in AssertWithTimeout(10):
            attempt(self.assertEqual, ["file1.txt"], llfuse.listdir(os.path.join(self.mounttmp)))


class FuseDeleteProjectEventTest(MountTestBase):
    def runTest(self):

        aproject = self.api.groups().create(body={
            "name": "aproject",
            "group_class": "project"
        }).execute()

        bproject = self.api.groups().create(body={
            "name": "bproject",
            "group_class": "project",
            "owner_uuid": aproject["uuid"]
        }).execute()

        self.make_mount(fuse.ProjectDirectory,
                        project_object=self.api.users().current().execute())

        self.operations.listen_for_events()

        d1 = llfuse.listdir(os.path.join(self.mounttmp, "aproject"))
        self.assertEqual(["bproject"], sorted(d1))

        self.api.groups().delete(uuid=bproject["uuid"]).execute()

        for attempt in AssertWithTimeout(10):
            attempt(self.assertEqual, [], llfuse.listdir(os.path.join(self.mounttmp, "aproject")))


def fuseFileConflictTestHelper(mounttmp, uuid, keeptmp, settings):
    class Test(unittest.TestCase):
        def runTest(self):
            os.environ['KEEP_LOCAL_STORE'] = keeptmp

            with open(os.path.join(mounttmp, "file1.txt"), "w") as f:
                with arvados.collection.Collection(uuid, api_client=arvados.api_from_config('v1', apiconfig=settings)) as collection2:
                    with collection2.open("file1.txt", "w") as f2:
                        f2.write("foo")
                f.write("bar")

            d1 = sorted(llfuse.listdir(os.path.join(mounttmp)))
            self.assertEqual(len(d1), 2)

            with open(os.path.join(mounttmp, "file1.txt"), "r") as f:
                self.assertEqual(f.read(), "bar")

            assertRegex(self, d1[1],
                r'file1\.txt~\d\d\d\d\d\d\d\d-\d\d\d\d\d\d~conflict~')

            with open(os.path.join(mounttmp, d1[1]), "r") as f:
                self.assertEqual(f.read(), "foo")

    Test().runTest()

class FuseFileConflictTest(MountTestBase):
    def runTest(self):
        collection = arvados.collection.Collection(api_client=self.api)
        collection.save_new()

        m = self.make_mount(fuse.CollectionDirectory)
        with llfuse.lock:
            m.new_collection(collection.api_response(), collection)

        d1 = llfuse.listdir(os.path.join(self.mounttmp))
        self.assertEqual([], sorted(d1))

        # See note in MountTestBase.setUp
        self.pool.apply(fuseFileConflictTestHelper, (self.mounttmp, collection.manifest_locator(), self.keeptmp, arvados.config.settings()))


def fuseUnlinkOpenFileTest(mounttmp):
    class Test(unittest.TestCase):
        def runTest(self):
            with open(os.path.join(mounttmp, "file1.txt"), "w+") as f:
                f.write("foo")

                d1 = llfuse.listdir(os.path.join(mounttmp))
                self.assertEqual(["file1.txt"], sorted(d1))

                os.remove(os.path.join(mounttmp, "file1.txt"))

                d1 = llfuse.listdir(os.path.join(mounttmp))
                self.assertEqual([], sorted(d1))

                f.seek(0)
                self.assertEqual(f.read(), "foo")
                f.write("bar")

                f.seek(0)
                self.assertEqual(f.read(), "foobar")

    Test().runTest()

class FuseUnlinkOpenFileTest(MountTestBase):
    def runTest(self):
        collection = arvados.collection.Collection(api_client=self.api)
        collection.save_new()

        m = self.make_mount(fuse.CollectionDirectory)
        with llfuse.lock:
            m.new_collection(collection.api_response(), collection)

        # See note in MountTestBase.setUp
        self.pool.apply(fuseUnlinkOpenFileTest, (self.mounttmp,))

        self.assertEqual(collection.manifest_text(), "")


def fuseMvFileBetweenCollectionsTest1(mounttmp, uuid1, uuid2):
    class Test(unittest.TestCase):
        def runTest(self):
            with open(os.path.join(mounttmp, uuid1, "file1.txt"), "w") as f:
                f.write("Hello world!")

            d1 = os.listdir(os.path.join(mounttmp, uuid1))
            self.assertEqual(["file1.txt"], sorted(d1))
            d1 = os.listdir(os.path.join(mounttmp, uuid2))
            self.assertEqual([], sorted(d1))

    Test().runTest()

def fuseMvFileBetweenCollectionsTest2(mounttmp, uuid1, uuid2):
    class Test(unittest.TestCase):
        def runTest(self):
            os.rename(os.path.join(mounttmp, uuid1, "file1.txt"), os.path.join(mounttmp, uuid2, "file2.txt"))

            d1 = os.listdir(os.path.join(mounttmp, uuid1))
            self.assertEqual([], sorted(d1))
            d1 = os.listdir(os.path.join(mounttmp, uuid2))
            self.assertEqual(["file2.txt"], sorted(d1))

    Test().runTest()

class FuseMvFileBetweenCollectionsTest(MountTestBase):
    def runTest(self):
        collection1 = arvados.collection.Collection(api_client=self.api)
        collection1.save_new()

        collection2 = arvados.collection.Collection(api_client=self.api)
        collection2.save_new()

        m = self.make_mount(fuse.MagicDirectory)

        # See note in MountTestBase.setUp
        self.pool.apply(fuseMvFileBetweenCollectionsTest1, (self.mounttmp,
                                                  collection1.manifest_locator(),
                                                  collection2.manifest_locator()))

        collection1.update()
        collection2.update()

        assertRegex(self, collection1.manifest_text(), r"\. 86fb269d190d2c85f6e0468ceca42a20\+12\+A\S+ 0:12:file1\.txt$")
        self.assertEqual(collection2.manifest_text(), "")

        self.pool.apply(fuseMvFileBetweenCollectionsTest2, (self.mounttmp,
                                                  collection1.manifest_locator(),
                                                  collection2.manifest_locator()))

        collection1.update()
        collection2.update()

        self.assertEqual(collection1.manifest_text(), "")
        assertRegex(self, collection2.manifest_text(), r"\. 86fb269d190d2c85f6e0468ceca42a20\+12\+A\S+ 0:12:file2\.txt$")

        collection1.stop_threads()
        collection2.stop_threads()


def fuseMvDirBetweenCollectionsTest1(mounttmp, uuid1, uuid2):
    class Test(unittest.TestCase):
        def runTest(self):
            os.mkdir(os.path.join(mounttmp, uuid1, "testdir"))
            with open(os.path.join(mounttmp, uuid1, "testdir", "file1.txt"), "w") as f:
                f.write("Hello world!")

            d1 = os.listdir(os.path.join(mounttmp, uuid1))
            self.assertEqual(["testdir"], sorted(d1))
            d1 = os.listdir(os.path.join(mounttmp, uuid1, "testdir"))
            self.assertEqual(["file1.txt"], sorted(d1))

            d1 = os.listdir(os.path.join(mounttmp, uuid2))
            self.assertEqual([], sorted(d1))

    Test().runTest()


def fuseMvDirBetweenCollectionsTest2(mounttmp, uuid1, uuid2):
    class Test(unittest.TestCase):
        def runTest(self):
            os.rename(os.path.join(mounttmp, uuid1, "testdir"), os.path.join(mounttmp, uuid2, "testdir2"))

            d1 = os.listdir(os.path.join(mounttmp, uuid1))
            self.assertEqual([], sorted(d1))

            d1 = os.listdir(os.path.join(mounttmp, uuid2))
            self.assertEqual(["testdir2"], sorted(d1))
            d1 = os.listdir(os.path.join(mounttmp, uuid2, "testdir2"))
            self.assertEqual(["file1.txt"], sorted(d1))

            with open(os.path.join(mounttmp, uuid2, "testdir2", "file1.txt"), "r") as f:
                self.assertEqual(f.read(), "Hello world!")

    Test().runTest()

class FuseMvDirBetweenCollectionsTest(MountTestBase):
    def runTest(self):
        collection1 = arvados.collection.Collection(api_client=self.api)
        collection1.save_new()

        collection2 = arvados.collection.Collection(api_client=self.api)
        collection2.save_new()

        m = self.make_mount(fuse.MagicDirectory)

        # See note in MountTestBase.setUp
        self.pool.apply(fuseMvDirBetweenCollectionsTest1, (self.mounttmp,
                                                  collection1.manifest_locator(),
                                                  collection2.manifest_locator()))

        collection1.update()
        collection2.update()

        assertRegex(self, collection1.manifest_text(), r"\./testdir 86fb269d190d2c85f6e0468ceca42a20\+12\+A\S+ 0:12:file1\.txt$")
        self.assertEqual(collection2.manifest_text(), "")

        self.pool.apply(fuseMvDirBetweenCollectionsTest2, (self.mounttmp,
                                                  collection1.manifest_locator(),
                                                  collection2.manifest_locator()))

        collection1.update()
        collection2.update()

        self.assertEqual(collection1.manifest_text(), "")
        assertRegex(self, collection2.manifest_text(), r"\./testdir2 86fb269d190d2c85f6e0468ceca42a20\+12\+A\S+ 0:12:file1\.txt$")

        collection1.stop_threads()
        collection2.stop_threads()

def fuseProjectMkdirTestHelper1(mounttmp):
    class Test(unittest.TestCase):
        def runTest(self):
            os.mkdir(os.path.join(mounttmp, "testcollection"))
            with self.assertRaises(OSError):
                os.mkdir(os.path.join(mounttmp, "testcollection"))
    Test().runTest()

def fuseProjectMkdirTestHelper2(mounttmp):
    class Test(unittest.TestCase):
        def runTest(self):
            with open(os.path.join(mounttmp, "testcollection", "file1.txt"), "w") as f:
                f.write("Hello world!")
            with self.assertRaises(OSError):
                os.rmdir(os.path.join(mounttmp, "testcollection"))
            os.remove(os.path.join(mounttmp, "testcollection", "file1.txt"))
            with self.assertRaises(OSError):
                os.remove(os.path.join(mounttmp, "testcollection"))
            os.rmdir(os.path.join(mounttmp, "testcollection"))
    Test().runTest()

class FuseProjectMkdirRmdirTest(MountTestBase):
    def runTest(self):
        self.make_mount(fuse.ProjectDirectory,
                        project_object=self.api.users().current().execute())

        d1 = llfuse.listdir(self.mounttmp)
        self.assertNotIn('testcollection', d1)

        self.pool.apply(fuseProjectMkdirTestHelper1, (self.mounttmp,))

        d1 = llfuse.listdir(self.mounttmp)
        self.assertIn('testcollection', d1)

        self.pool.apply(fuseProjectMkdirTestHelper2, (self.mounttmp,))

        d1 = llfuse.listdir(self.mounttmp)
        self.assertNotIn('testcollection', d1)


def fuseProjectMvTestHelper1(mounttmp):
    class Test(unittest.TestCase):
        def runTest(self):
            d1 = llfuse.listdir(mounttmp)
            self.assertNotIn('testcollection', d1)

            os.mkdir(os.path.join(mounttmp, "testcollection"))

            d1 = llfuse.listdir(mounttmp)
            self.assertIn('testcollection', d1)

            with self.assertRaises(OSError):
                os.rename(os.path.join(mounttmp, "testcollection"), os.path.join(mounttmp, 'Unrestricted public data'))

            os.rename(os.path.join(mounttmp, "testcollection"), os.path.join(mounttmp, 'Unrestricted public data', 'testcollection'))

            d1 = llfuse.listdir(mounttmp)
            self.assertNotIn('testcollection', d1)

            d1 = llfuse.listdir(os.path.join(mounttmp, 'Unrestricted public data'))
            self.assertIn('testcollection', d1)

    Test().runTest()

class FuseProjectMvTest(MountTestBase):
    def runTest(self):
        self.make_mount(fuse.ProjectDirectory,
                        project_object=self.api.users().current().execute())

        self.pool.apply(fuseProjectMvTestHelper1, (self.mounttmp,))


def fuseFsyncTestHelper(mounttmp, k):
    class Test(unittest.TestCase):
        def runTest(self):
            fd = os.open(os.path.join(mounttmp, k), os.O_RDONLY)
            os.fsync(fd)
            os.close(fd)

    Test().runTest()

class FuseFsyncTest(FuseMagicTest):
    def runTest(self):
        self.make_mount(fuse.MagicDirectory)
        self.pool.apply(fuseFsyncTestHelper, (self.mounttmp, self.testcollection))


class MagicDirApiError(FuseMagicTest):
    def setUp(self):
        api = mock.MagicMock()
        api.keep.block_cache = mock.MagicMock(cache_max=1)
        super(MagicDirApiError, self).setUp(api=api)
        api.collections().get().execute.side_effect = iter([
            Exception('API fail'),
            {
                "manifest_text": self.test_manifest,
                "portable_data_hash": self.test_manifest_pdh,
            },
        ])
        api.keep.get.side_effect = Exception('Keep fail')

    def runTest(self):
        with mock.patch('arvados_fuse.fresh.FreshBase._poll_time', new_callable=mock.PropertyMock, return_value=60) as mock_poll_time:
            self.make_mount(fuse.MagicDirectory)

            self.operations.inodes.inode_cache.cap = 1
            self.operations.inodes.inode_cache.min_entries = 2

            with self.assertRaises(OSError):
                llfuse.listdir(os.path.join(self.mounttmp, self.testcollection))

            llfuse.listdir(os.path.join(self.mounttmp, self.testcollection))


class SanitizeFilenameTest(MountTestBase):
    def test_sanitize_filename(self):
        pdir = fuse.ProjectDirectory(
            1, fuse.Inodes(None), self.api, 0, False, None,
            project_object=self.api.users().current().execute(),
        )
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
            self.assertEqual(f, pdir.sanitize_filename(f))
        for f in unacceptable:
            self.assertNotEqual(f, pdir.sanitize_filename(f))
            # The sanitized filename should be the same length, though.
            self.assertEqual(len(f), len(pdir.sanitize_filename(f)))
        # Special cases
        self.assertEqual("_", pdir.sanitize_filename(""))
        self.assertEqual("_", pdir.sanitize_filename("."))
        self.assertEqual("__", pdir.sanitize_filename(".."))


class FuseMagicTestPDHOnly(MountTestBase):
    def setUp(self, api=None):
        super(FuseMagicTestPDHOnly, self).setUp(api=api)

        cw = arvados.CollectionWriter()

        cw.start_new_file('thing1.txt')
        cw.write("data 1")

        self.testcollection = cw.finish()
        self.test_manifest = cw.manifest_text()
        created = self.api.collections().create(body={"manifest_text":self.test_manifest}).execute()
        self.testcollectionuuid = str(created['uuid'])

    def verify_pdh_only(self, pdh_only=False, skip_pdh_only=False):
        if skip_pdh_only is True:
            self.make_mount(fuse.MagicDirectory)    # in this case, the default by_id applies
        else:
            self.make_mount(fuse.MagicDirectory, pdh_only=pdh_only)

        mount_ls = llfuse.listdir(self.mounttmp)
        self.assertIn('README', mount_ls)
        self.assertFalse(any(arvados.util.keep_locator_pattern.match(fn) or
                             arvados.util.uuid_pattern.match(fn)
                             for fn in mount_ls),
                         "new FUSE MagicDirectory lists Collection")

        # look up using pdh should succeed in all cases
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

        for k, v in viewitems(files):
            with open(os.path.join(self.mounttmp, k), 'rb') as f:
                self.assertEqual(v, f.read().decode())

        # look up using uuid should fail when pdh_only is set
        if pdh_only is True:
            with self.assertRaises(OSError):
                self.assertDirContents(os.path.join('by_id', self.testcollectionuuid),
                               ['thing1.txt'])
        else:
            self.assertDirContents(os.path.join('by_id', self.testcollectionuuid),
                               ['thing1.txt'])

    def test_with_pdh_only_true(self):
        self.verify_pdh_only(pdh_only=True)

    def test_with_pdh_only_false(self):
        self.verify_pdh_only(pdh_only=False)

    def test_with_default_by_id(self):
        self.verify_pdh_only(skip_pdh_only=True)


class SlashSubstitutionTest(IntegrationTest):
    mnt_args = [
        '--read-write',
        '--mount-home', 'zzz',
        '--fsns', '[SLASH]'
    ]

    def setUp(self):
        super(SlashSubstitutionTest, self).setUp()

        self.api = arvados.safeapi.ThreadSafeApiCache(
            arvados.config.settings(),
            version='v1'
        )
        self.testcoll = self.api.collections().create(body={"name": "foo/bar/baz"}).execute()
        self.testcolleasy = self.api.collections().create(body={"name": "foo-bar-baz"}).execute()
        self.fusename = 'foo[SLASH]bar[SLASH]baz'

    @IntegrationTest.mount(argv=mnt_args)
    def test_slash_substitution_before_listing(self):
        self.pool_test(os.path.join(self.mnt, 'zzz'), self.fusename)
        self.checkContents()
    @staticmethod
    def _test_slash_substitution_before_listing(self, tmpdir, fusename):
        with open(os.path.join(tmpdir, 'foo-bar-baz', 'waz'), 'w') as f:
            f.write('xxx')
        with open(os.path.join(tmpdir, fusename, 'waz'), 'w') as f:
            f.write('foo')

    @IntegrationTest.mount(argv=mnt_args)
    @mock.patch('arvados.util.get_config_once')
    def test_slash_substitution_after_listing(self, get_config_once):
        get_config_once.return_value = {"Collections": {"ForwardSlashNameSubstitution": "[SLASH]"}}
        self.pool_test(os.path.join(self.mnt, 'zzz'), self.fusename)
        self.checkContents()
    @staticmethod
    def _test_slash_substitution_after_listing(self, tmpdir, fusename):
        with open(os.path.join(tmpdir, 'foo-bar-baz', 'waz'), 'w') as f:
            f.write('xxx')
        os.listdir(tmpdir)
        with open(os.path.join(tmpdir, fusename, 'waz'), 'w') as f:
            f.write('foo')

    def checkContents(self):
        self.assertRegexpMatches(self.api.collections().get(uuid=self.testcoll['uuid']).execute()['manifest_text'], ' acbd18db') # md5(foo)
        self.assertRegexpMatches(self.api.collections().get(uuid=self.testcolleasy['uuid']).execute()['manifest_text'], ' f561aaf6') # md5(xxx)

    @IntegrationTest.mount(argv=mnt_args)
    @mock.patch('arvados.util.get_config_once')
    def test_slash_substitution_conflict(self, get_config_once):
        self.testcollconflict = self.api.collections().create(body={"name": self.fusename}).execute()
        get_config_once.return_value = {"Collections": {"ForwardSlashNameSubstitution": "[SLASH]"}}
        self.pool_test(os.path.join(self.mnt, 'zzz'), self.fusename)
        self.assertRegexpMatches(self.api.collections().get(uuid=self.testcollconflict['uuid']).execute()['manifest_text'], ' acbd18db') # md5(foo)
        # foo/bar/baz collection unchanged, because it is masked by foo[SLASH]bar[SLASH]baz
        self.assertEqual(self.api.collections().get(uuid=self.testcoll['uuid']).execute()['manifest_text'], '')
    @staticmethod
    def _test_slash_substitution_conflict(self, tmpdir, fusename):
        with open(os.path.join(tmpdir, fusename, 'waz'), 'w') as f:
            f.write('foo')

class StorageClassesTest(IntegrationTest):
    mnt_args = [
        '--read-write',
        '--mount-home', 'homedir',
    ]

    def setUp(self):
        super(StorageClassesTest, self).setUp()
        self.api = arvados.safeapi.ThreadSafeApiCache(
            arvados.config.settings(),
            version='v1',
        )

    @IntegrationTest.mount(argv=mnt_args)
    def test_collection_default_storage_classes(self):
        coll_path = os.path.join(self.mnt, 'homedir', 'a_collection')
        self.api.collections().create(body={'name':'a_collection'}).execute()
        self.pool_test(coll_path)
    @staticmethod
    def _test_collection_default_storage_classes(self, coll):
        self.assertEqual(storage_classes_desired(coll), ['default'])

    @IntegrationTest.mount(argv=mnt_args+['--storage-classes', 'foo'])
    def test_collection_custom_storage_classes(self):
        coll_path = os.path.join(self.mnt, 'homedir', 'new_coll')
        os.mkdir(coll_path)
        self.pool_test(coll_path)
    @staticmethod
    def _test_collection_custom_storage_classes(self, coll):
        self.assertEqual(storage_classes_desired(coll), ['foo'])

def _readonlyCollectionTestHelper(mounttmp):
    f = open(os.path.join(mounttmp, 'thing1.txt'), 'rt')
    # Testing that close() doesn't raise an error.
    f.close()

class ReadonlyCollectionTest(MountTestBase):
    def setUp(self):
        super(ReadonlyCollectionTest, self).setUp()
        cw = arvados.collection.Collection()
        with cw.open('thing1.txt', 'wt') as f:
            f.write("data 1")
        cw.save_new(owner_uuid=run_test_server.fixture("groups")["aproject"]["uuid"])
        self.testcollection = cw.api_response()

    def runTest(self):
        settings = arvados.config.settings().copy()
        settings["ARVADOS_API_TOKEN"] = run_test_server.fixture("api_client_authorizations")["project_viewer"]["api_token"]
        self.api = arvados.safeapi.ThreadSafeApiCache(settings, version='v1')
        self.make_mount(fuse.CollectionDirectory, collection_record=self.testcollection, enable_write=False)

        self.pool.apply(_readonlyCollectionTestHelper, (self.mounttmp,))


@parameterized.parameterized_class([
    {'root_class': fusedir.ProjectDirectory, 'root_kwargs': {
        'project_object': run_test_server.fixture('users')['admin'],
    }},
    {'root_class': fusedir.ProjectDirectory, 'root_kwargs': {
        'project_object': run_test_server.fixture('groups')['public'],
    }},
])
class UnsupportedCreateTest(MountTestBase):
    root_class = None
    root_kwargs = {}

    def setUp(self):
        super().setUp()
        if 'prefs' in self.root_kwargs.get('project_object', ()):
            self.root_kwargs['project_object']['prefs'] = {}
        self.make_mount(self.root_class, **self.root_kwargs)
        # Make sure the directory knows about its top-level ents.
        os.listdir(self.mounttmp)

    def test_create(self):
        test_path = Path(self.mounttmp, 'test_create')
        with self.assertRaises(OSError) as exc_check:
            with test_path.open('w'):
                pass
        self.assertEqual(exc_check.exception.errno, errno.ENOTSUP)


# FIXME: IMO, for consistency with the "create inside a project" case,
# these operations should also return ENOTSUP instead of EPERM.
# Right now they're returning EPERM because the clasess' writable() method
# usually returns False, and the Operations class transforms that accordingly.
# However, for cases where the mount will never be writable, I think ENOTSUP
# is a clearer error: it lets the user know they can't fix the problem by
# adding permissions in Arvados, etc.
@parameterized.parameterized_class([
    {'root_class': fusedir.MagicDirectory,
     'preset_dir': 'by_id',
     'preset_file': 'README',
     },

    {'root_class': fusedir.SharedDirectory,
     'root_kwargs': {
         'exclude': run_test_server.fixture('users')['admin']['uuid'],
     },
     'preset_dir': 'Active User',
     },

    {'root_class': fusedir.TagDirectory,
     'root_kwargs': {
         'tag': run_test_server.fixture('links')['foo_collection_tag']['name'],
     },
     'preset_dir': run_test_server.fixture('collections')['foo_collection_in_aproject']['uuid'],
     },

    {'root_class': fusedir.TagsDirectory,
     'preset_dir': run_test_server.fixture('links')['foo_collection_tag']['name'],
     },
])
class UnsupportedOperationsTest(UnsupportedCreateTest):
    preset_dir = None
    preset_file = None

    def test_create(self):
        test_path = Path(self.mounttmp, 'test_create')
        with self.assertRaises(OSError) as exc_check:
            with test_path.open('w'):
                pass
        self.assertEqual(exc_check.exception.errno, errno.EPERM)

    def test_mkdir(self):
        test_path = Path(self.mounttmp, 'test_mkdir')
        with self.assertRaises(OSError) as exc_check:
            test_path.mkdir()
        self.assertEqual(exc_check.exception.errno, errno.EPERM)

    def test_rename(self):
        src_name = self.preset_dir or self.preset_file
        if src_name is None:
            return
        test_src = Path(self.mounttmp, src_name)
        test_dst = test_src.with_name('test_dst')
        with self.assertRaises(OSError) as exc_check:
            test_src.rename(test_dst)
        self.assertEqual(exc_check.exception.errno, errno.EPERM)

    def test_rmdir(self):
        if self.preset_dir is None:
            return
        test_path = Path(self.mounttmp, self.preset_dir)
        with self.assertRaises(OSError) as exc_check:
            test_path.rmdir()
        self.assertEqual(exc_check.exception.errno, errno.EPERM)

    def test_unlink(self):
        if self.preset_file is None:
            return
        test_path = Path(self.mounttmp, self.preset_file)
        with self.assertRaises(OSError) as exc_check:
            test_path.unlink()
        self.assertEqual(exc_check.exception.errno, errno.EPERM)
