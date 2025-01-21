#!/bin/bash
#
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -e
set -u
set -o pipefail

useradd --no-create-home foo-bar
useradd --no-create-home expired
chpasswd <<EOF
foo-bar:secret
expired:secret
EOF
usermod --expiredate 1970-01-07 expired
