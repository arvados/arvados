#!/bin/bash

read -rd "\000" helpmessage <<EOF
$(basename $0): Orchestrate run-build-packages.sh for one target

Syntax:
        WORKSPACE=/path/to/arvados $(basename $0) [options]

--target <target>
    Distribution to build packages for (default: debian7)
--command
    Build command to execute (default: use built-in Docker image command)
--test-packages
    Run package install test script "test-packages-$target.sh"
--debug
    Output debug information (default: false)
--only-test
    Test only a specific package

WORKSPACE=path         Path to the Arvados source tree to build packages from

EOF

set -e

if ! [[ -n "$WORKSPACE" ]]; then
  echo >&2 "$helpmessage"
  echo >&2
  echo >&2 "Error: WORKSPACE environment variable not set"
  echo >&2
  exit 1
fi

if ! [[ -d "$WORKSPACE" ]]; then
  echo >&2 "$helpmessage"
  echo >&2
  echo >&2 "Error: $WORKSPACE is not a directory"
  echo >&2
  exit 1
fi

PARSEDOPTS=$(getopt --name "$0" --longoptions \
    help,debug,test-packages,target:,command:,only-test: \
    -- "" "$@")
if [ $? -ne 0 ]; then
    exit 1
fi

TARGET=debian7
COMMAND=
DEBUG=

eval set -- "$PARSEDOPTS"
while [ $# -gt 0 ]; do
    case "$1" in
        --help)
            echo >&2 "$helpmessage"
            echo >&2
            exit 1
            ;;
        --target)
            TARGET="$2"; shift
            ;;
        --only-test)
            packages="$2"; shift
            ;;
        --debug)
            DEBUG=" --debug"
            ;;
        --command)
            COMMAND="$2"; shift
            ;;
        --test-packages)
            test_packages=1
            ;;
        --)
            if [ $# -gt 1 ]; then
                echo >&2 "$0: unrecognized argument '$2'. Try: $0 --help"
                exit 1
            fi
            ;;
    esac
    shift
done

set -e

if [[ -n "$test_packages" ]]; then
    if [[ -n "$(find $WORKSPACE/packages/$TARGET -name '*.rpm')" ]] ; then
        createrepo $WORKSPACE/packages/$TARGET
    fi

    if [[ -n "$(find $WORKSPACE/packages/$TARGET -name '*.deb')" ]] ; then
        (cd $WORKSPACE/packages/$TARGET
         dpkg-scanpackages .  2> >(grep -v 'warning' 1>&2) | gzip -c > Packages.gz
        )
    fi

    COMMAND="/jenkins/package-testing/test-packages-$TARGET.sh"
    IMAGE="arvados/package-test:$TARGET"
else
    IMAGE="arvados/build:$TARGET"
    if [[ "$COMMAND" != "" ]]; then
        COMMAND="/usr/local/rvm/bin/rvm-exec default bash /jenkins/$COMMAND --target $TARGET$DEBUG"
    fi
fi

JENKINS_DIR=$(dirname "$(readlink -e "$0")")

if [[ -n "$test_packages" ]]; then
    pushd "$JENKINS_DIR/package-test-dockerfiles"
else
    pushd "$JENKINS_DIR/package-build-dockerfiles"
    make "$TARGET/generated"
fi

echo $TARGET
cd $TARGET
time docker build --tag=$IMAGE .
popd

if test -z "$packages" ; then
    packages="arvados-api-server
        arvados-data-manager
        arvados-docker-cleaner
        arvados-git-httpd
        arvados-node-manager
        arvados-src
        arvados-workbench
        crunch-dispatch-local
        crunch-dispatch-slurm
        crunch-run
        crunchstat
        keep-balance
        keep-block-check
        keepproxy
        keep-rsync
        keepstore
        keep-web
        libarvados-perl"

    case "$TARGET" in
        centos6)
            packages="$packages python27-python-arvados-fuse
                  python27-python-arvados-python-client python27-python-arvados-cwl-runner"
            ;;
        *)
            packages="$packages python-arvados-fuse
                  python-arvados-python-client python-arvados-cwl-runner"
            ;;
    esac
fi

FINAL_EXITCODE=0

package_fails=""

mkdir -p "$WORKSPACE/apps/workbench/vendor/cache-$TARGET"
mkdir -p "$WORKSPACE/services/api/vendor/cache-$TARGET"

docker_volume_args=(
    -v "$JENKINS_DIR:/jenkins"
    -v "$WORKSPACE:/arvados"
    -v /arvados/services/api/vendor/bundle
    -v /arvados/apps/workbench/vendor/bundle
    -v "$WORKSPACE/services/api/vendor/cache-$TARGET:/arvados/services/api/vendor/cache"
    -v "$WORKSPACE/apps/workbench/vendor/cache-$TARGET:/arvados/apps/workbench/vendor/cache"
)

if [[ -n "$test_packages" ]]; then
    for p in $packages ; do
        echo
        echo "START: $p test on $IMAGE" >&2
        if docker run --rm \
            "${docker_volume_args[@]}" \
            --env ARVADOS_DEBUG=1 \
            --env "TARGET=$TARGET" \
            --env "WORKSPACE=/arvados" \
            "$IMAGE" $COMMAND $p
        then
            echo "OK: $p test on $IMAGE succeeded" >&2
        else
            FINAL_EXITCODE=$?
            package_fails="$package_fails $p"
            echo "ERROR: $p test on $IMAGE failed with exit status $FINAL_EXITCODE" >&2
        fi
    done
else
    echo
    echo "START: build packages on $IMAGE" >&2
    if docker run --rm \
        "${docker_volume_args[@]}" \
        --env ARVADOS_DEBUG=1 \
        "$IMAGE" $COMMAND
    then
        echo
        echo "OK: build packages on $IMAGE succeeded" >&2
    else
        FINAL_EXITCODE=$?
        echo "ERROR: build packages on $IMAGE failed with exit status $FINAL_EXITCODE" >&2
    fi
fi

if test -n "$package_fails" ; then
    echo "Failed package tests:$package_fails" >&2
fi

exit $FINAL_EXITCODE
