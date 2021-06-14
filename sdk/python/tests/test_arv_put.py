# -*- coding: utf-8 -*-

# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import absolute_import
from __future__ import division
from future import standard_library
standard_library.install_aliases()
from builtins import str
from builtins import range
from functools import partial
import apiclient
import ciso8601
import datetime
import hashlib
import json
import logging
import mock
import os
import pwd
import random
import re
import select
import shutil
import signal
import subprocess
import sys
import tempfile
import time
import unittest
import uuid
import yaml

import arvados
import arvados.commands.put as arv_put
from . import arvados_testutil as tutil

from .arvados_testutil import ArvadosBaseTestCase, fake_httplib2_response
from . import run_test_server

class ArvadosPutResumeCacheTest(ArvadosBaseTestCase):
    CACHE_ARGSET = [
        [],
        ['/dev/null'],
        ['/dev/null', '--filename', 'empty'],
        ['/tmp']
        ]

    def tearDown(self):
        super(ArvadosPutResumeCacheTest, self).tearDown()
        try:
            self.last_cache.destroy()
        except AttributeError:
            pass

    def cache_path_from_arglist(self, arglist):
        return arv_put.ResumeCache.make_path(arv_put.parse_arguments(arglist))

    def test_cache_names_stable(self):
        for argset in self.CACHE_ARGSET:
            self.assertEqual(self.cache_path_from_arglist(argset),
                              self.cache_path_from_arglist(argset),
                              "cache name changed for {}".format(argset))

    def test_cache_names_unique(self):
        results = []
        for argset in self.CACHE_ARGSET:
            path = self.cache_path_from_arglist(argset)
            self.assertNotIn(path, results)
            results.append(path)

    def test_cache_names_simple(self):
        # The goal here is to make sure the filename doesn't use characters
        # reserved by the filesystem.  Feel free to adjust this regexp as
        # long as it still does that.
        bad_chars = re.compile(r'[^-\.\w]')
        for argset in self.CACHE_ARGSET:
            path = self.cache_path_from_arglist(argset)
            self.assertFalse(bad_chars.search(os.path.basename(path)),
                             "path too exotic: {}".format(path))

    def test_cache_names_ignore_argument_order(self):
        self.assertEqual(
            self.cache_path_from_arglist(['a', 'b', 'c']),
            self.cache_path_from_arglist(['c', 'a', 'b']))
        self.assertEqual(
            self.cache_path_from_arglist(['-', '--filename', 'stdin']),
            self.cache_path_from_arglist(['--filename', 'stdin', '-']))

    def test_cache_names_differ_for_similar_paths(self):
        # This test needs names at / that don't exist on the real filesystem.
        self.assertNotEqual(
            self.cache_path_from_arglist(['/_arvputtest1', '/_arvputtest2']),
            self.cache_path_from_arglist(['/_arvputtest1/_arvputtest2']))

    def test_cache_names_ignore_irrelevant_arguments(self):
        # Workaround: parse_arguments bails on --filename with a directory.
        path1 = self.cache_path_from_arglist(['/tmp'])
        args = arv_put.parse_arguments(['/tmp'])
        args.filename = 'tmp'
        path2 = arv_put.ResumeCache.make_path(args)
        self.assertEqual(path1, path2,
                         "cache path considered --filename for directory")
        self.assertEqual(
            self.cache_path_from_arglist(['-']),
            self.cache_path_from_arglist(['-', '--max-manifest-depth', '1']),
            "cache path considered --max-manifest-depth for file")

    def test_cache_names_treat_negative_manifest_depths_identically(self):
        base_args = ['/tmp', '--max-manifest-depth']
        self.assertEqual(
            self.cache_path_from_arglist(base_args + ['-1']),
            self.cache_path_from_arglist(base_args + ['-2']))

    def test_cache_names_treat_stdin_consistently(self):
        self.assertEqual(
            self.cache_path_from_arglist(['-', '--filename', 'test']),
            self.cache_path_from_arglist(['/dev/stdin', '--filename', 'test']))

    def test_cache_names_identical_for_synonymous_names(self):
        self.assertEqual(
            self.cache_path_from_arglist(['.']),
            self.cache_path_from_arglist([os.path.realpath('.')]))
        testdir = self.make_tmpdir()
        looplink = os.path.join(testdir, 'loop')
        os.symlink(testdir, looplink)
        self.assertEqual(
            self.cache_path_from_arglist([testdir]),
            self.cache_path_from_arglist([looplink]))

    def test_cache_names_different_by_api_host(self):
        config = arvados.config.settings()
        orig_host = config.get('ARVADOS_API_HOST')
        try:
            name1 = self.cache_path_from_arglist(['.'])
            config['ARVADOS_API_HOST'] = 'x' + (orig_host or 'localhost')
            self.assertNotEqual(name1, self.cache_path_from_arglist(['.']))
        finally:
            if orig_host is None:
                del config['ARVADOS_API_HOST']
            else:
                config['ARVADOS_API_HOST'] = orig_host

    @mock.patch('arvados.keep.KeepClient.head')
    def test_resume_cache_with_current_stream_locators(self, keep_client_head):
        keep_client_head.side_effect = [True]
        thing = {}
        thing['_current_stream_locators'] = ['098f6bcd4621d373cade4e832627b4f6+4', '1f253c60a2306e0ee12fb6ce0c587904+6']
        with tempfile.NamedTemporaryFile() as cachefile:
            self.last_cache = arv_put.ResumeCache(cachefile.name)
        self.last_cache.save(thing)
        self.last_cache.close()
        resume_cache = arv_put.ResumeCache(self.last_cache.filename)
        self.assertNotEqual(None, resume_cache)

    @mock.patch('arvados.keep.KeepClient.head')
    def test_resume_cache_with_finished_streams(self, keep_client_head):
        keep_client_head.side_effect = [True]
        thing = {}
        thing['_finished_streams'] = [['.', ['098f6bcd4621d373cade4e832627b4f6+4', '1f253c60a2306e0ee12fb6ce0c587904+6']]]
        with tempfile.NamedTemporaryFile() as cachefile:
            self.last_cache = arv_put.ResumeCache(cachefile.name)
        self.last_cache.save(thing)
        self.last_cache.close()
        resume_cache = arv_put.ResumeCache(self.last_cache.filename)
        self.assertNotEqual(None, resume_cache)

    @mock.patch('arvados.keep.KeepClient.head')
    def test_resume_cache_with_finished_streams_error_on_head(self, keep_client_head):
        keep_client_head.side_effect = Exception('Locator not found')
        thing = {}
        thing['_finished_streams'] = [['.', ['098f6bcd4621d373cade4e832627b4f6+4', '1f253c60a2306e0ee12fb6ce0c587904+6']]]
        with tempfile.NamedTemporaryFile() as cachefile:
            self.last_cache = arv_put.ResumeCache(cachefile.name)
        self.last_cache.save(thing)
        self.last_cache.close()
        resume_cache = arv_put.ResumeCache(self.last_cache.filename)
        self.assertNotEqual(None, resume_cache)
        resume_cache.check_cache()

    def test_basic_cache_storage(self):
        thing = ['test', 'list']
        with tempfile.NamedTemporaryFile() as cachefile:
            self.last_cache = arv_put.ResumeCache(cachefile.name)
        self.last_cache.save(thing)
        self.assertEqual(thing, self.last_cache.load())

    def test_empty_cache(self):
        with tempfile.NamedTemporaryFile() as cachefile:
            cache = arv_put.ResumeCache(cachefile.name)
        self.assertRaises(ValueError, cache.load)

    def test_cache_persistent(self):
        thing = ['test', 'list']
        path = os.path.join(self.make_tmpdir(), 'cache')
        cache = arv_put.ResumeCache(path)
        cache.save(thing)
        cache.close()
        self.last_cache = arv_put.ResumeCache(path)
        self.assertEqual(thing, self.last_cache.load())

    def test_multiple_cache_writes(self):
        thing = ['short', 'list']
        with tempfile.NamedTemporaryFile() as cachefile:
            self.last_cache = arv_put.ResumeCache(cachefile.name)
        # Start writing an object longer than the one we test, to make
        # sure the cache file gets truncated.
        self.last_cache.save(['long', 'long', 'list'])
        self.last_cache.save(thing)
        self.assertEqual(thing, self.last_cache.load())

    def test_cache_is_locked(self):
        with tempfile.NamedTemporaryFile() as cachefile:
            _ = arv_put.ResumeCache(cachefile.name)
            self.assertRaises(arv_put.ResumeCacheConflict,
                              arv_put.ResumeCache, cachefile.name)

    def test_cache_stays_locked(self):
        with tempfile.NamedTemporaryFile() as cachefile:
            self.last_cache = arv_put.ResumeCache(cachefile.name)
            path = cachefile.name
        self.last_cache.save('test')
        self.assertRaises(arv_put.ResumeCacheConflict,
                          arv_put.ResumeCache, path)

    def test_destroy_cache(self):
        cachefile = tempfile.NamedTemporaryFile(delete=False)
        try:
            cache = arv_put.ResumeCache(cachefile.name)
            cache.save('test')
            cache.destroy()
            try:
                arv_put.ResumeCache(cachefile.name)
            except arv_put.ResumeCacheConflict:
                self.fail("could not load cache after destroying it")
            self.assertRaises(ValueError, cache.load)
        finally:
            if os.path.exists(cachefile.name):
                os.unlink(cachefile.name)

    def test_restart_cache(self):
        path = os.path.join(self.make_tmpdir(), 'cache')
        cache = arv_put.ResumeCache(path)
        cache.save('test')
        cache.restart()
        self.assertRaises(ValueError, cache.load)
        self.assertRaises(arv_put.ResumeCacheConflict,
                          arv_put.ResumeCache, path)


