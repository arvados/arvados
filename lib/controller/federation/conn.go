// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package federation

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"git.curoverse.com/arvados.git/lib/config"
	"git.curoverse.com/arvados.git/lib/controller/localdb"
	"git.curoverse.com/arvados.git/lib/controller/rpc"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/auth"
	"git.curoverse.com/arvados.git/sdk/go/ctxlog"
)

type Conn struct {
	cluster *arvados.Cluster
	local   backend
	remotes map[string]backend
}

func New(cluster *arvados.Cluster) *Conn {
	local := localdb.NewConn(cluster)
	remotes := map[string]backend{}
	for id, remote := range cluster.RemoteClusters {
		if !remote.Proxy {
			continue
		}
		remotes[id] = rpc.NewConn(id, &url.URL{Scheme: remote.Scheme, Host: remote.Host}, remote.Insecure, saltedTokenProvider(local, id))
	}

	return &Conn{
		cluster: cluster,
		local:   local,
		remotes: remotes,
	}
}

// Return a new rpc.TokenProvider that takes the client-provided
// tokens from an incoming request context, determines whether they
// should (and can) be salted for the given remoteID, and returns the
// resulting tokens.
func saltedTokenProvider(local backend, remoteID string) rpc.TokenProvider {
	return func(ctx context.Context) ([]string, error) {
		var tokens []string
		incoming, ok := auth.FromContext(ctx)
		if !ok {
			return nil, errors.New("no token provided")
		}
		for _, token := range incoming.Tokens {
			salted, err := auth.SaltToken(token, remoteID)
			switch err {
			case nil:
				tokens = append(tokens, salted)
			case auth.ErrSalted:
				tokens = append(tokens, token)
			case auth.ErrObsoleteToken:
				ctx := auth.NewContext(ctx, &auth.Credentials{Tokens: []string{token}})
				aca, err := local.APIClientAuthorizationCurrent(ctx, arvados.GetOptions{})
				if errStatus(err) == http.StatusUnauthorized {
					// pass through unmodified
					tokens = append(tokens, token)
					continue
				} else if err != nil {
					return nil, err
				}
				salted, err := auth.SaltToken(aca.TokenV2(), remoteID)
				if err != nil {
					return nil, err
				}
				tokens = append(tokens, salted)
			default:
				return nil, err
			}
		}
		return tokens, nil
	}
}

// Return suitable backend for a query about the given cluster ID
// ("aaaaa") or object UUID ("aaaaa-dz642-abcdefghijklmno").
func (conn *Conn) chooseBackend(id string) backend {
	if len(id) == 27 {
		id = id[:5]
	} else if len(id) != 5 {
		// PDH or bogus ID
		return conn.local
	}
	if id == conn.cluster.ClusterID {
		return conn.local
	} else if be, ok := conn.remotes[id]; ok {
		return be
	} else {
		// TODO: return an "always error" backend?
		return conn.local
	}
}

// Call fn with the local backend; then, if fn returned 404, call fn
// on the available remote backends (possibly concurrently) until one
// succeeds.
//
// The second argument to fn is the cluster ID of the remote backend,
// or "" for the local backend.
//
// A non-nil error means all backends failed.
func (conn *Conn) tryLocalThenRemotes(ctx context.Context, fn func(context.Context, string, backend) error) error {
	if err := fn(ctx, "", conn.local); err == nil || errStatus(err) != http.StatusNotFound {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errchan := make(chan error, len(conn.remotes))
	for remoteID, be := range conn.remotes {
		remoteID, be := remoteID, be
		go func() {
			errchan <- fn(ctx, remoteID, be)
		}()
	}
	all404 := true
	var errs []error
	for i := 0; i < cap(errchan); i++ {
		err := <-errchan
		if err == nil {
			return nil
		}
		all404 = all404 && errStatus(err) == http.StatusNotFound
		errs = append(errs, err)
	}
	if all404 {
		return notFoundError{}
	}
	return httpErrorf(http.StatusBadGateway, "errors: %v", errs)
}

func (conn *Conn) CollectionCreate(ctx context.Context, options arvados.CreateOptions) (arvados.Collection, error) {
	return conn.chooseBackend(options.ClusterID).CollectionCreate(ctx, options)
}

func (conn *Conn) CollectionUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.Collection, error) {
	return conn.chooseBackend(options.UUID).CollectionUpdate(ctx, options)
}

