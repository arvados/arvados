#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

read -rd "\000" helpmessage <<EOF
Build an arvados/jobs Docker image from local git tree.

Intended for use by developers working on arvados-python-client or
arvados-cwl-runner and need to run a crunch job with a custom package
version.  Also supports building custom cwltool if CWLTOOL is set.

Syntax:
        WORKSPACE=/path/to/arvados $(basename $0)

WORKSPACE=path         Path to the Arvados source tree to build packages from
CWLTOOL=path           (optional) Path to cwltool git repository.
SALAD=path             (optional) Path to schema_salad git repository.
CWL_UTILS=path         (optional) Path to cwl-utils git repository.

EOF

set -e

if [[ -z "$WORKSPACE" ]] ; then
    export WORKSPACE=$(readlink -f $(dirname $0)/..)
    echo "Using WORKSPACE $WORKSPACE"
fi

context_dir="$(mktemp --directory --tmpdir dev-jobs.XXXXXXXX)"
trap 'rm -rf "$context_dir"' EXIT INT TERM QUIT

for src_dir in "$WORKSPACE/sdk/python" "${CWLTOOL:-}" "${CWL_UTILS:-}" "${SALAD:-}" "$WORKSPACE/tools/crunchstat-summary" "$WORKSPACE/sdk/cwl"; do
    if [[ -z "$src_dir" ]]; then
        continue
    fi
    env -C "$src_dir" python3 setup.py sdist --dist-dir="$context_dir"
done

cd "$WORKSPACE"
. build/run-library.sh
# This defines python_sdk_version and cwl_runner_version with python-style
# package suffixes (.dev/rc)
calculate_python_sdk_cwl_package_versions

set -x
docker build --no-cache \
       -f "$WORKSPACE/sdk/dev-jobs.dockerfile" \
       -t arvados/jobs:$cwl_runner_version \
       "$context_dir"

arv-keepdocker arvados/jobs $cwl_runner_version
