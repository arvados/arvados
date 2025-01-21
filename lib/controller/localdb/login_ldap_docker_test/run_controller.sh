#!/bin/bash
#
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -e
set -u
set -o pipefail

if [[ -e /setup.sh ]]; then
    . /setup.sh
fi
exec arvados-server controller
