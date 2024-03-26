#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -e

arv-put --version >/dev/null

/usr/lib/python3-arvados-python-client/bin/python <<EOF
import arvados
print("Successfully imported arvados")
EOF
