# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import arvados
import arvados.collection
import arvados_fuse
import arvados_fuse.command
import json
import logging
import os
import tempfile
import unittest

from .integration_test import IntegrationTest
from .mount_test_base import MountTestBase

class CacheTest(IntegrationTest):
    mnt_args = ["--by-id", "--directory-cache=0"]

    @IntegrationTest.mount(argv=mnt_args)
    def test_cache_spill(self):
        pdh = []
        for i in range(0, 8):
            cw = arvados.collection.Collection()
            f = cw.open("blurg%i" % i, "w")
            f.write("bloop%i" % i)

            cw.mkdirs("dir%i" % i)
            f = cw.open("dir%i/blurg" % i, "w")
            f.write("dirbloop%i" % i)

            cw.save_new()
            pdh.append(cw.portable_data_hash())
        self.pool_test(self.mnt, pdh)

    @staticmethod
    def _test_cache_spill(self, mnt, pdh):
        for i,v in enumerate(pdh):
            j = os.path.join(mnt, "by_id", v, "blurg%i" % i)
            self.assertTrue(os.path.exists(j))
            j = os.path.join(mnt, "by_id", v, "dir%i/blurg" % i)
            self.assertTrue(os.path.exists(j))

        for i,v in enumerate(pdh):
            j = os.path.join(mnt, "by_id", v, "blurg%i" % i)
            self.assertTrue(os.path.exists(j))
            j = os.path.join(mnt, "by_id", v, "dir%i/blurg" % i)
            self.assertTrue(os.path.exists(j))
