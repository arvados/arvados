#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

export GOPATH=/var/lib/gopath
mkdir -p $GOPATH

cd /usr/src/arvados
flock /var/lib/gopath/gopath.lock go get -v -d ...
flock /var/lib/gopath/gopath.lock go get -t ./cmd/arvados-server
install $GOPATH/bin/arvados-server /usr/local/bin
