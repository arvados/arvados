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
        -h|--help)
            echo "$0 [--no-reset-container] [--leave-running] [--config dev|localdemo] [--tag docker_tag]"
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

if test $reset_container = 1 ; then
    arvbox stop
    docker rm $ARVBOX_CONTAINER
    arvbox reset -f
fi

arvbox start $config $tag

arvbox pipe <<EOF
set -eu -o pipefail

. /usr/local/lib/arvbox/common.sh

if test $config = dev ; then
  cd /usr/src/arvados/sdk/cwl
  python setup.py sdist
  pip_install \$(ls -r dist/arvados-cwl-runner-*.tar.gz | head -n1)
fi

pip install cwltest

mkdir -p /tmp/cwltest
cd /tmp/cwltest
if ! test -d common-workflow-language ; then
  git clone https://github.com/common-workflow-language/common-workflow-language.git
fi
cd common-workflow-language
git pull
export ARVADOS_API_HOST=localhost:8000
export ARVADOS_API_HOST_INSECURE=1
export ARVADOS_API_TOKEN=\$(cat /var/lib/arvados/superuser_token)


if test "$tag" = "latest" ; then
  arv-keepdocker --pull arvados/jobs $tag
else
  jobsimg=\$(curl http://versions.arvados.org/v1/commit/$tag | python -c "import json; import sys; sys.stdout.write(json.load(sys.stdin)['Versions']['Docker']['arvados/jobs'])")
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
exec ./run_test.sh RUNNER=/tmp/cwltest/arv-cwl-containers EXTRA=--compute-checksum $@
EOF

CODE=$?

if test $leave_running = 0 ; then
    arvbox stop
fi

exit $CODE
