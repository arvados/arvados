#!/bin/bash
#
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0 OR Apache-2.0

set -e
set -u
set -o pipefail

declare -a gradle_opts=()
declare -a gradle_tasks=(clean test jar install)

if ! grep -E '^signing\.[[:alpha:]]+=[^[:space:]]' gradle.properties >/dev/null
then
    gradle_opts+=(--exclude-task=signArchives)
fi

for arg in "$@"
do
    case "$arg" in
        -*) gradle_opts+=("$arg") ;;
        *) gradle_tasks+=("$arg") ;;
    esac
done

set -x
exec gradle "${gradle_opts[@]}" "${gradle_tasks[@]}"
