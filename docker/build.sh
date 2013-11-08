#! /bin/sh

# build the base wheezy image, if it doesn't already exist
(docker images | grep '^arvados/debian') || \
  ./mkimage-debootstrap.sh arvados/debian wheezy http://debian.lcs.mit.edu/debian/

# build the Docker images
docker build -rm -t arvados/base base
docker build -rm -t arvados/api api
