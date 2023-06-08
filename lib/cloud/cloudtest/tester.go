// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package cloudtest

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"git.arvados.org/arvados.git/lib/cloud"
	"git.arvados.org/arvados.git/lib/dispatchcloud/sshexecutor"
	"git.arvados.org/arvados.git/lib/dispatchcloud/worker"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

var (
	errTestInstanceNotFound = errors.New("test instance missing from cloud provider's list")
)

// A tester does a sequence of operations to test a cloud driver and
// configuration. Run() should be called only once, after assigning
// suitable values to public fields.
type tester struct {
	Logger              logrus.FieldLogger
	Tags                cloud.SharedResourceTags
	TagKeyPrefix        string
	SetID               cloud.InstanceSetID
	DestroyExisting     bool
	ProbeInterval       time.Duration
	SyncInterval        time.Duration
	TimeoutBooting      time.Duration
	Driver              cloud.Driver
	DriverParameters    json.RawMessage
	InstanceType        arvados.InstanceType
	ImageID             cloud.ImageID
	SSHKey              ssh.Signer
	SSHPort             string
	BootProbeCommand    string
	InstanceInitCommand cloud.InitCommand
	ShellCommand        string
	PauseBeforeDestroy  func()

	is              cloud.InstanceSet
	testInstance    *worker.TagVerifier
	secret          string
	executor        *sshexecutor.Executor
	showedLoginInfo bool

	failed bool
}

// Run the test suite as specified, clean up as needed, and return
// true (everything is OK) or false (something went wrong).
func (t *tester) Run() bool {
	// This flag gets set when we encounter a non-fatal error, so
	// we can continue doing more tests but remember to return
	// false (failure) at the end.
	deferredError := false

	var err error
	t.is, err = t.Driver.InstanceSet(t.DriverParameters, t.SetID, t.Tags, t.Logger)
	if err != nil {
		t.Logger.WithError(err).Info("error initializing driver")
		return false
	}

	for {
		// Don't send the driver any filters when getting the
		// initial instance list. This way we can log an
		// instance count (N=...)  that includes all instances
		// in this service account, even if they don't have
		// the same InstanceSetID.
		insts, err := t.getInstances(nil)
		if err != nil {
			t.Logger.WithError(err).Info("error getting list of instances")
			return false
		}

		foundExisting := false
		for _, i := range insts {
			if i.Tags()[t.TagKeyPrefix+"InstanceSetID"] != string(t.SetID) {
				continue
			}
			lgr := t.Logger.WithFields(logrus.Fields{
				"Instance":      i.ID(),
				"InstanceSetID": t.SetID,
			})
			foundExisting = true
			if t.DestroyExisting {
				lgr.Info("destroying existing instance with our InstanceSetID")
				t0 := time.Now()
				err := i.Destroy()
				lgr := lgr.WithField("Duration", time.Since(t0))
				if err != nil {
					lgr.WithError(err).Error("error destroying existing instance")
				} else {
					lgr.Info("Destroy() call succeeded")
				}
			} else {
				lgr.Error("found existing instance with our InstanceSetID")
			}
		}
		if !foundExisting {
			break
		} else if t.DestroyExisting {
			t.sleepSyncInterval()
		} else {
			t.Logger.Error("cannot continue with existing instances -- clean up manually, use -destroy-existing=true, or choose a different -instance-set-id")
			return false
		}
	}

	t.secret = randomHex(40)

	tags := cloud.InstanceTags{}
	for k, v := range t.Tags {
		tags[k] = v
	}
	tags[t.TagKeyPrefix+"InstanceSetID"] = string(t.SetID)
	tags[t.TagKeyPrefix+"InstanceSecret"] = t.secret

	defer t.destroyTestInstance()

	bootDeadline := time.Now().Add(t.TimeoutBooting)
	initCommand := worker.TagVerifier{Instance: nil, Secret: t.secret, ReportVerified: nil}.InitCommand() + "\n" + t.InstanceInitCommand

	t.Logger.WithFields(logrus.Fields{
		"InstanceType":         t.InstanceType.Name,
		"ProviderInstanceType": t.InstanceType.ProviderType,
		"ImageID":              t.ImageID,
		"Tags":                 tags,
		"InitCommand":          initCommand,
	}).Info("creating instance")
	t0 := time.Now()
	inst, err := t.is.Create(t.InstanceType, t.ImageID, tags, initCommand, t.SSHKey.PublicKey())
	lgrC := t.Logger.WithField("Duration", time.Since(t0))
	if err != nil {
		// Create() might have failed due to a bug or network
		// error even though the creation was successful, so
		// it's safer to wait a bit for an instance to appear.
		deferredError = true
		lgrC.WithError(err).Error("error creating test instance")
		t.Logger.WithField("Deadline", bootDeadline).Info("waiting for instance to appear anyway, in case the Create response was incorrect")
		for err = t.refreshTestInstance(); err != nil; err = t.refreshTestInstance() {
			if time.Now().After(bootDeadline) {
				t.Logger.Error("timed out")
				return false
			}
			t.sleepSyncInterval()
		}
		t.Logger.WithField("Instance", t.testInstance.ID()).Info("new instance appeared")
		t.showLoginInfo()
	} else {
		// Create() succeeded. Make sure the new instance
		// appears right away in the Instances() list.
		lgrC.WithField("Instance", inst.ID()).Info("created instance")
		t.testInstance = &worker.TagVerifier{Instance: inst, Secret: t.secret, ReportVerified: nil}
		t.showLoginInfo()
		err = t.refreshTestInstance()
		if err == errTestInstanceNotFound {
			t.Logger.WithError(err).Error("cloud/driver Create succeeded, but instance is not in list")
			deferredError = true
		} else if err != nil {
			t.Logger.WithError(err).Error("error getting list of instances")
			return false
		}
	}

	if !t.checkTags() {
		// checkTags() already logged the errors
		deferredError = true
	}

	if !t.waitForBoot(bootDeadline) {
		deferredError = true
	}

	if t.ShellCommand != "" {
		err = t.runShellCommand(t.ShellCommand)
		if err != nil {
			t.Logger.WithError(err).Error("shell command failed")
			deferredError = true
		}
	}

	if fn := t.PauseBeforeDestroy; fn != nil {
		t.showLoginInfo()
		fn()
	}

	return !deferredError
}

