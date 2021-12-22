// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package recovercollection

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"gopkg.in/check.v1"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&Suite{})

type Suite struct{}

func (*Suite) SetUpSuite(c *check.C) {
	arvadostest.StartKeep(2, true)
}

func (*Suite) TestUnrecoverableBlock(c *check.C) {
	tmp := c.MkDir()
	mfile := tmp + "/manifest"
	ioutil.WriteFile(mfile, []byte(". aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa+410 0:410:Gone\n"), 0777)
	var stdout, stderr bytes.Buffer
	exitcode := Command.RunCommand("recovercollection.test", []string{"-log-level=debug", mfile}, &bytes.Buffer{}, &stdout, &stderr)
	c.Check(exitcode, check.Equals, 1)
	c.Check(stdout.String(), check.Equals, "")
	c.Log(stderr.String())
	c.Check(stderr.String(), check.Matches, `(?ms).*msg="not found" block=aaaaa.*`)
	c.Check(stderr.String(), check.Matches, `(?ms).*msg="untrash failed" block=aaaaa.*`)
	c.Check(stderr.String(), check.Matches, `(?ms).*msg=unrecoverable block=aaaaa.*`)
	c.Check(stderr.String(), check.Matches, `(?ms).*msg="recovery failed".*`)
}

func (*Suite) TestUntrashAndTouchBlock(c *check.C) {
	tmp := c.MkDir()
	mfile := tmp + "/manifest"
	ioutil.WriteFile(mfile, []byte(". dcd0348cb2532ee90c99f1b846efaee7+13 0:13:test.txt\n"), 0777)

	logger := ctxlog.TestLogger(c)
	loader := config.NewLoader(&bytes.Buffer{}, logger)
	cfg, err := loader.Load()
	c.Assert(err, check.IsNil)
	cluster, err := cfg.GetCluster("")
	c.Assert(err, check.IsNil)
	var datadirs []string
	for _, v := range cluster.Volumes {
		var params struct {
			Root string
		}
		err := json.Unmarshal(v.DriverParameters, &params)
		c.Assert(err, check.IsNil)
		if params.Root != "" {
			datadirs = append(datadirs, params.Root)
			err := os.Remove(params.Root + "/dcd/dcd0348cb2532ee90c99f1b846efaee7")
			if err != nil && !os.IsNotExist(err) {
				c.Error(err)
			}
		}
	}
	c.Logf("keepstore datadirs are %q", datadirs)

	// Currently StartKeep(2, true) uses dirs called "keep0" and
	// "keep1" so we could just put our fake trashed file in keep0
	// ... but we don't want to rely on arvadostest's
	// implementation details, so we put a trashed file in every
	// dir that keepstore might be using.
	for _, datadir := range datadirs {
		if fi, err := os.Stat(datadir); err != nil || !fi.IsDir() {
			continue
		}
		c.Logf("placing backdated trashed block in datadir %q", datadir)
		trashfile := datadir + "/dcd/dcd0348cb2532ee90c99f1b846efaee7.trash.999999999"
		os.Mkdir(datadir+"/dcd", 0777)
		err = ioutil.WriteFile(trashfile, []byte("undelete test"), 0777)
		c.Assert(err, check.IsNil)
		t := time.Now().Add(-time.Hour * 24 * 365)
		err = os.Chtimes(trashfile, t, t)
		c.Assert(err, check.IsNil)
	}

	var stdout, stderr bytes.Buffer
	exitcode := Command.RunCommand("recovercollection.test", []string{"-log-level=debug", mfile}, &bytes.Buffer{}, &stdout, &stderr)
	c.Check(exitcode, check.Equals, 0)
	c.Check(stdout.String(), check.Matches, `zzzzz-4zz18-.{15}\n`)
	c.Log(stderr.String())
	c.Check(stderr.String(), check.Matches, `(?ms).*msg=untrashed block=dcd0348.*`)
	c.Check(stderr.String(), check.Matches, `(?ms).*msg="updated timestamp" block=dcd0348.*`)

	found := false
	for _, datadir := range datadirs {
		buf, err := ioutil.ReadFile(datadir + "/dcd/dcd0348cb2532ee90c99f1b846efaee7")
		if err == nil {
			found = true
			c.Check(buf, check.DeepEquals, []byte("undelete test"))
			fi, err := os.Stat(datadir + "/dcd/dcd0348cb2532ee90c99f1b846efaee7")
			if c.Check(err, check.IsNil) {
				c.Logf("recovered block's modtime is %s", fi.ModTime())
				c.Check(time.Now().Sub(fi.ModTime()) < time.Hour, check.Equals, true)
			}
		}
	}
	c.Check(found, check.Equals, true)
}

func (*Suite) TestUnusableManifestSourceArg(c *check.C) {
	for _, trial := range []struct {
		srcArg    string
		errRegexp string
	}{
		{"zzzzz-4zz18-aaaaaaaaaaaaaaa", `(?ms).*msg="log entry not found".*`},
		{"zzzzz-57u5n-aaaaaaaaaaaaaaa", `(?ms).*msg="log entry not found.*`},
		{"zzzzz-57u5n-containerlog006", `(?ms).*msg="log entry properties\.old_attributes\.manifest_text missing or empty".*`},
		{"zzzzz-j7d0g-aaaaaaaaaaaaaaa", `(?ms).*msg="looks like a UUID but not a log or collection UUID.*`},
	} {
		var stdout, stderr bytes.Buffer
		exitcode := Command.RunCommand("recovercollection.test", []string{"-log-level=debug", trial.srcArg}, &bytes.Buffer{}, &stdout, &stderr)
		c.Check(exitcode, check.Equals, 1)
		c.Check(stdout.String(), check.Equals, "")
		c.Log(stderr.String())
		c.Check(stderr.String(), check.Matches, trial.errRegexp)
	}
}
