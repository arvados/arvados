#!/usr/bin/env bash

# This script is intended to make Arvados installation easy. It will download the
# latest copy of the Arvados docker images as well as the arvdock command. It
# then uses arvdock to spin up Arvados on this computer.
#
# The latest version of this script is available at http://get.arvados.org, so that this
# command does the right thing:
#
#  $ \curl -sSL http://get.arvados.org | bash
#
# Prerequisites: working docker installation. Run this script as a user who is a member 
# of the docker group.

COLUMNS=80

fail () {
    title "$*"
    exit 1
}

title () {
  printf "\n%*s\n\n" $(((${#title}+$COLUMNS)/2)) "********** $1 **********"
}

docker_pull () {
  $DOCKER pull $*

  ECODE=$?

  if [[ "$ECODE" != "0" ]]; then
    title "$DOCKER pull $* failed"
    exit $ECODE
  fi
}

main () {

  \which which >/dev/null 2>&1 || fail "Error: could not find 'which' command."

  # find the docker binary
  DOCKER=`which docker.io`
  
  if [[ "$DOCKER" == "" ]]; then
    DOCKER=`which docker`
  fi

  if [[ "$DOCKER" == "" ]]; then
    fail "Error: you need to have docker installed. Could not find the docker executable."
  fi

  echo
  echo "If necessary, this command will download the latest arvados docker images."
  echo "The download can take a long time, depending on the speed of your internet connection."
  echo "When the images are downloaded, it will then start an Arvados environment on this computer."
  echo
  docker_pull arvados/workbench
  docker_pull arvados/doc
  docker_pull arvados/keep
  docker_pull arvados/shell
  docker_pull arvados/compute
  docker_pull arvados/keep
  docker_pull arvados/api
  docker_pull crosbymichael/skydns
  docker_pull crosbymichael/skydock

  # Now download arvdock and start the containers
  echo
  echo Downloading arvdock
  echo
  \curl -sSL https://raw.githubusercontent.com/curoverse/arvados/master/docker/arvdock -o arvdock
  chmod 755 arvdock

  echo
  echo Starting the docker containers
  echo
  ./arvdock start

  echo To stop the containers, run
  echo
  echo ./arvdock stop
  echo 
}

main
