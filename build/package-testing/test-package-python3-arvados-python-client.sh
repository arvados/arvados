#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -e

arv-put --version >/dev/null

PYTHON=`ls /usr/share/python3*/dist/python3-arvados-python-client/bin/python3 |head -n1`

$PYTHON << EOF
import arvados
print("Successfully imported arvados")
EOF
