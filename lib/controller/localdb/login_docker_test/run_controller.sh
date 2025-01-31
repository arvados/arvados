#!/bin/bash
#
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0
#
# This script is the entrypoint for test containers. If the test mounts a
# /setup.sh script in the container, it runs that first, then starts the
# controller.

set -e
set -u
set -o pipefail

if [[ -e /setup.sh ]]; then
    . /setup.sh
fi
exec arvados-server controller
