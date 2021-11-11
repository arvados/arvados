// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package federation

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/url"
	"os"
	"strings"

	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&UserSuite{})

type UserSuite struct {
	FederationSuite
}

func (s *UserSuite) TestLoginClusterUserList(c *check.C) {
	s.cluster.ClusterID = "local"
	s.cluster.Login.LoginCluster = "zzzzz"
	s.fed = New(s.cluster, nil)
	s.addDirectRemote(c, "zzzzz", rpc.NewConn("zzzzz", &url.URL{Scheme: "https", Host: os.Getenv("ARVADOS_API_HOST")}, true, rpc.PassthroughTokenProvider))

	for _, updateFail := range []bool{false, true} {
		for _, opts := range []arvados.ListOptions{
			{Offset: 0, Limit: -1, Select: nil},
			{Offset: 0, Limit: math.MaxInt64, Select: nil},
			{Offset: 1, Limit: 1, Select: nil},
			{Offset: 0, Limit: 2, Select: []string{"uuid"}},
			{Offset: 0, Limit: 2, Select: []string{"uuid", "email"}},
		} {
			c.Logf("updateFail %v, opts %#v", updateFail, opts)
			spy := arvadostest.NewProxy(c, s.cluster.Services.RailsAPI)
			stub := &arvadostest.APIStub{Error: errors.New("local cluster failure")}
			if updateFail {
				s.fed.local = stub
			} else {
				s.fed.local = rpc.NewConn(s.cluster.ClusterID, spy.URL, true, rpc.PassthroughTokenProvider)
			}
			userlist, err := s.fed.UserList(s.ctx, opts)
			if err != nil {
				c.Logf("... UserList failed %q", err)
			}
			if updateFail && err == nil {
				// All local updates fail, so the only
				// cases expected to succeed are the
				// ones with 0 results.
				c.Check(userlist.Items, check.HasLen, 0)
				c.Check(stub.Calls(nil), check.HasLen, 0)
			} else if updateFail {
				c.Logf("... err %#v", err)
				calls := stub.Calls(stub.UserBatchUpdate)
				if c.Check(calls, check.HasLen, 1) {
					c.Logf("... stub.UserUpdate called with options: %#v", calls[0].Options)
					shouldUpdate := map[string]bool{
						"uuid":       false,
						"email":      true,
						"first_name": true,
						"last_name":  true,
						"is_admin":   true,
						"is_active":  true,
						"prefs":      true,
						// can't safely update locally
						"owner_uuid":   false,
						"identity_url": false,
						// virtual attrs
						"full_name":  false,
						"is_invited": false,
					}
					if opts.Select != nil {
						// Only the selected
						// fields (minus uuid)
						// should be updated.
						for k := range shouldUpdate {
							shouldUpdate[k] = false
						}
						for _, k := range opts.Select {
							if k != "uuid" {
								shouldUpdate[k] = true
							}
						}
					}
					var uuid string
					for uuid = range calls[0].Options.(arvados.UserBatchUpdateOptions).Updates {
					}
					for k, shouldFind := range shouldUpdate {
						_, found := calls[0].Options.(arvados.UserBatchUpdateOptions).Updates[uuid][k]
						c.Check(found, check.Equals, shouldFind, check.Commentf("offending attr: %s", k))
					}
				}
			} else {
				updates := 0
				for _, d := range spy.RequestDumps {
					d := string(d)
					if strings.Contains(d, "PATCH /arvados/v1/users/batch") {
						c.Check(d, check.Matches, `(?ms).*Authorization: Bearer `+arvadostest.SystemRootToken+`.*`)
						updates++
					}
				}
				c.Check(err, check.IsNil)
				c.Check(updates, check.Equals, 1)
				c.Logf("... response items %#v", userlist.Items)
			}
		}
	}
}

