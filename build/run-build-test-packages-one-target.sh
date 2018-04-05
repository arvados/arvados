#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

read -rd "\000" helpmessage <<EOF
$(basename $0): Build, test and (optionally) upload packages for one target

Syntax:
        WORKSPACE=/path/to/arvados $(basename $0) [options]

--target <target>
    Distribution to build packages for (default: debian8)
--upload
    If the build and test steps are successful, upload the packages
    to a remote apt repository (default: false)
--build-version <version>
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

PARSEDOPTS=$(getopt --name "$0" --longoptions \
    help,upload,target:,build-version: \
    -- "" "$@")
if [ $? -ne 0 ]; then
    exit 1
fi

TARGET=debian8
UPLOAD=0

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
        --upload)
            UPLOAD=1
            ;;
        --build-version)
            build_args+=("$1" "$2")
            shift
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

build_args+=(--target "$TARGET")

exit_cleanly() {
    trap - INT
    report_outcomes
    exit ${#failures}
}

COLUMNS=80
. $WORKSPACE/build/run-library.sh

title "Start build packages"
timer_reset

$WORKSPACE/build/run-build-packages-one-target.sh "${build_args[@]}"

checkexit $? "build packages"
title "End of build packages (`timer`)"

title "Start test packages"
timer_reset

if [ ${#failures[@]} -eq 0 ]; then
  $WORKSPACE/build/run-build-packages-one-target.sh "${build_args[@]}" --test-packages
else
  echo "Skipping package upload, there were errors building the packages"
fi

checkexit $? "test packages"
title "End of test packages (`timer`)"

if [[ "$UPLOAD" != 0 ]]; then
  title "Start upload packages"
  timer_reset

  if [ ${#failures[@]} -eq 0 ]; then
    echo "/usr/local/arvados-dev/jenkins/run_upload_packages.py -H jenkinsapt@apt.arvados.org -o Port=2222 --workspace $WORKSPACE $TARGET"
    /usr/local/arvados-dev/jenkins/run_upload_packages.py -H jenkinsapt@apt.arvados.org -o Port=2222 --workspace $WORKSPACE $TARGET
  else
    echo "Skipping package upload, there were errors building and/or testing the packages"
  fi
  checkexit $? "upload packages"
  title "End of upload packages (`timer`)"
fi

exit_cleanly
