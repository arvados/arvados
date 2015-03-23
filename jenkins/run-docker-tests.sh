#!/bin/bash

EXITCODE=0

COLUMNS=80

title () {
  printf "\n%*s\n\n" $(((${#title}+$COLUMNS)/2)) "********** $1 **********"
}

docker_push () {
  # Sometimes docker push fails; retry it a few times if necessary.
  for i in `seq 1 5`; do
    $DOCKER push $*
    ECODE=$?
    if [[ "$ECODE" == "0" ]]; then
      break
    fi
  done

  if [[ "$ECODE" != "0" ]]; then
    title "!!!!!! docker push $* failed !!!!!!"
    EXITCODE=$(($EXITCODE + $ECODE))
  fi
}

timer_reset() {
    t0=$SECONDS
}

timer() {
    echo -n "$(($SECONDS - $t0))s"
}

# Sanity check
if ! [[ -n "$WORKSPACE" ]]; then
  echo >&2
  echo >&2 "Error: WORKSPACE environment variable not set"
  echo >&2
  exit 1
fi

echo $WORKSPACE

# find the docker binary
DOCKER=`which docker.io`

if [[ "$DOCKER" == "" ]]; then
  DOCKER=`which docker`
fi

if [[ "$DOCKER" == "" ]]; then
  title "Error: you need to have docker installed. Could not find the docker executable."
  exit 1
fi

# DOCKER
title "Starting docker build"

timer_reset

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

title "docker build complete (`timer`)"

title "uploading images"

timer_reset

if [[ "$ECODE" == "0" ]]; then
  docker_push arvados/api
  docker_push arvados/compute
  docker_push arvados/doc
  docker_push arvados/workbench
  docker_push arvados/keep
  docker_push arvados/keepproxy
  docker_push arvados/shell
  docker_push arvados/sso
else
  title "upload arvados images SKIPPED because build failed"
fi

title "upload arvados images complete (`timer`)"

title "Starting docker java-bwa-samtools build"

timer_reset

./build.sh java-bwa-samtools-image

ECODE=$?

if [[ "$ECODE" != "0" ]]; then
  title "!!!!!! docker java-bwa-samtools BUILD FAILED !!!!!!"
  EXITCODE=$(($EXITCODE + $ECODE))
fi

title "docker build java-bwa-samtools complete (`timer`)"

title "upload arvados/jobs image"

timer_reset
if [[ "$ECODE" == "0" ]]; then
  docker_push arvados/jobs
else
  title "upload arvados/jobs image SKIPPED because build failed"
fi

title "upload arvados/jobs image complete (`timer`)"

exit $EXITCODE
