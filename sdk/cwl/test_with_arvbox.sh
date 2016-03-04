#!/bin/sh

if ! which arvbox >/dev/null ; then
    export PATH=$PATH:$(readlink -f $(dirname $0)/../../tools/arvbox/bin)
fi

reset_container=1
leave_running=0

while test -n "$1" ; do
    arg="$1"
    case "$arg" in
        --no-reset-container)
            reset_container=0
            shift;
            ;;
        --leave-running)
            leave_running=1
            shift;
            ;;
        -*)
            break
            ;;
    esac
done

if test -z "$ARVBOX_CONTAINER" ; then
   export ARVBOX_CONTAINER=cwltest
fi

if test $reset_container = 1 ; then
    arvbox reset -f
    arvbox build dev
fi

arvbox start dev

arvbox pipe <<EOF
set -eu -o pipefail

. /usr/local/lib/arvbox/common.sh

cd /usr/src/arvados/sdk/cwl
python setup.py sdist
pip_install \$(ls dist/arvados-cwl-runner-*.tar.gz | tail -n1)

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
env
exec ./run_test.sh "$@"
EOF

CODE=$?

if test $leave_running = 0 ; then
    arvbox stop
fi

exit $CODE
