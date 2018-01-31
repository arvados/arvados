#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

mkdir -p /var/lib/gopath
cd /var/lib/gopath

export GOPATH=$PWD
mkdir -p "$GOPATH/src/git.curoverse.com"
ln -sfn "/usr/src/arvados" "$GOPATH/src/git.curoverse.com/arvados.git"

flock /var/lib/gopath/gopath.lock go get -t github.com/kardianos/govendor
cd "$GOPATH/src/git.curoverse.com/arvados.git"
flock /var/lib/gopath/gopath.lock go get -v -d ...
flock /var/lib/gopath/gopath.lock "$GOPATH/bin/govendor" sync
