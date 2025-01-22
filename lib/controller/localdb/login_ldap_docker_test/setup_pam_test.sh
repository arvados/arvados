#!/bin/bash
#
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0
#
# This script should be mounted in the PAM test controller at /setup.sh.
# It creates the user account fixtures necessary for the test in passwd.

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