class ArvPutUploadJobTest(run_test_server.TestCaseWithServers,
                          ArvadosBaseTestCase):

    def setUp(self):
        super(ArvPutUploadJobTest, self).setUp()
        run_test_server.authorize_with('active')
        # Temp files creation
        self.tempdir = tempfile.mkdtemp()
        subdir = os.path.join(self.tempdir, 'subdir')
        os.mkdir(subdir)
        data = "x" * 1024 # 1 KB
        for i in range(1, 5):
            with open(os.path.join(self.tempdir, str(i)), 'w') as f:
                f.write(data * i)
        with open(os.path.join(subdir, 'otherfile'), 'w') as f:
            f.write(data * 5)
        # Large temp file for resume test
        _, self.large_file_name = tempfile.mkstemp()
        fileobj = open(self.large_file_name, 'w')
        # Make sure to write just a little more than one block
        for _ in range((arvados.config.KEEP_BLOCK_SIZE>>20)+1):
            data = random.choice(['x', 'y', 'z']) * 1024 * 1024 # 1 MiB
            fileobj.write(data)
        fileobj.close()
        # Temp dir containing small files to be repacked
        self.small_files_dir = tempfile.mkdtemp()
        data = 'y' * 1024 * 1024 # 1 MB
        for i in range(1, 70):
            with open(os.path.join(self.small_files_dir, str(i)), 'w') as f:
                f.write(data + str(i))
        self.arvfile_write = getattr(arvados.arvfile.ArvadosFileWriter, 'write')
        # Temp dir to hold a symlink to other temp dir
        self.tempdir_with_symlink = tempfile.mkdtemp()
        os.symlink(self.tempdir, os.path.join(self.tempdir_with_symlink, 'linkeddir'))
        os.symlink(os.path.join(self.tempdir, '1'),
                   os.path.join(self.tempdir_with_symlink, 'linkedfile'))

    def tearDown(self):
        super(ArvPutUploadJobTest, self).tearDown()
        shutil.rmtree(self.tempdir)
        os.unlink(self.large_file_name)
        shutil.rmtree(self.small_files_dir)
        shutil.rmtree(self.tempdir_with_symlink)

    def test_symlinks_are_followed_by_default(self):
        self.assertTrue(os.path.islink(os.path.join(self.tempdir_with_symlink, 'linkeddir')))
        self.assertTrue(os.path.islink(os.path.join(self.tempdir_with_symlink, 'linkedfile')))
        cwriter = arv_put.ArvPutUploadJob([self.tempdir_with_symlink])
        cwriter.start(save_collection=False)
        self.assertIn('linkeddir', cwriter.manifest_text())
        self.assertIn('linkedfile', cwriter.manifest_text())
        cwriter.destroy_cache()

    def test_symlinks_are_not_followed_when_requested(self):
        self.assertTrue(os.path.islink(os.path.join(self.tempdir_with_symlink, 'linkeddir')))
        self.assertTrue(os.path.islink(os.path.join(self.tempdir_with_symlink, 'linkedfile')))
        cwriter = arv_put.ArvPutUploadJob([self.tempdir_with_symlink],
                                          follow_links=False)
        cwriter.start(save_collection=False)
        self.assertNotIn('linkeddir', cwriter.manifest_text())
        self.assertNotIn('linkedfile', cwriter.manifest_text())
        cwriter.destroy_cache()
        # Check for bug #17800: passed symlinks should also be ignored.
        linked_dir = os.path.join(self.tempdir_with_symlink, 'linkeddir')
        cwriter = arv_put.ArvPutUploadJob([linked_dir], follow_links=False)
        cwriter.start(save_collection=False)
        self.assertNotIn('linkeddir', cwriter.manifest_text())
        cwriter.destroy_cache()

    def test_no_empty_collection_saved(self):
        self.assertTrue(os.path.islink(os.path.join(self.tempdir_with_symlink, 'linkeddir')))
        linked_dir = os.path.join(self.tempdir_with_symlink, 'linkeddir')
        cwriter = arv_put.ArvPutUploadJob([linked_dir], follow_links=False)
        cwriter.start(save_collection=True)
        self.assertIsNone(cwriter.manifest_locator())
        self.assertEqual('', cwriter.manifest_text())
        cwriter.destroy_cache()

    def test_passing_nonexistant_path_raise_exception(self):
        uuid_str = str(uuid.uuid4())
        with self.assertRaises(arv_put.PathDoesNotExistError):
            arv_put.ArvPutUploadJob(["/this/path/does/not/exist/{}".format(uuid_str)])

    def test_writer_works_without_cache(self):
        cwriter = arv_put.ArvPutUploadJob(['/dev/null'], resume=False)
        cwriter.start(save_collection=False)
        self.assertEqual(". d41d8cd98f00b204e9800998ecf8427e+0 0:0:null\n", cwriter.manifest_text())

    def test_writer_works_with_cache(self):
        with tempfile.NamedTemporaryFile() as f:
            f.write(b'foo')
            f.flush()
            cwriter = arv_put.ArvPutUploadJob([f.name])
            cwriter.start(save_collection=False)
            self.assertEqual(0, cwriter.bytes_skipped)
            self.assertEqual(3, cwriter.bytes_written)
            # Don't destroy the cache, and start another upload
            cwriter_new = arv_put.ArvPutUploadJob([f.name])
            cwriter_new.start(save_collection=False)
            cwriter_new.destroy_cache()
            self.assertEqual(3, cwriter_new.bytes_skipped)
            self.assertEqual(3, cwriter_new.bytes_written)

    def make_progress_tester(self):
        progression = []
        def record_func(written, expected):
            progression.append((written, expected))
        return progression, record_func

    def test_progress_reporting(self):
        with tempfile.NamedTemporaryFile() as f:
            f.write(b'foo')
            f.flush()
            for expect_count in (None, 8):
                progression, reporter = self.make_progress_tester()
                cwriter = arv_put.ArvPutUploadJob([f.name],
                                                  reporter=reporter)
                cwriter.bytes_expected = expect_count
                cwriter.start(save_collection=False)
                cwriter.destroy_cache()
                self.assertIn((3, expect_count), progression)

    def test_writer_upload_directory(self):
        cwriter = arv_put.ArvPutUploadJob([self.tempdir])
        cwriter.start(save_collection=False)
        cwriter.destroy_cache()
        self.assertEqual(1024*(1+2+3+4+5), cwriter.bytes_written)

    def test_resume_large_file_upload(self):
        def wrapped_write(*args, **kwargs):
            data = args[1]
            # Exit only on last block
            if len(data) < arvados.config.KEEP_BLOCK_SIZE:
                # Simulate a checkpoint before quitting. Ensure block commit.
                self.writer._update(final=True)
                raise SystemExit("Simulated error")
            return self.arvfile_write(*args, **kwargs)

        with mock.patch('arvados.arvfile.ArvadosFileWriter.write',
                        autospec=True) as mocked_write:
            mocked_write.side_effect = wrapped_write
            writer = arv_put.ArvPutUploadJob([self.large_file_name],
                                             replication_desired=1)
            # We'll be accessing from inside the wrapper
            self.writer = writer
            with self.assertRaises(SystemExit):
                writer.start(save_collection=False)
            # Confirm that the file was partially uploaded
            self.assertGreater(writer.bytes_written, 0)
            self.assertLess(writer.bytes_written,
                            os.path.getsize(self.large_file_name))
        # Retry the upload
        writer2 = arv_put.ArvPutUploadJob([self.large_file_name],
                                          replication_desired=1)
        writer2.start(save_collection=False)
        self.assertEqual(writer.bytes_written + writer2.bytes_written - writer2.bytes_skipped,
                         os.path.getsize(self.large_file_name))
        writer2.destroy_cache()
        del(self.writer)

    # Test for bug #11002
    def test_graceful_exit_while_repacking_small_blocks(self):
        def wrapped_commit(*args, **kwargs):
            raise SystemExit("Simulated error")

        with mock.patch('arvados.arvfile._BlockManager.commit_bufferblock',
                        autospec=True) as mocked_commit:
            mocked_commit.side_effect = wrapped_commit
            # Upload a little more than 1 block, wrapped_commit will make the first block
            # commit to fail.
            # arv-put should not exit with an exception by trying to commit the collection
            # as it's in an inconsistent state.
            writer = arv_put.ArvPutUploadJob([self.small_files_dir],
                                             replication_desired=1)
            try:
                with self.assertRaises(SystemExit):
                    writer.start(save_collection=False)
            except arvados.arvfile.UnownedBlockError:
                self.fail("arv-put command is trying to use a corrupted BlockManager. See https://dev.arvados.org/issues/11002")
        writer.destroy_cache()

    def test_no_resume_when_asked(self):
        def wrapped_write(*args, **kwargs):
            data = args[1]
            # Exit only on last block
            if len(data) < arvados.config.KEEP_BLOCK_SIZE:
                # Simulate a checkpoint before quitting.
                self.writer._update()
                raise SystemExit("Simulated error")
            return self.arvfile_write(*args, **kwargs)

        with mock.patch('arvados.arvfile.ArvadosFileWriter.write',
                        autospec=True) as mocked_write:
            mocked_write.side_effect = wrapped_write
            writer = arv_put.ArvPutUploadJob([self.large_file_name],
                                             replication_desired=1)
            # We'll be accessing from inside the wrapper
            self.writer = writer
            with self.assertRaises(SystemExit):
                writer.start(save_collection=False)
            # Confirm that the file was partially uploaded
            self.assertGreater(writer.bytes_written, 0)
            self.assertLess(writer.bytes_written,
                            os.path.getsize(self.large_file_name))
        # Retry the upload, this time without resume
        writer2 = arv_put.ArvPutUploadJob([self.large_file_name],
                                          replication_desired=1,
                                          resume=False)
        writer2.start(save_collection=False)
        self.assertEqual(writer2.bytes_skipped, 0)
        self.assertEqual(writer2.bytes_written,
                         os.path.getsize(self.large_file_name))
        writer2.destroy_cache()
        del(self.writer)

    def test_no_resume_when_no_cache(self):
        def wrapped_write(*args, **kwargs):
            data = args[1]
            # Exit only on last block
            if len(data) < arvados.config.KEEP_BLOCK_SIZE:
                # Simulate a checkpoint before quitting.
                self.writer._update()
                raise SystemExit("Simulated error")
            return self.arvfile_write(*args, **kwargs)

        with mock.patch('arvados.arvfile.ArvadosFileWriter.write',
                        autospec=True) as mocked_write:
            mocked_write.side_effect = wrapped_write
            writer = arv_put.ArvPutUploadJob([self.large_file_name],
                                             replication_desired=1)
            # We'll be accessing from inside the wrapper
            self.writer = writer
            with self.assertRaises(SystemExit):
                writer.start(save_collection=False)
            # Confirm that the file was partially uploaded
            self.assertGreater(writer.bytes_written, 0)
            self.assertLess(writer.bytes_written,
                            os.path.getsize(self.large_file_name))
        # Retry the upload, this time without cache usage
        writer2 = arv_put.ArvPutUploadJob([self.large_file_name],
                                          replication_desired=1,
                                          resume=False,
                                          use_cache=False)
        writer2.start(save_collection=False)
        self.assertEqual(writer2.bytes_skipped, 0)
        self.assertEqual(writer2.bytes_written,
                         os.path.getsize(self.large_file_name))
        writer2.destroy_cache()
        del(self.writer)

    def test_dry_run_feature(self):
        def wrapped_write(*args, **kwargs):
            data = args[1]
            # Exit only on last block
            if len(data) < arvados.config.KEEP_BLOCK_SIZE:
                # Simulate a checkpoint before quitting.
                self.writer._update()
                raise SystemExit("Simulated error")
            return self.arvfile_write(*args, **kwargs)

        with mock.patch('arvados.arvfile.ArvadosFileWriter.write',
                        autospec=True) as mocked_write:
            mocked_write.side_effect = wrapped_write
            writer = arv_put.ArvPutUploadJob([self.large_file_name],
                                             replication_desired=1)
            # We'll be accessing from inside the wrapper
            self.writer = writer
            with self.assertRaises(SystemExit):
                writer.start(save_collection=False)
            # Confirm that the file was partially uploaded
            self.assertGreater(writer.bytes_written, 0)
            self.assertLess(writer.bytes_written,
                            os.path.getsize(self.large_file_name))
        with self.assertRaises(arv_put.ArvPutUploadIsPending):
            # Retry the upload using dry_run to check if there is a pending upload
            writer2 = arv_put.ArvPutUploadJob([self.large_file_name],
                                              replication_desired=1,
                                              dry_run=True)
        # Complete the pending upload
        writer3 = arv_put.ArvPutUploadJob([self.large_file_name],
                                          replication_desired=1)
        writer3.start(save_collection=False)
        with self.assertRaises(arv_put.ArvPutUploadNotPending):
            # Confirm there's no pending upload with dry_run=True
            writer4 = arv_put.ArvPutUploadJob([self.large_file_name],
                                              replication_desired=1,
                                              dry_run=True)
        # Test obvious cases
        with self.assertRaises(arv_put.ArvPutUploadIsPending):
            arv_put.ArvPutUploadJob([self.large_file_name],
                                    replication_desired=1,
                                    dry_run=True,
                                    resume=False,
                                    use_cache=False)
        with self.assertRaises(arv_put.ArvPutUploadIsPending):
            arv_put.ArvPutUploadJob([self.large_file_name],
                                    replication_desired=1,
                                    dry_run=True,
                                    resume=False)
        del(self.writer)

