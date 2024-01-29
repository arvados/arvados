#!/bin/bash
#
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -e -o pipefail

# Starting with a base debian buster system, like "docker run -it
# debian:10"...

apt update
apt upgrade
apt install --no-install-recommends build-essential ca-certificates git golang
git clone https://git.arvados.org/arvados.git
cd arvados
[[ -e lib/install ]] || git checkout origin/16053-install-deps
cd cmd/arvados-server
go run ./cmd/arvados-server install -type test
pg_isready || pg_ctlcluster 11 main start # only needed if there's no init process (as in docker)
build/run-tests.sh
