#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

function usage {
    echo >&2
    echo >&2 "usage: WORKSPACE=/path/to/arvados $0 [options]"
    echo >&2
    echo >&2 "$0 options:"
    echo >&2 "  -t, --tags                    version tag for docker"
    echo >&2 "  -r, --repo                    Arvados package repo to use: dev (default), testing, stable"
    echo >&2 "  -u, --upload                  Upload the images (docker push)"
    echo >&2 "  --no-cache                    Don't use build cache"
    echo >&2 "  -h, --help                    Display this help and exit"
    echo >&2
    echo >&2 "  WORKSPACE=path                Path to the Arvados source tree to build from"
    echo >&2
}
upload=false
REPO=dev

# NOTE: This requires GNU getopt (part of the util-linux package on Debian-based distros).
TEMP=`getopt -o hut:r: \
    --long help,upload,no-cache,tags:,repo: \
    -n "$0" -- "$@"`

if [ $? != 0 ] ; then echo "Use -h for help"; exit 1 ; fi
# Note the quotes around `$TEMP': they are essential!
eval set -- "$TEMP"

while [ $# -ge 1 ]
do
    case $1 in
        -u | --upload)
            upload=true
            shift
            ;;
        --no-cache)
            NOCACHE=--no-cache
            shift
            ;;
        -t | --tags)
            case "$2" in
                "")
                  echo "ERROR: --tags needs a parameter";
                  usage;
                  exit 1
                  ;;
                *)
                  version_tag="$2";
                  shift 2
                  ;;
            esac
            ;;
        -r | --repo)
            case "$2" in
                "")
                  echo "ERROR: --repo needs a parameter";
                  usage;
                  exit 1
                  ;;
                *)
                  REPO="$2";
                  shift 2
                  ;;
            esac
            ;;
        --)
            shift
            break
            ;;
        *)
            usage
            exit 1
            ;;
    esac
done

EXITCODE=0

exit_cleanly() {
    trap - INT
    report_outcomes
    exit $EXITCODE
}

# Sanity check
if ! [[ -n "$WORKSPACE" ]]; then
    usage;
    echo >&2 "Error: WORKSPACE environment variable not set"
    echo >&2
    exit 1
fi

echo $WORKSPACE

COLUMNS=80
. $WORKSPACE/build/run-library.sh

docker_push () {
    # Sometimes docker push fails; retry it a few times if necessary.
    for i in `seq 1 5`; do
        $DOCKER push $*
        ECODE=$?
        if [[ "$ECODE" == "0" ]]; then
            break
        fi
    done

    if [[ "$ECODE" != "0" ]]; then
        EXITCODE=$(($EXITCODE + $ECODE))
    fi
    checkexit $ECODE "docker push $*"
}

# find the docker binary
DOCKER=`which docker.io`

if [[ "$DOCKER" == "" ]]; then
    DOCKER=`which docker`
fi

if [[ "$DOCKER" == "" ]]; then
    title "Error: you need to have docker installed. Could not find the docker executable."
    exit 1
fi

# DOCKER
title "Starting docker build"

timer_reset

# clean up the docker build environment
cd "$WORKSPACE"

if [[ -z "$ARVADOS_BUILDING_VERSION" ]] && ! [[ -z "$version_tag" ]]; then
	export ARVADOS_BUILDING_VERSION="$version_tag"
	export ARVADOS_BUILDING_ITERATION="1"
fi

# This defines python_sdk_version and cwl_runner_version with python-style
# package suffixes (.dev/rc)
calculate_python_sdk_cwl_package_versions

if [[ -z "$cwl_runner_version" ]]; then
  echo "ERROR: cwl_runner_version is empty";
  exit 1
fi

echo cwl_runner_version $cwl_runner_version python_sdk_version $python_sdk_version

# For development and release candidate packages, the OS package has a "~dev"
# or "~rc" suffix, but Python requires a ".dev" or "rc" suffix.
#
# Arvados-cwl-runner will be expecting the Python-compatible version string
# when it tries to pull the Docker image, so we use that to tag the Docker
# image.
#
# The --build-arg docker invocation arguments are expecting the OS package
# version.
python_sdk_version_os=$(echo -n $python_sdk_version | sed s/.dev/~dev/g | sed s/rc/~rc/g)
cwl_runner_version_os=$(echo -n $cwl_runner_version | sed s/.dev/~dev/g | sed s/rc/~rc/g)

if [[ "${python_sdk_version}" != "${ARVADOS_BUILDING_VERSION}" ]]; then
	python_sdk_version_os="${python_sdk_version_os}-1"
else
	python_sdk_version_os="${ARVADOS_BUILDING_VERSION}-${ARVADOS_BUILDING_ITERATION}"
fi

if [[ "${cwl_runner_version_os}" != "${ARVADOS_BUILDING_VERSION}" ]]; then
	cwl_runner_version_os="${cwl_runner_version_os}-1"
else
	cwl_runner_version_os="${ARVADOS_BUILDING_VERSION}-${ARVADOS_BUILDING_ITERATION}"
fi

cd docker/jobs
docker build $NOCACHE \
       --build-arg python_sdk_version=${python_sdk_version_os} \
       --build-arg cwl_runner_version=${cwl_runner_version_os} \
       --build-arg repo_version=${REPO} \
       -t arvados/jobs:$cwl_runner_version .

ECODE=$?

if [[ "$ECODE" != "0" ]]; then
    EXITCODE=$(($EXITCODE + $ECODE))
fi

checkexit $ECODE "docker build"
title "docker build complete (`timer`)"

if [[ "$ECODE" != "0" ]]; then
  exit_cleanly
fi

timer_reset

if docker --version |grep " 1\.[0-9]\." ; then
    # Docker version prior 1.10 require -f flag
    # -f flag removed in Docker 1.12
    FORCE=-f
fi

title "uploading images"

timer_reset

if [[ "$EXITCODE" != "0" ]]; then
    title "upload arvados images SKIPPED because build or tag failed"
else
    if [[ $upload == true ]]; then
        ## 20150526 nico -- *sometimes* dockerhub needs re-login
        ## even though credentials are already in .dockercfg
        docker login -u arvados
        docker_push arvados/jobs:$cwl_runner_version
        title "upload arvados images finished (`timer`)"
    else
        title "upload arvados images SKIPPED because no --upload option set (`timer`)"
    fi
fi

exit_cleanly
