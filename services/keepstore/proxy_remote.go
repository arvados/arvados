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

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/auth"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
)

type remoteProxy struct {
	clients map[string]*keepclient.KeepClient
	mtx     sync.Mutex
}

func (rp *remoteProxy) Get(ctx context.Context, w http.ResponseWriter, r *http.Request, cluster *arvados.Cluster) {
	token := GetAPIToken(r)
	if token == "" {
		http.Error(w, "no token provided in Authorization header", http.StatusUnauthorized)
		return
	}
	if sign := r.Header.Get("X-Keep-Signature"); sign != "" {
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
		}
		defer rrc.Flush(ctx)
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
				http.Error(w, "remote cluster not configured", http.StatusBadGateway)
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
	Locator string
	Token   string
	Buffer  []byte
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

func (rrc *remoteResponseCacher) Flush(ctx context.Context) {
	if rrc.statusCode == 0 {
		rrc.statusCode = http.StatusOK
	} else if rrc.statusCode != http.StatusOK {
		rrc.ResponseWriter.WriteHeader(rrc.statusCode)
		rrc.ResponseWriter.Write(rrc.Buffer)
		return
	}
	_, err := PutBlock(ctx, rrc.Buffer, rrc.Locator[:32])
	if err == RequestHashError {
		http.Error(rrc.ResponseWriter, "checksum mismatch in remote response", http.StatusBadGateway)
		return
	}
	if err, ok := err.(*KeepError); ok {
		http.Error(rrc.ResponseWriter, err.Error(), err.HTTPCode)
		return
	}
	if err != nil {
		http.Error(rrc.ResponseWriter, err.Error(), http.StatusBadGateway)
		return
	}

	unsigned := localOrRemoteSignature.ReplaceAllLiteralString(rrc.Locator, "")
	signed := SignLocator(unsigned, rrc.Token, time.Now().Add(theConfig.BlobSignatureTTL.Duration()))
	if signed == unsigned {
		http.Error(rrc.ResponseWriter, "could not sign locator", http.StatusInternalServerError)
		return
	}
	rrc.Header().Set("X-Keep-Locator", signed)
	rrc.ResponseWriter.WriteHeader(rrc.statusCode)
	rrc.ResponseWriter.Write(rrc.Buffer)
}
