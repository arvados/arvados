#!/bin/bash
#
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -e
set -u
set -o pipefail

net_name="$1"; shift

docker network inspect "$net_name" |
    jq -r 'map(.Containers | keys) | flatten | join("\n")' |
    xargs -r -d\\n docker stop
docker network rm "$net_name"
