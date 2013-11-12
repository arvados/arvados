#! /bin/sh

# build the base wheezy image, if it doesn't already exist
(docker images | grep '^arvados/debian') || \
  ./mkimage-debootstrap.sh arvados/debian wheezy http://debian.lcs.mit.edu/debian/

# build the Docker images
docker build -t arvados/base base
docker build -t arvados/api api
docker build -t arvados/docserver docserver
