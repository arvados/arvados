#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

exec python <<EOF
import arvados_fuse
print "Successfully imported arvados_fuse"
EOF
