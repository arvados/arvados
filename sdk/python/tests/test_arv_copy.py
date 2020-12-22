# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

from __future__ import absolute_import
import os
import sys
import tempfile
import unittest
import shutil
import arvados.api
from arvados.collection import Collection, CollectionReader

import arvados.commands.arv_copy as arv_copy
from . import arvados_testutil as tutil
from . import run_test_server

class ArvCopyVersionTestCase(run_test_server.TestCaseWithServers, tutil.VersionChecker):
    MAIN_SERVER = {}
    KEEP_SERVER = {}

    def run_copy(self, args):
        sys.argv = ['arv-copy'] + args
        return arv_copy.main()

    def test_unsupported_arg(self):
        with self.assertRaises(SystemExit):
            self.run_copy(['-x=unknown'])

    def test_version_argument(self):
        with tutil.redirected_streams(
                stdout=tutil.StringIO, stderr=tutil.StringIO) as (out, err):
            with self.assertRaises(SystemExit):
                self.run_copy(['--version'])
        self.assertVersionOutput(out, err)

    def test_copy_project(self):
        api = arvados.api()
        src_proj = api.groups().create(body={"group": {"name": "arv-copy project", "group_class": "project"}}).execute()["uuid"]

        c = Collection()
        with c.open('foo', 'wt') as f:
            f.write('foo')
        c.save_new("arv-copy foo collection", owner_uuid=src_proj)

        dest_proj = api.groups().create(body={"group": {"name": "arv-copy dest project", "group_class": "project"}}).execute()["uuid"]

        tmphome = tempfile.mkdtemp()
        home_was = os.environ['HOME']
        os.environ['HOME'] = tmphome
        try:
            cfgdir = os.path.join(tmphome, ".config", "arvados")
            os.makedirs(cfgdir)
            with open(os.path.join(cfgdir, "zzzzz.conf"), "wt") as f:
                f.write("ARVADOS_API_HOST=%s\n" % os.environ["ARVADOS_API_HOST"])
                f.write("ARVADOS_API_TOKEN=%s\n" % os.environ["ARVADOS_API_TOKEN"])
                f.write("ARVADOS_API_HOST_INSECURE=1\n")

            contents = api.groups().list(filters=[["owner_uuid", "=", dest_proj]]).execute()
            assert len(contents["items"]) == 0

            try:
                self.run_copy(["--project-uuid", dest_proj, src_proj])
            except SystemExit as e:
                assert e.code == 0

            contents = api.groups().list(filters=[["owner_uuid", "=", dest_proj]]).execute()
            assert len(contents["items"]) == 1

            assert contents["items"][0]["name"] == "arv-copy project"
            copied_project = contents["items"][0]["uuid"]

            contents = api.collections().list(filters=[["owner_uuid", "=", copied_project]]).execute()
            assert len(contents["items"]) == 1

            assert contents["items"][0]["uuid"] != c.manifest_locator()
            assert contents["items"][0]["name"] == "arv-copy foo collection"
            assert contents["items"][0]["portable_data_hash"] == c.portable_data_hash()

        finally:
            os.environ['HOME'] = home_was
            shutil.rmtree(tmphome)
