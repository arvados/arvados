// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"io"
	"net/http"
	"strings"
	"sync"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/auth"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
)

type remoteProxy struct {
	clients map[string]*keepclient.KeepClient
	mtx     sync.Mutex
}

func (rp *remoteProxy) Get(w http.ResponseWriter, r *http.Request, cluster *arvados.Cluster) {
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
			token := GetAPIToken(r)
			if token == "" {
				http.Error(w, "no token provided in Authorization header", http.StatusUnauthorized)
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
