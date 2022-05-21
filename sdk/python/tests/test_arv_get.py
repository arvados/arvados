# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import absolute_import
from future.utils import listitems
import io
import logging
import mock
import os
import re
import shutil
import tempfile

import arvados
import arvados.collection as collection
import arvados.commands.get as arv_get
from . import run_test_server

from . import arvados_testutil as tutil
from .arvados_testutil import ArvadosBaseTestCase

class ArvadosGetTestCase(run_test_server.TestCaseWithServers,
                         tutil.VersionChecker,
                         ArvadosBaseTestCase):
    MAIN_SERVER = {}
    KEEP_SERVER = {}

    def setUp(self):
        super(ArvadosGetTestCase, self).setUp()
        self.tempdir = tempfile.mkdtemp()
        self.col_loc, self.col_pdh, self.col_manifest = self.write_test_collection()

        self.stdout = tutil.BytesIO()
        self.stderr = tutil.StringIO()
        self.loggingHandler = logging.StreamHandler(self.stderr)
        self.loggingHandler.setFormatter(logging.Formatter('%(levelname)s: %(message)s'))
        logging.getLogger().addHandler(self.loggingHandler)

    def tearDown(self):
        logging.getLogger().removeHandler(self.loggingHandler)
        super(ArvadosGetTestCase, self).tearDown()
        shutil.rmtree(self.tempdir)

    def write_test_collection(self,
                              strip_manifest=False,
                              contents = {
                                  'foo.txt' : 'foo',
                                  'bar.txt' : 'bar',
                                  'subdir/baz.txt' : 'baz',
                              }):
        api = arvados.api()
        c = collection.Collection(api_client=api)
        for path, data in listitems(contents):
            with c.open(path, 'wb') as f:
                f.write(data)
        c.save_new()

        api.close_connections()

        return (c.manifest_locator(),
                c.portable_data_hash(),
                c.manifest_text(strip=strip_manifest))

    def run_get(self, args):
        self.stdout.seek(0, 0)
        self.stdout.truncate(0)
        self.stderr.seek(0, 0)
        self.stderr.truncate(0)
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

    def test_get_block(self):
        # Get raw data using a block locator
        blk = re.search(' (acbd18\S+\+A\S+) ', self.col_manifest).group(1)
        r = self.run_get([blk, '-'])
        self.assertEqual(0, r)
        self.assertEqual(b'foo', self.stdout.getvalue())

    def test_get_multiple_files(self):
        # Download the entire collection to the temp directory
        r = self.run_get(["{}/".format(self.col_loc), self.tempdir])
        self.assertEqual(0, r)
        with open(os.path.join(self.tempdir, "foo.txt"), "r") as f:
            self.assertEqual("foo", f.read())
        with open(os.path.join(self.tempdir, "bar.txt"), "r") as f:
            self.assertEqual("bar", f.read())
        with open(os.path.join(self.tempdir, "subdir", "baz.txt"), "r") as f:
            self.assertEqual("baz", f.read())

    def test_get_collection_unstripped_manifest(self):
        dummy_token = "+Axxxxxxx"
        # Get the collection manifest by UUID
        r = self.run_get([self.col_loc, self.tempdir])
        self.assertEqual(0, r)
        m_from_collection = re.sub(r"\+A[0-9a-f@]+", dummy_token, self.col_manifest)
        with open(os.path.join(self.tempdir, self.col_loc), "r") as f:
            # Replace manifest tokens before comparison to avoid races
            m_from_file = re.sub(r"\+A[0-9a-f@]+", dummy_token, f.read())
            self.assertEqual(m_from_collection, m_from_file)
        # Get the collection manifest by PDH
        r = self.run_get([self.col_pdh, self.tempdir])
        self.assertEqual(0, r)
        with open(os.path.join(self.tempdir, self.col_pdh), "r") as f:
            # Replace manifest tokens before comparison to avoid races
            m_from_file = re.sub(r"\+A[0-9a-f@]+", dummy_token, f.read())
            self.assertEqual(m_from_collection, m_from_file)

    def test_get_collection_stripped_manifest(self):
        col_loc, col_pdh, col_manifest = self.write_test_collection(
            strip_manifest=True)
        # Get the collection manifest by UUID
        r = self.run_get(['--strip-manifest', col_loc, self.tempdir])
        self.assertEqual(0, r)
        with open(os.path.join(self.tempdir, col_loc), "r") as f:
            self.assertEqual(col_manifest, f.read())
        # Get the collection manifest by PDH
        r = self.run_get(['--strip-manifest', col_pdh, self.tempdir])
        self.assertEqual(0, r)
        with open(os.path.join(self.tempdir, col_pdh), "r") as f:
            self.assertEqual(col_manifest, f.read())

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
        with open(os.path.join(self.tempdir, "foo.txt"), "w") as f:
            f.write("another foo")
        r = self.run_get(["{}/foo.txt".format(self.col_loc), self.tempdir])
        self.assertNotEqual(0, r)
        with open(os.path.join(self.tempdir, "foo.txt"), "r") as f:
            self.assertEqual("another foo", f.read())

    def test_no_progress_when_stderr_not_a_tty(self):
        # Create a collection with a big file (>64MB) to force the progress
        # to be printed
        c = collection.Collection()
        with c.open('bigfile.txt', 'wb') as f:
            for _ in range(65):
                f.write("x" * 1024 * 1024)
        c.save_new()
        tmpdir = self.make_tmpdir()
        # Simulate a TTY stderr
        stderr = mock.MagicMock()
        stdout = tutil.BytesIO()

        # Confirm that progress is written to stderr when is a tty
        stderr.isatty.return_value = True
        r = arv_get.main(['{}/bigfile.txt'.format(c.manifest_locator()),
                          '{}/bigfile.txt'.format(tmpdir)],
                         stdout, stderr)
        self.assertEqual(0, r)
        self.assertEqual(b'', stdout.getvalue())
        self.assertTrue(stderr.write.called)

        # Clean up and reset stderr mock
        os.remove('{}/bigfile.txt'.format(tmpdir))
        stderr = mock.MagicMock()
        stdout = tutil.BytesIO()

        # Confirm that progress is not written to stderr when isn't a tty
        stderr.isatty.return_value = False
        r = arv_get.main(['{}/bigfile.txt'.format(c.manifest_locator()),
                          '{}/bigfile.txt'.format(tmpdir)],
                         stdout, stderr)
        self.assertEqual(0, r)
        self.assertEqual(b'', stdout.getvalue())
        self.assertFalse(stderr.write.called)

    request_id_regex = r'INFO: X-Request-Id: req-[a-z0-9]{20}\n'

    def test_request_id_logging_on(self):
        r = self.run_get(["-v", "{}/".format(self.col_loc), self.tempdir])
        self.assertEqual(0, r)
        self.assertRegex(self.stderr.getvalue(), self.request_id_regex)

    def test_request_id_logging_off(self):
        r = self.run_get(["{}/".format(self.col_loc), self.tempdir])
        self.assertEqual(0, r)
        self.assertNotRegex(self.stderr.getvalue(), self.request_id_regex)
