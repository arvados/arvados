// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"git.arvados.org/arvados.git/lib/ctrlctx"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
)

// CollectionGet defers to railsProxy for everything except blob
// signatures.
func (conn *Conn) CollectionGet(ctx context.Context, opts arvados.GetOptions) (arvados.Collection, error) {
	conn.logActivity(ctx)
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
	conn.logActivity(ctx)
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

// CollectionCreate defers to railsProxy for everything except blob
// signatures and vocabulary checking.
func (conn *Conn) CollectionCreate(ctx context.Context, opts arvados.CreateOptions) (arvados.Collection, error) {
	conn.logActivity(ctx)
	err := conn.checkProperties(ctx, opts.Attrs["properties"])
	if err != nil {
		return arvados.Collection{}, err
	}
	if len(opts.Select) > 0 {
		// We need to know IsTrashed and TrashAt to implement
		// signing properly, even if the caller doesn't want
		// them.
		opts.Select = append([]string{"is_trashed", "trash_at"}, opts.Select...)
	}
	if opts.Attrs, err = conn.applyReplaceFilesOption(ctx, "", opts.Attrs, opts.ReplaceFiles); err != nil {
		return arvados.Collection{}, err
	}
	resp, err := conn.railsProxy.CollectionCreate(ctx, opts)
	if err != nil {
		return resp, err
	}
	conn.signCollection(ctx, &resp)
	return resp, nil
}

// CollectionUpdate defers to railsProxy for everything except blob
// signatures and vocabulary checking.
func (conn *Conn) CollectionUpdate(ctx context.Context, opts arvados.UpdateOptions) (arvados.Collection, error) {
	conn.logActivity(ctx)
	err := conn.checkProperties(ctx, opts.Attrs["properties"])
	if err != nil {
		return arvados.Collection{}, err
	}
	if len(opts.Select) > 0 {
		// We need to know IsTrashed and TrashAt to implement
		// signing properly, even if the caller doesn't want
		// them.
		opts.Select = append([]string{"is_trashed", "trash_at"}, opts.Select...)
	}
	err = conn.lockUUID(ctx, opts.UUID)
	if err != nil {
		return arvados.Collection{}, err
	}
	if opts.Attrs, err = conn.applyReplaceFilesOption(ctx, opts.UUID, opts.Attrs, opts.ReplaceFiles); err != nil {
		return arvados.Collection{}, err
	}
	resp, err := conn.railsProxy.CollectionUpdate(ctx, opts)
	if err != nil {
		return resp, err
	}
	conn.signCollection(ctx, &resp)
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

func (conn *Conn) lockUUID(ctx context.Context, uuid string) error {
	tx, err := ctrlctx.CurrentTx(ctx)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `insert into uuid_locks (uuid) values ($1) on conflict (uuid) do update set n=uuid_locks.n+1`, uuid)
	if err != nil {
		return err
	}
	return nil
}

// If replaceFiles is non-empty, populate attrs["manifest_text"] by
// starting with the content of fromUUID (or an empty collection if
// fromUUID is empty) and applying the specified file/directory
// replacements.
//
// Return value is the (possibly modified) attrs map.
func (conn *Conn) applyReplaceFilesOption(ctx context.Context, fromUUID string, attrs map[string]interface{}, replaceFiles map[string]string) (map[string]interface{}, error) {
	if len(replaceFiles) == 0 {
		return attrs, nil
	} else if mtxt, ok := attrs["manifest_text"].(string); ok && len(mtxt) > 0 {
		return nil, httpserver.Errorf(http.StatusBadRequest, "ambiguous request: both 'replace_files' and attrs['manifest_text'] values provided")
	}

	// Load the current collection (if any) and set up an
	// in-memory filesystem.
	var dst arvados.Collection
	if _, replacingRoot := replaceFiles["/"]; !replacingRoot && fromUUID != "" {
		src, err := conn.CollectionGet(ctx, arvados.GetOptions{UUID: fromUUID})
		if err != nil {
			return nil, err
		}
		dst = src
	}
	dstfs, err := dst.FileSystem(&arvados.StubClient{}, &arvados.StubClient{})
	if err != nil {
		return nil, err
	}

	// Sort replacements by source collection to avoid redundant
	// reloads when a source collection is used more than
	// once. Note empty sources (which mean "delete target path")
	// sort first.
	dstTodo := make([]string, 0, len(replaceFiles))
	{
		srcid := make(map[string]string, len(replaceFiles))
		for dst, src := range replaceFiles {
			dstTodo = append(dstTodo, dst)
			if i := strings.IndexRune(src, '/'); i > 0 {
				srcid[dst] = src[:i]
			}
		}
		sort.Slice(dstTodo, func(i, j int) bool {
			return srcid[dstTodo[i]] < srcid[dstTodo[j]]
		})
	}

	// Reject attempt to replace a node as well as its descendant
	// (e.g., a/ and a/b/), which is unsupported, except where the
	// source for a/ is empty (i.e., delete).
	for _, dst := range dstTodo {
		if dst != "/" && (strings.HasSuffix(dst, "/") ||
			strings.HasSuffix(dst, "/.") ||
			strings.HasSuffix(dst, "/..") ||
			strings.Contains(dst, "//") ||
			strings.Contains(dst, "/./") ||
			strings.Contains(dst, "/../") ||
			!strings.HasPrefix(dst, "/")) {
			return nil, httpserver.Errorf(http.StatusBadRequest, "invalid replace_files target: %q", dst)
		}
		for i := 0; i < len(dst)-1; i++ {
			if dst[i] != '/' {
				continue
			}
			outerdst := dst[:i]
			if outerdst == "" {
				outerdst = "/"
			}
			if outersrc := replaceFiles[outerdst]; outersrc != "" {
				return nil, httpserver.Errorf(http.StatusBadRequest, "replace_files: cannot operate on target %q inside non-empty target %q", dst, outerdst)
			}
		}
	}

	var srcidloaded string
	var srcfs arvados.FileSystem
	// Apply the requested replacements.
	for _, dst := range dstTodo {
		src := replaceFiles[dst]
		if src == "" {
			if dst == "/" {
				// In this case we started with a
				// blank manifest, so there can't be
				// anything to delete.
				continue
			}
			err := dstfs.RemoveAll(dst)
			if err != nil {
				return nil, fmt.Errorf("RemoveAll(%s): %w", dst, err)
			}
			continue
		}
		srcspec := strings.SplitN(src, "/", 2)
		srcid, srcpath := srcspec[0], "/"
		if !arvadosclient.PDHMatch(srcid) {
			return nil, httpserver.Errorf(http.StatusBadRequest, "invalid source %q for replace_files[%q]: must be \"\" or \"PDH\" or \"PDH/path\"", src, dst)
		}
		if len(srcspec) == 2 && srcspec[1] != "" {
			srcpath = srcspec[1]
		}
		if srcidloaded != srcid {
			srcfs = nil
			srccoll, err := conn.CollectionGet(ctx, arvados.GetOptions{UUID: srcid})
			if err != nil {
				return nil, err
			}
			// We use StubClient here because we don't
			// want srcfs to read/write any file data or
			// sync collection state to/from the database.
			srcfs, err = srccoll.FileSystem(&arvados.StubClient{}, &arvados.StubClient{})
			if err != nil {
				return nil, err
			}
			srcidloaded = srcid
		}
		snap, err := arvados.Snapshot(srcfs, srcpath)
		if err != nil {
			return nil, httpserver.Errorf(http.StatusBadRequest, "error getting snapshot of %q from %q: %w", srcpath, srcid, err)
		}
		// Create intermediate dirs, in case dst is
		// "newdir1/newdir2/dst".
		for i := 1; i < len(dst)-1; i++ {
			if dst[i] == '/' {
				err = dstfs.Mkdir(dst[:i], 0777)
				if err != nil && !os.IsExist(err) {
					return nil, httpserver.Errorf(http.StatusBadRequest, "error creating parent dirs for %q: %w", dst, err)
				}
			}
		}
		err = arvados.Splice(dstfs, dst, snap)
		if err != nil {
			return nil, fmt.Errorf("error splicing snapshot onto path %q: %w", dst, err)
		}
	}
	mtxt, err := dstfs.MarshalManifest(".")
	if err != nil {
		return nil, err
	}
	if attrs == nil {
		attrs = make(map[string]interface{}, 1)
	}
	attrs["manifest_text"] = mtxt
	return attrs, nil
}
