from __future__ import absolute_import
from future.utils import listitems
import io
import shutil
import tempfile

import arvados
import arvados.collection as collection
import arvados.commands.get as arv_get
from . import run_test_server

from . import arvados_testutil as tutil

class ArvadosGetTestCase(run_test_server.TestCaseWithServers,
                         tutil.VersionChecker):
    MAIN_SERVER = {}
    KEEP_SERVER = {}

    def setUp(self):
        super(ArvadosGetTestCase, self).setUp()
        self.tempdir = tempfile.mkdtemp()
        self.col_loc, self.col_pdh, self.col_manifest = self.write_test_collection()

    def tearDown(self):
        super(ArvadosGetTestCase, self).tearDown()
        shutil.rmtree(self.tempdir)

    def write_test_collection(self,
                              contents = {
                                  'foo.txt' : 'foo',
                                  'bar.txt' : 'bar',
                                  'subdir/baz.txt' : 'baz',
                              }):
        c = collection.Collection()
        for path, data in listitems(contents):
            with c.open(path, 'wb') as f:
                f.write(data)
        c.save_new()
        return (c.manifest_locator(), c.portable_data_hash(), c.manifest_text())

    def run_get(self, args):
        self.stdout = tutil.BytesIO()
        self.stderr = tutil.StringIO()
        return arv_get.main(args, self.stdout, self.stderr)

    def test_version_argument(self):
        with tutil.redirected_streams(
                stdout=tutil.StringIO, stderr=tutil.StringIO) as (out, err):
            with self.assertRaises(SystemExit):
                self.run_get(['--version'])
        self.assertVersionOutput(out, err)

    def test_get_single_file(self):
        # Get the file using the collection's locator
        r = self.run_get(["{}/subdir/baz.txt".format(self.col_loc), '-'])
        self.assertEqual(0, r)
        self.assertEqual(b'baz', self.stdout.getvalue())
        # Then, try by PDH
        r = self.run_get(["{}/subdir/baz.txt".format(self.col_pdh), '-'])
        self.assertEqual(0, r)
        self.assertEqual(b'baz', self.stdout.getvalue())

    def test_get_multiple_files(self):
        # Download the entire collection to the temp directory
        r = self.run_get(["{}/".format(self.col_loc), self.tempdir])
        self.assertEqual(0, r)
        with open("{}/foo.txt".format(self.tempdir), "r") as f:
            self.assertEqual("foo", f.read())
        with open("{}/bar.txt".format(self.tempdir), "r") as f:
            self.assertEqual("bar", f.read())
        with open("{}/subdir/baz.txt".format(self.tempdir), "r") as f:
            self.assertEqual("baz", f.read())

    def test_get_collection_manifest(self):
        # Get the collection manifest
        r = self.run_get([self.col_loc, self.tempdir])
        self.assertEqual(0, r)
        with open("{}/{}".format(self.tempdir, self.col_loc), "r") as f:
            self.assertEqual(self.col_manifest, f.read())

    def test_invalid_collection(self):
        # Asking for an invalid collection should generate an error.
        r = self.run_get(['this-uuid-seems-to-be-fake', self.tempdir])
        self.assertNotEqual(0, r)

    def test_invalid_file_request(self):
        # Asking for an inexistant file within a collection should generate an error.
        r = self.run_get(["{}/im-not-here.txt".format(self.col_loc), self.tempdir])
        self.assertNotEqual(0, r)

    def test_invalid_destination(self):
        # Asking to place the collection's files on a non existant directory
        # should generate an error.
        r = self.run_get([self.col_loc, "/fake/subdir/"])
        self.assertNotEqual(0, r)

    def test_preexistent_destination(self):
        # Asking to place a file with the same path as a local one should
        # generate an error and avoid overwrites.
        with open("{}/foo.txt".format(self.tempdir), "w") as f:
            f.write("another foo")
        r = self.run_get(["{}/foo.txt".format(self.col_loc), self.tempdir])
        self.assertNotEqual(0, r)
        with open("{}/foo.txt".format(self.tempdir), "r") as f:
            self.assertEqual("another foo", f.read())

