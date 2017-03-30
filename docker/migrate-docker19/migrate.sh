#!/bin/bash

set -e

image_tar_keepref=$1
image_id=$2
image_repo=$3
image_tag=$4
project_uuid=$5
graph_driver=$6

function freespace() {
    df -B1 /var/lib/docker | tail -n1 | sed 's/  */ /g' | cut -d' ' -f4
}

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
    trap EXIT
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

UUID=$(arv-keepdocker --force-image-format --project-uuid=$project_uuid $image_repo $image_tag)

echo "Available space after arv-keepdocker is $(freespace)"

echo "Migrated uuid is $UUID"
