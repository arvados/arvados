#!/bin/bash
#
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

set -e -o pipefail

# Starting with a base debian bullseye system, like "docker run -it
# debian:11"...

apt update
apt upgrade
apt install --no-install-recommends build-essential ca-certificates git golang
git clone https://git.arvados.org/arvados.git
cd arvados/cmd/arvados-server
go run ./cmd/arvados-server install -type test
pg_isready || pg_ctlcluster 13 main start # only needed if there's no init process (as in docker)
build/run-tests.sh
