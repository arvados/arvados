#! /bin/sh

# create a arvados/debian base image
./mkimage-debootstrap.sh arvados/debian wheezy http://debian.lcs.mit.edu/debian/

# build the Docker base image
docker build -t arvados/base base
