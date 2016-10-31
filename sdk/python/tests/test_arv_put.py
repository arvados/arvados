#!/usr/bin/env python
# -*- coding: utf-8 -*-

import apiclient
import mock
import os
import pwd
import re
import shutil
import subprocess
import sys
import tempfile
import time
import unittest
import yaml
import threading
import hashlib
import random

from cStringIO import StringIO

import arvados
import arvados.commands.put as arv_put
import arvados_testutil as tutil

from arvados_testutil import ArvadosBaseTestCase, fake_httplib2_response
import run_test_server

class ArvadosPutResumeCacheTest(ArvadosBaseTestCase):
    CACHE_ARGSET = [
        [],
        ['/dev/null'],
        ['/dev/null', '--filename', 'empty'],
        ['/tmp'],
        ['/tmp', '--max-manifest-depth', '0'],
        ['/tmp', '--max-manifest-depth', '1']
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
        self.assertRaises(None, resume_cache.check_cache())

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
            cache = arv_put.ResumeCache(cachefile.name)
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
        for _ in range((arvados.config.KEEP_BLOCK_SIZE/(1024*1024))+1):
            data = random.choice(['x', 'y', 'z']) * 1024 * 1024 # 1 MB
            fileobj.write(data)
        fileobj.close()
        self.arvfile_write = getattr(arvados.arvfile.ArvadosFileWriter, 'write')

    def tearDown(self):
        super(ArvPutUploadJobTest, self).tearDown()
        shutil.rmtree(self.tempdir)
        os.unlink(self.large_file_name)

    def test_writer_works_without_cache(self):
        cwriter = arv_put.ArvPutUploadJob(['/dev/null'], resume=False)
        cwriter.start()
        self.assertEqual(". d41d8cd98f00b204e9800998ecf8427e+0 0:0:null\n", cwriter.manifest_text())

    def test_writer_works_with_cache(self):
        with tempfile.NamedTemporaryFile() as f:
            f.write('foo')
            f.flush()
            cwriter = arv_put.ArvPutUploadJob([f.name])
            cwriter.start()
            self.assertEqual(3, cwriter.bytes_written)
            # Don't destroy the cache, and start another upload
            cwriter_new = arv_put.ArvPutUploadJob([f.name])
            cwriter_new.start()
            cwriter_new.destroy_cache()
            self.assertEqual(0, cwriter_new.bytes_written)

    def make_progress_tester(self):
        progression = []
        def record_func(written, expected):
            progression.append((written, expected))
        return progression, record_func

    def test_progress_reporting(self):
        with tempfile.NamedTemporaryFile() as f:
            f.write('foo')
            f.flush()
            for expect_count in (None, 8):
                progression, reporter = self.make_progress_tester()
                cwriter = arv_put.ArvPutUploadJob([f.name],
                    reporter=reporter, bytes_expected=expect_count)
                cwriter.start()
                cwriter.destroy_cache()
                self.assertIn((3, expect_count), progression)

    def test_writer_upload_directory(self):
        cwriter = arv_put.ArvPutUploadJob([self.tempdir])
        cwriter.start()
        cwriter.destroy_cache()
        self.assertEqual(1024*(1+2+3+4+5), cwriter.bytes_written)

    def test_resume_large_file_upload(self):
        def wrapped_write(*args, **kwargs):
            data = args[1]
            # Exit only on last block
            if len(data) < arvados.config.KEEP_BLOCK_SIZE:
                raise SystemExit("Simulated error")
            return self.arvfile_write(*args, **kwargs)

        with mock.patch('arvados.arvfile.ArvadosFileWriter.write',
                        autospec=True) as mocked_write:
            mocked_write.side_effect = wrapped_write
            writer = arv_put.ArvPutUploadJob([self.large_file_name],
                                             replication_desired=1)
            with self.assertRaises(SystemExit):
                writer.start()
                self.assertLess(writer.bytes_written,
                                os.path.getsize(self.large_file_name))
        # Retry the upload
        writer2 = arv_put.ArvPutUploadJob([self.large_file_name],
                                          replication_desired=1)
        writer2.start()
        self.assertEqual(writer.bytes_written + writer2.bytes_written,
                         os.path.getsize(self.large_file_name))
        writer2.destroy_cache()


class ArvadosExpectedBytesTest(ArvadosBaseTestCase):
    TEST_SIZE = os.path.getsize(__file__)

    def test_expected_bytes_for_file(self):
        self.assertEqual(self.TEST_SIZE,
                          arv_put.expected_bytes_for([__file__]))

    def test_expected_bytes_for_tree(self):
        tree = self.make_tmpdir()
        shutil.copyfile(__file__, os.path.join(tree, 'one'))
        shutil.copyfile(__file__, os.path.join(tree, 'two'))
        self.assertEqual(self.TEST_SIZE * 2,
                          arv_put.expected_bytes_for([tree]))
        self.assertEqual(self.TEST_SIZE * 3,
                          arv_put.expected_bytes_for([tree, __file__]))

    def test_expected_bytes_for_device(self):
        self.assertIsNone(arv_put.expected_bytes_for(['/dev/null']))
        self.assertIsNone(arv_put.expected_bytes_for([__file__, '/dev/null']))


class ArvadosPutReportTest(ArvadosBaseTestCase):
    def test_machine_progress(self):
        for count, total in [(0, 1), (0, None), (1, None), (235, 9283)]:
            expect = ": {} written {} total\n".format(
                count, -1 if (total is None) else total)
            self.assertTrue(
                arv_put.machine_progress(count, total).endswith(expect))

    def test_known_human_progress(self):
        for count, total in [(0, 1), (2, 4), (45, 60)]:
            expect = '{:.1%}'.format(float(count) / total)
            actual = arv_put.human_progress(count, total)
            self.assertTrue(actual.startswith('\r'))
            self.assertIn(expect, actual)

    def test_unknown_human_progress(self):
        for count in [1, 20, 300, 4000, 50000]:
            self.assertTrue(re.search(r'\b{}\b'.format(count),
                                      arv_put.human_progress(count, None)))


class ArvadosPutTest(run_test_server.TestCaseWithServers, ArvadosBaseTestCase):
    MAIN_SERVER = {}
    Z_UUID = 'zzzzz-zzzzz-zzzzzzzzzzzzzzz'

    def call_main_with_args(self, args):
        self.main_stdout = StringIO()
        self.main_stderr = StringIO()
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

    def tearDown(self):
        for outbuf in ['main_stdout', 'main_stderr']:
            if hasattr(self, outbuf):
                getattr(self, outbuf).close()
                delattr(self, outbuf)
        super(ArvadosPutTest, self).tearDown()

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

    def test_api_error_handling(self):
        coll_save_mock = mock.Mock(name='arv.collection.Collection().save_new()')
        coll_save_mock.side_effect = arvados.errors.ApiError(
            fake_httplib2_response(403), '{}')
        with mock.patch('arvados.collection.Collection.save_new',
                        new=coll_save_mock):
            with self.assertRaises(SystemExit) as exc_test:
                self.call_main_with_args(['/dev/null'])
            self.assertLess(0, exc_test.exception.args[0])
            self.assertLess(0, coll_save_mock.call_count)
            self.assertEqual("", self.main_stdout.getvalue())


class ArvPutIntegrationTest(run_test_server.TestCaseWithServers,
                            ArvadosBaseTestCase):
    def _getKeepServerConfig():
        for config_file, mandatory in [
                ['application.yml', False], ['application.default.yml', True]]:
            path = os.path.join(run_test_server.SERVICES_SRC_DIR,
                                "api", "config", config_file)
            if not mandatory and not os.path.exists(path):
                continue
            with open(path) as f:
                rails_config = yaml.load(f.read())
                for config_section in ['test', 'common']:
                    try:
                        key = rails_config[config_section]["blob_signing_key"]
                    except (KeyError, TypeError):
                        pass
                    else:
                        return {'blob_signing_key': key,
                                'enforce_permissions': True}
        return {'blog_signing_key': None, 'enforce_permissions': False}

    MAIN_SERVER = {}
    KEEP_SERVER = _getKeepServerConfig()
    PROJECT_UUID = run_test_server.fixture('groups')['aproject']['uuid']

    @classmethod
    def setUpClass(cls):
        super(ArvPutIntegrationTest, cls).setUpClass()
        cls.ENVIRON = os.environ.copy()
        cls.ENVIRON['PYTHONPATH'] = ':'.join(sys.path)

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
            self.assertIn(BAD_UUID, error.message)
        else:
            self.assertFalse(result, "incorrectly found nonexistent project")

    def test_check_error_finding_nonexistent_project(self):
        BAD_UUID = 'zzzzz-tpzed-zzzzzzzzzzzzzzz'
        self.authorize_with('active')
        with self.assertRaises(apiclient.errors.HttpError):
            result = arv_put.desired_project_uuid(arv_put.api_client, BAD_UUID,
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
        pipe.stdin.write('stdin test\n')
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
        self.assertIn('4a9c8b735dce4b5fa3acf221a0b13628+11', pipe.stdout.read())

    def test_ArvPutSignedManifest(self):
        # ArvPutSignedManifest runs "arv-put foo" and then attempts to get
        # the newly created manifest from the API server, testing to confirm
        # that the block locators in the returned manifest are signed.
        self.authorize_with('active')

        # Before doing anything, demonstrate that the collection
        # we're about to create is not present in our test fixture.
        manifest_uuid = "00b4e9f40ac4dd432ef89749f1c01e74+47"
        with self.assertRaises(apiclient.errors.HttpError):
            notfound = arv_put.api_client.collections().get(
                uuid=manifest_uuid).execute()

        datadir = self.make_tmpdir()
        with open(os.path.join(datadir, "foo"), "w") as f:
            f.write("The quick brown fox jumped over the lazy dog")
        p = subprocess.Popen([sys.executable, arv_put.__file__, datadir],
                             stdout=subprocess.PIPE, env=self.ENVIRON)
        (arvout, arverr) = p.communicate()
        self.assertEqual(arverr, None)
        self.assertEqual(p.returncode, 0)

        # The manifest text stored in the API server under the same
        # manifest UUID must use signed locators.
        c = arv_put.api_client.collections().get(uuid=manifest_uuid).execute()
        self.assertRegexpMatches(
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
        stdout, stderr = pipe.communicate(text)
        search_key = ('portable_data_hash'
                      if '--portable-data-hash' in extra_args else 'uuid')
        collection_list = arvados.api('v1').collections().list(
            filters=[[search_key, '=', stdout.strip()]]).execute().get('items', [])
        self.assertEqual(1, len(collection_list))
        return collection_list[0]

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
        self.assertRegexpMatches(
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


if __name__ == '__main__':
    unittest.main()
