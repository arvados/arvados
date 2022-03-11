#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

export GOPATH=/var/lib/gopath
mkdir -p $GOPATH

cd /usr/src/arvados
if [[ $UID = 0 ]] ; then
  RUNSU="/usr/local/lib/arvbox/runsu.sh"
fi

if [[ ! -f /usr/local/bin/arvados-server ]]; then
  $RUNSU flock /var/lib/gopath/gopath.lock go mod download
  $RUNSU flock /var/lib/gopath/gopath.lock go mod vendor
  $RUNSU flock /var/lib/gopath/gopath.lock go install git.arvados.org/arvados.git/cmd/arvados-server
  $RUNSU flock /var/lib/gopath/gopath.lock install $GOPATH/bin/arvados-server /usr/local/bin
fi
