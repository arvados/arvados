#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

export GOPATH=/var/lib/gopath
mkdir -p $GOPATH

cd /usr/src/arvados
if [[ $UID = 0 ]] ; then
    /usr/local/lib/arvbox/runsu.sh flock /var/lib/gopath/gopath.lock go mod download
    /usr/local/lib/arvbox/runsu.sh flock /var/lib/gopath/gopath.lock go get ./cmd/arvados-server
else
    flock /var/lib/gopath/gopath.lock go mod download
    flock /var/lib/gopath/gopath.lock go get ./cmd/arvados-server
fi
install $GOPATH/bin/arvados-server /usr/local/bin
