#!/usr/bin/env python
# -*- coding: utf-8 -*-

import arvados
import hashlib
import io
import mock
import os
import subprocess
import sys
import tempfile
import unittest

import arvados.commands.keepdocker as arv_keepdocker
import arvados_testutil as tutil
import run_test_server


class StopTest(Exception):
    pass


class ArvKeepdockerTestCase(unittest.TestCase):
    def run_arv_keepdocker(self, args):
        sys.argv = ['arv-keepdocker'] + args
        return arv_keepdocker.main()

    def test_unsupported_arg(self):
        with self.assertRaises(SystemExit):
            self.run_arv_keepdocker(['-x=unknown'])

    def test_version_argument(self):
        err = io.BytesIO()
        out = io.BytesIO()
        with tutil.redirected_streams(stdout=out, stderr=err):
            with self.assertRaises(SystemExit):
                self.run_arv_keepdocker(['--version'])
        self.assertEqual(out.getvalue(), '')
        self.assertRegexpMatches(err.getvalue(), "[0-9]+\.[0-9]+\.[0-9]+")

    def test_migrate19(self):
        try:
            sys.argv = ['arv-migrate-docker19']

            added = arv_keepdocker.migrate19()
            self.assertEqual(len(added), 1)
            self.assertEqual(added[0]['link_class'], 'docker_image_migration')
            self.assertEqual(added[0]['name'], 'migrate_1.9_1.10')
            self.assertEqual(added[0]['tail_uuid'], 'fa3c1a9cb6783f85f2ecda037e07b8c3+167')
            self.assertEqual(added[0]['head_uuid'], 'd740a57097711e08eb9b2a93518f20ab+174')

            added = arv_keepdocker.migrate19()
            self.assertEqual(added, [])
        finally:
            run_test_server.reset()

    @mock.patch('arvados.commands.keepdocker.find_image_hashes',
                return_value=['abc123'])
    @mock.patch('arvados.commands.keepdocker.find_one_image_hash',
                return_value='abc123')
    def test_image_format_compatibility(self, _1, _2):
        old_id = hashlib.sha256('old').hexdigest()
        new_id = 'sha256:'+hashlib.sha256('new').hexdigest()
        for supported, img_id, expect_ok in [
                (['v1'], old_id, True),
                (['v1'], new_id, False),
                (None, old_id, True),
                ([], old_id, True),
                ([], new_id, True),
                (['v1', 'v2'], new_id, True),
                (['v1'], new_id, False),
                (['v2'], new_id, True)]:

            fakeDD = arvados.api('v1')._rootDesc
            if supported is None:
                del fakeDD['dockerImageFormats']
            else:
                fakeDD['dockerImageFormats'] = supported

            err = io.BytesIO()
            out = io.BytesIO()

            with tutil.redirected_streams(stdout=out, stderr=err), \
                 mock.patch('arvados.api') as api, \
                 mock.patch('arvados.commands.keepdocker.popen_docker',
                            return_value=subprocess.Popen(
                                ['echo', img_id],
                                stdout=subprocess.PIPE)), \
                 mock.patch('arvados.commands.keepdocker.prep_image_file',
                            side_effect=StopTest), \
                 self.assertRaises(StopTest if expect_ok else SystemExit):

                api()._rootDesc = fakeDD
                self.run_arv_keepdocker(['--force', 'testimage'])

            self.assertEqual(out.getvalue(), '')
            if expect_ok:
                self.assertNotRegexpMatches(
                    err.getvalue(), "refusing to store",
                    msg=repr((supported, img_id)))
            else:
                self.assertRegexpMatches(
                    err.getvalue(), "refusing to store",
                    msg=repr((supported, img_id)))
            if not supported:
                self.assertRegexpMatches(
                    err.getvalue(),
                    "server does not specify supported image formats",
                    msg=repr((supported, img_id)))

        fakeDD = arvados.api('v1')._rootDesc
        fakeDD['dockerImageFormats'] = ['v1']
        err = io.BytesIO()
        out = io.BytesIO()
        with tutil.redirected_streams(stdout=out, stderr=err), \
             mock.patch('arvados.api') as api, \
             mock.patch('arvados.commands.keepdocker.popen_docker',
                        return_value=subprocess.Popen(
                            ['echo', new_id],
                            stdout=subprocess.PIPE)), \
             mock.patch('arvados.commands.keepdocker.prep_image_file',
                        side_effect=StopTest), \
             self.assertRaises(StopTest):
            api()._rootDesc = fakeDD
            self.run_arv_keepdocker(
                ['--force', '--force-image-format', 'testimage'])
        self.assertRegexpMatches(err.getvalue(), "forcing incompatible image")
