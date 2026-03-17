#!/bin/sh

# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

set -e
set -u

SECRET_ID="$1"
OUT_FILE="${2:-$RUNTIME_DIRECTORY/$SECRET_ID}"

while true; do
    aws secretsmanager get-secret-value --secret-id "$SECRET_ID" |
        jq -r .SecretString >"$OUT_FILE"
    sleep 1
done
