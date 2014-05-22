#!/usr/bin/env python

import errno
import os
import shutil
import tempfile
import unittest

class ArvadosBaseTestCase(unittest.TestCase):
    # This class provides common utility functions for our tests.

    def setUp(self):
        self._tempdirs = []

    def tearDown(self):
        for workdir in self._tempdirs:
            shutil.rmtree(workdir, ignore_errors=True)

    def make_tmpdir(self):
        self._tempdirs.append(tempfile.mkdtemp())
        return self._tempdirs[-1]

    def data_file(self, filename):
        try:
            basedir = os.path.dirname(__file__)
        except NameError:
            basedir = '.'
        return open(os.path.join(basedir, 'data', filename))


class ArvadosKeepLocalStoreTestCase(ArvadosBaseTestCase):
    def setUp(self):
        super(ArvadosKeepLocalStoreTestCase, self).setUp()
        self._orig_keep_local_store = os.environ.get('KEEP_LOCAL_STORE')
        os.environ['KEEP_LOCAL_STORE'] = self.make_tmpdir()

    def tearDown(self):
        if self._orig_keep_local_store is None:
            del os.environ['KEEP_LOCAL_STORE']
        else:
            os.environ['KEEP_LOCAL_STORE'] = self._orig_keep_local_store
        super(ArvadosKeepLocalStoreTestCase, self).tearDown()

    def build_directory_tree(self, tree):
        tree_root = self.make_tmpdir()
        for leaf in tree:
            path = os.path.join(tree_root, leaf)
            try:
                os.makedirs(os.path.dirname(path))
            except OSError as error:
                if error.errno != errno.EEXIST:
                    raise
            with open(path, 'w') as tmpfile:
                tmpfile.write(leaf)
        return tree_root

    def make_test_file(self, text="test"):
        testfile = tempfile.NamedTemporaryFile()
        testfile.write(text)
        testfile.flush()
        return testfile