class CachedManifestValidationTest(ArvadosBaseTestCase):
    class MockedPut(arv_put.ArvPutUploadJob):
        def __init__(self, cached_manifest=None):
            self._state = arv_put.ArvPutUploadJob.EMPTY_STATE
            self._state['manifest'] = cached_manifest
            self._api_client = mock.MagicMock()
            self.logger = mock.MagicMock()
            self.num_retries = 1

    def datetime_to_hex(self, dt):
        return hex(int(time.mktime(dt.timetuple())))[2:]

    def setUp(self):
        super(CachedManifestValidationTest, self).setUp()
        self.block1 = "fdba98970961edb29f88241b9d99d890" # foo
        self.block2 = "37b51d194a7513e45b56f6524f2d51f2" # bar
        self.template = ". "+self.block1+"+3+Asignature@%s "+self.block2+"+3+Anothersignature@%s 0:3:foofile.txt 3:6:barfile.txt\n"

    def test_empty_cached_manifest_is_valid(self):
        put_mock = self.MockedPut()
        self.assertEqual(None, put_mock._state.get('manifest'))
        self.assertTrue(put_mock._cached_manifest_valid())
        put_mock._state['manifest'] = ''
        self.assertTrue(put_mock._cached_manifest_valid())

    def test_signature_cases(self):
        now = datetime.datetime.utcnow()
        yesterday = now - datetime.timedelta(days=1)
        lastweek = now - datetime.timedelta(days=7)
        tomorrow = now + datetime.timedelta(days=1)
        nextweek = now + datetime.timedelta(days=7)

        def mocked_head(blocks={}, loc=None):
            blk = loc.split('+', 1)[0]
            if blocks.get(blk):
                return True
            raise arvados.errors.KeepRequestError("mocked error - block invalid")

        # Block1_expiration, Block2_expiration, Block1_HEAD, Block2_HEAD, Expectation
        cases = [
            # All expired, reset cache - OK
            (yesterday, lastweek, False, False, True),
            (lastweek, yesterday, False, False, True),
            # All non-expired valid blocks - OK
            (tomorrow, nextweek, True, True, True),
            (nextweek, tomorrow, True, True, True),
            # All non-expired invalid blocks - Not OK
            (tomorrow, nextweek, False, False, False),
            (nextweek, tomorrow, False, False, False),
            # One non-expired valid block - OK
            (tomorrow, yesterday, True, False, True),
            (yesterday, tomorrow, False, True, True),
            # One non-expired invalid block - Not OK
            (tomorrow, yesterday, False, False, False),
            (yesterday, tomorrow, False, False, False),
        ]
        for case in cases:
            b1_expiration, b2_expiration, b1_valid, b2_valid, outcome = case
            head_responses = {
                self.block1: b1_valid,
                self.block2: b2_valid,
            }
            cached_manifest = self.template % (
                self.datetime_to_hex(b1_expiration),
                self.datetime_to_hex(b2_expiration),
            )
            arvput = self.MockedPut(cached_manifest)
            with mock.patch('arvados.collection.KeepClient.head') as head_mock:
                head_mock.side_effect = partial(mocked_head, head_responses)
                self.assertEqual(outcome, arvput._cached_manifest_valid(),
                    "Case '%s' should have produced outcome '%s'" % (case, outcome)
                )
                if b1_expiration > now or b2_expiration > now:
                    # A HEAD request should have been done
                    head_mock.assert_called_once()
                else:
                    head_mock.assert_not_called()


