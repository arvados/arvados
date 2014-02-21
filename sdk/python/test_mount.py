import unittest
import arvados
import arvados.fuse as fuse
import threading
import time
import os
import llfuse
import tempfile
import shutil
import subprocess
import glob

class FuseMountTest(unittest.TestCase):
    def setUp(self):
        self.keeptmp = tempfile.mkdtemp()
        os.environ['KEEP_LOCAL_STORE'] = self.keeptmp

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
        e = operations.inodes.add_entry(fuse.Directory(llfuse.ROOT_INODE))
        operations.inodes.load_collection(e, arvados.CollectionReader(arvados.Keep.get(self.testcollection)))

        self.mounttmp = tempfile.mkdtemp()

        llfuse.init(operations, self.mounttmp, [])
        t = threading.Thread(None, lambda: llfuse.main())
        t.start()

        # wait until the driver is finished initializing
        operations.initlock.wait()

        # now check some stuff
        d1 = os.listdir(self.mounttmp)
        d1.sort()
        self.assertEqual(d1, ['dir1', 'dir2', 'thing1.txt', 'thing2.txt'])

        d2 = os.listdir(os.path.join(self.mounttmp, 'dir1'))
        d2.sort()
        self.assertEqual(d2, ['thing3.txt', 'thing4.txt'])

        d3 = os.listdir(os.path.join(self.mounttmp, 'dir2'))
        d3.sort()
        self.assertEqual(d3, ['dir3', 'thing5.txt', 'thing6.txt'])

        d4 = os.listdir(os.path.join(self.mounttmp, 'dir2/dir3'))
        d4.sort()
        self.assertEqual(d4, ['thing7.txt', 'thing8.txt'])
        
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
                self.assertEqual(f.read(), v)
        

    def tearDown(self):
        # llfuse.close is buggy, so use fusermount instead.
        #llfuse.close(unmount=True)
        subprocess.call(["fusermount", "-u", self.mounttmp])

        os.rmdir(self.mounttmp)
        shutil.rmtree(self.keeptmp)

class FuseMagicTest(unittest.TestCase):
    def setUp(self):
        self.keeptmp = tempfile.mkdtemp()
        os.environ['KEEP_LOCAL_STORE'] = self.keeptmp

        cw = arvados.CollectionWriter()

        cw.start_new_file('thing1.txt')
        cw.write("data 1")

        self.testcollection = cw.finish()

    def runTest(self):
        # Create the request handler
        operations = fuse.Operations(os.getuid(), os.getgid())
        e = operations.inodes.add_entry(fuse.MagicDirectory(llfuse.ROOT_INODE, operations.inodes))

        self.mounttmp = tempfile.mkdtemp()

        llfuse.init(operations, self.mounttmp, [])
        t = threading.Thread(None, lambda: llfuse.main())
        t.start()

        # wait until the driver is finished initializing
        operations.initlock.wait()

        # now check some stuff
        d1 = os.listdir(self.mounttmp)
        d1.sort()
        self.assertEqual(d1, [])

        d2 = os.listdir(os.path.join(self.mounttmp, self.testcollection))
        d2.sort()
        self.assertEqual(d2, ['thing1.txt'])

        d3 = os.listdir(self.mounttmp)
        d3.sort()
        self.assertEqual(d3, [self.testcollection])
        
        files = {}
        files[os.path.join(self.mounttmp, self.testcollection, 'thing1.txt')] = 'data 1'

        for k, v in files.items():
            with open(os.path.join(self.mounttmp, k)) as f:
                self.assertEqual(f.read(), v)
        

    def tearDown(self):
        # llfuse.close is buggy, so use fusermount instead.
        #llfuse.close(unmount=True)
        subprocess.call(["fusermount", "-u", self.mounttmp])

        os.rmdir(self.mounttmp)
        shutil.rmtree(self.keeptmp)
