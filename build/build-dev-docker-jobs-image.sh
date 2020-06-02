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
PYCMD=pythonexec       (optional) Specify the python executable to use in the docker image. Defaults to "python3".

EOF

set -e

if [[ -z "$WORKSPACE" ]] ; then
    export WORKSPACE=$(readlink -f $(dirname $0)/..)
    echo "Using WORKSPACE $WORKSPACE"
fi

if [[ -z "$ARVADOS_API_HOST" || -z "$ARVADOS_API_TOKEN" ]] ; then
    echo "$helpmessage"
    echo
    echo "Must set ARVADOS_API_HOST and ARVADOS_API_TOKEN"
    exit 1
fi

cd "$WORKSPACE"

py=python3
pipcmd=pip
if [[ -n "$PYCMD" ]] ; then
    py="$PYCMD"
fi
if [[ $py = python3 ]] ; then
    pipcmd=pip3
fi

(cd sdk/python && python setup.py sdist)
sdk=$(cd sdk/python/dist && ls -t arvados-python-client-*.tar.gz | head -n1)

(cd sdk/cwl && python setup.py sdist)
runner=$(cd sdk/cwl/dist && ls -t arvados-cwl-runner-*.tar.gz | head -n1)

rm -rf sdk/cwl/salad_dist
mkdir -p sdk/cwl/salad_dist
if [[ -n "$SALAD" ]] ; then
    (cd "$SALAD" && python setup.py sdist)
    salad=$(cd "$SALAD/dist" && ls -t schema-salad-*.tar.gz | head -n1)
    cp "$SALAD/dist/$salad" $WORKSPACE/sdk/cwl/salad_dist
fi

rm -rf sdk/cwl/cwltool_dist
mkdir -p sdk/cwl/cwltool_dist
if [[ -n "$CWLTOOL" ]] ; then
    (cd "$CWLTOOL" && python setup.py sdist)
    cwltool=$(cd "$CWLTOOL/dist" && ls -t cwltool-*.tar.gz | head -n1)
    cp "$CWLTOOL/dist/$cwltool" $WORKSPACE/sdk/cwl/cwltool_dist
fi

. build/run-library.sh

calculate_python_sdk_cwl_package_versions

set -x
docker build --no-cache --build-arg sdk=$sdk --build-arg runner=$runner --build-arg salad=$salad --build-arg cwltool=$cwltool --build-arg pythoncmd=$py --build-arg pipcmd=$pipcmd -f "$WORKSPACE/sdk/dev-jobs.dockerfile" -t arvados/jobs:$cwl_runner_version "$WORKSPACE/sdk"
echo arv-keepdocker arvados/jobs $cwl_runner_version
arv-keepdocker arvados/jobs $cwl_runner_version