class ArvadosExpectedBytesTest(ArvadosBaseTestCase):
    TEST_SIZE = os.path.getsize(__file__)

    def test_expected_bytes_for_file(self):
        writer = arv_put.ArvPutUploadJob([__file__])
        self.assertEqual(self.TEST_SIZE,
                         writer.bytes_expected)

    def test_expected_bytes_for_tree(self):
        tree = self.make_tmpdir()
        shutil.copyfile(__file__, os.path.join(tree, 'one'))
        shutil.copyfile(__file__, os.path.join(tree, 'two'))

        writer = arv_put.ArvPutUploadJob([tree])
        self.assertEqual(self.TEST_SIZE * 2,
                         writer.bytes_expected)
        writer = arv_put.ArvPutUploadJob([tree, __file__])
        self.assertEqual(self.TEST_SIZE * 3,
                         writer.bytes_expected)

    def test_expected_bytes_for_device(self):
        writer = arv_put.ArvPutUploadJob(['/dev/null'], use_cache=False, resume=False)
        self.assertIsNone(writer.bytes_expected)
        writer = arv_put.ArvPutUploadJob([__file__, '/dev/null'])
        self.assertIsNone(writer.bytes_expected)


class ArvadosPutReportTest(ArvadosBaseTestCase):
    def test_machine_progress(self):
        for count, total in [(0, 1), (0, None), (1, None), (235, 9283)]:
            expect = ": {} written {} total\n".format(
                count, -1 if (total is None) else total)
            self.assertTrue(
                arv_put.machine_progress(count, total).endswith(expect))

    def test_known_human_progress(self):
        for count, total in [(0, 1), (2, 4), (45, 60)]:
            expect = '{:.1%}'.format(1.0*count/total)
            actual = arv_put.human_progress(count, total)
            self.assertTrue(actual.startswith('\r'))
            self.assertIn(expect, actual)

    def test_unknown_human_progress(self):
        for count in [1, 20, 300, 4000, 50000]:
            self.assertTrue(re.search(r'\b{}\b'.format(count),
                                      arv_put.human_progress(count, None)))


class ArvPutLogFormatterTest(ArvadosBaseTestCase):
    matcher = r'\(X-Request-Id: req-[a-z0-9]{20}\)'

    def setUp(self):
        super(ArvPutLogFormatterTest, self).setUp()
        self.stderr = tutil.StringIO()
        self.loggingHandler = logging.StreamHandler(self.stderr)
        self.loggingHandler.setFormatter(
            arv_put.ArvPutLogFormatter(arvados.util.new_request_id()))
        self.logger = logging.getLogger()
        self.logger.addHandler(self.loggingHandler)
        self.logger.setLevel(logging.DEBUG)

    def tearDown(self):
        self.logger.removeHandler(self.loggingHandler)
        self.stderr.close()
        self.stderr = None
        super(ArvPutLogFormatterTest, self).tearDown()

    def test_request_id_logged_only_once_on_error(self):
        self.logger.error('Ooops, something bad happened.')
        self.logger.error('Another bad thing just happened.')
        log_lines = self.stderr.getvalue().split('\n')[:-1]
        self.assertEqual(2, len(log_lines))
        self.assertRegex(log_lines[0], self.matcher)
        self.assertNotRegex(log_lines[1], self.matcher)

    def test_request_id_logged_only_once_on_debug(self):
        self.logger.debug('This is just a debug message.')
        self.logger.debug('Another message, move along.')
        log_lines = self.stderr.getvalue().split('\n')[:-1]
        self.assertEqual(2, len(log_lines))
        self.assertRegex(log_lines[0], self.matcher)
        self.assertNotRegex(log_lines[1], self.matcher)

    def test_request_id_not_logged_on_info(self):
        self.logger.info('This should be a useful message')
        log_lines = self.stderr.getvalue().split('\n')[:-1]
        self.assertEqual(1, len(log_lines))
        self.assertNotRegex(log_lines[0], self.matcher)

