// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"encoding/json"
	"net"
	"net/http"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/lib/controller/railsproxy"
	"git.arvados.org/arvados.git/lib/ctrlctx"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/bradleypeabody/godap"
	"github.com/jmoiron/sqlx"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&LDAPSuite{})

type LDAPSuite struct {
	cluster *arvados.Cluster
	ctrl    *ldapLoginController
	ldap    *godap.LDAPServer // fake ldap server that accepts auth goodusername/goodpassword
	db      *sqlx.DB

	// transaction context
	ctx      context.Context
	rollback func() error
}

func (s *LDAPSuite) TearDownSuite(c *check.C) {
	// Undo any changes/additions to the user database so they
	// don't affect subsequent tests.
	arvadostest.ResetEnv()
	c.Check(arvados.NewClientFromEnv().RequestAndDecode(nil, "POST", "database/reset", nil, nil), check.IsNil)
}

func (s *LDAPSuite) SetUpSuite(c *check.C) {
	cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
	c.Assert(err, check.IsNil)
	s.cluster, err = cfg.GetCluster("")
	c.Assert(err, check.IsNil)

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
	s.cluster.Login.LDAP.StartTLS = false
	s.cluster.Login.LDAP.SearchBindUser = "cn=goodusername,dc=example,dc=com"
	s.cluster.Login.LDAP.SearchBindPassword = "goodpassword"
	s.cluster.Login.LDAP.SearchBase = "dc=example,dc=com"
	c.Assert(err, check.IsNil)
	s.ctrl = &ldapLoginController{
		Cluster: s.cluster,
		Parent:  &Conn{railsProxy: railsproxy.NewConn(s.cluster)},
	}
	s.db = arvadostest.DB(c, s.cluster)
}

func (s *LDAPSuite) SetUpTest(c *check.C) {
	tx, err := s.db.Beginx()
	c.Assert(err, check.IsNil)
	s.ctx = ctrlctx.NewWithTransaction(context.Background(), tx)
	s.rollback = tx.Rollback
}

func (s *LDAPSuite) TearDownTest(c *check.C) {
	if s.rollback != nil {
		s.rollback()
	}
}

func (s *LDAPSuite) TestLoginSuccess(c *check.C) {
	conn := NewConn(context.Background(), s.cluster, (&ctrlctx.DBConnector{PostgreSQL: s.cluster.PostgreSQL}).GetDB)
	conn.loginController = s.ctrl
	resp, err := conn.UserAuthenticate(s.ctx, arvados.UserAuthenticateOptions{
		Username: "goodusername",
		Password: "goodpassword",
	})
	c.Check(err, check.IsNil)
	c.Check(resp.APIToken, check.Not(check.Equals), "")
	c.Check(resp.UUID, check.Matches, `zzzzz-gj3su-.*`)
	c.Check(resp.Scopes, check.DeepEquals, []string{"all"})

	ctx := auth.NewContext(s.ctx, &auth.Credentials{Tokens: []string{"v2/" + resp.UUID + "/" + resp.APIToken}})
	user, err := railsproxy.NewConn(s.cluster).UserGetCurrent(ctx, arvados.GetOptions{})
	c.Check(err, check.IsNil)
	c.Check(user.Email, check.Equals, "goodusername@example.com")
	c.Check(user.Username, check.Equals, "goodusername")
}

func (s *LDAPSuite) TestLoginFailure(c *check.C) {
	// search returns no results
	s.cluster.Login.LDAP.SearchBase = "dc=example,dc=invalid"
	resp, err := s.ctrl.UserAuthenticate(s.ctx, arvados.UserAuthenticateOptions{
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
	resp, err = s.ctrl.UserAuthenticate(s.ctx, arvados.UserAuthenticateOptions{
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
