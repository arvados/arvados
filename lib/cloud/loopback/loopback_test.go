// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package loopback

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"testing"
	"time"

	"git.arvados.org/arvados.git/lib/cloud"
	"git.arvados.org/arvados.git/lib/dispatchcloud/sshexecutor"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"golang.org/x/crypto/ssh"
	check "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type suite struct{}

var _ = check.Suite(&suite{})

func (*suite) TestCreateListExecDestroy(c *check.C) {
	logger := ctxlog.TestLogger(c)
	is, err := Driver.InstanceSet(json.RawMessage("{}"), "testInstanceSetID", cloud.SharedResourceTags{"sharedTag": "sharedTagValue"}, logger)
	c.Assert(err, check.IsNil)

	clientRSAKey, err := rsa.GenerateKey(rand.Reader, 1024)
	c.Assert(err, check.IsNil)
	clientSSHKey, err := ssh.NewSignerFromKey(clientRSAKey)
	c.Assert(err, check.IsNil)
	clientSSHPubKey, err := ssh.NewPublicKey(clientRSAKey.Public())
	c.Assert(err, check.IsNil)

	it := arvados.InstanceType{
		Name:         "localhost",
		ProviderType: "localhost",
		RAM:          1002003004,
		VCPUs:        5,
	}

	// First call to Create should succeed, and the returned
	// instance's SSH target address should be available in << 1s.
	inst, err := is.Create(it, "testImageID", cloud.InstanceTags{"instanceTag": "instanceTagValue"}, "testInitCommand", clientSSHPubKey)
	c.Assert(err, check.IsNil)
	for deadline := time.Now().Add(time.Second); inst.Address() == ""; time.Sleep(time.Second / 100) {
		if deadline.Before(time.Now()) {
			c.Fatal("timed out")
		}
	}

	// Another call to Create should fail with a quota error.
	inst2, err := is.Create(it, "testImageID", cloud.InstanceTags{"instanceTag": "instanceTagValue"}, "testInitCommand", clientSSHPubKey)
	c.Check(inst2, check.IsNil)
	qerr, ok := err.(cloud.QuotaError)
	if c.Check(ok, check.Equals, true, check.Commentf("expect cloud.QuotaError, got %#v", err)) {
		c.Check(qerr.IsQuotaError(), check.Equals, true)
	}

	// Instance list should now have one entry, for the new
	// instance.
	list, err := is.Instances(nil)
	c.Assert(err, check.IsNil)
	c.Assert(list, check.HasLen, 1)
	inst = list[0]
	c.Check(inst.String(), check.Equals, "localhost")

	// Instance's SSH server should execute shell commands.
	exr := sshexecutor.New(inst)
	exr.SetSigners(clientSSHKey)

	stdout, stderr, err := exr.Execute(nil, "echo ok", nil)
	c.Check(err, check.IsNil)
	c.Check(string(stdout), check.Equals, "ok\n")
	c.Check(string(stderr), check.Equals, "")

	// SSH server should propagate stderr and non-zero exit
	// status.
	stdout, stderr, err = exr.Execute(nil, "echo fail && echo -n fail2 >&2 && false", nil)
	c.Check(err, check.FitsTypeOf, &ssh.ExitError{})
	c.Check(string(stdout), check.Equals, "fail\n")
	c.Check(string(stderr), check.Equals, "fail2")

	// SSH server should strip "sudo" from the front of the
	// command.
	withoutsudo, _, err := exr.Execute(nil, "whoami", nil)
	c.Check(err, check.IsNil)
	withsudo, _, err := exr.Execute(nil, "sudo whoami", nil)
	c.Check(err, check.IsNil)
	c.Check(string(withsudo), check.Equals, string(withoutsudo))

	// SSH server should reject keys other than the one whose
	// public key we passed to Create.
	badRSAKey, err := rsa.GenerateKey(rand.Reader, 1024)
	c.Assert(err, check.IsNil)
	badSSHKey, err := ssh.NewSignerFromKey(badRSAKey)
	c.Assert(err, check.IsNil)
	// Create a new executor here, otherwise Execute would reuse
	// the existing connection instead of authenticating with
	// badRSAKey.
	exr = sshexecutor.New(inst)
	exr.SetSigners(badSSHKey)
	stdout, stderr, err = exr.Execute(nil, "true", nil)
	c.Check(err, check.ErrorMatches, `.*unable to authenticate.*`)

	// Destroying the instance causes it to disappear from the
	// list, and allows us to create one more.
	err = inst.Destroy()
	c.Check(err, check.IsNil)
	list, err = is.Instances(nil)
	c.Assert(err, check.IsNil)
	c.Assert(list, check.HasLen, 0)
	_, err = is.Create(it, "testImageID", cloud.InstanceTags{"instanceTag": "instanceTagValue"}, "testInitCommand", clientSSHPubKey)
	c.Check(err, check.IsNil)
	_, err = is.Create(it, "testImageID", cloud.InstanceTags{"instanceTag": "instanceTagValue"}, "testInitCommand", clientSSHPubKey)
	c.Check(err, check.NotNil)
	list, err = is.Instances(nil)
	c.Assert(err, check.IsNil)
	c.Assert(list, check.HasLen, 1)
}
