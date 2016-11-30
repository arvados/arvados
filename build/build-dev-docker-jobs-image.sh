#!/bin/bash

read -rd "\000" helpmessage <<EOF
Build an arvados/jobs Docker image from local git tree.

Intended for use by developers working on arvados-python-client or
arvados-cwl-runner and need to run a crunch job with a custom package
version.  Also supports building custom cwltool if CWLTOOL is set.

Syntax:
        WORKSPACE=/path/to/arvados $(basename $0)

WORKSPACE=path         Path to the Arvados source tree to build packages from
CWLTOOL=path           (optional) Path to cwltool git repository.

EOF

set -e

if [[ -z "$WORKSPACE" ]] ; then
    echo "$helpmessage"
    echo
    echo "Must set WORKSPACE"
    exit 1
fi

if [[ -z "$ARVADOS_API_HOST" || -z "$ARVADOS_API_TOKEN" ]] ; then
    echo "$helpmessage"
    echo
    echo "Must set ARVADOS_API_HOST and ARVADOS_API_TOKEN"
    exit 1
fi

(cd "$WORKSPACE/sdk/python" && python setup.py sdist)
sdk=$(cd "$WORKSPACE/sdk/python/dist" && ls -t arvados-python-client-*.tar.gz | head -n1)

(cd "$WORKSPACE/sdk/cwl" && python setup.py sdist)
runner=$(cd "$WORKSPACE/sdk/cwl/dist" && ls -t arvados-cwl-runner-*.tar.gz | head -n1)

rm -rf "$WORKSPACE/sdk/cwl/cwltool_dist"
mkdir -p "$WORKSPACE/sdk/cwl/cwltool_dist"
if [[ -n "$CWLTOOL" ]] ; then
    (cd "$CWLTOOL" && python setup.py sdist)
    cwltool=$(cd "$CWLTOOL/dist" && ls -t cwltool-*.tar.gz | head -n1)
    cp "$CWLTOOL/dist/$cwltool" $WORKSPACE/sdk/cwl/cwltool_dist
fi

gittag=$(cd "$WORKSPACE/sdk/cwl" && git log --first-parent --max-count=1 --format=format:%H)
docker build --build-arg sdk=$sdk --build-arg runner=$runner --build-arg cwltool=$cwltool -f "$WORKSPACE/sdk/dev-jobs.dockerfile" -t arvados/jobs:$gittag "$WORKSPACE/sdk"
echo arv-keepdocker arvados/jobs $gittag
arv-keepdocker arvados/jobs $gittag
