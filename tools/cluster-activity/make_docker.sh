#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

read -rd "\000" helpmessage <<EOF
Build an arvados/cluster-activity Docker image from local git tree.

Intended for use by developers working on arvados/cluster-activity or
and need to run a crunch job with a custom package version.

Syntax:
        WORKSPACE=/path/to/arvados $(basename $0)

WORKSPACE=path         Path to the Arvados source tree to build packages from

EOF

set -e

if [[ -z "$WORKSPACE" ]] ; then
    export WORKSPACE=$(readlink -f $(dirname $0)/../..)
    echo "Using WORKSPACE $WORKSPACE"
fi

context_dir="$(mktemp --directory --tmpdir dev-jobs.XXXXXXXX)"
trap 'rm -rf "$context_dir"' EXIT INT TERM QUIT

cluster_activity_version=$(cd $WORKSPACE/tools/cluster-activity && python3 arvados_version.py)

for src_dir in "$WORKSPACE/sdk/python" "$WORKSPACE/tools/crunchstat-summary" "$WORKSPACE/tools/cluster-activity" ; do
    env -C "$src_dir" python3 setup.py sdist --dist-dir="$context_dir"
done

set -x
docker build --no-cache \
       -f "$WORKSPACE/tools/cluster-activity/cluster-activity.dockerfile" \
       -t arvados/cluster-activity:$cluster_activity_version \
       "$context_dir"
