#!/bin/bash

set -e

#/root/dnd.sh &
/migrator/dnd.sh &
sleep 2

arv-get $1 | docker load

docker tag $2 $3:$4

docker images -a

kill $(cat /var/run/docker.pid)
sleep 1

cd /root/pkgs
dpkg -i libltdl7_2.4.2-1.11+b1_amd64.deb  docker-engine_1.13.1-0~debian-jessie_amd64.deb

/migrator/dnd.sh &
sleep 2

docker images -a

UUID=$(arv-keepdocker --project-uuid=$5 $3 $4)

kill $(cat /var/run/docker.pid)
sleep 1

chmod ugo+rwx -R /var/lib/docker

echo "Migrated uuid is $UUID"
