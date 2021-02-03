// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/url"
	"os"
	"os/exec"

	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/lib/crunchrun"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
)

func (s *ClientSuite) TestShellGatewayNotAvailable(c *check.C) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("go", "run", ".", "shell", arvadostest.QueuedContainerUUID, "-o", "controlpath=none", "echo", "ok")
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, "ARVADOS_API_TOKEN="+arvadostest.ActiveTokenV2)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	c.Check(cmd.Run(), check.NotNil)
	c.Log(stderr.String())
	c.Check(stderr.String(), check.Matches, `(?ms).*container is not running yet \(state is "Queued"\).*`)
}

func (s *ClientSuite) TestShellGateway(c *check.C) {
	defer func() {
		c.Check(arvados.NewClientFromEnv().RequestAndDecode(nil, "POST", "database/reset", nil, nil), check.IsNil)
	}()
	uuid := arvadostest.QueuedContainerUUID
	h := hmac.New(sha256.New, []byte(arvadostest.SystemRootToken))
	fmt.Fprint(h, uuid)
	authSecret := fmt.Sprintf("%x", h.Sum(nil))
	dcid := "theperthcountyconspiracy"
	gw := crunchrun.Gateway{
		DockerContainerID: &dcid,
		ContainerUUID:     uuid,
		Address:           "0.0.0.0:0",
		AuthSecret:        authSecret,
	}
	err := gw.Start()
	c.Assert(err, check.IsNil)

	rpcconn := rpc.NewConn("",
		&url.URL{
			Scheme: "https",
			Host:   os.Getenv("ARVADOS_API_HOST"),
		},
		true,
		func(context.Context) ([]string, error) {
			return []string{arvadostest.SystemRootToken}, nil
		})
	_, err = rpcconn.ContainerUpdate(context.TODO(), arvados.UpdateOptions{UUID: uuid, Attrs: map[string]interface{}{
		"state": arvados.ContainerStateLocked,
	}})
	c.Assert(err, check.IsNil)
	_, err = rpcconn.ContainerUpdate(context.TODO(), arvados.UpdateOptions{UUID: uuid, Attrs: map[string]interface{}{
		"state":           arvados.ContainerStateRunning,
		"gateway_address": gw.Address,
	}})
	c.Assert(err, check.IsNil)

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("go", "run", ".", "shell", uuid, "-o", "controlpath=none", "-o", "userknownhostsfile="+c.MkDir()+"/known_hosts", "echo", "ok")
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, "ARVADOS_API_TOKEN="+arvadostest.ActiveTokenV2)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	c.Check(cmd.Run(), check.NotNil)
	c.Log(stderr.String())
	c.Check(stderr.String(), check.Matches, `(?ms).*(No such container: theperthcountyconspiracy|exec: \"docker\": executable file not found in \$PATH).*`)
}