func rewriteManifest(mt, remoteID string) string {
	return regexp.MustCompile(` [0-9a-f]{32}\+[^ ]*`).ReplaceAllStringFunc(mt, func(tok string) string {
		return strings.Replace(tok, "+A", "+R"+remoteID+"-", -1)
	})
}

// this could be in sdk/go/arvados
func portableDataHash(mt string) string {
	h := md5.New()
	blkRe := regexp.MustCompile(`^ [0-9a-f]{32}\+\d+`)
	size := 0
	_ = regexp.MustCompile(` ?[^ ]*`).ReplaceAllFunc([]byte(mt), func(tok []byte) []byte {
		if m := blkRe.Find(tok); m != nil {
			// write hash+size, ignore remaining block hints
			tok = m
		}
		n, err := h.Write(tok)
		if err != nil {
			panic(err)
		}
		size += n
		return nil
	})
	return fmt.Sprintf("%x+%d", h.Sum(nil), size)
}

func (conn *Conn) ConfigGet(ctx context.Context) (json.RawMessage, error) {
	var buf bytes.Buffer
	err := config.ExportJSON(&buf, conn.cluster)
	return json.RawMessage(buf.Bytes()), err
}

func (conn *Conn) Login(ctx context.Context, options arvados.LoginOptions) (arvados.LoginResponse, error) {
	if id := conn.cluster.Login.LoginCluster; id != "" && id != conn.cluster.ClusterID {
		// defer entire login procedure to designated cluster
		remote, ok := conn.remotes[id]
		if !ok {
			return arvados.LoginResponse{}, fmt.Errorf("configuration problem: designated login cluster %q is not defined", id)
		}
		baseURL := remote.BaseURL()
		target, err := baseURL.Parse(arvados.EndpointLogin.Path)
		if err != nil {
			return arvados.LoginResponse{}, fmt.Errorf("internal error getting redirect target: %s", err)
		}
		target.RawQuery = url.Values{
			"return_to": []string{options.ReturnTo},
			"remote":    []string{options.Remote},
		}.Encode()
		return arvados.LoginResponse{
			RedirectLocation: target.String(),
		}, nil
	} else {
		return conn.local.Login(ctx, options)
	}
}

func (conn *Conn) CollectionGet(ctx context.Context, options arvados.GetOptions) (arvados.Collection, error) {
	if len(options.UUID) == 27 {
		// UUID is really a UUID
		c, err := conn.chooseBackend(options.UUID).CollectionGet(ctx, options)
		if err == nil && options.UUID[:5] != conn.cluster.ClusterID {
			c.ManifestText = rewriteManifest(c.ManifestText, options.UUID[:5])
		}
		return c, err
	} else {
		// UUID is a PDH
		first := make(chan arvados.Collection, 1)
		err := conn.tryLocalThenRemotes(ctx, func(ctx context.Context, remoteID string, be backend) error {
			c, err := be.CollectionGet(ctx, options)
			if err != nil {
				return err
			}
			// options.UUID is either hash+size or
			// hash+size+hints; only hash+size need to
			// match the computed PDH.
			if pdh := portableDataHash(c.ManifestText); pdh != options.UUID && !strings.HasPrefix(options.UUID, pdh+"+") {
				err = httpErrorf(http.StatusBadGateway, "bad portable data hash %q received from remote %q (expected %q)", pdh, remoteID, options.UUID)
				ctxlog.FromContext(ctx).Warn(err)
				return err
			}
			if remoteID != "" {
				c.ManifestText = rewriteManifest(c.ManifestText, remoteID)
			}
			select {
			case first <- c:
				return nil
			default:
				// lost race, return value doesn't matter
				return nil
			}
		})
		if err != nil {
			return arvados.Collection{}, err
		}
		return <-first, nil
	}
}

