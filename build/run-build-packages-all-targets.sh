#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

read -rd "\000" helpmessage <<EOF
$(basename $0): Orchestrate run-build-packages.sh for every target

Syntax:
        WORKSPACE=/path/to/arvados $(basename $0) [options]

Options:

--command
    Build command to execute (default: use built-in Docker image command)
--test-packages
    Run package install tests
--debug
    Output debug information (default: false)
--build-version <string>
    Version to build (default:
    \$ARVADOS_BUILDING_VERSION-\$ARVADOS_BUILDING_ITERATION or
    0.1.timestamp.commithash)

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

set -e

PARSEDOPTS=$(getopt --name "$0" --longoptions \
    help,test-packages,debug,command:,only-test:,build-version: \
    -- "" "$@")
if [ $? -ne 0 ]; then
    exit 1
fi

COMMAND=
DEBUG=
TEST_PACKAGES=
ONLY_TEST=

eval set -- "$PARSEDOPTS"
while [ $# -gt 0 ]; do
    case "$1" in
        --help)
            echo >&2 "$helpmessage"
            echo >&2
            exit 1
            ;;
        --debug)
            DEBUG="--debug"
            ;;
        --command)
            COMMAND="$2"; shift
            ;;
        --test-packages)
            TEST_PACKAGES="--test-packages"
            ;;
        --only-test)
            ONLY_TEST="$1 $2"; shift
            ;;
        --build-version)
            ARVADOS_BUILDING_VERSION="$2"; shift
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

cd $(dirname $0)

FINAL_EXITCODE=0

for dockerfile_path in $(find -name Dockerfile | grep package-build-dockerfiles); do
    if ./run-build-packages-one-target.sh --target "$(basename $(dirname "$dockerfile_path"))" --command "$COMMAND" --build-version "$ARVADOS_BUILDING_VERSION" $DEBUG $TEST_PACKAGES $ONLY_TEST ; then
        true
    else
        FINAL_EXITCODE=$?
        echo
        echo "Build packages failed for $(basename $(dirname "$dockerfile_path"))"
        echo
    fi
done

if test $FINAL_EXITCODE != 0 ; then
    echo
    echo "Build packages failed with code $FINAL_EXITCODE" >&2
    echo
fi

exit $FINAL_EXITCODE
