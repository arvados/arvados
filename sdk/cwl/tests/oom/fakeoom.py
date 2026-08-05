# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

import argparse
import sys

import arvados

MB = 1_000_000

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

arv = arvados.api('v1')
ctr = arv.containers().current().execute()
ctr_ram = ctr['runtime_constraints']['ram']
try:
    exit_status = 0 if ctr_ram >= args.fail_under * MB else int(args.fail_with)
except ValueError:
    exit_status = args.fail_with
print("container {}: have {:.01f}MB RAM, want {:.01f}MB: exiting {!r}".format(
    ctr['uuid'], ctr_ram / MB, args.fail_under, exit_status,
), file=sys.stderr)

exit(exit_status)