class ArvadosPutTest(run_test_server.TestCaseWithServers,
                     ArvadosBaseTestCase,
                     tutil.VersionChecker):
    MAIN_SERVER = {}
    Z_UUID = 'zzzzz-zzzzz-zzzzzzzzzzzzzzz'

    def call_main_with_args(self, args):
        self.main_stdout.seek(0, 0)
        self.main_stdout.truncate(0)
        self.main_stderr.seek(0, 0)
        self.main_stderr.truncate(0)
        return arv_put.main(args, self.main_stdout, self.main_stderr)

    def call_main_on_test_file(self, args=[]):
        with self.make_test_file() as testfile:
            path = testfile.name
            self.call_main_with_args(['--stream', '--no-progress'] + args + [path])
        self.assertTrue(
            os.path.exists(os.path.join(os.environ['KEEP_LOCAL_STORE'],
                                        '098f6bcd4621d373cade4e832627b4f6')),
            "did not find file stream in Keep store")

    def setUp(self):
        super(ArvadosPutTest, self).setUp()
        run_test_server.authorize_with('active')
        arv_put.api_client = None
        self.main_stdout = tutil.StringIO()
        self.main_stderr = tutil.StringIO()
        self.loggingHandler = logging.StreamHandler(self.main_stderr)
        self.loggingHandler.setFormatter(
            arv_put.ArvPutLogFormatter(arvados.util.new_request_id()))
        logging.getLogger().addHandler(self.loggingHandler)

    def tearDown(self):
        logging.getLogger().removeHandler(self.loggingHandler)
        for outbuf in ['main_stdout', 'main_stderr']:
            if hasattr(self, outbuf):
                getattr(self, outbuf).close()
                delattr(self, outbuf)
        super(ArvadosPutTest, self).tearDown()

    def test_version_argument(self):
        with tutil.redirected_streams(
                stdout=tutil.StringIO, stderr=tutil.StringIO) as (out, err):
            with self.assertRaises(SystemExit):
                self.call_main_with_args(['--version'])
        self.assertVersionOutput(out, err)

    def test_simple_file_put(self):
        self.call_main_on_test_file()

    def test_put_with_unwriteable_cache_dir(self):
        orig_cachedir = arv_put.ResumeCache.CACHE_DIR
        cachedir = self.make_tmpdir()
        os.chmod(cachedir, 0o0)
        arv_put.ResumeCache.CACHE_DIR = cachedir
        try:
            self.call_main_on_test_file()
        finally:
            arv_put.ResumeCache.CACHE_DIR = orig_cachedir
            os.chmod(cachedir, 0o700)

    def test_put_with_unwritable_cache_subdir(self):
        orig_cachedir = arv_put.ResumeCache.CACHE_DIR
        cachedir = self.make_tmpdir()
        os.chmod(cachedir, 0o0)
        arv_put.ResumeCache.CACHE_DIR = os.path.join(cachedir, 'cachedir')
        try:
            self.call_main_on_test_file()
        finally:
            arv_put.ResumeCache.CACHE_DIR = orig_cachedir
            os.chmod(cachedir, 0o700)

    def test_put_block_replication(self):
        self.call_main_on_test_file()
        with mock.patch('arvados.collection.KeepClient.local_store_put') as put_mock:
            put_mock.return_value = 'acbd18db4cc2f85cedef654fccc4a4d8+3'
            self.call_main_on_test_file(['--replication', '1'])
            self.call_main_on_test_file(['--replication', '4'])
            self.call_main_on_test_file(['--replication', '5'])
            self.assertEqual(
                [x[-1].get('copies') for x in put_mock.call_args_list],
                [1, 4, 5])

    def test_normalize(self):
        testfile1 = self.make_test_file()
        testfile2 = self.make_test_file()
        test_paths = [testfile1.name, testfile2.name]
        # Reverse-sort the paths, so normalization must change their order.
        test_paths.sort(reverse=True)
        self.call_main_with_args(['--stream', '--no-progress', '--normalize'] +
                                 test_paths)
        manifest = self.main_stdout.getvalue()
        # Assert the second file we specified appears first in the manifest.
        file_indices = [manifest.find(':' + os.path.basename(path))
                        for path in test_paths]
        self.assertGreater(*file_indices)

    def test_error_name_without_collection(self):
        self.assertRaises(SystemExit, self.call_main_with_args,
                          ['--name', 'test without Collection',
                           '--stream', '/dev/null'])

    def test_error_when_project_not_found(self):
        self.assertRaises(SystemExit,
                          self.call_main_with_args,
                          ['--project-uuid', self.Z_UUID])

    def test_error_bad_project_uuid(self):
        self.assertRaises(SystemExit,
                          self.call_main_with_args,
                          ['--project-uuid', self.Z_UUID, '--stream'])

    def test_error_when_multiple_storage_classes_specified(self):
        self.assertRaises(SystemExit,
                          self.call_main_with_args,
                          ['--storage-classes', 'hot,cold'])

    def test_error_when_excluding_absolute_path(self):
        tmpdir = self.make_tmpdir()
        self.assertRaises(SystemExit,
                          self.call_main_with_args,
                          ['--exclude', '/some/absolute/path/*',
                           tmpdir])

    def test_api_error_handling(self):
        coll_save_mock = mock.Mock(name='arv.collection.Collection().save_new()')
        coll_save_mock.side_effect = arvados.errors.ApiError(
            fake_httplib2_response(403), b'{}')
        with mock.patch('arvados.collection.Collection.save_new',
                        new=coll_save_mock):
            with self.assertRaises(SystemExit) as exc_test:
                self.call_main_with_args(['/dev/null'])
            self.assertLess(0, exc_test.exception.args[0])
            self.assertLess(0, coll_save_mock.call_count)
            self.assertEqual("", self.main_stdout.getvalue())

    def test_request_id_logging_on_error(self):
        matcher = r'\(X-Request-Id: req-[a-z0-9]{20}\)\n'
        coll_save_mock = mock.Mock(name='arv.collection.Collection().save_new()')
        coll_save_mock.side_effect = arvados.errors.ApiError(
            fake_httplib2_response(403), b'{}')
        with mock.patch('arvados.collection.Collection.save_new',
                        new=coll_save_mock):
            with self.assertRaises(SystemExit):
                self.call_main_with_args(['/dev/null'])
            self.assertRegex(
                self.main_stderr.getvalue(), matcher)


