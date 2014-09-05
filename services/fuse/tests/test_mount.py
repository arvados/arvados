import unittest
import arvados
import arvados_fuse as fuse
import threading
import time
import os
import llfuse
import tempfile
import shutil
import subprocess
import glob
import run_test_server
import json

class MountTestBase(unittest.TestCase):
    def setUp(self):
        self.keeptmp = tempfile.mkdtemp()
        os.environ['KEEP_LOCAL_STORE'] = self.keeptmp
        self.mounttmp = tempfile.mkdtemp()

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

        self.testcollection = cw.finish()

    def runTest(self):
        # Create the request handler
        operations = fuse.Operations(os.getuid(), os.getgid())
        e = operations.inodes.add_entry(fuse.CollectionDirectory(llfuse.ROOT_INODE, operations.inodes, None, self.testcollection))

        llfuse.init(operations, self.mounttmp, [])
        t = threading.Thread(None, lambda: llfuse.main())
        t.start()

        # wait until the driver is finished initializing
        operations.initlock.wait()

        # now check some stuff
        d1 = os.listdir(self.mounttmp)
        d1.sort()
        self.assertEqual(['dir1', 'dir2', 'thing1.txt', 'thing2.txt'], d1)

        d2 = os.listdir(os.path.join(self.mounttmp, 'dir1'))
        d2.sort()
        self.assertEqual(['thing3.txt', 'thing4.txt'], d2)

        d3 = os.listdir(os.path.join(self.mounttmp, 'dir2'))
        d3.sort()
        self.assertEqual(['dir3', 'thing5.txt', 'thing6.txt'], d3)

        d4 = os.listdir(os.path.join(self.mounttmp, 'dir2/dir3'))
        d4.sort()
        self.assertEqual(['thing7.txt', 'thing8.txt'], d4)

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


class FuseMagicTest(MountTestBase):
    def setUp(self):
        super(FuseMagicTest, self).setUp()

        cw = arvados.CollectionWriter()

        cw.start_new_file('thing1.txt')
        cw.write("data 1")

        self.testcollection = cw.finish()

    def runTest(self):
        # Create the request handler
        operations = fuse.Operations(os.getuid(), os.getgid())
        e = operations.inodes.add_entry(fuse.MagicDirectory(llfuse.ROOT_INODE, operations.inodes, None))

        self.mounttmp = tempfile.mkdtemp()

        llfuse.init(operations, self.mounttmp, [])
        t = threading.Thread(None, lambda: llfuse.main())
        t.start()

        # wait until the driver is finished initializing
        operations.initlock.wait()

        # now check some stuff
        d1 = os.listdir(self.mounttmp)
        d1.sort()
        self.assertEqual(['README'], d1)

        d2 = os.listdir(os.path.join(self.mounttmp, self.testcollection))
        d2.sort()
        self.assertEqual(['thing1.txt'], d2)

        d3 = os.listdir(self.mounttmp)
        d3.sort()
        self.assertEqual(['README', self.testcollection], d3)

        files = {}
        files[os.path.join(self.mounttmp, self.testcollection, 'thing1.txt')] = 'data 1'

        for k, v in files.items():
            with open(os.path.join(self.mounttmp, k)) as f:
                self.assertEqual(v, f.read())

#
# Restore these tests when working on issue #3644
#
# class FuseTagsTest(MountTestBase):
#     def setUp(self):
#         super(FuseTagsTest, self).setUp()

#         cw = arvados.CollectionWriter()

#         cw.start_new_file('foo')
#         cw.write("foo")

#         self.testcollection = cw.finish()

#         run_test_server.run()

#     def runTest(self):
#         run_test_server.authorize_with("admin")
#         api = arvados.api('v1', cache=False)

#         operations = fuse.Operations(os.getuid(), os.getgid())
#         e = operations.inodes.add_entry(fuse.TagsDirectory(llfuse.ROOT_INODE, operations.inodes, api))

#         llfuse.init(operations, self.mounttmp, [])
#         t = threading.Thread(None, lambda: llfuse.main())
#         t.start()

#         # wait until the driver is finished initializing
#         operations.initlock.wait()

#         d1 = os.listdir(self.mounttmp)
#         d1.sort()
#         self.assertEqual(['foo_tag'], d1)

#         d2 = os.listdir(os.path.join(self.mounttmp, 'foo_tag'))
#         d2.sort()
#         self.assertEqual(['1f4b0bc7583c2a7f9102c395f4ffc5e3+45'], d2)

#         d3 = os.listdir(os.path.join(self.mounttmp, 'foo_tag', '1f4b0bc7583c2a7f9102c395f4ffc5e3+45'))
#         d3.sort()
#         self.assertEqual(['foo'], d3)

#         files = {}
#         files[os.path.join(self.mounttmp, 'foo_tag', '1f4b0bc7583c2a7f9102c395f4ffc5e3+45', 'foo')] = 'foo'

#         for k, v in files.items():
#             with open(os.path.join(self.mounttmp, k)) as f:
#                 self.assertEqual(v, f.read())


#     def tearDown(self):
#         run_test_server.stop()

#         super(FuseTagsTest, self).tearDown()

# class FuseTagsUpdateTestBase(MountTestBase):

