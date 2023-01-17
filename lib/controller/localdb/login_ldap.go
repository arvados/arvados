// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"github.com/go-ldap/ldap"
)

type ldapLoginController struct {
	Cluster *arvados.Cluster
	Parent  *Conn
}

func (ctrl *ldapLoginController) Logout(ctx context.Context, opts arvados.LogoutOptions) (arvados.LogoutResponse, error) {
	return logout(ctx, ctrl.Cluster, opts)
}

func (ctrl *ldapLoginController) Login(ctx context.Context, opts arvados.LoginOptions) (arvados.LoginResponse, error) {
	return arvados.LoginResponse{}, errors.New("interactive login is not available")
}

func (ctrl *ldapLoginController) UserAuthenticate(ctx context.Context, opts arvados.UserAuthenticateOptions) (arvados.APIClientAuthorization, error) {
	log := ctxlog.FromContext(ctx)
	conf := ctrl.Cluster.Login.LDAP
	errFailed := httpserver.ErrorWithStatus(fmt.Errorf("LDAP: Authentication failure (with username %q and password)", opts.Username), http.StatusUnauthorized)

	if conf.SearchAttribute == "" {
		return arvados.APIClientAuthorization{}, errors.New("config error: SearchAttribute is blank")
	}
	if opts.Password == "" {
		log.WithField("username", opts.Username).Error("refusing to authenticate with empty password")
		return arvados.APIClientAuthorization{}, errFailed
	}

	log = log.WithField("URL", conf.URL.String())
	var l *ldap.Conn
	var err error
	if conf.URL.Scheme == "ldaps" {
		// ldap.DialURL does not currently allow us to control
		// tls.Config, so we need to figure out the port
		// ourselves and call DialTLS.
		host, port, err := net.SplitHostPort(conf.URL.Host)
		if err != nil {
			// Assume error means no port given
			host = conf.URL.Host
			port = ldap.DefaultLdapsPort
		}
		l, err = ldap.DialTLS("tcp", net.JoinHostPort(host, port), &tls.Config{
			ServerName: host,
			MinVersion: uint16(conf.MinTLSVersion),
		})
	} else {
		l, err = ldap.DialURL(conf.URL.String())
	}
	if err != nil {
		log.WithError(err).Error("ldap connection failed")
		return arvados.APIClientAuthorization{}, err
	}
	defer l.Close()

	if conf.StartTLS {
		var tlsconfig tls.Config
		tlsconfig.MinVersion = uint16(conf.MinTLSVersion)
		if conf.InsecureTLS {
			tlsconfig.InsecureSkipVerify = true
		} else {
			if host, _, err := net.SplitHostPort(conf.URL.Host); err != nil {
				// Assume SplitHostPort error means
				// port was not specified
				tlsconfig.ServerName = conf.URL.Host
			} else {
				tlsconfig.ServerName = host
			}
		}
		err = l.StartTLS(&tlsconfig)
		if err != nil {
			log.WithError(err).Error("ldap starttls failed")
			return arvados.APIClientAuthorization{}, err
		}
	}

	username := opts.Username
	if at := strings.Index(username, "@"); at >= 0 {
		if conf.StripDomain == "*" || strings.ToLower(conf.StripDomain) == strings.ToLower(username[at+1:]) {
			username = username[:at]
		}
	}
	if conf.AppendDomain != "" && !strings.Contains(username, "@") {
		username = username + "@" + conf.AppendDomain
	}

	if conf.SearchBindUser != "" {
		err = l.Bind(conf.SearchBindUser, conf.SearchBindPassword)
		if err != nil {
			log.WithError(err).WithField("user", conf.SearchBindUser).Error("ldap authentication failed")
			return arvados.APIClientAuthorization{}, err
		}
	}

	search := fmt.Sprintf("(%s=%s)", ldap.EscapeFilter(conf.SearchAttribute), ldap.EscapeFilter(username))
	if conf.SearchFilters != "" {
		search = fmt.Sprintf("(&%s%s)", conf.SearchFilters, search)
	}
	log = log.WithField("search", search)
	req := ldap.NewSearchRequest(
		conf.SearchBase,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 1, 0, false,
		search,
		[]string{"DN", "givenName", "SN", conf.EmailAttribute, conf.UsernameAttribute},
		nil)
	resp, err := l.Search(req)
	if ldap.IsErrorWithCode(err, ldap.LDAPResultNoResultsReturned) ||
		ldap.IsErrorWithCode(err, ldap.LDAPResultNoSuchObject) ||
		(err == nil && len(resp.Entries) == 0) {
		log.WithError(err).Info("ldap lookup returned no results")
		return arvados.APIClientAuthorization{}, errFailed
	} else if err != nil {
		log.WithError(err).Error("ldap lookup failed")
		return arvados.APIClientAuthorization{}, err
	}
	userdn := resp.Entries[0].DN
	if userdn == "" {
		log.Warn("refusing to authenticate with empty dn")
		return arvados.APIClientAuthorization{}, errFailed
	}
	log = log.WithField("DN", userdn)

	attrs := map[string]string{}
	for _, attr := range resp.Entries[0].Attributes {
		if attr == nil || len(attr.Values) == 0 {
			continue
		}
		attrs[strings.ToLower(attr.Name)] = attr.Values[0]
	}
	log.WithField("attrs", attrs).Debug("ldap search succeeded")

	// Now that we have the DN, try authenticating.
	err = l.Bind(userdn, opts.Password)
	if err != nil {
		log.WithError(err).Info("ldap user authentication failed")
		return arvados.APIClientAuthorization{}, errFailed
	}
	log.Debug("ldap authentication succeeded")

	email := attrs[strings.ToLower(conf.EmailAttribute)]
	if email == "" {
		log.Errorf("ldap returned no email address in %q attribute", conf.EmailAttribute)
		return arvados.APIClientAuthorization{}, errors.New("authentication succeeded but ldap returned no email address")
	}

	return ctrl.Parent.CreateAPIClientAuthorization(ctx, ctrl.Cluster.SystemRootToken, rpc.UserSessionAuthInfo{
		Email:     email,
		FirstName: attrs["givenname"],
		LastName:  attrs["sn"],
		Username:  attrs[strings.ToLower(conf.UsernameAttribute)],
	})
}
