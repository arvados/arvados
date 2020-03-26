// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
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
		return arvados.LoginResponse{}, err
	}
	err = tx.Authenticate(pam.DisallowNullAuthtok)
	if err != nil {
		return arvados.LoginResponse{}, httpserver.ErrorWithStatus(err, http.StatusUnauthorized)
	}
	if errorMessage != "" {
		return arvados.LoginResponse{}, httpserver.ErrorWithStatus(errors.New(errorMessage), http.StatusUnauthorized)
	}
	user, err := tx.GetItem(pam.User)
	if err != nil {
		return arvados.LoginResponse{}, err
	}
	email := user
	if domain := ctrl.Cluster.Login.PAMDefaultEmailDomain; domain != "" && !strings.Contains(email, "@") {
		email = email + "@" + domain
	}
	ctxlog.FromContext(ctx).WithFields(logrus.Fields{"user": user, "email": email}).Debug("pam authentication succeeded")
	ctxRoot := auth.NewContext(ctx, &auth.Credentials{Tokens: []string{ctrl.Cluster.SystemRootToken}})
	resp, err := ctrl.RailsProxy.UserSessionCreate(ctxRoot, rpc.UserSessionCreateOptions{
		// Send a fake ReturnTo value instead of the caller's
		// opts.ReturnTo. We won't follow the resulting
		// redirect target anyway.
		ReturnTo: opts.Remote + ",https://none.invalid",
		AuthInfo: rpc.UserSessionAuthInfo{
			Username: user,
			Email:    email,
		},
	})
	if err != nil {
		return arvados.LoginResponse{}, err
	}
	target, err := url.Parse(resp.RedirectLocation)
	if err != nil {
		return arvados.LoginResponse{}, err
	}
	resp.Token = target.Query().Get("api_token")
	resp.RedirectLocation = ""
	return resp, err
}
