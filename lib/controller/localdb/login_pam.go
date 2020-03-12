// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/msteinert/pam"
	"github.com/sirupsen/logrus"
)

type pamLoginController struct {
	Cluster    *arvados.Cluster
	RailsProxy *railsProxy
}

func (ctrl *pamLoginController) Logout(ctx context.Context, opts arvados.LogoutOptions) (arvados.LogoutResponse, error) {
	return noopLogout(ctrl.Cluster, opts)
}

func (ctrl *pamLoginController) Login(ctx context.Context, opts arvados.LoginOptions) (arvados.LoginResponse, error) {
	errorMessage := ""
	tx, err := pam.StartFunc(ctrl.Cluster.Login.PAMService, opts.Username, func(style pam.Style, message string) (string, error) {
		ctxlog.FromContext(ctx).Debugf("pam conversation: style=%v message=%q", style, message)
		switch style {
		case pam.ErrorMsg:
			ctxlog.FromContext(ctx).WithField("Message", message).Info("pam.ErrorMsg")
			errorMessage = message
			return "", nil
		case pam.TextInfo:
			ctxlog.FromContext(ctx).WithField("Message", message).Info("pam.TextInfo")
			return "", nil
		case pam.PromptEchoOn, pam.PromptEchoOff:
			return opts.Password, nil
		default:
			return "", fmt.Errorf("unrecognized message style %d", style)
		}
	})
	if err != nil {
		return arvados.LoginResponse{Message: err.Error()}, nil
	}
	err = tx.Authenticate(pam.DisallowNullAuthtok)
	if err != nil {
		return arvados.LoginResponse{Message: err.Error()}, nil
	}
	if errorMessage != "" {
		return arvados.LoginResponse{Message: errorMessage}, nil
	}
	user, err := tx.GetItem(pam.User)
	if err != nil {
		return arvados.LoginResponse{Message: err.Error()}, nil
	}
	email := user
	if domain := ctrl.Cluster.Login.PAMDefaultEmailDomain; domain != "" && !strings.Contains(email, "@") {
		email = email + "@" + domain
	}
	ctxlog.FromContext(ctx).WithFields(logrus.Fields{"user": user, "email": email}).Debug("pam authentication succeeded")
	ctxRoot := auth.NewContext(ctx, &auth.Credentials{Tokens: []string{ctrl.Cluster.SystemRootToken}})
	resp, err := ctrl.RailsProxy.UserSessionCreate(ctxRoot, rpc.UserSessionCreateOptions{
		ReturnTo: opts.Remote + "," + opts.ReturnTo,
		AuthInfo: rpc.UserSessionAuthInfo{
			Username: user,
			Email:    email,
		},
	})
	if err != nil {
		return arvados.LoginResponse{Message: err.Error()}, nil
	}
	target, err := url.Parse(resp.RedirectLocation)
	if err != nil {
		return arvados.LoginResponse{Message: err.Error()}, nil
	}
	resp.Token = target.Query().Get("api_token")
	resp.RedirectLocation = ""
	return resp, err
}
