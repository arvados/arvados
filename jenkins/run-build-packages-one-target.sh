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

WORKSPACE=path         Path to the Arvados source tree to build packages from

EOF

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
    help,debug,test-packages,target:,command: \
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
    if [[ -n "$(find $WORKSPACE/packages/$TARGET -name *.rpm)" ]] ; then
        createrepo $WORKSPACE/packages/$TARGET
    fi

    if [[ -n "$(find $WORKSPACE/packages/$TARGET -name *.deb)" ]] ; then
        (cd $WORKSPACE/packages/$TARGET
         dpkg-scanpackages .  2> >(grep -v 'warning' 1>&2) | gzip -c > Packages.gz
        )
    fi

    COMMAND="/jenkins/test-packages-$TARGET.sh"
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
    pushd "$JENKINS_DIR/dockerfiles"
fi

echo $TARGET
cd $TARGET
time docker build --tag=$IMAGE .
popd

FINAL_EXITCODE=0

if docker run --rm -v "$JENKINS_DIR:/jenkins" -v "$WORKSPACE:/arvados" \
          --env ARVADOS_DEBUG=1 "$IMAGE" $COMMAND; then
    # Success - nothing more to do.
    true
else
    FINAL_EXITCODE=$?
    echo "ERROR: $tag build failed with exit status $FINAL_EXITCODE." >&2
fi

exit $FINAL_EXITCODE
