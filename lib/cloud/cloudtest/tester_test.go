// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package cloudtest

import (
	"bytes"
	"testing"
	"time"

	"git.arvados.org/arvados.git/lib/cloud"
	"git.arvados.org/arvados.git/lib/dispatchcloud/test"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"golang.org/x/crypto/ssh"
	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&TesterSuite{})

type TesterSuite struct {
	stubDriver *test.StubDriver
	cluster    *arvados.Cluster
	tester     *tester
	log        bytes.Buffer
}

func (s *TesterSuite) SetUpTest(c *check.C) {
	pubkey, privkey := test.LoadTestKey(c, "../../dispatchcloud/test/sshkey_dispatch")
	_, privhostkey := test.LoadTestKey(c, "../../dispatchcloud/test/sshkey_vm")
	s.stubDriver = &test.StubDriver{
		HostKey:                   privhostkey,
		AuthorizedKeys:            []ssh.PublicKey{pubkey},
		ErrorRateDestroy:          0.1,
		MinTimeBetweenCreateCalls: time.Millisecond,
	}
	tagKeyPrefix := "tagprefix:"
	s.cluster = &arvados.Cluster{
		ManagementToken: "test-management-token",
		Containers: arvados.ContainersConfig{
			CloudVMs: arvados.CloudVMsConfig{
				SyncInterval:   arvados.Duration(10 * time.Millisecond),
				TimeoutBooting: arvados.Duration(150 * time.Millisecond),
				TimeoutProbe:   arvados.Duration(15 * time.Millisecond),
				ProbeInterval:  arvados.Duration(5 * time.Millisecond),
				ResourceTags:   map[string]string{"testtag": "test value"},
			},
		},
		InstanceTypes: arvados.InstanceTypeMap{
			test.InstanceType(1).Name: test.InstanceType(1),
			test.InstanceType(2).Name: test.InstanceType(2),
			test.InstanceType(3).Name: test.InstanceType(3),
		},
	}
	s.tester = &tester{
		Logger:           ctxlog.New(&s.log, "text", "info"),
		Tags:             cloud.SharedResourceTags{"testtagkey": "testtagvalue"},
		TagKeyPrefix:     tagKeyPrefix,
		SetID:            cloud.InstanceSetID("test-instance-set-id"),
		ProbeInterval:    5 * time.Millisecond,
		SyncInterval:     10 * time.Millisecond,
		TimeoutBooting:   150 * time.Millisecond,
		Driver:           s.stubDriver,
		DriverParameters: nil,
		InstanceType:     test.InstanceType(2),
		ImageID:          "test-image-id",
		SSHKey:           privkey,
		BootProbeCommand: "crunch-run --list",
		ShellCommand:     "true",
	}
}

func (s *TesterSuite) TestSuccess(c *check.C) {
	s.tester.Logger = ctxlog.TestLogger(c)
	ok := s.tester.Run()
	c.Check(ok, check.Equals, true)
}

func (s *TesterSuite) TestBootFail(c *check.C) {
	s.tester.BootProbeCommand = "falsey"
	ok := s.tester.Run()
	c.Check(ok, check.Equals, false)
	c.Check(s.log.String(), check.Matches, `(?ms).*\\"falsey\\": command not found.*`)
}

func (s *TesterSuite) TestShellCommandFail(c *check.C) {
	s.tester.ShellCommand = "falsey"
	ok := s.tester.Run()
	c.Check(ok, check.Equals, false)
	c.Check(s.log.String(), check.Matches, `(?ms).*\\"falsey\\": command not found.*`)
}
