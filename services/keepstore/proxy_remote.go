// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
)

type remoteProxy struct {
	clients map[string]*keepclient.KeepClient
	mtx     sync.Mutex
}

func (rp *remoteProxy) Get(ctx context.Context, w http.ResponseWriter, r *http.Request, cluster *arvados.Cluster, volmgr *RRVolumeManager) {
	// Intervening proxies must not return a cached GET response
	// to a prior request if a X-Keep-Signature request header has
	// been added or changed.
	w.Header().Add("Vary", "X-Keep-Signature")

	token := GetAPIToken(r)
	if token == "" {
		http.Error(w, "no token provided in Authorization header", http.StatusUnauthorized)
		return
	}
	if strings.SplitN(r.Header.Get("X-Keep-Signature"), ",", 2)[0] == "local" {
		buf, err := getBufferWithContext(ctx, bufs, BlockSize)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		defer bufs.Put(buf)
		rrc := &remoteResponseCacher{
			Locator:        r.URL.Path[1:],
			Token:          token,
			Buffer:         buf[:0],
			ResponseWriter: w,
			Context:        ctx,
			Cluster:        cluster,
			VolumeManager:  volmgr,
		}
		defer rrc.Close()
		w = rrc
	}
	var remoteClient *keepclient.KeepClient
	var parts []string
	for i, part := range strings.Split(r.URL.Path[1:], "+") {
		switch {
		case i == 0:
			// don't try to parse hash part as hint
		case strings.HasPrefix(part, "A"):
			// drop local permission hint
			continue
		case len(part) > 7 && part[0] == 'R' && part[6] == '-':
			remoteID := part[1:6]
			remote, ok := cluster.RemoteClusters[remoteID]
			if !ok {
				http.Error(w, "remote cluster not configured", http.StatusBadRequest)
				return
			}
			kc, err := rp.remoteClient(remoteID, remote, token)
			if err == auth.ErrObsoleteToken {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			} else if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			remoteClient = kc
			part = "A" + part[7:]
		}
		parts = append(parts, part)
	}
	if remoteClient == nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	locator := strings.Join(parts, "+")
	rdr, _, _, err := remoteClient.Get(locator)
	switch err.(type) {
	case nil:
		defer rdr.Close()
		io.Copy(w, rdr)
	case *keepclient.ErrNotFound:
		http.Error(w, err.Error(), http.StatusNotFound)
	default:
		http.Error(w, err.Error(), http.StatusBadGateway)
	}
}

func (rp *remoteProxy) remoteClient(remoteID string, remoteCluster arvados.RemoteCluster, token string) (*keepclient.KeepClient, error) {
	rp.mtx.Lock()
	kc, ok := rp.clients[remoteID]
	rp.mtx.Unlock()
	if !ok {
		c := &arvados.Client{
			APIHost:   remoteCluster.Host,
			AuthToken: "xxx",
			Insecure:  remoteCluster.Insecure,
		}
		ac, err := arvadosclient.New(c)
		if err != nil {
			return nil, err
		}
		kc, err = keepclient.MakeKeepClient(ac)
		if err != nil {
			return nil, err
		}

		rp.mtx.Lock()
		if rp.clients == nil {
			rp.clients = map[string]*keepclient.KeepClient{remoteID: kc}
		} else {
			rp.clients[remoteID] = kc
		}
		rp.mtx.Unlock()
	}
	accopy := *kc.Arvados
	accopy.ApiToken = token
	kccopy := *kc
	kccopy.Arvados = &accopy
	token, err := auth.SaltToken(token, remoteID)
	if err != nil {
		return nil, err
	}
	kccopy.Arvados.ApiToken = token
	return &kccopy, nil
}

var localOrRemoteSignature = regexp.MustCompile(`\+[AR][^\+]*`)

// remoteResponseCacher wraps http.ResponseWriter. It buffers the
// response data in the provided buffer, writes/touches a copy on a
// local volume, adds a response header with a locally-signed locator,
// and finally writes the data through.
type remoteResponseCacher struct {
	Locator       string
	Token         string
	Buffer        []byte
	Context       context.Context
	Cluster       *arvados.Cluster
	VolumeManager *RRVolumeManager
	http.ResponseWriter
	statusCode int
}

func (rrc *remoteResponseCacher) Write(p []byte) (int, error) {
	if len(rrc.Buffer)+len(p) > cap(rrc.Buffer) {
		return 0, errors.New("buffer full")
	}
	rrc.Buffer = append(rrc.Buffer, p...)
	return len(p), nil
}

func (rrc *remoteResponseCacher) WriteHeader(statusCode int) {
	rrc.statusCode = statusCode
}

func (rrc *remoteResponseCacher) Close() error {
	if rrc.statusCode == 0 {
		rrc.statusCode = http.StatusOK
	} else if rrc.statusCode != http.StatusOK {
		rrc.ResponseWriter.WriteHeader(rrc.statusCode)
		rrc.ResponseWriter.Write(rrc.Buffer)
		return nil
	}
	_, err := PutBlock(rrc.Context, rrc.VolumeManager, rrc.Buffer, rrc.Locator[:32], nil)
	if rrc.Context.Err() != nil {
		// If caller hung up, log that instead of subsequent/misleading errors.
		http.Error(rrc.ResponseWriter, rrc.Context.Err().Error(), http.StatusGatewayTimeout)
		return err
	}
	if err == RequestHashError {
		http.Error(rrc.ResponseWriter, "checksum mismatch in remote response", http.StatusBadGateway)
		return err
	}
	if err, ok := err.(*KeepError); ok {
		http.Error(rrc.ResponseWriter, err.Error(), err.HTTPCode)
		return err
	}
	if err != nil {
		http.Error(rrc.ResponseWriter, err.Error(), http.StatusBadGateway)
		return err
	}

	unsigned := localOrRemoteSignature.ReplaceAllLiteralString(rrc.Locator, "")
	expiry := time.Now().Add(rrc.Cluster.Collections.BlobSigningTTL.Duration())
	signed := SignLocator(rrc.Cluster, unsigned, rrc.Token, expiry)
	if signed == unsigned {
		err = errors.New("could not sign locator")
		http.Error(rrc.ResponseWriter, err.Error(), http.StatusInternalServerError)
		return err
	}
	rrc.Header().Set("X-Keep-Locator", signed)
	rrc.ResponseWriter.WriteHeader(rrc.statusCode)
	_, err = rrc.ResponseWriter.Write(rrc.Buffer)
	return err
}
