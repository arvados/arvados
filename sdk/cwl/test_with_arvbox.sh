#!/bin/sh
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

set -x

if ! which arvbox >/dev/null ; then
    export PATH=$PATH:$(readlink -f $(dirname $0)/../../tools/arvbox/bin)
fi

reset_container=1
leave_running=0
config=dev
tag="latest"
pythoncmd=python
suite=conformance
runapi=containers

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
            build=1
            shift
            ;;
        --pythoncmd)
            pythoncmd=$2
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
            echo "$0 [--no-reset-container] [--leave-running] [--config dev|localdemo] [--tag docker_tag] [--build] [--pythoncmd python(2|3)] [--suite (integration|conformance-v1.0|conformance-v1.1)]"
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

if [[ "$suite" = "conformance" ]] ; then
  suite=conformance-v1.0
fi

if test $reset_container = 1 ; then
    arvbox stop
    docker rm $ARVBOX_CONTAINER
    arvbox reset -f
fi

arvbox start $config $tag

arvbox pipe <<EOF
set -eu -o pipefail

. /usr/local/lib/arvbox/common.sh

export PYCMD=$pythoncmd

if test $config = dev ; then
  cd /usr/src/arvados/sdk/cwl
  \$PYCMD setup.py sdist
  pip_install \$(ls -r dist/arvados-cwl-runner-*.tar.gz | head -n1)
fi

set -x

if [ \$PYCMD = "python3" ]; then
    pip3 install cwltest
else
    pip install cwltest
fi

mkdir -p /tmp/cwltest
cd /tmp/cwltest

if [[ "$suite" = "conformance-v1.0" ]] ; then
   if ! test -d common-workflow-language ; then
     git clone https://github.com/common-workflow-language/common-workflow-language.git
   fi
   cd common-workflow-language
elif [[ "$suite" = "conformance-v1.1" ]] ; then
   if ! test -d cwl-v1.1 ; then
     git clone https://github.com/common-workflow-language/cwl-v1.1.git
   fi
   cd cwl-v1.1
fi

if [[ "$suite" != "integration" ]] ; then
  git pull
fi

export ARVADOS_API_HOST=localhost:8000
export ARVADOS_API_HOST_INSECURE=1
export ARVADOS_API_TOKEN=\$(cat /var/lib/arvados/superuser_token)

if test -n "$build" ; then
  /usr/src/arvados/build/build-dev-docker-jobs-image.sh
elif test "$tag" = "latest" ; then
  arv-keepdocker --pull arvados/jobs $tag
else
  jobsimg=\$(curl https://versions.arvados.org/v1/commit/$tag | python -c "import json; import sys; sys.stdout.write(json.load(sys.stdin)['Versions']['Docker']['arvados/jobs'])")
  arv-keepdocker --pull arvados/jobs \$jobsimg
  docker tag arvados/jobs:\$jobsimg arvados/jobs:latest
  arv-keepdocker arvados/jobs latest
fi

cat >/tmp/cwltest/arv-cwl-jobs <<EOF2
#!/bin/sh
exec arvados-cwl-runner --api=jobs \\\$@
EOF2
chmod +x /tmp/cwltest/arv-cwl-jobs

cat >/tmp/cwltest/arv-cwl-containers <<EOF2
#!/bin/sh
exec arvados-cwl-runner --api=containers \\\$@
EOF2
chmod +x /tmp/cwltest/arv-cwl-containers

env
if [[ "$suite" = "integration" ]] ; then
   cd /usr/src/arvados/sdk/cwl/tests
   exec ./arvados-tests.sh $@
else
   exec ./run_test.sh RUNNER=/tmp/cwltest/arv-cwl-${runapi} EXTRA=--compute-checksum $@
fi
EOF

CODE=$?

if test $leave_running = 0 ; then
    arvbox stop
fi

exit $CODE
