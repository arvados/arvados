#!/bin/bash

set -e

function cleanup {
    kill $(cat /var/run/docker.pid)
    sleep 1
    rm -rf /var/lib/docker/*
    rm -rf /root/.cache/arvados/docker/*
}

trap cleanup EXIT

/root/dnd.sh &
sleep 2

image_tar_keepref=$1
image_id=$2
image_repo=$3
image_tag=$4
project_uuid=$5

arv-get $image_tar_keepref | docker load

docker tag $image_id $image_repo:$image_tag

docker images -a

kill $(cat /var/run/docker.pid)
sleep 1

cd /root/pkgs
dpkg -i libltdl7_2.4.2-1.11+b1_amd64.deb  docker-engine_1.13.1-0~debian-jessie_amd64.deb

/root/dnd.sh &
sleep 2

docker images -a

UUID=$(arv-keepdocker --force-image-format --project-uuid=$project_uuid $image_repo $image_tag)

echo "Migrated uuid is $UUID"
