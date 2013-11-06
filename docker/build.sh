#! /bin/sh

# build the base wheezy image
./mkimage-debootstrap.sh arvados/debian wheezy http://debian.lcs.mit.edu/debian/

# build the Docker images
docker build -t arvados/base base
docker build -t arvados/api api
