#!/bin/bash

EXITCODE=0

COLUMNS=80

title () {
  printf "\n%*s\n\n" $(((${#title}+$COLUMNS)/2)) "********** $1 **********"
}

echo $WORKSPACE

# DOCKER
title "Starting docker build"

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

title "Starting docker java-bwa-samtools build"

./build.sh java-bwa-samtools-image

ECODE=$?

if [[ "$ECODE" != "0" ]]; then
  title "!!!!!! docker java-bwa-samtools BUILD FAILED !!!!!!"
  EXITCODE=$(($EXITCODE + $ECODE))
fi

title "docker build java-bwa-samtools complete"

exit $EXITCODE
