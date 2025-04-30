#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

arv-put --version >/dev/null || exit

. /usr/lib/python3-arvados-python-client/bin/activate
python <<EOF
import arvados
print("Successfully imported arvados")
EOF
