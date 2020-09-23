#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

export GOPATH=/var/lib/gopath
mkdir -p $GOPATH

cd /usr/src/arvados
if [[ $UID = 0 ]] ; then
  /usr/local/lib/arvbox/runsu.sh flock /var/lib/gopath/gopath.lock go mod download
  if [[ ! -f /usr/local/bin/arvados-server ]]; then
    /usr/local/lib/arvbox/runsu.sh flock /var/lib/gopath/gopath.lock go install git.arvados.org/arvados.git/cmd/arvados-server
  fi
else
  flock /var/lib/gopath/gopath.lock go mod download
  if [[ ! -f /usr/local/bin/arvados-server ]]; then
    flock /var/lib/gopath/gopath.lock go install git.arvados.org/arvados.git/cmd/arvados-server
  fi
fi
install $GOPATH/bin/arvados-server /usr/local/bin
