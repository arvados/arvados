#! /bin/sh

# create a cfi/debian base image
./mkimage-debootstrap.sh cfi/debian wheezy http://debian.lcs.mit.edu/debian/

# build the Docker base image
docker build -t cfi/base base