func (s *UserSuite) TestLoginClusterUserGet(c *check.C) {
	s.cluster.ClusterID = "local"
	s.cluster.Login.LoginCluster = "zzzzz"
	s.fed = New(s.cluster, nil)
	s.addDirectRemote(c, "zzzzz", rpc.NewConn("zzzzz", &url.URL{Scheme: "https", Host: os.Getenv("ARVADOS_API_HOST")}, true, rpc.PassthroughTokenProvider))

	opts := arvados.GetOptions{UUID: "zzzzz-tpzed-xurymjxw79nv3jz", Select: []string{"uuid", "email"}}

	stub := &arvadostest.APIStub{Error: errors.New("local cluster failure")}
	s.fed.local = stub
	s.fed.UserGet(s.ctx, opts)

	calls := stub.Calls(stub.UserBatchUpdate)
	if c.Check(calls, check.HasLen, 1) {
		c.Logf("... stub.UserUpdate called with options: %#v", calls[0].Options)
		shouldUpdate := map[string]bool{
			"uuid":       false,
			"email":      true,
			"first_name": true,
			"last_name":  true,
			"is_admin":   true,
			"is_active":  true,
			"prefs":      true,
			// can't safely update locally
			"owner_uuid":   false,
			"identity_url": false,
			// virtual attrs
			"full_name":  false,
			"is_invited": false,
		}
		if opts.Select != nil {
			// Only the selected
			// fields (minus uuid)
			// should be updated.
			for k := range shouldUpdate {
				shouldUpdate[k] = false
			}
			for _, k := range opts.Select {
				if k != "uuid" {
					shouldUpdate[k] = true
				}
			}
		}
		var uuid string
		for uuid = range calls[0].Options.(arvados.UserBatchUpdateOptions).Updates {
		}
		for k, shouldFind := range shouldUpdate {
			_, found := calls[0].Options.(arvados.UserBatchUpdateOptions).Updates[uuid][k]
			c.Check(found, check.Equals, shouldFind, check.Commentf("offending attr: %s", k))
		}
	}

}

func (s *UserSuite) TestLoginClusterUserListBypassFederation(c *check.C) {
	s.cluster.ClusterID = "local"
	s.cluster.Login.LoginCluster = "zzzzz"
	s.fed = New(s.cluster, nil)
	s.addDirectRemote(c, "zzzzz", rpc.NewConn("zzzzz", &url.URL{Scheme: "https", Host: os.Getenv("ARVADOS_API_HOST")},
		true, rpc.PassthroughTokenProvider))

	spy := arvadostest.NewProxy(c, s.cluster.Services.RailsAPI)
	s.fed.local = rpc.NewConn(s.cluster.ClusterID, spy.URL, true, rpc.PassthroughTokenProvider)

	_, err := s.fed.UserList(s.ctx, arvados.ListOptions{Offset: 0, Limit: math.MaxInt64, Select: nil, BypassFederation: true})
	// this will fail because it is not using a root token
	c.Check(err.(*arvados.TransactionError).StatusCode, check.Equals, 403)

	// Now use SystemRootToken
	ctx := context.Background()
	ctx = ctxlog.Context(ctx, ctxlog.TestLogger(c))
	ctx = auth.NewContext(ctx, &auth.Credentials{Tokens: []string{arvadostest.SystemRootToken}})

	// Assert that it did not try to batch update users.
	_, err = s.fed.UserList(ctx, arvados.ListOptions{Offset: 0, Limit: math.MaxInt64, Select: nil, BypassFederation: true})
	for _, d := range spy.RequestDumps {
		d := string(d)
		if strings.Contains(d, "PATCH /arvados/v1/users/batch") {
			c.Fail()
		}
	}
	c.Check(err, check.IsNil)
}

// userAttrsCachedFromLoginCluster must have an entry for every field
// in the User struct.
func (s *UserSuite) TestUserAttrsUpdateWhitelist(c *check.C) {
	buf, err := json.Marshal(&arvados.User{})
	c.Assert(err, check.IsNil)
	var allFields map[string]interface{}
	err = json.Unmarshal(buf, &allFields)
	c.Assert(err, check.IsNil)
	for k := range allFields {
		_, ok := userAttrsCachedFromLoginCluster[k]
		c.Check(ok, check.Equals, true, check.Commentf("field name %q missing from userAttrsCachedFromLoginCluster", k))
	}
}
