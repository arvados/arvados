#!/usr/bin/env python

import errno
import httplib
import httplib2
import mock
import os
import shutil
import tempfile
import unittest

# Use this hostname when you want to make sure the traffic will be
# instantly refused.  100::/64 is a dedicated black hole.
TEST_HOST = '100::'

skip_sleep = mock.patch('time.sleep', lambda n: None)  # clown'll eat me

def fake_httplib2_response(code, **headers):
    headers.update(status=str(code),
                   reason=httplib.responses.get(code, "Unknown Response"))
    return httplib2.Response(headers)

def mock_responses(body, *codes, **headers):
    return mock.patch('httplib2.Http.request', side_effect=(
            (fake_httplib2_response(code, **headers), body) for code in codes))

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
