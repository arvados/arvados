#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

make

GOVERSION=$(grep 'const goversion =' ../../lib/install/deps.go |awk -F'"' '{print $2}')

for target in `find -maxdepth 1 -type d |grep -v generated`; do
  if [[ "$target" == "." ]]; then
    continue
  fi
  target=${target#./}
  echo $target
  cd $target
  docker build --tag arvados/build:$target \
    --build-arg HOSTTYPE=$HOSTTYPE \
    --build-arg BRANCH=$(git rev-parse --abbrev-ref HEAD) \
    --build-arg GOVERSION=$GOVERSION --no-cache .
  cd ..
done
