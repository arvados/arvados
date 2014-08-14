#!/bin/bash

EXITCODE=0

COLUMNS=80

title () {
  printf "\n%*s\n\n" $(((${#title}+$COLUMNS)/2)) "********** $1 **********"
}

echo $WORKSPACE

# DOCKER
title "Starting docker build"

# clean up existing docker containers and images
docker.io stop $(docker.io ps -a -q)
docker.io rm $(docker.io ps -a -q)
docker.io rmi $(docker.io images -q)

# clean up the docker build environment
cd "$WORKSPACE"
cd docker
./build.sh realclean

rm -f config.yml

# Get test config.yml file
cp $HOME/docker/config.yml .

./build.sh

ECODE=$?

if [[ "$ECODE" != "0" ]]; then
  title "!!!!!! docker BUILD FAILED !!!!!!"
  EXITCODE=$(($EXITCODE + $ECODE))
fi

title "docker build complete"

exit $EXITCODE
