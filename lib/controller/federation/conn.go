// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package federation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/lib/controller/localdb"
	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/health"
	"github.com/jmoiron/sqlx"
)

type Conn struct {
	bgCtx   context.Context
	cluster *arvados.Cluster
	local   backend
	remotes map[string]backend
}

func New(bgCtx context.Context, cluster *arvados.Cluster, healthFuncs *map[string]health.Func, getdb func(context.Context) (*sqlx.DB, error)) *Conn {
	local := localdb.NewConn(bgCtx, cluster, getdb)
	remotes := map[string]backend{}
	for id, remote := range cluster.RemoteClusters {
		if !remote.Proxy || id == cluster.ClusterID {
			continue
		}
		conn := rpc.NewConn(id, &url.URL{Scheme: remote.Scheme, Host: remote.Host}, remote.Insecure, saltedTokenProvider(cluster, local, id))
		// Older versions of controller rely on the Via header
		// to detect loops.
		conn.SendHeader = http.Header{"Via": {"HTTP/1.1 arvados-controller"}}
		remotes[id] = conn
	}

	if healthFuncs != nil {
		hf := map[string]health.Func{"vocabulary": local.LastVocabularyError}
		*healthFuncs = hf
	}

	return &Conn{
		bgCtx:   bgCtx,
		cluster: cluster,
		local:   local,
		remotes: remotes,
	}
}

