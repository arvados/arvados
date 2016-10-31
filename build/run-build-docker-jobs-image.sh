#!/bin/bash

function usage {
    echo >&2
    echo >&2 "usage: $0 [options]"
    echo >&2
    echo >&2 "$0 options:"
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
                  echo "WARNING: --tags is deprecated and doesn't do anything";
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

python_sdk_version=$(cd sdk/python && nohash_version_from_git 0.1)-2
cwl_runner_version=$(cd sdk/cwl && nohash_version_from_git 1.0)-3

if [[ $python_sdk_ts -gt $cwl_runner_ts ]]; then
    cwl_runner_version=$(cd sdk/python && nohash_version_from_git 1.0)-3
    gittag=$(cd sdk/python && git log --first-parent --max-count=1 --format=format:%H)
else
    gittag=$(cd sdk/cwl && git log --first-parent --max-count=1 --format=format:%H)
fi

echo cwl_runner_version $cwl_runner_version python_sdk_version $python_sdk_version

cd docker/jobs
docker build $NOCACHE \
       --build-arg python_sdk_version=$python_sdk_version \
       --build-arg cwl_runner_version=$cwl_runner_version \
       -t arvados/jobs:$gittag .

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

docker tag $FORCE arvados/jobs:$gittag arvados/jobs:latest

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

        docker_push arvados/jobs:$gittag
        docker_push arvados/jobs:latest
        title "upload arvados images finished (`timer`)"
    else
        title "upload arvados images SKIPPED because no --upload option set (`timer`)"
    fi
fi

exit_cleanly
