// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// Skip this slow test unless invoked as "go test -tags docker".
// Depending on host/network speed, Go's default 10m test timeout
// might be too short; recommend "go test -timeout 20m -tags docker".
//
//go:build docker
// +build docker

package install

import (
	"os"

	"gopkg.in/check.v1"
)

func (*Suite) TestInstallDeps(c *check.C) {
	tmp := c.MkDir()
	script := `
set -x
tmp="` + tmp + `"
sourcepath="$(realpath ../..)"
(cd ${sourcepath} && go build -o ${tmp} ./cmd/arvados-server)
docker run -i --rm --workdir /arvados \
       -v ${tmp}/arvados-server:/arvados-server:ro \
       -v ${sourcepath}:/arvados:ro \
       -v /arvados/services/api/.bundle \
       -v /arvados/services/api/tmp \
       --env http_proxy \
       --env https_proxy \
       debian:11 \
       bash -c "/arvados-server install -type test &&
           git config --global --add safe.directory /arvados &&
           /arvados-server boot -type test -config doc/examples/config/zzzzz.yml -own-temporary-database -shutdown -timeout 9m"
`
	c.Check((&installCommand{}).runBash(script, os.Stdout, os.Stderr), check.IsNil)
}
