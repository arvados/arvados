#!/usr/bin/env python
# -*- coding: utf-8 -*-

import apiclient
import os
import re
import shutil
import subprocess
import sys
import tempfile
import time
import unittest
import yaml

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
        self.assertEquals(". 0:0:null\n", cwriter.manifest_text())

    def test_writer_works_without_cache(self):
        cwriter = arv_put.ArvPutCollectionWriter()
        cwriter.write_file('/dev/null')
        self.assertEquals(". 0:0:null\n", cwriter.manifest_text())

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
        self.assertEquals(". 0:0:null\n", new_writer.manifest_text())

    def test_new_writer_from_empty_cache(self):
        cwriter = arv_put.ArvPutCollectionWriter.from_cache(self.cache)
        cwriter.write_file('/dev/null')
        self.assertEquals(". 0:0:null\n", cwriter.manifest_text())

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


class ArvadosPutTest(ArvadosKeepLocalStoreTestCase):
    def test_simple_file_put(self):
        with self.make_test_file() as testfile:
            path = testfile.name
            arv_put.main(['--stream', '--no-progress', path])
        self.assertTrue(
            os.path.exists(os.path.join(os.environ['KEEP_LOCAL_STORE'],
                                        '098f6bcd4621d373cade4e832627b4f6')),
            "did not find file stream in Keep store")

    def test_short_put_from_stdin(self):
        # Have to run this separately since arv-put can't read from the
        # tests' stdin.
        # arv-put usually can't stat(os.path.realpath('/dev/stdin')) in this
        # case, because the /proc entry is already gone by the time it tries.
        pipe = subprocess.Popen(
            [sys.executable, arv_put.__file__, '--stream'],
            stdin=subprocess.PIPE, stdout=subprocess.PIPE,
            stderr=open('/dev/null', 'w'))
        pipe.stdin.write('stdin test\n')
        pipe.stdin.close()
        deadline = time.time() + 5
        while (pipe.poll() is None) and (time.time() < deadline):
            time.sleep(.1)
        if pipe.returncode is None:
            pipe.terminate()
            self.fail("arv-put did not PUT from stdin within 5 seconds")
        self.assertEquals(pipe.returncode, 0)
        self.assertIn('4a9c8b735dce4b5fa3acf221a0b13628+11', pipe.stdout.read())


class ArvPutIntegrationTest(unittest.TestCase):
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
        config_blob_signing_key = rails_config["test"]["blob_signing_key"]
        run_test_server.run()
        run_test_server.run_keep(blob_signing_key=config_blob_signing_key,
                                 enforce_permissions=True)

    @classmethod
    def tearDownClass(cls):
        run_test_server.stop()
        run_test_server.stop_keep()

    def test_ArvPutSignedManifest(self):
        # ArvPutSignedManifest runs "arv-put foo" and then attempts to get
        # the newly created manifest from the API server, testing to confirm
        # that the block locators in the returned manifest are signed.
        run_test_server.authorize_with('active')
        for v in ["ARVADOS_API_HOST",
                  "ARVADOS_API_HOST_INSECURE",
                  "ARVADOS_API_TOKEN"]:
            os.environ[v] = arvados.config.settings()[v]

        # Before doing anything, demonstrate that the collection
        # we're about to create is not present in our test fixture.
        api = arvados.api('v1', cache=False)
        manifest_uuid = "00b4e9f40ac4dd432ef89749f1c01e74+47"
        with self.assertRaises(apiclient.errors.HttpError):
            notfound = api.collections().get(uuid=manifest_uuid).execute()

        datadir = tempfile.mkdtemp()
        with open(os.path.join(datadir, "foo"), "w") as f:
            f.write("The quick brown fox jumped over the lazy dog")
        p = subprocess.Popen(["./bin/arv-put", datadir],
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


if __name__ == '__main__':
    unittest.main()
