// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

//go:build !static

package localdb

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"github.com/msteinert/pam"
	"github.com/sirupsen/logrus"
)

type pamLoginController struct {
	Cluster *arvados.Cluster
	Parent  *Conn
}

func (ctrl *pamLoginController) Logout(ctx context.Context, opts arvados.LogoutOptions) (arvados.LogoutResponse, error) {
	return logout(ctx, ctrl.Cluster, opts)
}

func (ctrl *pamLoginController) Login(ctx context.Context, opts arvados.LoginOptions) (arvados.LoginResponse, error) {
	return arvados.LoginResponse{}, errors.New("interactive login is not available")
}

func (ctrl *pamLoginController) UserAuthenticate(ctx context.Context, opts arvados.UserAuthenticateOptions) (arvados.APIClientAuthorization, error) {
	errorMessage := ""
	sentPassword := false
	tx, err := pam.StartFunc(ctrl.Cluster.Login.PAM.Service, opts.Username, func(style pam.Style, message string) (string, error) {
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
			sentPassword = true
			return opts.Password, nil
		default:
			return "", fmt.Errorf("unrecognized message style %d", style)
		}
	})
	if err != nil {
		return arvados.APIClientAuthorization{}, err
	}
	// Check that the given credentials are valid.
	err = tx.Authenticate(pam.DisallowNullAuthtok)
	if err != nil {
		err = fmt.Errorf("PAM: %s", err)
		if errorMessage != "" {
			// Perhaps the error message in the
			// conversation is helpful.
			err = fmt.Errorf("%s; %q", err, errorMessage)
		}
		if sentPassword {
			err = fmt.Errorf("%s (with username %q and password)", err, opts.Username)
		} else {
			// This might hint that the username was
			// invalid.
			err = fmt.Errorf("%s (with username %q; password was never requested by PAM service)", err, opts.Username)
		}
		return arvados.APIClientAuthorization{}, httpserver.ErrorWithStatus(err, http.StatusUnauthorized)
	}
	if errorMessage != "" {
		return arvados.APIClientAuthorization{}, httpserver.ErrorWithStatus(errors.New(errorMessage), http.StatusUnauthorized)
	}
	// Check that the account/user is permitted to access this host.
	err = tx.AcctMgmt(pam.DisallowNullAuthtok)
	if err != nil {
		err = fmt.Errorf("PAM: %s", err)
		if errorMessage != "" {
			err = fmt.Errorf("%s; %q", err, errorMessage)
		}
		return arvados.APIClientAuthorization{}, httpserver.ErrorWithStatus(err, http.StatusUnauthorized)
	}
	user, err := tx.GetItem(pam.User)
	if err != nil {
		return arvados.APIClientAuthorization{}, err
	}
	email := user
	if domain := ctrl.Cluster.Login.PAM.DefaultEmailDomain; domain != "" && !strings.Contains(email, "@") {
		email = email + "@" + domain
	}
	ctxlog.FromContext(ctx).WithFields(logrus.Fields{
		"user":  user,
		"email": email,
	}).Debug("pam authentication succeeded")
	return ctrl.Parent.CreateAPIClientAuthorization(ctx, ctrl.Cluster.SystemRootToken, rpc.UserSessionAuthInfo{
		Username: user,
		Email:    email,
	})
}
