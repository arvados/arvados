// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package deduplicationreport

import (
	"bytes"
	"testing"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"gopkg.in/check.v1"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&Suite{})

type Suite struct{}

func (s *Suite) TearDownSuite(c *check.C) {
	// Undo any changes/additions to the database so they don't affect subsequent tests.
	arvadostest.ResetEnv()
}

func (*Suite) TestUsage(c *check.C) {
	var stdout, stderr bytes.Buffer
	exitcode := Command.RunCommand("deduplicationreport.test", []string{"-h", "-log-level=debug"}, &bytes.Buffer{}, &stdout, &stderr)
	c.Check(exitcode, check.Equals, 0)
	c.Check(stdout.String(), check.Equals, "")
	c.Log(stderr.String())
	c.Check(stderr.String(), check.Matches, `(?ms).*Usage:.*`)
}

func (*Suite) TestTwoIdenticalUUIDs(c *check.C) {
	var stdout, stderr bytes.Buffer
	// Run dedupreport with 2 identical uuids
	exitcode := Command.RunCommand("deduplicationreport.test", []string{arvadostest.FooCollection, arvadostest.FooCollection}, &bytes.Buffer{}, &stdout, &stderr)
	c.Check(exitcode, check.Equals, 0)
	c.Check(stdout.String(), check.Matches, "(?ms).*Collections:[[:space:]]+1.*")
	c.Check(stdout.String(), check.Matches, "(?ms).*Nominal size of stored data:[[:space:]]+3 bytes \\(3 B\\).*")
	c.Check(stdout.String(), check.Matches, "(?ms).*Actual size of stored data:[[:space:]]+3 bytes \\(3 B\\).*")
	c.Check(stdout.String(), check.Matches, "(?ms).*Saved by Keep deduplication:[[:space:]]+0 bytes \\(0 B\\).*")
	c.Log(stderr.String())
}

func (*Suite) TestTwoUUIDsInvalidPDH(c *check.C) {
	var stdout, stderr bytes.Buffer
	// Run dedupreport with pdh,uuid where pdh does not match
	exitcode := Command.RunCommand("deduplicationreport.test", []string{arvadostest.FooAndBarFilesInDirPDH + "," + arvadostest.FooCollection, arvadostest.FooCollection}, &bytes.Buffer{}, &stdout, &stderr)
	c.Check(exitcode, check.Equals, 1)
	c.Check(stdout.String(), check.Equals, "")
	c.Log(stderr.String())
	c.Check(stderr.String(), check.Matches, `(?ms).*Error: the collection with UUID zzzzz-4zz18-fy296fx3hot09f7 has PDH 1f4b0bc7583c2a7f9102c395f4ffc5e3\+45, but a different PDH was provided in the arguments: 870369fc72738603c2fad16664e50e2d\+58.*`)
}

func (*Suite) TestNonExistentCollection(c *check.C) {
	var stdout, stderr bytes.Buffer
	// Run dedupreport with many UUIDs
	exitcode := Command.RunCommand("deduplicationreport.test", []string{arvadostest.FooCollection, arvadostest.NonexistentCollection}, &bytes.Buffer{}, &stdout, &stderr)
	c.Check(exitcode, check.Equals, 1)
	c.Check(stdout.String(), check.Equals, "Collection zzzzz-4zz18-fy296fx3hot09f7: pdh 1f4b0bc7583c2a7f9102c395f4ffc5e3+45; nominal size 3 (3 B)\n")
	c.Log(stderr.String())
	c.Check(stderr.String(), check.Matches, `(?ms).*Error: unable to retrieve collection:.*404 Not Found.*`)
}

func (*Suite) TestManyUUIDsNoOverlap(c *check.C) {
	var stdout, stderr bytes.Buffer
	// Run dedupreport with 5 UUIDs
	exitcode := Command.RunCommand("deduplicationreport.test", []string{arvadostest.FooCollection, arvadostest.HelloWorldCollection, arvadostest.FooBarDirCollection, arvadostest.WazVersion1Collection, arvadostest.UserAgreementCollection}, &bytes.Buffer{}, &stdout, &stderr)
	c.Check(exitcode, check.Equals, 0)
	c.Check(stdout.String(), check.Matches, "(?ms).*Collections:[[:space:]]+5.*")
	c.Check(stdout.String(), check.Matches, "(?ms).*Nominal size of stored data:[[:space:]]+249049 bytes \\(243 KiB\\).*")
	c.Check(stdout.String(), check.Matches, "(?ms).*Actual size of stored data:[[:space:]]+249049 bytes \\(243 KiB\\).*")
	c.Check(stdout.String(), check.Matches, "(?ms).*Saved by Keep deduplication:[[:space:]]+0 bytes \\(0 B\\).*")
	c.Log(stderr.String())
	c.Check(stderr.String(), check.Equals, "")
}

func (*Suite) TestTwoOverlappingCollections(c *check.C) {
	var stdout, stderr bytes.Buffer
	// Create two collections
	arv := arvados.NewClientFromEnv()

	var c1 arvados.Collection
	err := arv.RequestAndDecode(&c1, "POST", "arvados/v1/collections", nil, map[string]interface{}{"collection": map[string]interface{}{"manifest_text": ". d3b07384d113edec49eaa6238ad5ff00+4 0:4:foo\n"}})
	c.Assert(err, check.Equals, nil)

	var c2 arvados.Collection
	err = arv.RequestAndDecode(&c2, "POST", "arvados/v1/collections", nil, map[string]interface{}{"collection": map[string]interface{}{"manifest_text": ". c157a79031e1c40f85931829bc5fc552+4 d3b07384d113edec49eaa6238ad5ff00+4 0:4:bar 4:4:foo\n"}})
	c.Assert(err, check.Equals, nil)

	for _, trial := range []struct {
		field1 string
		field2 string
	}{
		{
			// Run dedupreport with 2 arguments: uuid uuid
			field1: c1.UUID,
			field2: c2.UUID,
		},
		{
			// Run dedupreport with 2 arguments: pdh,uuid uuid
			field1: c1.PortableDataHash + "," + c1.UUID,
			field2: c2.UUID,
		},
	} {
		exitcode := Command.RunCommand("deduplicationreport.test", []string{trial.field1, trial.field2}, &bytes.Buffer{}, &stdout, &stderr)
		c.Check(exitcode, check.Equals, 0)
		c.Check(stdout.String(), check.Matches, "(?ms).*Nominal size of stored data:[[:space:]]+12 bytes \\(12 B\\).*")
		c.Check(stdout.String(), check.Matches, "(?ms).*Actual size of stored data:[[:space:]]+8 bytes \\(8 B\\).*")
		c.Check(stdout.String(), check.Matches, "(?ms).*Saved by Keep deduplication:[[:space:]]+4 bytes \\(4 B\\).*")
		c.Log(stderr.String())
		c.Check(stderr.String(), check.Equals, "")
	}
}
