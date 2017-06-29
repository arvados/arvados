#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

# This script is called by arv-migrate-docker19 to perform the actual migration
# of a single image.  This works by running Docker-in-Docker (dnd.sh) to
# download the image using Docker 1.9 and then upgrading to Docker 1.13 and
# uploading the converted image.

# When using bash in pid 1 and using "trap on EXIT"
# it will sometimes go into an 100% CPU infinite loop.
#
# Using workaround from here:
#
# https://github.com/docker/docker/issues/4854
if [ "$$" = 1 ]; then
  $0 "$@"
  exit $?
fi

# -x           show script
# -e           exit on error
# -o pipefail  use exit code from 1st failure in pipeline, not last
set -x -e -o pipefail

image_tar_keepref=$1
image_id=$2
image_repo=$3
image_tag=$4
project_uuid=$5
graph_driver=$6

if [[ "$image_repo" = "<none>" ]] ; then
  image_repo=none
  image_tag=latest
fi

# Print free space in /var/lib/docker
function freespace() {
    df -B1 /var/lib/docker | tail -n1 | sed 's/  */ /g' | cut -d' ' -f4
}

# Run docker-in-docker script and then wait for it to come up
function start_docker {
    /root/dnd.sh $graph_driver &
    for i in $(seq 1 10) ; do
        if docker version >/dev/null 2>/dev/null ; then
            return
        fi
        sleep 1
    done
    false
}

# Kill docker from pid then wait for it to be down
function kill_docker {
    if test -f /var/run/docker.pid ; then
        kill $(cat /var/run/docker.pid)
    fi
    for i in $(seq 1 10) ; do
        if ! docker version >/dev/null 2>/dev/null ; then
            return
        fi
        sleep 1
    done
    false
}

# Ensure that we clean up docker graph and/or lingering cache files on exit
function cleanup {
    kill_docker
    rm -rf /var/lib/docker/*
    rm -rf /root/.cache/arvados/docker/*
    echo "Available space after cleanup is $(freespace)"
}

trap cleanup EXIT

start_docker

echo "Initial available space is $(freespace)"

arv-get $image_tar_keepref | docker load


docker tag $image_id $image_repo:$image_tag

docker images -a

kill_docker

echo "Available space after image load is $(freespace)"

cd /root/pkgs
dpkg -i libltdl7_2.4.2-1.11+b1_amd64.deb docker-engine_1.13.1-0~debian-jessie_amd64.deb

echo "Available space after image upgrade is $(freespace)"

start_docker

docker images -a

if [[ "$image_repo" = "none" ]] ; then
  image_repo=$(docker images -a --no-trunc | sed 's/  */ /g' | grep ^none | cut -d' ' -f3)
  image_tag=""
fi

UUID=$(arv-keepdocker --force-image-format --project-uuid=$project_uuid $image_repo $image_tag)

echo "Available space after arv-keepdocker is $(freespace)"

echo "Migrated uuid is $UUID"
