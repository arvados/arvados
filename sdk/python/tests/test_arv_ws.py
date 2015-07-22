#!/usr/bin/env python

import unittest
import arvados.errors as arv_error
import arvados.commands.ws as arv_ws

class ArvWsTestCase(unittest.TestCase):
    def run_ws(self, args):
        return arv_ws.main(args)

    def test_unsupported_arg(self):
        with self.assertRaises(SystemExit):
            self.run_ws(['-x=unknown'])
