#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

function usage {
    echo >&2
    echo >&2 "usage: $0 [options]"
    echo >&2
    echo >&2 "$0 options:"
    echo >&2 "  -t, --tags [csv_tags]         comma separated tags"
    echo >&2 "  -i, --images [dev,demo]       Choose which images to build (default: dev and demo)"
    echo >&2 "  -u, --upload                  Upload the images (docker push)"
    echo >&2 "  -h, --help                    Display this help and exit"
    echo >&2
    echo >&2 "  If no options are given, just builds the images."
}

upload=false
images=dev,demo

# NOTE: This requires GNU getopt (part of the util-linux package on Debian-based distros).
TEMP=`getopt -o hut:i: \
    --long help,upload,tags:,images: \
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
        -i | --images)
            case "$2" in
                "")
                  echo "ERROR: --images needs a parameter";
                  usage;
                  exit 1
                  ;;
                *)
                  images=$2;
                  shift 2
                  ;;
            esac
            ;;
        -t | --tags)
            case "$2" in
                "")
                  echo "ERROR: --tags needs a parameter";
                  usage;
                  exit 1
                  ;;
                *)
                  tags=$2;
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

COLUMNS=80

title () {
    printf "\n%*s\n\n" $(((${#title}+$COLUMNS)/2)) "********** $1 **********"
}

docker_push () {
    # docker always creates a local 'latest' tag, and we don't want to push that
    # tag in every case. Remove it.
    docker rmi $1:latest

    GITHEAD=$(cd $WORKSPACE && git log --format=%H -n1 HEAD)

    if [[ ! -z "$tags" ]]
    then
        for tag in $( echo $tags|tr "," " " )
        do
             $DOCKER tag $1:$GITHEAD $1:$tag
        done
    fi

    # Sometimes docker push fails; retry it a few times if necessary.
    for i in `seq 1 5`; do
        $DOCKER push $*
        ECODE=$?
        if [[ "$ECODE" == "0" ]]; then
            break
        fi
    done

    if [[ "$ECODE" != "0" ]]; then
        title "!!!!!! docker push $* failed !!!!!!"
        EXITCODE=$(($EXITCODE + $ECODE))
    fi
}

timer_reset() {
    t0=$SECONDS
}

timer() {
    echo -n "$(($SECONDS - $t0))s"
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

if [[ "$images" =~ demo ]]; then
  title "Starting arvbox build localdemo"

  tools/arvbox/bin/arvbox build localdemo
  ECODE=$?

  if [[ "$ECODE" != "0" ]]; then
      title "!!!!!! docker BUILD FAILED !!!!!!"
      EXITCODE=$(($EXITCODE + $ECODE))
  fi
fi

if [[ "$images" =~ dev ]]; then
  title "Starting arvbox build dev"

  tools/arvbox/bin/arvbox build dev

  ECODE=$?

  if [[ "$ECODE" != "0" ]]; then
      title "!!!!!! docker BUILD FAILED !!!!!!"
      EXITCODE=$(($EXITCODE + $ECODE))
  fi
fi

title "docker build complete (`timer`)"

if [[ "$EXITCODE" != "0" ]]; then
    title "upload arvados images SKIPPED because build failed"
else
    if [[ $upload == true ]]; then
        title "uploading images"
        timer_reset

        ## 20150526 nico -- *sometimes* dockerhub needs re-login
        ## even though credentials are already in .dockercfg
        docker login -u arvados

        if [[ "$images" =~ dev ]]; then
          docker_push arvados/arvbox-dev
        fi
        if [[ "$images" =~ demo ]]; then
          docker_push arvados/arvbox-demo
        fi
        title "upload arvados images complete (`timer`)"
    else
        title "upload arvados images SKIPPED because no --upload option set"
    fi
fi

exit $EXITCODE