// If the test instance has an address, log an "ssh user@host" command
// line that the operator can paste into another terminal, and set
// t.showedLoginInfo.
//
// If the test instance doesn't have an address yet, do nothing.
func (t *tester) showLoginInfo() {
	t.updateExecutor()
	host, port := t.executor.TargetHostPort()
	if host == "" {
		return
	}
	user := t.testInstance.RemoteUser()
	t.Logger.WithField("Command", fmt.Sprintf("ssh -p%s %s@%s", port, user, host)).Info("showing login information")
	t.showedLoginInfo = true
}

// Get the latest instance list from the driver. If our test instance
// is found, assign it to t.testIntance.
func (t *tester) refreshTestInstance() error {
	insts, err := t.getInstances(cloud.InstanceTags{t.TagKeyPrefix + "InstanceSetID": string(t.SetID)})
	if err != nil {
		return err
	}
	for _, i := range insts {
		if t.testInstance == nil {
			// Filter by InstanceSetID tag value
			if i.Tags()[t.TagKeyPrefix+"InstanceSetID"] != string(t.SetID) {
				continue
			}
		} else {
			// Filter by instance ID
			if i.ID() != t.testInstance.ID() {
				continue
			}
		}
		t.Logger.WithFields(logrus.Fields{
			"Instance": i.ID(),
			"Address":  i.Address(),
		}).Info("found our instance in returned list")
		t.testInstance = &worker.TagVerifier{Instance: i, Secret: t.secret, ReportVerified: nil}
		if !t.showedLoginInfo {
			t.showLoginInfo()
		}
		return nil
	}
	return errTestInstanceNotFound
}

