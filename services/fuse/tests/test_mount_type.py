# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

import logging
import subprocess

from .integration_test import IntegrationTest

logger = logging.getLogger('arvados.arv-mount')


class MountTypeTest(IntegrationTest):
    @IntegrationTest.mount(argv=["--subtype=arv-mount-test"])
    def test_mount_type(self):
        self.pool_test(self.mnt)

    @staticmethod
    def _test_mount_type(self, mnt):
        self.assertEqual(["fuse.arv-mount-test"], [
            toks[4]
            for toks in [
                line.split(' ')
                for line in subprocess.check_output("mount").split("\n")
            ]
            if len(toks) > 4 and toks[2] == mnt
        ])
