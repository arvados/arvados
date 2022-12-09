# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import absolute_import
import arvados
import collections
import copy
import hashlib
import mock
import os
import subprocess
import sys
import tempfile
import unittest
import logging

import arvados.commands.keepdocker as arv_keepdocker
from . import arvados_testutil as tutil
from . import run_test_server


class StopTest(Exception):
    pass


class ArvKeepdockerTestCase(unittest.TestCase, tutil.VersionChecker):
    def run_arv_keepdocker(self, args, err, **kwargs):
        sys.argv = ['arv-keepdocker'] + args
        log_handler = logging.StreamHandler(err)
        arv_keepdocker.logger.addHandler(log_handler)
        try:
            return arv_keepdocker.main(**kwargs)
        finally:
            arv_keepdocker.logger.removeHandler(log_handler)

    def test_unsupported_arg(self):
        out = tutil.StringIO()
        with tutil.redirected_streams(stdout=out, stderr=out), \
             self.assertRaises(SystemExit):
            self.run_arv_keepdocker(['-x=unknown'], sys.stderr)
        self.assertRegex(out.getvalue(), 'unrecognized arguments')

    def test_version_argument(self):
        with tutil.redirected_streams(
                stdout=tutil.StringIO, stderr=tutil.StringIO) as (out, err):
            with self.assertRaises(SystemExit):
                self.run_arv_keepdocker(['--version'], sys.stderr)
        self.assertVersionOutput(out, err)

    @mock.patch('arvados.commands.keepdocker.list_images_in_arv',
                return_value=[])
    @mock.patch('arvados.commands.keepdocker.find_image_hashes',
                return_value=['abc123'])
    @mock.patch('arvados.commands.keepdocker.find_one_image_hash',
                return_value='abc123')
    def test_image_format_compatibility(self, _1, _2, _3):
        old_id = hashlib.sha256(b'old').hexdigest()
        new_id = 'sha256:'+hashlib.sha256(b'new').hexdigest()
        for supported, img_id, expect_ok in [
                (['v1'], old_id, True),
                (['v1'], new_id, False),
                (None, old_id, False),
                ([], old_id, False),
                ([], new_id, False),
                (['v1', 'v2'], new_id, True),
                (['v1'], new_id, False),
                (['v2'], new_id, True)]:

            fakeDD = arvados.api('v1')._rootDesc
            if supported is None:
                del fakeDD['dockerImageFormats']
            else:
                fakeDD['dockerImageFormats'] = supported

            err = tutil.StringIO()
            out = tutil.StringIO()

            with tutil.redirected_streams(stdout=out), \
                 mock.patch('arvados.api') as api, \
                 mock.patch('arvados.commands.keepdocker.popen_docker',
                            return_value=subprocess.Popen(
                                ['echo', img_id],
                                stdout=subprocess.PIPE)), \
                 mock.patch('arvados.commands.keepdocker.prep_image_file',
                            side_effect=StopTest), \
                 self.assertRaises(StopTest if expect_ok else SystemExit):

                api()._rootDesc = fakeDD
                self.run_arv_keepdocker(['--force', 'testimage'], err)

            self.assertEqual(out.getvalue(), '')
            if expect_ok:
                self.assertNotRegex(
                    err.getvalue(), "refusing to store",
                    msg=repr((supported, img_id)))
            else:
                self.assertRegex(
                    err.getvalue(), "refusing to store",
                    msg=repr((supported, img_id)))
            if not supported:
                self.assertRegex(
                    err.getvalue(),
                    "server does not specify supported image formats",
                    msg=repr((supported, img_id)))

        fakeDD = arvados.api('v1')._rootDesc
        fakeDD['dockerImageFormats'] = ['v1']
        err = tutil.StringIO()
        out = tutil.StringIO()
        with tutil.redirected_streams(stdout=out), \
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
                ['--force', '--force-image-format', 'testimage'], err)
        self.assertRegex(err.getvalue(), "forcing incompatible image")

    def test_tag_given_twice(self):
        with tutil.redirected_streams(stdout=tutil.StringIO, stderr=tutil.StringIO) as (out, err):
            with self.assertRaises(SystemExit):
                self.run_arv_keepdocker(['myrepo:mytag', 'extratag'], sys.stderr)
            self.assertRegex(err.getvalue(), "cannot add tag argument 'extratag'")

    def test_image_given_as_repo_colon_tag(self):
        with self.assertRaises(StopTest), \
             mock.patch('arvados.commands.keepdocker.find_one_image_hash',
                        side_effect=StopTest) as find_image_mock:
            self.run_arv_keepdocker(['repo:tag'], sys.stderr)
        find_image_mock.assert_called_with('repo', 'tag')

    def test_image_given_as_registry_repo_colon_tag(self):
        with self.assertRaises(StopTest), \
             mock.patch('arvados.commands.keepdocker.find_one_image_hash',
                        side_effect=StopTest) as find_image_mock:
            self.run_arv_keepdocker(['myreg.example:8888/repo/img:tag'], sys.stderr)
        find_image_mock.assert_called_with('myreg.example:8888/repo/img', 'tag')

        with self.assertRaises(StopTest), \
             mock.patch('arvados.commands.keepdocker.find_one_image_hash',
                        side_effect=StopTest) as find_image_mock:
            self.run_arv_keepdocker(['registry.hub.docker.com:443/library/debian:bullseye-slim'], sys.stderr)
        find_image_mock.assert_called_with('registry.hub.docker.com/library/debian', 'bullseye-slim')

    def test_image_has_colons(self):
        with self.assertRaises(StopTest), \
             mock.patch('arvados.commands.keepdocker.find_one_image_hash',
                        side_effect=StopTest) as find_image_mock:
            self.run_arv_keepdocker(['[::1]:8888/repo/img'], sys.stderr)
        find_image_mock.assert_called_with('[::1]:8888/repo/img', 'latest')

        with self.assertRaises(StopTest), \
             mock.patch('arvados.commands.keepdocker.find_one_image_hash',
                        side_effect=StopTest) as find_image_mock:
            self.run_arv_keepdocker(['[::1]/repo/img'], sys.stderr)
        find_image_mock.assert_called_with('[::1]/repo/img', 'latest')

        with self.assertRaises(StopTest), \
             mock.patch('arvados.commands.keepdocker.find_one_image_hash',
                        side_effect=StopTest) as find_image_mock:
            self.run_arv_keepdocker(['[::1]:8888/repo/img:tag'], sys.stderr)
        find_image_mock.assert_called_with('[::1]:8888/repo/img', 'tag')

    def test_list_images_with_host_and_port(self):
        api = arvados.api('v1')
        taglink = api.links().create(body={'link': {
            'link_class': 'docker_image_repo+tag',
            'name': 'registry.example:1234/repo:latest',
            'head_uuid': 'zzzzz-4zz18-1v45jub259sjjgb',
        }}).execute()
        try:
            out = tutil.StringIO()
            with self.assertRaises(SystemExit):
                self.run_arv_keepdocker([], sys.stderr, stdout=out)
            self.assertRegex(out.getvalue(), '\nregistry.example:1234/repo +latest ')
        finally:
            api.links().delete(uuid=taglink['uuid']).execute()

    @mock.patch('arvados.commands.keepdocker.list_images_in_arv',
                return_value=[])
    @mock.patch('arvados.commands.keepdocker.find_image_hashes',
                return_value=['abc123'])
    @mock.patch('arvados.commands.keepdocker.find_one_image_hash',
                return_value='abc123')
    def test_collection_property_update(self, _1, _2, _3):
        image_id = 'sha256:'+hashlib.sha256(b'image').hexdigest()
        fakeDD = arvados.api('v1')._rootDesc
        fakeDD['dockerImageFormats'] = ['v2']

        err = tutil.StringIO()
        out = tutil.StringIO()
        File = collections.namedtuple('File', ['name'])
        mocked_file = File(name='docker_image')
        mocked_collection = {
            'uuid': 'new-collection-uuid',
            'properties': {
                'responsible_person_uuid': 'person_uuid',
            }
        }

        with tutil.redirected_streams(stdout=out), \
             mock.patch('arvados.api') as api, \
             mock.patch('arvados.commands.keepdocker.popen_docker',
                        return_value=subprocess.Popen(
                            ['echo', image_id],
                            stdout=subprocess.PIPE)), \
             mock.patch('arvados.commands.keepdocker.prep_image_file',
                        return_value=(mocked_file, False)), \
             mock.patch('arvados.commands.put.main',
                        return_value='new-collection-uuid'), \
             self.assertRaises(StopTest):

            api()._rootDesc = fakeDD
            api().collections().get().execute.return_value = copy.deepcopy(mocked_collection)
            api().collections().update().execute.side_effect = StopTest
            self.run_arv_keepdocker(['--force', 'testimage'], err)

        updated_properties = mocked_collection['properties']
        updated_properties.update({'docker-image-repo-tag': 'testimage:latest'})
        api().collections().update.assert_called_with(
            uuid=mocked_collection['uuid'],
            body={'properties': updated_properties})
