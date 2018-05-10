#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

function usage {
    echo >&2
    echo >&2 "usage: $0 [options]"
    echo >&2
    echo >&2 "$0 options:"
    echo >&2 "  -t, --tags                    version tag for docker"
    echo >&2 "  -u, --upload                  Upload the images (docker push)"
    echo >&2 "  --no-cache                    Don't use build cache"
    echo >&2 "  -h, --help                    Display this help and exit"
    echo >&2
    echo >&2 "  If no options are given, just builds the images."
}
upload=false

# NOTE: This requires GNU getopt (part of the util-linux package on Debian-based distros).
TEMP=`getopt -o hut: \
    --long help,upload,no-cache,tags: \
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

# Sanity check
if ! [[ -n "$WORKSPACE" ]]; then
    echo >&2
    echo >&2 "Error: WORKSPACE environment variable not set"
    echo >&2
    exit 1
fi

echo $WORKSPACE

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

python_sdk_ts=$(cd sdk/python && timestamp_from_git)
cwl_runner_ts=$(cd sdk/cwl && timestamp_from_git)

python_sdk_version=$(cd sdk/python && nohash_version_from_git 0.1)
cwl_runner_version=$(cd sdk/cwl && nohash_version_from_git 1.0)

if [[ $python_sdk_ts -gt $cwl_runner_ts ]]; then
    cwl_runner_version=$(cd sdk/python && nohash_version_from_git 1.0)
fi

echo cwl_runner_version $cwl_runner_version python_sdk_version $python_sdk_version

cd docker/jobs
docker build $NOCACHE \
       --build-arg python_sdk_version=${python_sdk_version}-2 \
       --build-arg cwl_runner_version=${cwl_runner_version}-4 \
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
if ! [[ -z "$version_tag" ]]; then
    docker tag $FORCE arvados/jobs:$cwl_runner_version arvados/jobs:"$version_tag"
else
    docker tag $FORCE arvados/jobs:$cwl_runner_version arvados/jobs:latest
fi

ECODE=$?

if [[ "$ECODE" != "0" ]]; then
    EXITCODE=$(($EXITCODE + $ECODE))
fi

checkexit $ECODE "docker tag"
title "docker tag complete (`timer`)"

title "uploading images"

timer_reset

if [[ "$ECODE" != "0" ]]; then
    title "upload arvados images SKIPPED because build or tag failed"
else
    if [[ $upload == true ]]; then
        ## 20150526 nico -- *sometimes* dockerhub needs re-login
        ## even though credentials are already in .dockercfg
        docker login -u arvados
        if ! [[ -z "$version_tag" ]]; then
            docker_push arvados/jobs:"$version_tag"
        else
           docker_push arvados/jobs:$cwl_runner_version
           docker_push arvados/jobs:latest
        fi
        title "upload arvados images finished (`timer`)"
    else
        title "upload arvados images SKIPPED because no --upload option set (`timer`)"
    fi
fi

exit_cleanly
