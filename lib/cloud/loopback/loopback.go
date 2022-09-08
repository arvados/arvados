// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package loopback

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"sync"
	"syscall"

	"git.arvados.org/arvados.git/lib/cloud"
	"git.arvados.org/arvados.git/lib/dispatchcloud/test"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// Driver is the loopback implementation of the cloud.Driver interface.
var Driver = cloud.DriverFunc(newInstanceSet)

var (
	errUnimplemented = errors.New("function not implemented by loopback driver")
	errQuota         = quotaError("loopback driver is always at quota")
)

type quotaError string

func (e quotaError) IsQuotaError() bool { return true }
func (e quotaError) Error() string      { return string(e) }

type instanceSet struct {
	instanceSetID cloud.InstanceSetID
	logger        logrus.FieldLogger
	instances     []*instance
	mtx           sync.Mutex
}

func newInstanceSet(config json.RawMessage, instanceSetID cloud.InstanceSetID, _ cloud.SharedResourceTags, logger logrus.FieldLogger) (cloud.InstanceSet, error) {
	is := &instanceSet{
		instanceSetID: instanceSetID,
		logger:        logger,
	}
	return is, nil
}

func (is *instanceSet) Create(it arvados.InstanceType, _ cloud.ImageID, tags cloud.InstanceTags, _ cloud.InitCommand, pubkey ssh.PublicKey) (cloud.Instance, error) {
	is.mtx.Lock()
	defer is.mtx.Unlock()
	if len(is.instances) > 0 {
		return nil, errQuota
	}
	// A crunch-run process running in a previous instance may
	// have marked the node as broken. In the loopback scenario a
	// destroy+create cycle doesn't fix whatever was broken -- but
	// nothing else will either, so the best we can do is remove
	// the "broken" flag and try again.
	if err := os.Remove("/var/lock/crunch-run-broken"); err == nil {
		is.logger.Info("removed /var/lock/crunch-run-broken")
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	hostRSAKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}
	hostKey, err := ssh.NewSignerFromKey(hostRSAKey)
	if err != nil {
		return nil, err
	}
	hostPubKey, err := ssh.NewPublicKey(hostRSAKey.Public())
	if err != nil {
		return nil, err
	}
	inst := &instance{
		is:           is,
		instanceType: it,
		adminUser:    u.Username,
		tags:         tags,
		hostPubKey:   hostPubKey,
		sshService: test.SSHService{
			HostKey:        hostKey,
			AuthorizedUser: u.Username,
			AuthorizedKeys: []ssh.PublicKey{pubkey},
		},
	}
	inst.sshService.Exec = inst.sshExecFunc
	go inst.sshService.Start()
	is.instances = []*instance{inst}
	return inst, nil
}

func (is *instanceSet) Instances(cloud.InstanceTags) ([]cloud.Instance, error) {
	is.mtx.Lock()
	defer is.mtx.Unlock()
	var ret []cloud.Instance
	for _, inst := range is.instances {
		ret = append(ret, inst)
	}
	return ret, nil
}

func (is *instanceSet) Stop() {
	is.mtx.Lock()
	defer is.mtx.Unlock()
	for _, inst := range is.instances {
		inst.sshService.Close()
	}
}

type instance struct {
	is           *instanceSet
	instanceType arvados.InstanceType
	adminUser    string
	tags         cloud.InstanceTags
	hostPubKey   ssh.PublicKey
	sshService   test.SSHService
}

func (i *instance) ID() cloud.InstanceID     { return cloud.InstanceID(i.instanceType.ProviderType) }
func (i *instance) String() string           { return i.instanceType.ProviderType }
func (i *instance) ProviderType() string     { return i.instanceType.ProviderType }
func (i *instance) Address() string          { return i.sshService.Address() }
func (i *instance) RemoteUser() string       { return i.adminUser }
func (i *instance) Tags() cloud.InstanceTags { return i.tags }
func (i *instance) SetTags(tags cloud.InstanceTags) error {
	i.tags = tags
	return nil
}
func (i *instance) Destroy() error {
	i.is.mtx.Lock()
	defer i.is.mtx.Unlock()
	i.is.instances = i.is.instances[:0]
	return nil
}
func (i *instance) VerifyHostKey(pubkey ssh.PublicKey, _ *ssh.Client) error {
	if !bytes.Equal(pubkey.Marshal(), i.hostPubKey.Marshal()) {
		return errors.New("host key mismatch")
	}
	return nil
}
func (i *instance) sshExecFunc(env map[string]string, command string, stdin io.Reader, stdout, stderr io.Writer) uint32 {
	cmd := exec.Command("sh", "-c", strings.TrimPrefix(command, "sudo "))
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	// Prevent child process from using our tty.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	err := cmd.Run()
	if err == nil {
		return 0
	} else if err, ok := err.(*exec.ExitError); !ok {
		return 1
	} else if code := err.ExitCode(); code < 0 {
		return 1
	} else {
		return uint32(code)
	}
}
