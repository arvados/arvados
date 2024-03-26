#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

read -rd "\000" helpmessage <<EOF
$(basename $0): Build, test and (optionally) upload packages for one target

Syntax:
        WORKSPACE=/path/to/arvados $(basename $0) --target <target> [options]

--target <target>
    Distribution to build packages for
--only-build <package>
    Build only a specific package (or ONLY_BUILD from environment)
--arch <arch>
    Build a specific architecture (or ARCH from environment, defaults to native architecture)
--force-build
    Build even if the package exists upstream or if it has already been
    built locally
--force-test
    Test even if there is no new untested package
--upload
    If the build and test steps are successful, upload the packages
    to a remote apt repository (default: false)
--debug
    Output debug information (default: false)
--rc
    Optional Parameter to build Release Candidate
--build-version <version>
    Version to build (default:
    \$ARVADOS_BUILDING_VERSION-\$ARVADOS_BUILDING_ITERATION or
    0.1.timestamp.commithash)
--skip-docker-build
    Don't try to build Docker images

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
    help,debug,upload,rc,target:,force-test,only-build:,force-build,arch:,build-version:,skip-docker-build \
    -- "" "$@")
if [ $? -ne 0 ]; then
    exit 1
fi

UPLOAD=0
RC=0
DEBUG=
TARGET=

declare -a build_args=()

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
        --force-test)
            FORCE_TEST=1
            ;;
        --force-build)
            FORCE_BUILD=1
            ;;
        --only-build)
            ONLY_BUILD="$2"; shift
            ;;
        --arch)
            ARCH="$2"; shift
            ;;
        --debug)
            DEBUG=" --debug"
            ;;
        --upload)
            UPLOAD=1
            ;;
        --rc)
            RC=1
            ;;
        --build-version)
            build_args+=("$1" "$2")
            shift
            ;;
        --skip-docker-build)
            SKIP_DOCKER_BUILD=1
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

if [[ -z "$TARGET" ]]; then
    echo "FATAL: --target must be specified" >&2
    exit 2
elif [[ ! -d "$WORKSPACE/build/package-build-dockerfiles/$TARGET" ]]; then
    echo "FATAL: unknown build target '$TARGET'" >&2
    exit 2
fi

build_args+=(--target "$TARGET")

if [[ -n "$ONLY_BUILD" ]]; then
  build_args+=(--only-build "$ONLY_BUILD")
fi

if [[ -n "$FORCE_BUILD" ]]; then
  build_args+=(--force-build)
fi

if [[ -n "$FORCE_TEST" ]]; then
  build_args+=(--force-test)
fi

if [[ "$SKIP_DOCKER_BUILD" = 1 ]]; then
  build_args+=(--skip-docker-build)
fi

if [[ -n "$ARCH" ]]; then
  build_args+=(--arch "$ARCH")
fi

exit_cleanly() {
    trap - INT
    report_outcomes
    exit ${#failures}
}

COLUMNS=80
. $WORKSPACE/build/run-library.sh

title "Start build packages"
timer_reset

$WORKSPACE/build/run-build-packages-one-target.sh "${build_args[@]}"$DEBUG

checkexit $? "build packages"
title "End of build packages (`timer`)"

title "Start test packages"
timer_reset

if [ ${#failures[@]} -eq 0 ]; then
  $WORKSPACE/build/run-build-packages-one-target.sh "${build_args[@]}" --test-packages$DEBUG
else
  echo "Skipping package upload, there were errors building the packages"
fi

checkexit $? "test packages"
title "End of test packages (`timer`)"

if [[ "$UPLOAD" != 0 ]]; then
  title "Start upload packages"
  timer_reset

  if [ ${#failures[@]} -eq 0 ]; then
    if [[ "$RC" != 0 ]]; then
      echo "/usr/local/arvados-dev/jenkins/run_upload_packages.py --repo testing -H jenkinsapt@apt.arvados.org -o Port=2222 --workspace $WORKSPACE $TARGET"
      /usr/local/arvados-dev/jenkins/run_upload_packages.py --repo testing -H jenkinsapt@apt.arvados.org -o Port=2222 --workspace $WORKSPACE $TARGET
    else
      echo "/usr/local/arvados-dev/jenkins/run_upload_packages.py --repo dev -H jenkinsapt@apt.arvados.org -o Port=2222 --workspace $WORKSPACE $TARGET"
      /usr/local/arvados-dev/jenkins/run_upload_packages.py --repo dev -H jenkinsapt@apt.arvados.org -o Port=2222 --workspace $WORKSPACE $TARGET
    fi
  else
    echo "Skipping package upload, there were errors building and/or testing the packages"
  fi
  checkexit $? "upload packages"
  title "End of upload packages (`timer`)"
fi

exit_cleanly
