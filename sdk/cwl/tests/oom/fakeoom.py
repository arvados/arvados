# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import argparse

import arvados

argparser = argparse.ArgumentParser()
argparser.add_argument(
    '--fail-under', '-t',
    type=int,
    default=500,
    help="Fail when the container has less than this much RAM in SI MB (default %(default)s)",
)
argparser.add_argument(
    '--fail-with', '-f',
    default='',
    help="Fail with this exit code (when numeric) or message (otherwise)",
)
args = argparser.parse_args()

arv = arvados.api()
ctr = arv.containers().current().execute()

if ctr['runtime_constraints']['ram'] >= (args.fail_under * 1_000_000):
    exit()
try:
    exit_status = int(args.fail_with)
except ValueError:
    exit_status = args.fail_with
exit(exit_status)
