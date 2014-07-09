#!/usr/bin/env python
# -*- coding: utf-8 -*-

import apiclient
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

from cStringIO import StringIO

import arvados
import arvados.commands.put as arv_put

from arvados_testutil import ArvadosBaseTestCase, ArvadosKeepLocalStoreTestCase
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
            self.assertEquals(self.cache_path_from_arglist(argset),
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
        self.assertEquals(
            self.cache_path_from_arglist(['a', 'b', 'c']),
            self.cache_path_from_arglist(['c', 'a', 'b']))
        self.assertEquals(
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
        self.assertEquals(path1, path2,
                         "cache path considered --filename for directory")
        self.assertEquals(
            self.cache_path_from_arglist(['-']),
            self.cache_path_from_arglist(['-', '--max-manifest-depth', '1']),
            "cache path considered --max-manifest-depth for file")

    def test_cache_names_treat_negative_manifest_depths_identically(self):
        base_args = ['/tmp', '--max-manifest-depth']
        self.assertEquals(
            self.cache_path_from_arglist(base_args + ['-1']),
            self.cache_path_from_arglist(base_args + ['-2']))

    def test_cache_names_treat_stdin_consistently(self):
        self.assertEquals(
            self.cache_path_from_arglist(['-', '--filename', 'test']),
            self.cache_path_from_arglist(['/dev/stdin', '--filename', 'test']))

    def test_cache_names_identical_for_synonymous_names(self):
        self.assertEquals(
            self.cache_path_from_arglist(['.']),
            self.cache_path_from_arglist([os.path.realpath('.')]))
        testdir = self.make_tmpdir()
        looplink = os.path.join(testdir, 'loop')
        os.symlink(testdir, looplink)
        self.assertEquals(
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

    def test_basic_cache_storage(self):
        thing = ['test', 'list']
        with tempfile.NamedTemporaryFile() as cachefile:
            self.last_cache = arv_put.ResumeCache(cachefile.name)
        self.last_cache.save(thing)
        self.assertEquals(thing, self.last_cache.load())

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
        self.assertEquals(thing, self.last_cache.load())

    def test_multiple_cache_writes(self):
        thing = ['short', 'list']
        with tempfile.NamedTemporaryFile() as cachefile:
            self.last_cache = arv_put.ResumeCache(cachefile.name)
        # Start writing an object longer than the one we test, to make
        # sure the cache file gets truncated.
        self.last_cache.save(['long', 'long', 'list'])
        self.last_cache.save(thing)
        self.assertEquals(thing, self.last_cache.load())

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


class ArvadosPutCollectionWriterTest(ArvadosKeepLocalStoreTestCase):
    def setUp(self):
        super(ArvadosPutCollectionWriterTest, self).setUp()
        with tempfile.NamedTemporaryFile(delete=False) as cachefile:
            self.cache = arv_put.ResumeCache(cachefile.name)
            self.cache_filename = cachefile.name

    def tearDown(self):
        super(ArvadosPutCollectionWriterTest, self).tearDown()
        if os.path.exists(self.cache_filename):
            self.cache.destroy()
        self.cache.close()

    def test_writer_caches(self):
        cwriter = arv_put.ArvPutCollectionWriter(self.cache)
        cwriter.write_file('/dev/null')
        cwriter.cache_state()
        self.assertTrue(self.cache.load())
        self.assertEquals(". d41d8cd98f00b204e9800998ecf8427e+0 0:0:null\n", cwriter.manifest_text())

    def test_writer_works_without_cache(self):
        cwriter = arv_put.ArvPutCollectionWriter()
        cwriter.write_file('/dev/null')
        self.assertEquals(". d41d8cd98f00b204e9800998ecf8427e+0 0:0:null\n", cwriter.manifest_text())

    def test_writer_resumes_from_cache(self):
        cwriter = arv_put.ArvPutCollectionWriter(self.cache)
        with self.make_test_file() as testfile:
            cwriter.write_file(testfile.name, 'test')
            cwriter.cache_state()
            new_writer = arv_put.ArvPutCollectionWriter.from_cache(
                self.cache)
            self.assertEquals(
                ". 098f6bcd4621d373cade4e832627b4f6+4 0:4:test\n",
                new_writer.manifest_text())

    def test_new_writer_from_stale_cache(self):
        cwriter = arv_put.ArvPutCollectionWriter(self.cache)
        with self.make_test_file() as testfile:
            cwriter.write_file(testfile.name, 'test')
        new_writer = arv_put.ArvPutCollectionWriter.from_cache(self.cache)
        new_writer.write_file('/dev/null')
        self.assertEquals(". d41d8cd98f00b204e9800998ecf8427e+0 0:0:null\n", new_writer.manifest_text())

    def test_new_writer_from_empty_cache(self):
        cwriter = arv_put.ArvPutCollectionWriter.from_cache(self.cache)
        cwriter.write_file('/dev/null')
        self.assertEquals(". d41d8cd98f00b204e9800998ecf8427e+0 0:0:null\n", cwriter.manifest_text())

    def test_writer_resumable_after_arbitrary_bytes(self):
        cwriter = arv_put.ArvPutCollectionWriter(self.cache)
        # These bytes are intentionally not valid UTF-8.
        with self.make_test_file('\x00\x07\xe2') as testfile:
            cwriter.write_file(testfile.name, 'test')
            cwriter.cache_state()
            new_writer = arv_put.ArvPutCollectionWriter.from_cache(
                self.cache)
        self.assertEquals(cwriter.manifest_text(), new_writer.manifest_text())

    def make_progress_tester(self):
        progression = []
        def record_func(written, expected):
            progression.append((written, expected))
        return progression, record_func

    def test_progress_reporting(self):
        for expect_count in (None, 8):
            progression, reporter = self.make_progress_tester()
            cwriter = arv_put.ArvPutCollectionWriter(
                reporter=reporter, bytes_expected=expect_count)
            with self.make_test_file() as testfile:
                cwriter.write_file(testfile.name, 'test')
            cwriter.finish_current_stream()
            self.assertIn((4, expect_count), progression)

    def test_resume_progress(self):
        cwriter = arv_put.ArvPutCollectionWriter(self.cache, bytes_expected=4)
        with self.make_test_file() as testfile:
            # Set up a writer with some flushed bytes.
            cwriter.write_file(testfile.name, 'test')
            cwriter.finish_current_stream()
            cwriter.cache_state()
            new_writer = arv_put.ArvPutCollectionWriter.from_cache(self.cache)
            self.assertEqual(new_writer.bytes_written, 4)


class ArvadosExpectedBytesTest(ArvadosBaseTestCase):
    TEST_SIZE = os.path.getsize(__file__)

    def test_expected_bytes_for_file(self):
        self.assertEquals(self.TEST_SIZE,
                          arv_put.expected_bytes_for([__file__]))

    def test_expected_bytes_for_tree(self):
        tree = self.make_tmpdir()
        shutil.copyfile(__file__, os.path.join(tree, 'one'))
        shutil.copyfile(__file__, os.path.join(tree, 'two'))
        self.assertEquals(self.TEST_SIZE * 2,
                          arv_put.expected_bytes_for([tree]))
        self.assertEquals(self.TEST_SIZE * 3,
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


class ArvadosPutProjectLinkTest(ArvadosBaseTestCase):
    Z_UUID = 'zzzzz-zzzzz-zzzzzzzzzzzzzzz'

    def setUp(self):
        self.stderr = StringIO()
        super(ArvadosPutProjectLinkTest, self).setUp()

    def tearDown(self):
        self.stderr.close()
        super(ArvadosPutProjectLinkTest, self).tearDown()

    def prep_link_from_arguments(self, args, uuid_found=True):
        try:
            link = arv_put.prep_project_link(arv_put.parse_arguments(args),
                                             self.stderr,
                                             lambda uuid: uuid_found)
        finally:
            self.stderr.seek(0)
        return link

    def check_link(self, link, project_uuid, link_name=None):
        self.assertEqual(project_uuid, link.get('tail_uuid'))
        self.assertEqual('name', link.get('link_class'))
        if link_name is None:
            self.assertNotIn('name', link)
        else:
            self.assertEqual(link_name, link.get('name'))
        self.assertNotIn('head_uuid', link)

    def check_stderr_empty(self):
        self.assertEqual('', self.stderr.getvalue())

    def test_project_link_with_name(self):
        link = self.prep_link_from_arguments(['--project-uuid', self.Z_UUID,
                                              '--name', 'test link AAA'])
        self.check_link(link, self.Z_UUID, 'test link AAA')
        self.check_stderr_empty()

    def test_project_link_without_name(self):
        link = self.prep_link_from_arguments(['--project-uuid', self.Z_UUID])
        self.check_link(link, self.Z_UUID)
        self.check_stderr_empty()

    def test_collection_without_project_warned(self):
        self.assertIsNone(self.prep_link_from_arguments([]))
        for line in self.stderr:
            if "--project-uuid or --name" in line:
                break
        else:
            self.fail("no warning emitted about the lack of project name")

    def test_no_link_or_warning_with_no_collection(self):
        self.assertIsNone(self.prep_link_from_arguments(['--raw']))
        self.check_stderr_empty()

    def test_error_when_project_not_found(self):
        self.assertRaises(ValueError,
                          self.prep_link_from_arguments,
                          ['--project-uuid', self.Z_UUID], False)

    def test_name_without_project_is_error(self):
        self.assertRaises(ValueError,
                          self.prep_link_from_arguments,
                          ['--name', 'test'])

    def test_link_without_collection_is_error(self):
        self.assertRaises(ValueError,
                          self.prep_link_from_arguments,
                          ['--project-uuid', self.Z_UUID, '--stream'])


class ArvadosPutTest(ArvadosKeepLocalStoreTestCase):
    def call_main_with_args(self, args):
        self.main_stdout = StringIO()
        self.main_stderr = StringIO()
        return arv_put.main(args, self.main_stdout, self.main_stderr)

    def call_main_on_test_file(self):
        with self.make_test_file() as testfile:
            path = testfile.name
            self.call_main_with_args(['--stream', '--no-progress', path])
        self.assertTrue(
            os.path.exists(os.path.join(os.environ['KEEP_LOCAL_STORE'],
                                        '098f6bcd4621d373cade4e832627b4f6')),
            "did not find file stream in Keep store")

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

    def test_short_put_from_stdin(self):
        # Have to run this separately since arv-put can't read from the
        # tests' stdin.
        # arv-put usually can't stat(os.path.realpath('/dev/stdin')) in this
        # case, because the /proc entry is already gone by the time it tries.
        pipe = subprocess.Popen(
            [sys.executable, arv_put.__file__, '--stream'],
            stdin=subprocess.PIPE, stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT)
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

    def test_link_without_project_uuid_aborts(self):
        self.assertRaises(SystemExit, self.call_main_with_args,
                          ['--name', 'test without project UUID', '/dev/null'])

    def test_link_without_collection_aborts(self):
        self.assertRaises(SystemExit, self.call_main_with_args,
                          ['--name', 'test without Collection',
                           '--stream', '/dev/null'])

class ArvPutIntegrationTest(unittest.TestCase):
    PROJECT_UUID = run_test_server.fixture('groups')['aproject']['uuid']

    @classmethod
    def setUpClass(cls):
        try:
            del os.environ['KEEP_LOCAL_STORE']
        except KeyError:
            pass

        # Use the blob_signing_key from the Rails "test" configuration
        # to provision the Keep server.
        with open(os.path.join(os.path.dirname(__file__),
                               run_test_server.ARV_API_SERVER_DIR,
                               "config",
                               "application.yml")) as f:
            rails_config = yaml.load(f.read())
        try:
            config_blob_signing_key = rails_config["test"]["blob_signing_key"]
        except KeyError:
            config_blob_signing_key = rails_config["common"]["blob_signing_key"]
        run_test_server.run()
        run_test_server.run_keep(blob_signing_key=config_blob_signing_key,
                                 enforce_permissions=True)

    @classmethod
    def tearDownClass(cls):
        run_test_server.stop()
        run_test_server.stop_keep()

    def authorize_with(self, token_name):
        run_test_server.authorize_with(token_name)
        for v in ["ARVADOS_API_HOST",
                  "ARVADOS_API_HOST_INSECURE",
                  "ARVADOS_API_TOKEN"]:
            os.environ[v] = arvados.config.settings()[v]

    def test_check_real_project_found(self):
        self.assertTrue(arv_put.check_project_exists(self.PROJECT_UUID),
                        "did not correctly find test fixture project")

    def test_check_error_finding_nonexistent_project(self):
        BAD_UUID = 'zzzzz-zzzzz-zzzzzzzzzzzzzzz'
        try:
            result = arv_put.check_project_exists(BAD_UUID)
        except ValueError as error:
            self.assertIn(BAD_UUID, error.message)
        else:
            self.assertFalse(result, "incorrectly found nonexistent project")

    def test_ArvPutSignedManifest(self):
        # ArvPutSignedManifest runs "arv-put foo" and then attempts to get
        # the newly created manifest from the API server, testing to confirm
        # that the block locators in the returned manifest are signed.
        self.authorize_with('active')

        # Before doing anything, demonstrate that the collection
        # we're about to create is not present in our test fixture.
        api = arvados.api('v1', cache=False)
        manifest_uuid = "00b4e9f40ac4dd432ef89749f1c01e74+47"
        with self.assertRaises(apiclient.errors.HttpError):
            notfound = api.collections().get(uuid=manifest_uuid).execute()

        datadir = tempfile.mkdtemp()
        with open(os.path.join(datadir, "foo"), "w") as f:
            f.write("The quick brown fox jumped over the lazy dog")
        p = subprocess.Popen([sys.executable, arv_put.__file__, datadir],
                             stdout=subprocess.PIPE)
        (arvout, arverr) = p.communicate()
        self.assertEqual(p.returncode, 0)
        self.assertEqual(arverr, None)
        self.assertEqual(arvout.strip(), manifest_uuid)

        # The manifest text stored in the API server under the same
        # manifest UUID must use signed locators.
        c = api.collections().get(uuid=manifest_uuid).execute()
        self.assertRegexpMatches(
            c['manifest_text'],
            r'^\. 08a008a01d498c404b0c30852b39d3b8\+44\+A[0-9a-f]+@[0-9a-f]+ 0:44:foo\n')

        os.remove(os.path.join(datadir, "foo"))
        os.rmdir(datadir)

    def run_and_find_link(self, text, extra_args=[]):
        self.authorize_with('active')
        pipe = subprocess.Popen(
            [sys.executable, arv_put.__file__,
             '--project-uuid', self.PROJECT_UUID] + extra_args,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        stdout, stderr = pipe.communicate(text)
        link_list = arvados.api('v1', cache=False).links().list(
            filters=[['head_uuid', '=', stdout.strip()],
                     ['tail_uuid', '=', self.PROJECT_UUID],
                     ['link_class', '=', 'name']]).execute().get('items', [])
        self.assertEqual(1, len(link_list))
        return link_list[0]

    def test_put_collection_with_unnamed_project_link(self):
        link = self.run_and_find_link("Test unnamed collection")
        username = pwd.getpwuid(os.getuid()).pw_name
        self.assertRegexpMatches(
            link['name'],
            r'^Collection saved by {}@'.format(re.escape(username)))

    def test_put_collection_with_named_project_link(self):
        link_name = 'Test auto Collection Link'
        link = self.run_and_find_link("Test named collection",
                                      ['--name', link_name])
        self.assertEqual(link_name, link['name'])


if __name__ == '__main__':
    unittest.main()
