#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -e

arv-put --version

/usr/bin/python3 << EOF
import arvados
print("Successfully imported arvados")
EOF
