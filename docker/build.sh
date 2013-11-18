#! /bin/sh

# build the base wheezy image, if it doesn't already exist
(docker images | grep '^arvados/debian') || \
  ./mkimage-debootstrap.sh arvados/debian wheezy http://debian.lcs.mit.edu/debian/

# build the Docker images
docker build -t arvados/base base

mkdir -p api/generated
tar -c -z -f api/generated/api.tar.gz -C ../services api
docker build -t arvados/api api

mkdir -p docserver/generated
tar -c -z -f docserver/generated/doc.tar.gz -C .. doc
docker build -t arvados/docserver docserver

mkdir -p workbench/generated
tar -c -z -f workbench/generated/workbench.tar.gz -C ../apps workbench
docker build -t arvados/workbench workbench

docker build -t arvados/warehouse warehouse