// Return a new rpc.TokenProvider that takes the client-provided
// tokens from an incoming request context, determines whether they
// should (and can) be salted for the given remoteID, and returns the
// resulting tokens.
func saltedTokenProvider(cluster *arvados.Cluster, local backend, remoteID string) rpc.TokenProvider {
	return func(ctx context.Context) ([]string, error) {
		var tokens []string
		incoming, ok := auth.FromContext(ctx)
		if !ok {
			return nil, errors.New("no token provided")
		}
		for _, token := range incoming.Tokens {
			if strings.HasPrefix(token, "v2/"+cluster.ClusterID+"-") &&
				!strings.HasPrefix(token, "v2/"+cluster.ClusterID+"-gj3su-anonymouspublic/") &&
				remoteID == cluster.Login.LoginCluster {
				// If we did this, the login cluster would call back to us and then
				// reject our response because the user UUID prefix (i.e., the
				// LoginCluster prefix) won't match the token UUID prefix (i.e., our
				// prefix). The anonymous token is OK to forward, because (unlike other
				// local tokens for real users) the validation callback will return the
				// locally issued anonymous user ID instead of a login-cluster user ID.
				// That anonymous user ID gets mapped to the local anonymous user
				// automatically on the login cluster.
				return nil, httpErrorf(http.StatusUnauthorized, "cannot use a locally issued token to forward a request to our login cluster (%s)", remoteID)
			}
			salted, err := auth.SaltToken(token, remoteID)
			switch err {
			case nil:
				tokens = append(tokens, salted)
			case auth.ErrSalted:
				tokens = append(tokens, token)
			case auth.ErrTokenFormat:
				// pass through unmodified (assume it's an OIDC access token)
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
				if strings.HasPrefix(aca.UUID, remoteID) {
					// We have it cached here, but
					// the token belongs to the
					// remote target itself, so
					// pass it through unmodified.
					tokens = append(tokens, token)
					continue
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

func (conn *Conn) localOrLoginCluster() backend {
	if conn.cluster.Login.LoginCluster != "" {
		return conn.chooseBackend(conn.cluster.Login.LoginCluster)
	}
	return conn.local
}

// Call fn with the local backend; then, if fn returned 404, call fn
// on the available remote backends (possibly concurrently) until one
// succeeds.
//
// The second argument to fn is the cluster ID of the remote backend,
// or "" for the local backend.
//
// A non-nil error means all backends failed.
func (conn *Conn) tryLocalThenRemotes(ctx context.Context, forwardedFor string, fn func(context.Context, string, backend) error) error {
	if err := fn(ctx, "", conn.local); err == nil || errStatus(err) != http.StatusNotFound || forwardedFor != "" {
		// Note: forwardedFor != "" means this request came
		// from a remote cluster, so we don't take a second
		// hop. This avoids cycles, redundant calls to a
		// mutually reachable remote, and use of double-salted
		// tokens.
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
	returncode := http.StatusNotFound
	var errs []error
	for i := 0; i < cap(errchan); i++ {
		err := <-errchan
		if err == nil {
			return nil
		}
		errs = append(errs, err)
		if code := errStatus(err); code >= 500 || code == http.StatusTooManyRequests {
			// If any of the remotes have a retryable
			// error (and none succeed) we'll return 502.
			returncode = http.StatusBadGateway
		} else if code != http.StatusNotFound && returncode != http.StatusBadGateway {
			// If some of the remotes have non-retryable
			// non-404 errors (and none succeed or have
			// retryable errors) we'll return 422.
			returncode = http.StatusUnprocessableEntity
		}
	}
	if returncode == http.StatusNotFound {
		return notFoundError{}
	}
	return httpErrorf(returncode, "errors: %v", errs)
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

func (conn *Conn) ConfigGet(ctx context.Context) (json.RawMessage, error) {
	var buf bytes.Buffer
	err := config.ExportJSON(&buf, conn.cluster)
	return json.RawMessage(buf.Bytes()), err
}

func (conn *Conn) VocabularyGet(ctx context.Context) (arvados.Vocabulary, error) {
	return conn.local.VocabularyGet(ctx)
}

func (conn *Conn) DiscoveryDocument(ctx context.Context) (arvados.DiscoveryDocument, error) {
	return conn.local.DiscoveryDocument(ctx)
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
		params := url.Values{
			"return_to": []string{options.ReturnTo},
		}
		if options.Remote != "" {
			params.Set("remote", options.Remote)
		}
		target.RawQuery = params.Encode()
		return arvados.LoginResponse{
			RedirectLocation: target.String(),
		}, nil
	}
	return conn.local.Login(ctx, options)
}

var v2TokenRegexp = regexp.MustCompile(`^v2/[a-z0-9]{5}-gj3su-[a-z0-9]{15}/`)

func (conn *Conn) Logout(ctx context.Context, options arvados.LogoutOptions) (arvados.LogoutResponse, error) {
	// If the token was issued by another cluster, we want to issue a logout
	// request to the issuing instance to invalidate the token federation-wide.
	// If this federation has a login cluster, that's always considered the
	// issuing cluster.
	// Otherwise, if this is a v2 token, use the UUID to find the issuing
	// cluster.
	// Note that remoteBE may still be conn.local even *after* one of these
	// conditions is true.
	var remoteBE backend = conn.local
	if conn.cluster.Login.LoginCluster != "" {
		remoteBE = conn.chooseBackend(conn.cluster.Login.LoginCluster)
	} else {
		reqauth, ok := auth.FromContext(ctx)
		if ok && len(reqauth.Tokens) > 0 && v2TokenRegexp.MatchString(reqauth.Tokens[0]) {
			remoteBE = conn.chooseBackend(reqauth.Tokens[0][3:8])
		}
	}

	// We always want to invalidate the token locally. Start that process.
	var localResponse arvados.LogoutResponse
	var localErr error
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		localResponse, localErr = conn.local.Logout(ctx, options)
		wg.Done()
	}()

	// If the token was issued by another cluster, log out there too.
	if remoteBE != conn.local {
		response, err := remoteBE.Logout(ctx, options)
		// If the issuing cluster returns a redirect or error, that's more
		// important to return to the user than anything that happens locally.
		if response.RedirectLocation != "" || err != nil {
			return response, err
		}
	}

	// Either the local cluster is the issuing cluster, or the issuing cluster's
	// response was uninteresting.
	wg.Wait()
	return localResponse, localErr
}

func (conn *Conn) AuthorizedKeyCreate(ctx context.Context, options arvados.CreateOptions) (arvados.AuthorizedKey, error) {
	return conn.chooseBackend(options.ClusterID).AuthorizedKeyCreate(ctx, options)
}

func (conn *Conn) AuthorizedKeyUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.AuthorizedKey, error) {
	return conn.chooseBackend(options.UUID).AuthorizedKeyUpdate(ctx, options)
}

func (conn *Conn) AuthorizedKeyGet(ctx context.Context, options arvados.GetOptions) (arvados.AuthorizedKey, error) {
	return conn.chooseBackend(options.UUID).AuthorizedKeyGet(ctx, options)
}

func (conn *Conn) AuthorizedKeyList(ctx context.Context, options arvados.ListOptions) (arvados.AuthorizedKeyList, error) {
	return conn.generated_AuthorizedKeyList(ctx, options)
}

func (conn *Conn) AuthorizedKeyDelete(ctx context.Context, options arvados.DeleteOptions) (arvados.AuthorizedKey, error) {
	return conn.chooseBackend(options.UUID).AuthorizedKeyDelete(ctx, options)
}

func (conn *Conn) CollectionGet(ctx context.Context, options arvados.GetOptions) (arvados.Collection, error) {
	if len(options.UUID) == 27 {
		// UUID is really a UUID
		c, err := conn.chooseBackend(options.UUID).CollectionGet(ctx, options)
		if err == nil && options.UUID[:5] != conn.cluster.ClusterID {
			c.ManifestText = rewriteManifest(c.ManifestText, options.UUID[:5])
		}
		return c, err
	}
	if len(options.UUID) < 34 || options.UUID[32] != '+' {
		return arvados.Collection{}, httpErrorf(http.StatusNotFound, "invalid UUID or PDH %q", options.UUID)
	}
	// UUID is a PDH
	first := make(chan arvados.Collection, 1)
	err := conn.tryLocalThenRemotes(ctx, options.ForwardedFor, func(ctx context.Context, remoteID string, be backend) error {
		remoteOpts := options
		remoteOpts.ForwardedFor = conn.cluster.ClusterID + "-" + options.ForwardedFor
		c, err := be.CollectionGet(ctx, remoteOpts)
		if err != nil {
			return err
		}
		haveManifest := true
		if options.Select != nil {
			haveManifest = false
			for _, s := range options.Select {
				if s == "manifest_text" {
					haveManifest = true
					break
				}
			}
		}
		if haveManifest {
			pdh := arvados.PortableDataHash(c.ManifestText)
			// options.UUID is either hash+size or
			// hash+size+hints; only hash+size need to
			// match the computed PDH.
			if pdh != options.UUID && !strings.HasPrefix(options.UUID, pdh+"+") {
				err = httpErrorf(http.StatusBadGateway, "bad portable data hash %q received from remote %q (expected %q)", pdh, remoteID, options.UUID)
				ctxlog.FromContext(ctx).Warn(err)
				return err
			}
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

func (conn *Conn) CollectionList(ctx context.Context, options arvados.ListOptions) (arvados.CollectionList, error) {
	return conn.generated_CollectionList(ctx, options)
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

func (conn *Conn) ContainerList(ctx context.Context, options arvados.ListOptions) (arvados.ContainerList, error) {
	return conn.generated_ContainerList(ctx, options)
}

func (conn *Conn) ContainerCreate(ctx context.Context, options arvados.CreateOptions) (arvados.Container, error) {
	return conn.chooseBackend(options.ClusterID).ContainerCreate(ctx, options)
}

func (conn *Conn) ContainerUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.Container, error) {
	return conn.chooseBackend(options.UUID).ContainerUpdate(ctx, options)
}

func (conn *Conn) ContainerPriorityUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.Container, error) {
	return conn.chooseBackend(options.UUID).ContainerPriorityUpdate(ctx, options)
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

func (conn *Conn) ContainerSSH(ctx context.Context, options arvados.ContainerSSHOptions) (arvados.ConnectionResponse, error) {
	return conn.chooseBackend(options.UUID).ContainerSSH(ctx, options)
}

func (conn *Conn) ContainerGatewayTunnel(ctx context.Context, options arvados.ContainerGatewayTunnelOptions) (arvados.ConnectionResponse, error) {
	return conn.chooseBackend(options.UUID).ContainerGatewayTunnel(ctx, options)
}

func (conn *Conn) ContainerRequestList(ctx context.Context, options arvados.ListOptions) (arvados.ContainerRequestList, error) {
	return conn.generated_ContainerRequestList(ctx, options)
}

func (conn *Conn) ContainerRequestCreate(ctx context.Context, options arvados.CreateOptions) (arvados.ContainerRequest, error) {
	be := conn.chooseBackend(options.ClusterID)
	if be == conn.local {
		return be.ContainerRequestCreate(ctx, options)
	}
	if _, ok := options.Attrs["runtime_token"]; !ok {
		// If runtime_token is not set, create a new token
		aca, err := conn.local.APIClientAuthorizationCurrent(ctx, arvados.GetOptions{})
		if err != nil {
			// This should probably be StatusUnauthorized
			// (need to update test in
			// lib/controller/federation_test.go):
			// When RoR is out of the picture this should be:
			// return arvados.ContainerRequest{}, httpErrorf(http.StatusUnauthorized, "%w", err)
			return arvados.ContainerRequest{}, httpErrorf(http.StatusForbidden, "%s", "invalid API token")
		}
		user, err := conn.local.UserGetCurrent(ctx, arvados.GetOptions{})
		if err != nil {
			return arvados.ContainerRequest{}, err
		}
		if len(aca.Scopes) == 0 || aca.Scopes[0] != "all" {
			return arvados.ContainerRequest{}, httpErrorf(http.StatusForbidden, "token scope is not [all]")
		}
		if strings.HasPrefix(aca.UUID, conn.cluster.ClusterID) {
			// Local user, submitting to a remote cluster.
			// Create a new time-limited token.
			local, ok := conn.local.(*localdb.Conn)
			if !ok {
				return arvados.ContainerRequest{}, httpErrorf(http.StatusInternalServerError, "bug: local backend is a %T, not a *localdb.Conn", conn.local)
			}
			aca, err = local.CreateAPIClientAuthorization(ctx, conn.cluster.SystemRootToken, rpc.UserSessionAuthInfo{UserUUID: user.UUID,
				ExpiresAt: time.Now().UTC().Add(conn.cluster.Collections.BlobSigningTTL.Duration())})
			if err != nil {
				return arvados.ContainerRequest{}, err
			}
			options.Attrs["runtime_token"] = aca.TokenV2()
		} else {
			// Remote user. Container request will use the
			// current token, minus the trailing portion
			// (optional container uuid).
			options.Attrs["runtime_token"] = aca.TokenV2()
		}
	}
	return be.ContainerRequestCreate(ctx, options)
}

func (conn *Conn) ContainerRequestUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.ContainerRequest, error) {
	return conn.chooseBackend(options.UUID).ContainerRequestUpdate(ctx, options)
}

func (conn *Conn) ContainerRequestGet(ctx context.Context, options arvados.GetOptions) (arvados.ContainerRequest, error) {
	return conn.chooseBackend(options.UUID).ContainerRequestGet(ctx, options)
}

func (conn *Conn) ContainerRequestDelete(ctx context.Context, options arvados.DeleteOptions) (arvados.ContainerRequest, error) {
	return conn.chooseBackend(options.UUID).ContainerRequestDelete(ctx, options)
}

func (conn *Conn) ContainerRequestContainerStatus(ctx context.Context, options arvados.GetOptions) (arvados.ContainerStatus, error) {
	return conn.chooseBackend(options.UUID).ContainerRequestContainerStatus(ctx, options)
}

func (conn *Conn) ContainerRequestLog(ctx context.Context, options arvados.ContainerLogOptions) (http.Handler, error) {
	return conn.chooseBackend(options.UUID).ContainerRequestLog(ctx, options)
}

func (conn *Conn) GroupCreate(ctx context.Context, options arvados.CreateOptions) (arvados.Group, error) {
	return conn.chooseBackend(options.ClusterID).GroupCreate(ctx, options)
}

func (conn *Conn) GroupUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.Group, error) {
	return conn.chooseBackend(options.UUID).GroupUpdate(ctx, options)
}

func (conn *Conn) GroupGet(ctx context.Context, options arvados.GetOptions) (arvados.Group, error) {
	return conn.chooseBackend(options.UUID).GroupGet(ctx, options)
}

func (conn *Conn) GroupList(ctx context.Context, options arvados.ListOptions) (arvados.GroupList, error) {
	return conn.generated_GroupList(ctx, options)
}

var userUuidRe = regexp.MustCompile(`^[0-9a-z]{5}-tpzed-[0-9a-z]{15}$`)

func (conn *Conn) GroupContents(ctx context.Context, options arvados.GroupContentsOptions) (arvados.ObjectList, error) {
	if options.ClusterID != "" {
		// explicitly selected cluster
		return conn.chooseBackend(options.ClusterID).GroupContents(ctx, options)
	} else if userUuidRe.MatchString(options.UUID) {
		// user, get the things they own on the local cluster
		return conn.local.GroupContents(ctx, options)
	} else {
		// a group, potentially want to make federated request
		return conn.chooseBackend(options.UUID).GroupContents(ctx, options)
	}
}

func (conn *Conn) GroupShared(ctx context.Context, options arvados.ListOptions) (arvados.GroupList, error) {
	return conn.chooseBackend(options.ClusterID).GroupShared(ctx, options)
}

func (conn *Conn) GroupDelete(ctx context.Context, options arvados.DeleteOptions) (arvados.Group, error) {
	return conn.chooseBackend(options.UUID).GroupDelete(ctx, options)
}

func (conn *Conn) GroupTrash(ctx context.Context, options arvados.DeleteOptions) (arvados.Group, error) {
	return conn.chooseBackend(options.UUID).GroupTrash(ctx, options)
}

func (conn *Conn) GroupUntrash(ctx context.Context, options arvados.UntrashOptions) (arvados.Group, error) {
	return conn.chooseBackend(options.UUID).GroupUntrash(ctx, options)
}

func (conn *Conn) LinkCreate(ctx context.Context, options arvados.CreateOptions) (arvados.Link, error) {
	return conn.chooseBackend(options.ClusterID).LinkCreate(ctx, options)
}

func (conn *Conn) LinkUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.Link, error) {
	return conn.chooseBackend(options.UUID).LinkUpdate(ctx, options)
}

func (conn *Conn) LinkGet(ctx context.Context, options arvados.GetOptions) (arvados.Link, error) {
	return conn.chooseBackend(options.UUID).LinkGet(ctx, options)
}

func (conn *Conn) LinkList(ctx context.Context, options arvados.ListOptions) (arvados.LinkList, error) {
	return conn.generated_LinkList(ctx, options)
}

func (conn *Conn) LinkDelete(ctx context.Context, options arvados.DeleteOptions) (arvados.Link, error) {
	return conn.chooseBackend(options.UUID).LinkDelete(ctx, options)
}

func (conn *Conn) LogCreate(ctx context.Context, options arvados.CreateOptions) (arvados.Log, error) {
	return conn.chooseBackend(options.ClusterID).LogCreate(ctx, options)
}

func (conn *Conn) LogUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.Log, error) {
	return conn.chooseBackend(options.UUID).LogUpdate(ctx, options)
}

func (conn *Conn) LogGet(ctx context.Context, options arvados.GetOptions) (arvados.Log, error) {
	return conn.chooseBackend(options.UUID).LogGet(ctx, options)
}

func (conn *Conn) LogList(ctx context.Context, options arvados.ListOptions) (arvados.LogList, error) {
	return conn.generated_LogList(ctx, options)
}

func (conn *Conn) LogDelete(ctx context.Context, options arvados.DeleteOptions) (arvados.Log, error) {
	return conn.chooseBackend(options.UUID).LogDelete(ctx, options)
}

func (conn *Conn) SpecimenList(ctx context.Context, options arvados.ListOptions) (arvados.SpecimenList, error) {
	return conn.generated_SpecimenList(ctx, options)
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

func (conn *Conn) SysTrashSweep(ctx context.Context, options struct{}) (struct{}, error) {
	return conn.local.SysTrashSweep(ctx, options)
}

var userAttrsCachedFromLoginCluster = map[string]bool{
	"created_at":  true,
	"email":       true,
	"first_name":  true,
	"is_active":   true,
	"is_admin":    true,
	"is_invited":  true,
	"last_name":   true,
	"modified_at": true,
	"prefs":       true,
	"username":    true,
	"kind":        true,

	"etag":                    false,
	"full_name":               false,
	"identity_url":            false,
	"modified_by_client_uuid": false,
	"modified_by_user_uuid":   false,
	"owner_uuid":              false,
	"uuid":                    false,
	"writable_by":             false,
	"can_write":               false,
	"can_manage":              false,
}

func (conn *Conn) batchUpdateUsers(ctx context.Context,
	options arvados.ListOptions,
	items []arvados.User,
	includeAdminAndInvited bool) (err error) {

	id := conn.cluster.Login.LoginCluster
	logger := ctxlog.FromContext(ctx)
	batchOpts := arvados.UserBatchUpdateOptions{Updates: map[string]map[string]interface{}{}}
	for _, user := range items {
		if !strings.HasPrefix(user.UUID, id) {
			continue
		}
		logger.Debugf("cache user info for uuid %q", user.UUID)

		// If the remote cluster has null timestamps
		// (e.g., test server with incomplete
		// fixtures) use dummy timestamps (instead of
		// the zero time, which causes a Rails API
		// error "year too big to marshal: 1 UTC").
		if user.ModifiedAt.IsZero() {
			user.ModifiedAt = time.Now()
		}
		if user.CreatedAt.IsZero() {
			user.CreatedAt = time.Now()
		}

		var allFields map[string]interface{}
		buf, err := json.Marshal(user)
		if err != nil {
			return fmt.Errorf("error encoding user record from remote response: %s", err)
		}
		err = json.Unmarshal(buf, &allFields)
		if err != nil {
			return fmt.Errorf("error transcoding user record from remote response: %s", err)
		}
		updates := allFields
		if len(options.Select) > 0 {
			updates = map[string]interface{}{}
			for _, k := range options.Select {
				if v, ok := allFields[k]; ok && userAttrsCachedFromLoginCluster[k] {
					updates[k] = v
				}
			}
		} else {
			for k := range updates {
				if !userAttrsCachedFromLoginCluster[k] {
					delete(updates, k)
				}
			}
		}
		if !includeAdminAndInvited {
			// make sure we don't send these fields.
			delete(updates, "is_admin")
			delete(updates, "is_invited")
		}
		batchOpts.Updates[user.UUID] = updates
	}
	if len(batchOpts.Updates) > 0 {
		ctxRoot := auth.NewContext(ctx, &auth.Credentials{Tokens: []string{conn.cluster.SystemRootToken}})
		_, err = conn.local.UserBatchUpdate(ctxRoot, batchOpts)
		if err != nil {
			return fmt.Errorf("error updating local user records: %s", err)
		}
	}
	return nil
}

func (conn *Conn) includeAdminAndInvitedInBatchUpdate(ctx context.Context, be backend, updateUserUUID string) (bool, error) {
	// API versions prior to 20231117 would only include the
	// is_invited and is_admin fields if the current user is an
	// admin, or is requesting their own user record.  If those
	// fields aren't actually valid then we don't want to
	// send them in the batch update.
	dd, err := be.DiscoveryDocument(ctx)
	if err != nil {
		// couldn't get discovery document
		return false, err
	}
	if dd.Revision >= "20231117" {
		// newer version, fields are valid.
		return true, nil
	}
	selfuser, err := be.UserGetCurrent(ctx, arvados.GetOptions{})
	if err != nil {
		// couldn't get our user record
		return false, err
	}
	if selfuser.IsAdmin || selfuser.UUID == updateUserUUID {
		// we are an admin, or the current user is the same as
		// the user that we are updating.
		return true, nil
	}
	// Better safe than sorry.
	return false, nil
}

func (conn *Conn) UserList(ctx context.Context, options arvados.ListOptions) (arvados.UserList, error) {
	if id := conn.cluster.Login.LoginCluster; id != "" && id != conn.cluster.ClusterID && !options.BypassFederation {
		be := conn.chooseBackend(id)
		resp, err := be.UserList(ctx, options)
		if err != nil {
			return resp, err
		}
		includeAdminAndInvited, err := conn.includeAdminAndInvitedInBatchUpdate(ctx, be, "")
		if err != nil {
			return arvados.UserList{}, err
		}
		err = conn.batchUpdateUsers(ctx, options, resp.Items, includeAdminAndInvited)
		if err != nil {
			return arvados.UserList{}, err
		}
		return resp, nil
	}
	return conn.generated_UserList(ctx, options)
}

func (conn *Conn) UserCreate(ctx context.Context, options arvados.CreateOptions) (arvados.User, error) {
	return conn.chooseBackend(options.ClusterID).UserCreate(ctx, options)
}

func (conn *Conn) UserUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.User, error) {
	if options.BypassFederation {
		return conn.local.UserUpdate(ctx, options)
	}
	be := conn.chooseBackend(options.UUID)
	resp, err := be.UserUpdate(ctx, options)
	if err != nil {
		return resp, err
	}
	if !strings.HasPrefix(options.UUID, conn.cluster.ClusterID) {
		includeAdminAndInvited, err := conn.includeAdminAndInvitedInBatchUpdate(ctx, be, options.UUID)
		if err != nil {
			return arvados.User{}, err
		}
		// Copy the updated user record to the local cluster
		err = conn.batchUpdateUsers(ctx, arvados.ListOptions{}, []arvados.User{resp}, includeAdminAndInvited)
		if err != nil {
			return arvados.User{}, err
		}
	}
	return resp, err
}

func (conn *Conn) UserMerge(ctx context.Context, options arvados.UserMergeOptions) (arvados.User, error) {
	return conn.local.UserMerge(ctx, options)
}

func (conn *Conn) UserActivate(ctx context.Context, options arvados.UserActivateOptions) (arvados.User, error) {
	return conn.localOrLoginCluster().UserActivate(ctx, options)
}

func (conn *Conn) UserSetup(ctx context.Context, options arvados.UserSetupOptions) (map[string]interface{}, error) {
	upstream := conn.localOrLoginCluster()
	if upstream != conn.local {
		// When LoginCluster is in effect, and we're setting
		// up a remote user, and we want to give that user
		// access to a local VM, we can't include the VM in
		// the setup call, because the remote cluster won't
		// recognize it.

		// Similarly, if we want to create a git repo,
		// it should be created on the local cluster,
		// not the remote one.

		upstreamOptions := options
		upstreamOptions.VMUUID = ""
		upstreamOptions.RepoName = ""

		ret, err := upstream.UserSetup(ctx, upstreamOptions)
		if err != nil {
			return ret, err
		}
	}

	return conn.local.UserSetup(ctx, options)
}

func (conn *Conn) UserUnsetup(ctx context.Context, options arvados.GetOptions) (arvados.User, error) {
	return conn.localOrLoginCluster().UserUnsetup(ctx, options)
}

func (conn *Conn) UserGet(ctx context.Context, options arvados.GetOptions) (arvados.User, error) {
	be := conn.chooseBackend(options.UUID)
	resp, err := be.UserGet(ctx, options)
	if err != nil {
		return resp, err
	}
	if options.UUID != resp.UUID {
		return arvados.User{}, httpErrorf(http.StatusBadGateway, "Had requested %v but response was for %v", options.UUID, resp.UUID)
	}
	if options.UUID[:5] != conn.cluster.ClusterID {
		includeAdminAndInvited, err := conn.includeAdminAndInvitedInBatchUpdate(ctx, be, options.UUID)
		if err != nil {
			return arvados.User{}, err
		}
		err = conn.batchUpdateUsers(ctx, arvados.ListOptions{Select: options.Select}, []arvados.User{resp}, includeAdminAndInvited)
		if err != nil {
			return arvados.User{}, err
		}
	}
	return resp, nil
}

func (conn *Conn) UserGetCurrent(ctx context.Context, options arvados.GetOptions) (arvados.User, error) {
	return conn.local.UserGetCurrent(ctx, options)
}

func (conn *Conn) UserGetSystem(ctx context.Context, options arvados.GetOptions) (arvados.User, error) {
	return conn.chooseBackend(options.UUID).UserGetSystem(ctx, options)
}

func (conn *Conn) UserDelete(ctx context.Context, options arvados.DeleteOptions) (arvados.User, error) {
	return conn.chooseBackend(options.UUID).UserDelete(ctx, options)
}

func (conn *Conn) UserBatchUpdate(ctx context.Context, options arvados.UserBatchUpdateOptions) (arvados.UserList, error) {
	return conn.local.UserBatchUpdate(ctx, options)
}

func (conn *Conn) UserAuthenticate(ctx context.Context, options arvados.UserAuthenticateOptions) (arvados.APIClientAuthorization, error) {
	return conn.local.UserAuthenticate(ctx, options)
}

func (conn *Conn) APIClientAuthorizationCurrent(ctx context.Context, options arvados.GetOptions) (arvados.APIClientAuthorization, error) {
	return conn.chooseBackend(options.UUID).APIClientAuthorizationCurrent(ctx, options)
}

func (conn *Conn) APIClientAuthorizationCreate(ctx context.Context, options arvados.CreateOptions) (arvados.APIClientAuthorization, error) {
	if conn.cluster.Login.LoginCluster != "" {
		return conn.chooseBackend(conn.cluster.Login.LoginCluster).APIClientAuthorizationCreate(ctx, options)
	}
	ownerUUID, ok := options.Attrs["owner_uuid"].(string)
	if ok && ownerUUID != "" {
		return conn.chooseBackend(ownerUUID).APIClientAuthorizationCreate(ctx, options)
	}
	return conn.local.APIClientAuthorizationCreate(ctx, options)
}

func (conn *Conn) APIClientAuthorizationUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.APIClientAuthorization, error) {
	if options.BypassFederation {
		return conn.local.APIClientAuthorizationUpdate(ctx, options)
	}
	return conn.chooseBackend(options.UUID).APIClientAuthorizationUpdate(ctx, options)
}

func (conn *Conn) APIClientAuthorizationDelete(ctx context.Context, options arvados.DeleteOptions) (arvados.APIClientAuthorization, error) {
	return conn.chooseBackend(options.UUID).APIClientAuthorizationDelete(ctx, options)
}

func (conn *Conn) APIClientAuthorizationList(ctx context.Context, options arvados.ListOptions) (arvados.APIClientAuthorizationList, error) {
	if id := conn.cluster.Login.LoginCluster; id != "" && id != conn.cluster.ClusterID && !options.BypassFederation {
		return conn.chooseBackend(conn.cluster.Login.LoginCluster).APIClientAuthorizationList(ctx, options)
	}
	return conn.generated_APIClientAuthorizationList(ctx, options)
}

func (conn *Conn) APIClientAuthorizationGet(ctx context.Context, options arvados.GetOptions) (arvados.APIClientAuthorization, error) {
	return conn.chooseBackend(options.UUID).APIClientAuthorizationGet(ctx, options)
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
	}
	return http.StatusInternalServerError
}
