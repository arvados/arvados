// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"golang.org/x/crypto/ssh"
)

// AuthorizedKeyCreate checks that the provided public key is valid,
// then proxies to railsproxy.
func (conn *Conn) AuthorizedKeyCreate(ctx context.Context, opts arvados.CreateOptions) (arvados.AuthorizedKey, error) {
	if err := validateKey(opts.Attrs); err != nil {
		return arvados.AuthorizedKey{}, httpserver.ErrorWithStatus(err, http.StatusBadRequest)
	}
	return conn.railsProxy.AuthorizedKeyCreate(ctx, opts)
}

// AuthorizedKeyUpdate checks that the provided public key is valid,
// then proxies to railsproxy.
func (conn *Conn) AuthorizedKeyUpdate(ctx context.Context, opts arvados.UpdateOptions) (arvados.AuthorizedKey, error) {
	if err := validateKey(opts.Attrs); err != nil {
		return arvados.AuthorizedKey{}, httpserver.ErrorWithStatus(err, http.StatusBadRequest)
	}
	return conn.railsProxy.AuthorizedKeyUpdate(ctx, opts)
}

func validateKey(attrs map[string]interface{}) error {
	in, _ := attrs["public_key"].(string)
	if in == "" {
		return nil
	}
	in = strings.TrimSpace(in)
	if strings.IndexAny(in, "\r\n") >= 0 {
		return errors.New("Public key does not appear to be valid: extra data after key")
	}
	pubkey, _, _, rest, err := ssh.ParseAuthorizedKey([]byte(in))
	if err != nil {
		return fmt.Errorf("Public key does not appear to be valid: %w", err)
	}
	if len(rest) > 0 {
		return errors.New("Public key does not appear to be valid: extra data after key")
	}
	if i := strings.Index(in, " "); i < 0 {
		return errors.New("Public key does not appear to be valid: no leading type field")
	} else if in[:i] != pubkey.Type() {
		return fmt.Errorf("Public key does not appear to be valid: leading type field %q does not match actual key type %q", in[:i], pubkey.Type())
	}
	return nil
}
