// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"fmt"
	"strings"

	"git.arvados.org/arvados.git/lib/controller/railsproxy"
	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/sdk/go/arvados"
)

type railsProxy = rpc.Conn

type Conn struct {
	cluster     *arvados.Cluster
	*railsProxy // handles API methods that aren't defined on Conn itself
	loginController
}

func NewConn(cluster *arvados.Cluster) *Conn {
	railsProxy := railsproxy.NewConn(cluster)
	var conn Conn
	conn = Conn{
		cluster:    cluster,
		railsProxy: railsProxy,
	}
	conn.loginController = chooseLoginController(cluster, &conn)
	return &conn
}

// Logout handles the logout of conn giving to the appropriate loginController
func (conn *Conn) Logout(ctx context.Context, opts arvados.LogoutOptions) (arvados.LogoutResponse, error) {
	return conn.loginController.Logout(ctx, opts)
}

// Login handles the login of conn giving to the appropriate loginController
func (conn *Conn) Login(ctx context.Context, opts arvados.LoginOptions) (arvados.LoginResponse, error) {
	return conn.loginController.Login(ctx, opts)
}

// UserAuthenticate handles the User Authentication of conn giving to the appropriate loginController
func (conn *Conn) UserAuthenticate(ctx context.Context, opts arvados.UserAuthenticateOptions) (arvados.APIClientAuthorization, error) {
	return conn.loginController.UserAuthenticate(ctx, opts)
}

func (conn *Conn) GroupContents(ctx context.Context, options arvados.GroupContentsOptions) (arvados.ObjectList, error) {
	// The requested UUID can be a user (virtual home project), which we just pass on to
	// the API server.
	if strings.Index(options.UUID, "-j7d0g-") != 5 {
		return conn.railsProxy.GroupContents(ctx, options)
	}

	var resp arvados.ObjectList

	// Get the group object
	respGroup, err := conn.GroupGet(ctx, arvados.GetOptions{UUID: options.UUID})
	if err != nil {
		return resp, err
	}

	// If the group has groupClass 'filter', apply the filters before getting the contents.
	if respGroup.GroupClass == "filter" {
		if filters, ok := respGroup.Properties["filters"].([]interface{}); ok {
			for _, f := range filters {
				// f is supposed to be a []string
				tmp, ok2 := f.([]interface{})
				if !ok2 || len(tmp) < 3 {
					return resp, fmt.Errorf("filter unparsable: %T, %+v, original field: %T, %+v\n", tmp, tmp, f, f)
				}
				var filter arvados.Filter
				if attr, ok2 := tmp[0].(string); ok2 {
					filter.Attr = attr
				} else {
					return resp, fmt.Errorf("filter unparsable: attribute must be string: %T, %+v, filter: %T, %+v\n", tmp[0], tmp[0], f, f)
				}
				if operator, ok2 := tmp[1].(string); ok2 {
					filter.Operator = operator
				} else {
					return resp, fmt.Errorf("filter unparsable: operator must be string: %T, %+v, filter: %T, %+v\n", tmp[1], tmp[1], f, f)
				}
				filter.Operand = tmp[2]
				options.Filters = append(options.Filters, filter)
			}
		}
		// Use the generic /groups/contents endpoint for filter groups
		options.UUID = ""
	}

	return conn.railsProxy.GroupContents(ctx, options)
}
