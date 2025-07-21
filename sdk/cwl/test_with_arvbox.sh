#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

set -x

cwldir=$(readlink -f $(dirname $0))

if ! which arvbox >/dev/null ; then
    export PATH=$PATH:$cwldir/../../tools/arvbox/bin
fi

reset_container=1
leave_running=0
config=dev
devcwl=0
tag="latest"
suite=conformance
runapi=containers
reinstall=0

while test -n "$1" ; do
    arg="$1"
    case "$arg" in
        --no-reset-container)
            reset_container=0
            shift
            ;;
        --leave-running)
            leave_running=1
            shift
            ;;
        --config)
            config=$2
            shift ; shift
            ;;
        --tag)
            tag=$2
            shift ; shift
            ;;
        --build)
            echo "warning: --build option is no longer supported; ignored" >&2
            shift
            ;;
        --devcwl)
            devcwl=1
            shift
            ;;
        --reinstall)
            reinstall=1
            shift
            ;;
        --pythoncmd)
            echo "warning: --pythoncmd option is no longer supported; ignored" >&2
            shift ; shift
            ;;
        --suite)
            suite=$2
            shift ; shift
            ;;
	--api)
	    runapi=$2
            shift ; shift
            ;;
        -h|--help)
            echo "$0 [--no-reset-container] [--leave-running] [--config dev|localdemo] [--tag docker_tag] [--suite (integration|conformance-v1.0|conformance-*)]"
            exit
            ;;
        *)
            break
            ;;
    esac
done

if test -z "$ARVBOX_CONTAINER" ; then
   export ARVBOX_CONTAINER=cwltest
fi

if test "$suite" = "conformance" ; then
  suite=conformance-v1.0
fi

if test $reset_container = 1 ; then
    arvbox stop
    docker rm $ARVBOX_CONTAINER
    arvbox reset -f
fi

arvbox start $config $tag

githead=$(git rev-parse HEAD)

arvbox pipe <<EOF
. /opt/arvados-py/bin/activate
set -eu -o pipefail

. /usr/local/lib/arvbox/common.sh

# Switch to the branch that the outer script is running from,
# this ensures we get the right version of tests and a-c-r
cd /usr/src/arvados
git config --global --add safe.directory /usr/src/arvados
git fetch -a
git checkout -f $githead

pip install -r build/requirements.tests.txt
if test $config = dev -o $reinstall = 1; then
  pip_install_sdist sdk/python sdk/cwl
fi

set -x

export ARVADOS_API_HOST=localhost:8000
export ARVADOS_API_HOST_INSECURE=1
export ARVADOS_API_TOKEN=\$(cat /var/lib/arvados-arvbox/superuser_token)
env

arvados-cwl-runner --version
cwltest --version

cd /usr/src/arvados/sdk/cwl
if [[ "$suite" = "integration" ]] ; then
   pytest -v -m "integration and not cwl_conformance"
elif [[ "$suite" = "conformance-v1.2" ]] ; then
   pytest tests/test_conformance.py::test_conformance_1_2
elif [[ "$suite" = "conformance-v1.1" ]] ; then
   pytest tests/test_conformance.py::test_conformance_1_1
elif [[ "$suite" = "conformance-v1.0" ]] ; then
   pytest tests/test_conformance.py::test_conformance_1_0
else
   echo "ERROR: unknown suite '$suite'" >&2
   exit 2
fi
EOF

CODE=$?

# FIXME?: see comment in tests/__init__.py::run_cwltest
# docker cp -L $ARVBOX_CONTAINER:/tmp/badges $cwldir/badges

if test $leave_running = 0 ; then
    arvbox stop
fi

exit $CODE
