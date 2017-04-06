#!/bin/bash

set -e

function start_docker {
    /root/dnd.sh &
    for i in $(seq 1 10) ; do
        if docker version >/dev/null 2>/dev/null ; then
            return
        fi
        sleep 1
    done
    false
}

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

function cleanup {
    kill_docker
    rm -rf /var/lib/docker/*
    rm -rf /root/.cache/arvados/docker/*
}

trap cleanup EXIT

start_docker

image_tar_keepref=$1
image_id=$2
image_repo=$3
image_tag=$4
project_uuid=$5

arv-get $image_tar_keepref | docker load

docker tag $image_id $image_repo:$image_tag

docker images -a

kill_docker

cd /root/pkgs
dpkg -i libltdl7_2.4.2-1.11+b1_amd64.deb docker-engine_1.13.1-0~debian-jessie_amd64.deb

start_docker

docker images -a

UUID=$(arv-keepdocker --force-image-format --project-uuid=$project_uuid $image_repo $image_tag)

echo "Migrated uuid is $UUID"