class ArvPutIntegrationTest(run_test_server.TestCaseWithServers,
                            ArvadosBaseTestCase):
    MAIN_SERVER = {}
    KEEP_SERVER = {'blob_signing': True}
    PROJECT_UUID = run_test_server.fixture('groups')['aproject']['uuid']

    @classmethod
    def setUpClass(cls):
        super(ArvPutIntegrationTest, cls).setUpClass()
        cls.ENVIRON = os.environ.copy()
        cls.ENVIRON['PYTHONPATH'] = ':'.join(sys.path)

    def datetime_to_hex(self, dt):
        return hex(int(time.mktime(dt.timetuple())))[2:]

    def setUp(self):
        super(ArvPutIntegrationTest, self).setUp()
        arv_put.api_client = None

    def authorize_with(self, token_name):
        run_test_server.authorize_with(token_name)
        for v in ["ARVADOS_API_HOST",
                  "ARVADOS_API_HOST_INSECURE",
                  "ARVADOS_API_TOKEN"]:
            self.ENVIRON[v] = arvados.config.settings()[v]
        arv_put.api_client = arvados.api('v1')

    def current_user(self):
        return arv_put.api_client.users().current().execute()

    def test_check_real_project_found(self):
        self.authorize_with('active')
        self.assertTrue(arv_put.desired_project_uuid(arv_put.api_client, self.PROJECT_UUID, 0),
                        "did not correctly find test fixture project")

    def test_check_error_finding_nonexistent_uuid(self):
        BAD_UUID = 'zzzzz-zzzzz-zzzzzzzzzzzzzzz'
        self.authorize_with('active')
        try:
            result = arv_put.desired_project_uuid(arv_put.api_client, BAD_UUID,
                                                  0)
        except ValueError as error:
            self.assertIn(BAD_UUID, str(error))
        else:
            self.assertFalse(result, "incorrectly found nonexistent project")

    def test_check_error_finding_nonexistent_project(self):
        BAD_UUID = 'zzzzz-tpzed-zzzzzzzzzzzzzzz'
        self.authorize_with('active')
        with self.assertRaises(apiclient.errors.HttpError):
            arv_put.desired_project_uuid(arv_put.api_client, BAD_UUID,
                                                  0)

    def test_short_put_from_stdin(self):
        # Have to run this as an integration test since arv-put can't
        # read from the tests' stdin.
        # arv-put usually can't stat(os.path.realpath('/dev/stdin')) in this
        # case, because the /proc entry is already gone by the time it tries.
        pipe = subprocess.Popen(
            [sys.executable, arv_put.__file__, '--stream'],
            stdin=subprocess.PIPE, stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT, env=self.ENVIRON)
        pipe.stdin.write(b'stdin test\xa6\n')
        pipe.stdin.close()
        deadline = time.time() + 5
        while (pipe.poll() is None) and (time.time() < deadline):
            time.sleep(.1)
        returncode = pipe.poll()
        if returncode is None:
            pipe.terminate()
            self.fail("arv-put did not PUT from stdin within 5 seconds")
        elif returncode != 0:
            sys.stdout.write(pipe.stdout.read())
            self.fail("arv-put returned exit code {}".format(returncode))
        self.assertIn('1cb671b355a0c23d5d1c61d59cdb1b2b+12',
                      pipe.stdout.read().decode())

    def test_sigint_logs_request_id(self):
        # Start arv-put, give it a chance to start up, send SIGINT,
        # and check that its output includes the X-Request-Id.
        input_stream = subprocess.Popen(
            ['sleep', '10'],
            stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        pipe = subprocess.Popen(
            [sys.executable, arv_put.__file__, '--stream'],
            stdin=input_stream.stdout, stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT, env=self.ENVIRON)
        # Wait for arv-put child process to print something (i.e., a
        # log message) so we know its signal handler is installed.
        select.select([pipe.stdout], [], [], 10)
        pipe.send_signal(signal.SIGINT)
        deadline = time.time() + 5
        while (pipe.poll() is None) and (time.time() < deadline):
            time.sleep(.1)
        returncode = pipe.poll()
        input_stream.terminate()
        if returncode is None:
            pipe.terminate()
            self.fail("arv-put did not exit within 5 seconds")
        self.assertRegex(pipe.stdout.read().decode(), r'\(X-Request-Id: req-[a-z0-9]{20}\)')

    def test_ArvPutSignedManifest(self):
        # ArvPutSignedManifest runs "arv-put foo" and then attempts to get
        # the newly created manifest from the API server, testing to confirm
        # that the block locators in the returned manifest are signed.
        self.authorize_with('active')

        # Before doing anything, demonstrate that the collection
        # we're about to create is not present in our test fixture.
        manifest_uuid = "00b4e9f40ac4dd432ef89749f1c01e74+47"
        with self.assertRaises(apiclient.errors.HttpError):
            arv_put.api_client.collections().get(
                uuid=manifest_uuid).execute()

        datadir = self.make_tmpdir()
        with open(os.path.join(datadir, "foo"), "w") as f:
            f.write("The quick brown fox jumped over the lazy dog")
        p = subprocess.Popen([sys.executable, arv_put.__file__,
                              os.path.join(datadir, 'foo')],
                             stdout=subprocess.PIPE,
                             stderr=subprocess.PIPE,
                             env=self.ENVIRON)
        (_, err) = p.communicate()
        self.assertRegex(err.decode(), r'INFO: Collection saved as ')
        self.assertEqual(p.returncode, 0)

        # The manifest text stored in the API server under the same
        # manifest UUID must use signed locators.
        c = arv_put.api_client.collections().get(uuid=manifest_uuid).execute()
        self.assertRegex(
            c['manifest_text'],
            r'^\. 08a008a01d498c404b0c30852b39d3b8\+44\+A[0-9a-f]+@[0-9a-f]+ 0:44:foo\n')

        os.remove(os.path.join(datadir, "foo"))
        os.rmdir(datadir)

    def run_and_find_collection(self, text, extra_args=[]):
        self.authorize_with('active')
        pipe = subprocess.Popen(
            [sys.executable, arv_put.__file__] + extra_args,
            stdin=subprocess.PIPE, stdout=subprocess.PIPE,
            stderr=subprocess.PIPE, env=self.ENVIRON)
        stdout, stderr = pipe.communicate(text.encode())
        self.assertRegex(stderr.decode(), r'INFO: Collection (updated:|saved as)')
        search_key = ('portable_data_hash'
                      if '--portable-data-hash' in extra_args else 'uuid')
        collection_list = arvados.api('v1').collections().list(
            filters=[[search_key, '=', stdout.decode().strip()]]
        ).execute().get('items', [])
        self.assertEqual(1, len(collection_list))
        return collection_list[0]

    def test_all_expired_signatures_invalidates_cache(self):
        self.authorize_with('active')
        tmpdir = self.make_tmpdir()
        with open(os.path.join(tmpdir, 'somefile.txt'), 'w') as f:
            f.write('foo')
        # Upload a directory and get the cache file name
        p = subprocess.Popen([sys.executable, arv_put.__file__, tmpdir],
                             stdout=subprocess.PIPE,
                             stderr=subprocess.PIPE,
                             env=self.ENVIRON)
        (_, err) = p.communicate()
        self.assertRegex(err.decode(), r'INFO: Creating new cache file at ')
        self.assertEqual(p.returncode, 0)
        cache_filepath = re.search(r'INFO: Creating new cache file at (.*)',
                                   err.decode()).groups()[0]
        self.assertTrue(os.path.isfile(cache_filepath))
        # Load the cache file contents and modify the manifest to simulate
        # an expired access token
        with open(cache_filepath, 'r') as c:
            cache = json.load(c)
        self.assertRegex(cache['manifest'], r'\+A\S+\@')
        a_month_ago = datetime.datetime.now() - datetime.timedelta(days=30)
        cache['manifest'] = re.sub(
            r'\@.*? ',
            "@{} ".format(self.datetime_to_hex(a_month_ago)),
            cache['manifest'])
        with open(cache_filepath, 'w') as c:
            c.write(json.dumps(cache))
        # Re-run the upload and expect to get an invalid cache message
        p = subprocess.Popen([sys.executable, arv_put.__file__, tmpdir],
                             stdout=subprocess.PIPE,
                             stderr=subprocess.PIPE,
                             env=self.ENVIRON)
        (_, err) = p.communicate()
        self.assertRegex(
            err.decode(),
            r'INFO: Cache expired, starting from scratch.*')
        self.assertEqual(p.returncode, 0)

    def test_invalid_signature_invalidates_cache(self):
        self.authorize_with('active')
        tmpdir = self.make_tmpdir()
        with open(os.path.join(tmpdir, 'somefile.txt'), 'w') as f:
            f.write('foo')
        # Upload a directory and get the cache file name
        p = subprocess.Popen([sys.executable, arv_put.__file__, tmpdir],
                             stdout=subprocess.PIPE,
                             stderr=subprocess.PIPE,
                             env=self.ENVIRON)
        (_, err) = p.communicate()
        self.assertRegex(err.decode(), r'INFO: Creating new cache file at ')
        self.assertEqual(p.returncode, 0)
        cache_filepath = re.search(r'INFO: Creating new cache file at (.*)',
                                   err.decode()).groups()[0]
        self.assertTrue(os.path.isfile(cache_filepath))
        # Load the cache file contents and modify the manifest to simulate
        # an invalid access token
        with open(cache_filepath, 'r') as c:
            cache = json.load(c)
        self.assertRegex(cache['manifest'], r'\+A\S+\@')
        cache['manifest'] = re.sub(
            r'\+A.*\@',
            "+Aabcdef0123456789abcdef0123456789abcdef01@",
            cache['manifest'])
        with open(cache_filepath, 'w') as c:
            c.write(json.dumps(cache))
        # Re-run the upload and expect to get an invalid cache message
        p = subprocess.Popen([sys.executable, arv_put.__file__, tmpdir],
                             stdout=subprocess.PIPE,
                             stderr=subprocess.PIPE,
                             env=self.ENVIRON)
        (_, err) = p.communicate()
        self.assertRegex(
            err.decode(),
            r'ERROR: arv-put: Resume cache contains invalid signature.*')
        self.assertEqual(p.returncode, 1)

    def test_single_expired_signature_reuploads_file(self):
        self.authorize_with('active')
        tmpdir = self.make_tmpdir()
        with open(os.path.join(tmpdir, 'foofile.txt'), 'w') as f:
            f.write('foo')
        # Write a second file on its own subdir to force a new stream
        os.mkdir(os.path.join(tmpdir, 'bar'))
        with open(os.path.join(tmpdir, 'bar', 'barfile.txt'), 'w') as f:
            f.write('bar')
        # Upload a directory and get the cache file name
        p = subprocess.Popen([sys.executable, arv_put.__file__, tmpdir],
                             stdout=subprocess.PIPE,
                             stderr=subprocess.PIPE,
                             env=self.ENVIRON)
        (_, err) = p.communicate()
        self.assertRegex(err.decode(), r'INFO: Creating new cache file at ')
        self.assertEqual(p.returncode, 0)
        cache_filepath = re.search(r'INFO: Creating new cache file at (.*)',
                                   err.decode()).groups()[0]
        self.assertTrue(os.path.isfile(cache_filepath))
        # Load the cache file contents and modify the manifest to simulate
        # an expired access token
        with open(cache_filepath, 'r') as c:
            cache = json.load(c)
        self.assertRegex(cache['manifest'], r'\+A\S+\@')
        a_month_ago = datetime.datetime.now() - datetime.timedelta(days=30)
        # Make one of the signatures appear to have expired
        cache['manifest'] = re.sub(
            r'\@.*? 3:3:barfile.txt',
            "@{} 3:3:barfile.txt".format(self.datetime_to_hex(a_month_ago)),
            cache['manifest'])
        with open(cache_filepath, 'w') as c:
            c.write(json.dumps(cache))
        # Re-run the upload and expect to get an invalid cache message
        p = subprocess.Popen([sys.executable, arv_put.__file__, tmpdir],
                             stdout=subprocess.PIPE,
                             stderr=subprocess.PIPE,
                             env=self.ENVIRON)
        (_, err) = p.communicate()
        self.assertRegex(
            err.decode(),
            r'WARNING: Uploaded file \'.*barfile.txt\' access token expired, will re-upload it from scratch')
        self.assertEqual(p.returncode, 0)
        # Confirm that the resulting cache is different from the last run.
        with open(cache_filepath, 'r') as c2:
            new_cache = json.load(c2)
        self.assertNotEqual(cache['manifest'], new_cache['manifest'])

    def test_put_collection_with_later_update(self):
        tmpdir = self.make_tmpdir()
        with open(os.path.join(tmpdir, 'file1'), 'w') as f:
            f.write('Relaxing in basins at the end of inlets terminates the endless tests from the box')
        col = self.run_and_find_collection("", ['--no-progress', tmpdir])
        self.assertNotEqual(None, col['uuid'])
        # Add a new file to the directory
        with open(os.path.join(tmpdir, 'file2'), 'w') as f:
            f.write('The quick brown fox jumped over the lazy dog')
        updated_col = self.run_and_find_collection("", ['--no-progress', '--update-collection', col['uuid'], tmpdir])
        self.assertEqual(col['uuid'], updated_col['uuid'])
        # Get the manifest and check that the new file is being included
        c = arv_put.api_client.collections().get(uuid=updated_col['uuid']).execute()
        self.assertRegex(c['manifest_text'], r'^\..* .*:44:file2\n')

    def test_put_collection_with_utc_expiring_datetime(self):
        tmpdir = self.make_tmpdir()
        trash_at = (datetime.datetime.utcnow() + datetime.timedelta(days=90)).strftime('%Y%m%dT%H%MZ')
        with open(os.path.join(tmpdir, 'file1'), 'w') as f:
            f.write('Relaxing in basins at the end of inlets terminates the endless tests from the box')
        col = self.run_and_find_collection(
            "",
            ['--no-progress', '--trash-at', trash_at, tmpdir])
        self.assertNotEqual(None, col['uuid'])
        c = arv_put.api_client.collections().get(uuid=col['uuid']).execute()
        self.assertEqual(ciso8601.parse_datetime(trash_at),
            ciso8601.parse_datetime(c['trash_at']))

    def test_put_collection_with_timezone_aware_expiring_datetime(self):
        tmpdir = self.make_tmpdir()
        trash_at = (datetime.datetime.utcnow() + datetime.timedelta(days=90)).strftime('%Y%m%dT%H%M-0300')
        with open(os.path.join(tmpdir, 'file1'), 'w') as f:
            f.write('Relaxing in basins at the end of inlets terminates the endless tests from the box')
        col = self.run_and_find_collection(
            "",
            ['--no-progress', '--trash-at', trash_at, tmpdir])
        self.assertNotEqual(None, col['uuid'])
        c = arv_put.api_client.collections().get(uuid=col['uuid']).execute()
        self.assertEqual(
            ciso8601.parse_datetime(trash_at).replace(tzinfo=None) + datetime.timedelta(hours=3),
            ciso8601.parse_datetime(c['trash_at']).replace(tzinfo=None))

    def test_put_collection_with_timezone_naive_expiring_datetime(self):
        tmpdir = self.make_tmpdir()
        trash_at = (datetime.datetime.utcnow() + datetime.timedelta(days=90)).strftime('%Y%m%dT%H%M')
        with open(os.path.join(tmpdir, 'file1'), 'w') as f:
            f.write('Relaxing in basins at the end of inlets terminates the endless tests from the box')
        col = self.run_and_find_collection(
            "",
            ['--no-progress', '--trash-at', trash_at, tmpdir])
        self.assertNotEqual(None, col['uuid'])
        c = arv_put.api_client.collections().get(uuid=col['uuid']).execute()
        if time.daylight:
            offset = datetime.timedelta(seconds=time.altzone)
        else:
            offset = datetime.timedelta(seconds=time.timezone)
        self.assertEqual(
            ciso8601.parse_datetime(trash_at) + offset,
            ciso8601.parse_datetime(c['trash_at']).replace(tzinfo=None))

    def test_put_collection_with_expiring_date_only(self):
        tmpdir = self.make_tmpdir()
        trash_at = '2140-01-01'
        end_of_day = datetime.timedelta(hours=23, minutes=59, seconds=59)
        with open(os.path.join(tmpdir, 'file1'), 'w') as f:
            f.write('Relaxing in basins at the end of inlets terminates the endless tests from the box')
        col = self.run_and_find_collection(
            "",
            ['--no-progress', '--trash-at', trash_at, tmpdir])
        self.assertNotEqual(None, col['uuid'])
        c = arv_put.api_client.collections().get(uuid=col['uuid']).execute()
        if time.daylight:
            offset = datetime.timedelta(seconds=time.altzone)
        else:
            offset = datetime.timedelta(seconds=time.timezone)
        self.assertEqual(
            ciso8601.parse_datetime(trash_at) + end_of_day + offset,
            ciso8601.parse_datetime(c['trash_at']).replace(tzinfo=None))

    def test_put_collection_with_invalid_absolute_expiring_datetimes(self):
        cases = ['2100', '210010','2100-10', '2100-Oct']
        tmpdir = self.make_tmpdir()
        with open(os.path.join(tmpdir, 'file1'), 'w') as f:
            f.write('Relaxing in basins at the end of inlets terminates the endless tests from the box')
        for test_datetime in cases:
            with self.assertRaises(AssertionError):
                self.run_and_find_collection(
                    "",
                    ['--no-progress', '--trash-at', test_datetime, tmpdir])

    def test_put_collection_with_relative_expiring_datetime(self):
        expire_after = 7
        dt_before = datetime.datetime.utcnow() + datetime.timedelta(days=expire_after)
        tmpdir = self.make_tmpdir()
        with open(os.path.join(tmpdir, 'file1'), 'w') as f:
            f.write('Relaxing in basins at the end of inlets terminates the endless tests from the box')
        col = self.run_and_find_collection(
            "",
            ['--no-progress', '--trash-after', str(expire_after), tmpdir])
        self.assertNotEqual(None, col['uuid'])
        dt_after = datetime.datetime.utcnow() + datetime.timedelta(days=expire_after)
        c = arv_put.api_client.collections().get(uuid=col['uuid']).execute()
        trash_at = ciso8601.parse_datetime(c['trash_at']).replace(tzinfo=None)
        self.assertTrue(dt_before < trash_at)
        self.assertTrue(dt_after > trash_at)

    def test_put_collection_with_invalid_relative_expiring_datetime(self):
        expire_after = 0 # Must be >= 1
        tmpdir = self.make_tmpdir()
        with open(os.path.join(tmpdir, 'file1'), 'w') as f:
            f.write('Relaxing in basins at the end of inlets terminates the endless tests from the box')
        with self.assertRaises(AssertionError):
            self.run_and_find_collection(
                "",
                ['--no-progress', '--trash-after', str(expire_after), tmpdir])

    def test_upload_directory_reference_without_trailing_slash(self):
        tmpdir1 = self.make_tmpdir()
        tmpdir2 = self.make_tmpdir()
        with open(os.path.join(tmpdir1, 'foo'), 'w') as f:
            f.write('This is foo')
        with open(os.path.join(tmpdir2, 'bar'), 'w') as f:
            f.write('This is not foo')
        # Upload one directory and one file
        col = self.run_and_find_collection("", ['--no-progress',
                                                tmpdir1,
                                                os.path.join(tmpdir2, 'bar')])
        self.assertNotEqual(None, col['uuid'])
        c = arv_put.api_client.collections().get(uuid=col['uuid']).execute()
        # Check that 'foo' was written inside a subcollection
        # OTOH, 'bar' should have been directly uploaded on the root collection
        self.assertRegex(c['manifest_text'], r'^\. .*:15:bar\n\./.+ .*:11:foo\n')

    def test_upload_directory_reference_with_trailing_slash(self):
        tmpdir1 = self.make_tmpdir()
        tmpdir2 = self.make_tmpdir()
        with open(os.path.join(tmpdir1, 'foo'), 'w') as f:
            f.write('This is foo')
        with open(os.path.join(tmpdir2, 'bar'), 'w') as f:
            f.write('This is not foo')
        # Upload one directory (with trailing slash) and one file
        col = self.run_and_find_collection("", ['--no-progress',
                                                tmpdir1 + os.sep,
                                                os.path.join(tmpdir2, 'bar')])
        self.assertNotEqual(None, col['uuid'])
        c = arv_put.api_client.collections().get(uuid=col['uuid']).execute()
        # Check that 'foo' and 'bar' were written at the same level
        self.assertRegex(c['manifest_text'], r'^\. .*:15:bar .*:11:foo\n')

    def test_put_collection_with_high_redundancy(self):
        # Write empty data: we're not testing CollectionWriter, just
        # making sure collections.create tells the API server what our
        # desired replication level is.
        collection = self.run_and_find_collection("", ['--replication', '4'])
        self.assertEqual(4, collection['replication_desired'])

    def test_put_collection_with_default_redundancy(self):
        collection = self.run_and_find_collection("")
        self.assertEqual(None, collection['replication_desired'])

    def test_put_collection_with_unnamed_project_link(self):
        link = self.run_and_find_collection(
            "Test unnamed collection",
            ['--portable-data-hash', '--project-uuid', self.PROJECT_UUID])
        username = pwd.getpwuid(os.getuid()).pw_name
        self.assertRegex(
            link['name'],
            r'^Saved at .* by {}@'.format(re.escape(username)))

    def test_put_collection_with_name_and_no_project(self):
        link_name = 'Test Collection Link in home project'
        collection = self.run_and_find_collection(
            "Test named collection in home project",
            ['--portable-data-hash', '--name', link_name])
        self.assertEqual(link_name, collection['name'])
        my_user_uuid = self.current_user()['uuid']
        self.assertEqual(my_user_uuid, collection['owner_uuid'])

    def test_put_collection_with_named_project_link(self):
        link_name = 'Test auto Collection Link'
        collection = self.run_and_find_collection("Test named collection",
                                      ['--portable-data-hash',
                                       '--name', link_name,
                                       '--project-uuid', self.PROJECT_UUID])
        self.assertEqual(link_name, collection['name'])

    def test_put_collection_with_storage_classes_specified(self):
        collection = self.run_and_find_collection("", ['--storage-classes', 'hot'])

        self.assertEqual(len(collection['storage_classes_desired']), 1)
        self.assertEqual(collection['storage_classes_desired'][0], 'hot')

    def test_put_collection_without_storage_classes_specified(self):
        collection = self.run_and_find_collection("")

        self.assertEqual(len(collection['storage_classes_desired']), 1)
        self.assertEqual(collection['storage_classes_desired'][0], 'default')

    def test_exclude_filename_pattern(self):
        tmpdir = self.make_tmpdir()
        tmpsubdir = os.path.join(tmpdir, 'subdir')
        os.mkdir(tmpsubdir)
        for fname in ['file1', 'file2', 'file3']:
            with open(os.path.join(tmpdir, "%s.txt" % fname), 'w') as f:
                f.write("This is %s" % fname)
            with open(os.path.join(tmpsubdir, "%s.txt" % fname), 'w') as f:
                f.write("This is %s" % fname)
        col = self.run_and_find_collection("", ['--no-progress',
                                                '--exclude', '*2.txt',
                                                '--exclude', 'file3.*',
                                                 tmpdir])
        self.assertNotEqual(None, col['uuid'])
        c = arv_put.api_client.collections().get(uuid=col['uuid']).execute()
        # None of the file2.txt & file3.txt should have been uploaded
        self.assertRegex(c['manifest_text'], r'^.*:file1.txt')
        self.assertNotRegex(c['manifest_text'], r'^.*:file2.txt')
        self.assertNotRegex(c['manifest_text'], r'^.*:file3.txt')

    def test_exclude_filepath_pattern(self):
        tmpdir = self.make_tmpdir()
        tmpsubdir = os.path.join(tmpdir, 'subdir')
        os.mkdir(tmpsubdir)
        for fname in ['file1', 'file2', 'file3']:
            with open(os.path.join(tmpdir, "%s.txt" % fname), 'w') as f:
                f.write("This is %s" % fname)
            with open(os.path.join(tmpsubdir, "%s.txt" % fname), 'w') as f:
                f.write("This is %s" % fname)
        col = self.run_and_find_collection("", ['--no-progress',
                                                '--exclude', 'subdir/*2.txt',
                                                '--exclude', './file1.*',
                                                 tmpdir])
        self.assertNotEqual(None, col['uuid'])
        c = arv_put.api_client.collections().get(uuid=col['uuid']).execute()
        # Only tmpdir/file1.txt & tmpdir/subdir/file2.txt should have been excluded
        self.assertNotRegex(c['manifest_text'],
                            r'^\./%s.*:file1.txt' % os.path.basename(tmpdir))
        self.assertNotRegex(c['manifest_text'],
                            r'^\./%s/subdir.*:file2.txt' % os.path.basename(tmpdir))
        self.assertRegex(c['manifest_text'],
                         r'^\./%s.*:file2.txt' % os.path.basename(tmpdir))
        self.assertRegex(c['manifest_text'], r'^.*:file3.txt')

    def test_unicode_on_filename(self):
        tmpdir = self.make_tmpdir()
        fname = u"i❤arvados.txt"
        with open(os.path.join(tmpdir, fname), 'w') as f:
            f.write("This is a unicode named file")
        col = self.run_and_find_collection("", ['--no-progress', tmpdir])
        self.assertNotEqual(None, col['uuid'])
        c = arv_put.api_client.collections().get(uuid=col['uuid']).execute()
        self.assertTrue(fname in c['manifest_text'], u"{} does not include {}".format(c['manifest_text'], fname))

    def test_silent_mode_no_errors(self):
        self.authorize_with('active')
        tmpdir = self.make_tmpdir()
        with open(os.path.join(tmpdir, 'test.txt'), 'w') as f:
            f.write('hello world')
        pipe = subprocess.Popen(
            [sys.executable, arv_put.__file__] + ['--silent', tmpdir],
            stdin=subprocess.PIPE, stdout=subprocess.PIPE,
            stderr=subprocess.PIPE, env=self.ENVIRON)
        stdout, stderr = pipe.communicate()
        # No console output should occur on normal operations
        self.assertNotRegex(stderr.decode(), r'.+')
        self.assertNotRegex(stdout.decode(), r'.+')

    def test_silent_mode_does_not_avoid_error_messages(self):
        self.authorize_with('active')
        pipe = subprocess.Popen(
            [sys.executable, arv_put.__file__] + ['--silent',
                                                  '/path/not/existant'],
            stdin=subprocess.PIPE, stdout=subprocess.PIPE,
            stderr=subprocess.PIPE, env=self.ENVIRON)
        stdout, stderr = pipe.communicate()
        # Error message should be displayed when errors happen
        self.assertRegex(stderr.decode(), r'.*ERROR:.*')
        self.assertNotRegex(stdout.decode(), r'.+')


if __name__ == '__main__':
    unittest.main()
