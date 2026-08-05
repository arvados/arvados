# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import os
import sys
import tempfile
import unittest

import arvados.errors as arv_error
import arvados.commands.ws as arv_ws
from . import arvados_testutil as tutil

class ArvWsTestCase(unittest.TestCase):
    def run_ws(self, args):
        return arv_ws.main(args)

    def test_ctrl_c(self):
        with (
            self.assertRaises(SystemExit) as cm,
            unittest.mock.patch(
                "arvados.events.EventClient.run_forever",
                unittest.mock.Mock(side_effect=KeyboardInterrupt)
            )
        ):
            self.run_ws([])
        self.assertEqual(cm.exception.code, 0)
