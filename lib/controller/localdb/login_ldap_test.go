// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"encoding/json"
	"net"
	"net/http"

	"git.arvados.org/arvados.git/lib/controller/railsproxy"
	"git.arvados.org/arvados.git/lib/ctrlctx"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/bradleypeabody/godap"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&LDAPSuite{})

type LDAPSuite struct {
	localdbSuite
	ldap *godap.LDAPServer // fake ldap server that accepts auth goodusername/goodpassword
}

func (s *LDAPSuite) SetUpTest(c *check.C) {
	s.localdbSuite.SetUpTest(c)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	c.Assert(err, check.IsNil)
	s.ldap = &godap.LDAPServer{
		Listener: ln,
		Handlers: []godap.LDAPRequestHandler{
			&godap.LDAPBindFuncHandler{
				LDAPBindFunc: func(binddn string, bindpw []byte) bool {
					return binddn == "cn=goodusername,dc=example,dc=com" && string(bindpw) == "goodpassword"
				},
			},
			&godap.LDAPSimpleSearchFuncHandler{
				LDAPSimpleSearchFunc: func(req *godap.LDAPSimpleSearchRequest) []*godap.LDAPSimpleSearchResultEntry {
					if req.FilterAttr != "uid" || req.BaseDN != "dc=example,dc=com" {
						return []*godap.LDAPSimpleSearchResultEntry{}
					}
					return []*godap.LDAPSimpleSearchResultEntry{
						{
							DN: "cn=" + req.FilterValue + "," + req.BaseDN,
							Attrs: map[string]interface{}{
								"SN":   req.FilterValue,
								"CN":   req.FilterValue,
								"uid":  req.FilterValue,
								"mail": req.FilterValue + "@example.com",
							},
						},
					}
				},
			},
		},
	}
	go func() {
		ctxlog.TestLogger(c).Print(s.ldap.Serve())
	}()

	s.cluster.Login.LDAP.Enable = true
	err = json.Unmarshal([]byte(`"ldap://`+ln.Addr().String()+`"`), &s.cluster.Login.LDAP.URL)
	c.Assert(err, check.IsNil)
	s.cluster.Login.LDAP.StartTLS = false
	s.cluster.Login.LDAP.SearchBindUser = "cn=goodusername,dc=example,dc=com"
	s.cluster.Login.LDAP.SearchBindPassword = "goodpassword"
	s.cluster.Login.LDAP.SearchBase = "dc=example,dc=com"
	s.localdb.loginController = &ldapLoginController{
		Cluster: s.cluster,
		Parent:  s.localdb,
	}
}

func (s *LDAPSuite) TestLoginSuccess(c *check.C) {
	resp, err := s.localdb.UserAuthenticate(s.ctx, arvados.UserAuthenticateOptions{
		Username: "goodusername",
		Password: "goodpassword",
	})
	c.Check(err, check.IsNil)
	c.Check(resp.APIToken, check.Not(check.Equals), "")
	c.Check(resp.UUID, check.Matches, `zzzzz-gj3su-.*`)
	c.Check(resp.Scopes, check.DeepEquals, []string{"all"})

	ctx := ctrlctx.NewWithToken(s.ctx, s.cluster, "v2/"+resp.UUID+"/"+resp.APIToken)
	user, err := railsproxy.NewConn(s.cluster).UserGetCurrent(ctx, arvados.GetOptions{})
	c.Check(err, check.IsNil)
	c.Check(user.Email, check.Equals, "goodusername@example.com")
	c.Check(user.Username, check.Equals, "goodusername")
}

func (s *LDAPSuite) TestLoginFailure(c *check.C) {
	// search returns no results
	s.cluster.Login.LDAP.SearchBase = "dc=example,dc=invalid"
	resp, err := s.localdb.UserAuthenticate(s.ctx, arvados.UserAuthenticateOptions{
		Username: "goodusername",
		Password: "goodpassword",
	})
	c.Check(err, check.ErrorMatches, `LDAP: Authentication failure \(with username "goodusername" and password\)`)
	hs, ok := err.(interface{ HTTPStatus() int })
	if c.Check(ok, check.Equals, true) {
		c.Check(hs.HTTPStatus(), check.Equals, http.StatusUnauthorized)
	}
	c.Check(resp.APIToken, check.Equals, "")

	// search returns result, but auth fails
	s.cluster.Login.LDAP.SearchBase = "dc=example,dc=com"
	resp, err = s.localdb.UserAuthenticate(s.ctx, arvados.UserAuthenticateOptions{
		Username: "badusername",
		Password: "badpassword",
	})
	c.Check(err, check.ErrorMatches, `LDAP: Authentication failure \(with username "badusername" and password\)`)
	hs, ok = err.(interface{ HTTPStatus() int })
	if c.Check(ok, check.Equals, true) {
		c.Check(hs.HTTPStatus(), check.Equals, http.StatusUnauthorized)
	}
	c.Check(resp.APIToken, check.Equals, "")
}
