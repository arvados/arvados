// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package ws

import (
	"context"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&permSuite{})

type permSuite struct{}

func (s *permSuite) TestCheck(c *check.C) {
	client := arvados.NewClientFromEnv()
	// Disable auto-retry
	client.Timeout = 0

	pc := newPermChecker(*client).(*cachingPermChecker)
	setToken := func(label, token string) {
		c.Logf("...%s token %q", label, token)
		pc.SetToken(token)
	}
	wantError := func(uuid string) {
		c.Log(uuid)
		ok, err := pc.Check(context.Background(), uuid)
		c.Check(ok, check.Equals, false)
		c.Check(err, check.NotNil)
	}
	wantYes := func(uuid string) {
		c.Log(uuid)
		ok, err := pc.Check(context.Background(), uuid)
		c.Check(ok, check.Equals, true)
		c.Check(err, check.IsNil)
	}
	wantNo := func(uuid string) {
		c.Log(uuid)
		ok, err := pc.Check(context.Background(), uuid)
		c.Check(ok, check.Equals, false)
		c.Check(err, check.IsNil)
	}

	setToken("no", "")
	wantNo(arvadostest.UserAgreementCollection)
	wantNo(arvadostest.UserAgreementPDH)
	wantNo(arvadostest.FooBarDirCollection)

	setToken("anonymous", arvadostest.AnonymousToken)
	wantYes(arvadostest.UserAgreementCollection)
	wantYes(arvadostest.UserAgreementPDH)
	wantNo(arvadostest.FooBarDirCollection)
	wantNo(arvadostest.FooCollection)

	setToken("active", arvadostest.ActiveToken)
	wantYes(arvadostest.UserAgreementCollection)
	wantYes(arvadostest.UserAgreementPDH)
	wantYes(arvadostest.FooBarDirCollection)
	wantYes(arvadostest.FooCollection)

	setToken("admin", arvadostest.AdminToken)
	wantYes(arvadostest.UserAgreementCollection)
	wantYes(arvadostest.UserAgreementPDH)
	wantYes(arvadostest.FooBarDirCollection)
	wantYes(arvadostest.FooCollection)

	// hack to empty the cache
	pc.SetToken("")
	pc.SetToken(arvadostest.ActiveToken)

	c.Log("...network error")
	pc.Client.APIHost = "127.0.0.1:9"
	wantError(arvadostest.UserAgreementCollection)
	wantError(arvadostest.FooBarDirCollection)

	c.Logf("%d checks, %d misses, %d invalid, %d cached", pc.nChecks, pc.nMisses, pc.nInvalid, len(pc.cache))
}
