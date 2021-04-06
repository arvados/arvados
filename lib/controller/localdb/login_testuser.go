// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"bytes"
	"context"
	"fmt"
	"html/template"

	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/sirupsen/logrus"
)

type testLoginController struct {
	Cluster *arvados.Cluster
	Parent  *Conn
}

func (ctrl *testLoginController) Logout(ctx context.Context, opts arvados.LogoutOptions) (arvados.LogoutResponse, error) {
	return logout(ctx, ctrl.Cluster, opts)
}

func (ctrl *testLoginController) Login(ctx context.Context, opts arvados.LoginOptions) (arvados.LoginResponse, error) {
	tmpl, err := template.New("form").Parse(loginform)
	if err != nil {
		return arvados.LoginResponse{}, err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, opts)
	if err != nil {
		return arvados.LoginResponse{}, err
	}
	return arvados.LoginResponse{HTML: buf}, nil
}

func (ctrl *testLoginController) UserAuthenticate(ctx context.Context, opts arvados.UserAuthenticateOptions) (arvados.APIClientAuthorization, error) {
	for username, user := range ctrl.Cluster.Login.Test.Users {
		if (opts.Username == username || opts.Username == user.Email) && opts.Password == user.Password {
			ctxlog.FromContext(ctx).WithFields(logrus.Fields{
				"username": username,
				"email":    user.Email,
			}).Debug("test authentication succeeded")
			return ctrl.Parent.CreateAPIClientAuthorization(ctx, ctrl.Cluster.SystemRootToken, rpc.UserSessionAuthInfo{
				Username: username,
				Email:    user.Email,
			})
		}
	}
	return arvados.APIClientAuthorization{}, fmt.Errorf("authentication failed for user %q with password len=%d", opts.Username, len(opts.Password))
}

const loginform = `
<!doctype html>
<html>
  <head><title>Arvados test login</title>
    <script>
      async function authenticate(event) {
        event.preventDefault()
	document.getElementById('error').innerHTML = ''
	const resp = await fetch('/arvados/v1/users/authenticate', {
	  method: 'POST',
	  mode: 'same-origin',
	  headers: {'Content-Type': 'application/json'},
	  body: JSON.stringify({
	    username: document.getElementById('username').value,
	    password: document.getElementById('password').value,
	  }),
	})
	if (!resp.ok) {
	  document.getElementById('error').innerHTML = '<p>Authentication failed.</p><p>The "test login" users are defined in Clusters.[ClusterID].Login.Test.Users section of config.yml</p><p>If you are using arvbox, use "arvbox adduser" to add users.</p>'
	  return
	}
	var redir = document.getElementById('return_to').value
	if (redir.indexOf('?') > 0) {
	  redir += '&'
	} else {
	  redir += '?'
	}
        const respj = await resp.json()
	document.location = redir + "api_token=v2/" + respj.uuid + "/" + respj.api_token
      }
    </script>
  </head>
  <body>
    <h3>Arvados test login</h3>
    <form method="POST">
      <input id="return_to" type="hidden" name="return_to" value="{{.ReturnTo}}">
      username <input id="username" type="text" name="username" autofocus size=16>
      password <input id="password" type="password" name="password" size=16>
      <input type="submit" value="Log in">
      <br>
      <p id="error"></p>
    </form>
  </body>
  <script>
    document.getElementsByTagName('form')[0].onsubmit = authenticate
  </script>
</html>
`
