#!/bin/bash

EXITCODE=0

COLUMNS=80

title () {
  printf "\n%*s\n\n" $(((${#title}+$COLUMNS)/2)) "********** $1 **********"
}

echo $WORKSPACE

# clean up existing docker containers and images
docker.io stop $(docker.io ps -a -q)
docker.io rm $(docker.io ps -a -q)
docker.io rmi $(docker.io images -q)

# clean up build files so we can re-build
rm -f $WORKSPACE/docker/*-image

rm -f docker/config.yml

# Get test config.yml file
cp $HOME/docker/config.yml docker/

# DOCS
title "Starting docker build"
cd "$WORKSPACE"
cd docker
./build.sh

ECODE=$?

if [[ "$ECODE" != "0" ]]; then
  title "!!!!!! docker BUILD FAILED !!!!!!"
  EXITCODE=$(($EXITCODE + $ECODE))
fi

title "docker build complete"

exit $EXITCODE