#     def runRealTest(self):
#         run_test_server.authorize_with("admin")
#         api = arvados.api('v1', cache=False)

#         operations = fuse.Operations(os.getuid(), os.getgid())
#         e = operations.inodes.add_entry(fuse.TagsDirectory(llfuse.ROOT_INODE, operations.inodes, api, poll_time=1))

#         llfuse.init(operations, self.mounttmp, [])
#         t = threading.Thread(None, lambda: llfuse.main())
#         t.start()

#         # wait until the driver is finished initializing
#         operations.initlock.wait()

#         d1 = os.listdir(self.mounttmp)
#         d1.sort()
#         self.assertEqual(['foo_tag'], d1)

#         api.links().create(body={'link': {
#             'head_uuid': 'fa7aeb5140e2848d39b416daeef4ffc5+45',
#             'link_class': 'tag',
#             'name': 'bar_tag'
#         }}).execute()

#         time.sleep(1)

#         d2 = os.listdir(self.mounttmp)
#         d2.sort()
#         self.assertEqual(['bar_tag', 'foo_tag'], d2)

#         d3 = os.listdir(os.path.join(self.mounttmp, 'bar_tag'))
#         d3.sort()
#         self.assertEqual(['fa7aeb5140e2848d39b416daeef4ffc5+45'], d3)

#         l = api.links().create(body={'link': {
#             'head_uuid': 'ea10d51bcf88862dbcc36eb292017dfd+45',
#             'link_class': 'tag',
#             'name': 'bar_tag'
#         }}).execute()

#         time.sleep(1)

#         d4 = os.listdir(os.path.join(self.mounttmp, 'bar_tag'))
#         d4.sort()
#         self.assertEqual(['ea10d51bcf88862dbcc36eb292017dfd+45', 'fa7aeb5140e2848d39b416daeef4ffc5+45'], d4)

#         api.links().delete(uuid=l['uuid']).execute()

#         time.sleep(1)

#         d5 = os.listdir(os.path.join(self.mounttmp, 'bar_tag'))
#         d5.sort()
#         self.assertEqual(['fa7aeb5140e2848d39b416daeef4ffc5+45'], d5)


# class FuseTagsUpdateTestWebsockets(FuseTagsUpdateTestBase):
#     def setUp(self):
#         super(FuseTagsUpdateTestWebsockets, self).setUp()
#         run_test_server.run(True)

#     def runTest(self):
#         self.runRealTest()

#     def tearDown(self):
#         run_test_server.stop()
#         super(FuseTagsUpdateTestWebsockets, self).tearDown()


# class FuseTagsUpdateTestPoll(FuseTagsUpdateTestBase):
#     def setUp(self):
#         super(FuseTagsUpdateTestPoll, self).setUp()
#         run_test_server.run(False)

#     def runTest(self):
#         self.runRealTest()

#     def tearDown(self):
#         run_test_server.stop()
#         super(FuseTagsUpdateTestPoll, self).tearDown()


# class FuseGroupsTest(MountTestBase):
#     def setUp(self):
#         super(FuseGroupsTest, self).setUp()
#         run_test_server.run()

#     def runTest(self):
#         run_test_server.authorize_with("admin")
#         api = arvados.api('v1', cache=False)

#         operations = fuse.Operations(os.getuid(), os.getgid())
#         e = operations.inodes.add_entry(fuse.GroupsDirectory(llfuse.ROOT_INODE, operations.inodes, api))

#         llfuse.init(operations, self.mounttmp, [])
#         t = threading.Thread(None, lambda: llfuse.main())
#         t.start()

#         # wait until the driver is finished initializing
#         operations.initlock.wait()

#         d1 = os.listdir(self.mounttmp)
#         d1.sort()
#         self.assertIn('zzzzz-j7d0g-v955i6s2oi1cbso', d1)

#         d2 = os.listdir(os.path.join(self.mounttmp, 'zzzzz-j7d0g-v955i6s2oi1cbso'))
#         d2.sort()
#         self.assertEqual(['1f4b0bc7583c2a7f9102c395f4ffc5e3+45 added sometime',
#                           "I'm a job in a project",
#                           "I'm a template in a project",
#                           "zzzzz-j58dm-5gid26432uujf79",
#                           "zzzzz-j58dm-7r18rnd5nzhg5yk",
#                           "zzzzz-j58dm-ypsjlol9dofwijz",
#                           "zzzzz-j7d0g-axqo7eu9pwvna1x"
#                       ], d2)

#         d3 = os.listdir(os.path.join(self.mounttmp, 'zzzzz-j7d0g-v955i6s2oi1cbso', 'zzzzz-j7d0g-axqo7eu9pwvna1x'))
#         d3.sort()
#         self.assertEqual(["I'm in a subproject, too",
#                           "ea10d51bcf88862dbcc36eb292017dfd+45 added sometime",
#                           "zzzzz-j58dm-c40lddwcqqr1ffs"
#                       ], d3)

#         with open(os.path.join(self.mounttmp, 'zzzzz-j7d0g-v955i6s2oi1cbso', "I'm a template in a project")) as f:
#             j = json.load(f)
#             self.assertEqual("Two Part Pipeline Template", j['name'])

#     def tearDown(self):
#         run_test_server.stop()
#         super(FuseGroupsTest, self).tearDown()