// Get the list of instances, passing the given tags to the cloud
// driver to filter results.
//
// Return only the instances that have our InstanceSetID tag.
func (t *tester) getInstances(tags cloud.InstanceTags) ([]cloud.Instance, error) {
	var ret []cloud.Instance
	t.Logger.WithField("FilterTags", tags).Info("getting instance list")
	t0 := time.Now()
	insts, err := t.is.Instances(tags)
	if err != nil {
		return nil, err
	}
	t.Logger.WithFields(logrus.Fields{
		"Duration": time.Since(t0),
		"N":        len(insts),
	}).Info("got instance list")
	for _, i := range insts {
		if i.Tags()[t.TagKeyPrefix+"InstanceSetID"] == string(t.SetID) {
			ret = append(ret, i)
		}
	}
	return ret, nil
}

// Check that t.testInstance has every tag in t.Tags. If not, log an
// error and return false.
func (t *tester) checkTags() bool {
	ok := true
	for k, v := range t.Tags {
		if got := t.testInstance.Tags()[k]; got != v {
			ok = false
			t.Logger.WithFields(logrus.Fields{
				"Key":           k,
				"ExpectedValue": v,
				"GotValue":      got,
			}).Error("tag is missing from test instance")
		}
	}
	if ok {
		t.Logger.Info("all expected tags are present")
	}
	return ok
}

// Run t.BootProbeCommand on t.testInstance until it succeeds or the
// deadline arrives.
func (t *tester) waitForBoot(deadline time.Time) bool {
	for time.Now().Before(deadline) {
		err := t.runShellCommand(t.BootProbeCommand)
		if err == nil {
			return true
		}
		t.sleepProbeInterval()
		t.refreshTestInstance()
	}
	t.Logger.Error("timed out")
	return false
}

// Create t.executor and/or update its target to t.testInstance's
// current address.
func (t *tester) updateExecutor() {
	if t.executor == nil {
		t.executor = sshexecutor.New(t.testInstance)
		t.executor.SetTargetPort(t.SSHPort)
		t.executor.SetSigners(t.SSHKey)
	} else {
		t.executor.SetTarget(t.testInstance)
	}
}

func (t *tester) runShellCommand(cmd string) error {
	t.updateExecutor()
	t.Logger.WithFields(logrus.Fields{
		"Command": cmd,
	}).Info("executing remote command")
	t0 := time.Now()
	stdout, stderr, err := t.executor.Execute(nil, cmd, nil)
	lgr := t.Logger.WithFields(logrus.Fields{
		"Duration": time.Since(t0),
		"Command":  cmd,
		"stdout":   string(stdout),
		"stderr":   string(stderr),
	})
	if err != nil {
		lgr.WithError(err).Info("remote command failed")
	} else {
		lgr.Info("remote command succeeded")
	}
	return err
}

// currently, this tries forever until it can return true (success).
func (t *tester) destroyTestInstance() bool {
	if t.testInstance == nil {
		return true
	}
	for {
		lgr := t.Logger.WithField("Instance", t.testInstance.ID())
		lgr.Info("destroying instance")
		t0 := time.Now()

		err := t.testInstance.Destroy()
		lgrDur := lgr.WithField("Duration", time.Since(t0))
		if err != nil {
			lgrDur.WithError(err).Error("error destroying instance")
		} else {
			lgrDur.Info("destroyed instance")
		}

		err = t.refreshTestInstance()
		if err == errTestInstanceNotFound {
			lgr.Info("instance no longer appears in list")
			t.testInstance = nil
			return true
		} else if err == nil {
			lgr.Info("instance still exists after calling Destroy")
			t.sleepSyncInterval()
			continue
		} else {
			t.Logger.WithError(err).Error("error getting list of instances")
			continue
		}
	}
}

func (t *tester) sleepSyncInterval() {
	t.Logger.WithField("Duration", t.SyncInterval).Info("waiting SyncInterval")
	time.Sleep(t.SyncInterval)
}

func (t *tester) sleepProbeInterval() {
	t.Logger.WithField("Duration", t.ProbeInterval).Info("waiting ProbeInterval")
	time.Sleep(t.ProbeInterval)
}

// Return a random string of n hexadecimal digits (n*4 random bits). n
// must be even.
func randomHex(n int) string {
	buf := make([]byte, n/2)
	_, err := rand.Read(buf)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", buf)
}