func (conn *Conn) CollectionProvenance(ctx context.Context, options arvados.GetOptions) (map[string]interface{}, error) {
	return conn.chooseBackend(options.UUID).CollectionProvenance(ctx, options)
}

func (conn *Conn) CollectionUsedBy(ctx context.Context, options arvados.GetOptions) (map[string]interface{}, error) {
	return conn.chooseBackend(options.UUID).CollectionUsedBy(ctx, options)
}

func (conn *Conn) CollectionDelete(ctx context.Context, options arvados.DeleteOptions) (arvados.Collection, error) {
	return conn.chooseBackend(options.UUID).CollectionDelete(ctx, options)
}

func (conn *Conn) CollectionTrash(ctx context.Context, options arvados.DeleteOptions) (arvados.Collection, error) {
	return conn.chooseBackend(options.UUID).CollectionTrash(ctx, options)
}

func (conn *Conn) CollectionUntrash(ctx context.Context, options arvados.UntrashOptions) (arvados.Collection, error) {
	return conn.chooseBackend(options.UUID).CollectionUntrash(ctx, options)
}

func (conn *Conn) ContainerCreate(ctx context.Context, options arvados.CreateOptions) (arvados.Container, error) {
	return conn.chooseBackend(options.ClusterID).ContainerCreate(ctx, options)
}

func (conn *Conn) ContainerUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.Container, error) {
	return conn.chooseBackend(options.UUID).ContainerUpdate(ctx, options)
}

func (conn *Conn) ContainerGet(ctx context.Context, options arvados.GetOptions) (arvados.Container, error) {
	return conn.chooseBackend(options.UUID).ContainerGet(ctx, options)
}

func (conn *Conn) ContainerDelete(ctx context.Context, options arvados.DeleteOptions) (arvados.Container, error) {
	return conn.chooseBackend(options.UUID).ContainerDelete(ctx, options)
}

func (conn *Conn) ContainerLock(ctx context.Context, options arvados.GetOptions) (arvados.Container, error) {
	return conn.chooseBackend(options.UUID).ContainerLock(ctx, options)
}

func (conn *Conn) ContainerUnlock(ctx context.Context, options arvados.GetOptions) (arvados.Container, error) {
	return conn.chooseBackend(options.UUID).ContainerUnlock(ctx, options)
}

func (conn *Conn) SpecimenCreate(ctx context.Context, options arvados.CreateOptions) (arvados.Specimen, error) {
	return conn.chooseBackend(options.ClusterID).SpecimenCreate(ctx, options)
}

func (conn *Conn) SpecimenUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.Specimen, error) {
	return conn.chooseBackend(options.UUID).SpecimenUpdate(ctx, options)
}

func (conn *Conn) SpecimenGet(ctx context.Context, options arvados.GetOptions) (arvados.Specimen, error) {
	return conn.chooseBackend(options.UUID).SpecimenGet(ctx, options)
}

func (conn *Conn) SpecimenDelete(ctx context.Context, options arvados.DeleteOptions) (arvados.Specimen, error) {
	return conn.chooseBackend(options.UUID).SpecimenDelete(ctx, options)
}

func (conn *Conn) APIClientAuthorizationCurrent(ctx context.Context, options arvados.GetOptions) (arvados.APIClientAuthorization, error) {
	return conn.chooseBackend(options.UUID).APIClientAuthorizationCurrent(ctx, options)
}

type backend interface {
	arvados.API
	BaseURL() url.URL
}

type notFoundError struct{}

func (notFoundError) HTTPStatus() int { return http.StatusNotFound }
func (notFoundError) Error() string   { return "not found" }

func errStatus(err error) int {
	if httpErr, ok := err.(interface{ HTTPStatus() int }); ok {
		return httpErr.HTTPStatus()
	} else {
		return http.StatusInternalServerError
	}
}
