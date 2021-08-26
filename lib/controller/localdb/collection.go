// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/auth"
)

// CollectionGet defers to railsProxy for everything except blob
// signatures.
func (conn *Conn) CollectionGet(ctx context.Context, opts arvados.GetOptions) (arvados.Collection, error) {
	if len(opts.Select) > 0 {
		// We need to know IsTrashed and TrashAt to implement
		// signing properly, even if the caller doesn't want
		// them.
		opts.Select = append([]string{"is_trashed", "trash_at"}, opts.Select...)
	}
	resp, err := conn.railsProxy.CollectionGet(ctx, opts)
	if err != nil {
		return resp, err
	}
	conn.signCollection(ctx, &resp)
	return resp, nil
}

// CollectionList defers to railsProxy for everything except blob
// signatures.
func (conn *Conn) CollectionList(ctx context.Context, opts arvados.ListOptions) (arvados.CollectionList, error) {
	if len(opts.Select) > 0 {
		// We need to know IsTrashed and TrashAt to implement
		// signing properly, even if the caller doesn't want
		// them.
		opts.Select = append([]string{"is_trashed", "trash_at"}, opts.Select...)
	}
	resp, err := conn.railsProxy.CollectionList(ctx, opts)
	if err != nil {
		return resp, err
	}
	for i := range resp.Items {
		conn.signCollection(ctx, &resp.Items[i])
	}
	return resp, nil
}

func (conn *Conn) signCollection(ctx context.Context, coll *arvados.Collection) {
	if coll.IsTrashed || coll.ManifestText == "" || !conn.cluster.Collections.BlobSigning {
		return
	}
	var token string
	if creds, ok := auth.FromContext(ctx); ok && len(creds.Tokens) > 0 {
		token = creds.Tokens[0]
	}
	if token == "" {
		return
	}
	ttl := conn.cluster.Collections.BlobSigningTTL.Duration()
	exp := time.Now().Add(ttl)
	if coll.TrashAt != nil && !coll.TrashAt.IsZero() && coll.TrashAt.Before(exp) {
		exp = *coll.TrashAt
	}
	coll.ManifestText = arvados.SignManifest(coll.ManifestText, token, exp, ttl, []byte(conn.cluster.Collections.BlobSigningKey))
}
