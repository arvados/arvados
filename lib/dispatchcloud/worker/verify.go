// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package worker

import (
	"bytes"
	"errors"
	"fmt"

	"git.curoverse.com/arvados.git/lib/cloud"
	"golang.org/x/crypto/ssh"
)

var (
	errBadInstanceSecret = errors.New("bad instance secret")

	// filename on instance, as given to shell (quoted accordingly)
	instanceSecretFilename = "/var/run/arvados-instance-secret"
	instanceSecretLength   = 40 // hex digits
)

type tagVerifier struct {
	cloud.Instance
}

func (tv tagVerifier) VerifyHostKey(pubKey ssh.PublicKey, client *ssh.Client) error {
	expectSecret := tv.Instance.Tags()[tagKeyInstanceSecret]
	if err := tv.Instance.VerifyHostKey(pubKey, client); err != cloud.ErrNotImplemented || expectSecret == "" {
		// If the wrapped instance indicates it has a way to
		// verify the key, return that decision.
		return err
	}
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	var stdout, stderr bytes.Buffer
	session.Stdin = bytes.NewBuffer(nil)
	session.Stdout = &stdout
	session.Stderr = &stderr
	cmd := fmt.Sprintf("cat %s", instanceSecretFilename)
	if u := tv.RemoteUser(); u != "root" {
		cmd = "sudo " + cmd
	}
	err = session.Run(cmd)
	if err != nil {
		return err
	}
	if stdout.String() != expectSecret {
		return errBadInstanceSecret
	}
	return nil
}
