// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// Package railsproxy implements Arvados APIs by proxying to the
// RailsAPI server on the local machine.
package railsproxy

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"git.curoverse.com/arvados.git/lib/controller/rpc"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/auth"
)

// For now, FindRailsAPI always uses the rails API running on this
// node.
func FindRailsAPI(cluster *arvados.Cluster) (*url.URL, bool, error) {
	var best *url.URL
	for target := range cluster.Services.RailsAPI.InternalURLs {
		target := url.URL(target)
		best = &target
		if strings.HasPrefix(target.Host, "localhost:") || strings.HasPrefix(target.Host, "127.0.0.1:") || strings.HasPrefix(target.Host, "[::1]:") {
			break
		}
	}
	if best == nil {
		return nil, false, fmt.Errorf("Services.RailsAPI.InternalURLs is empty")
	}
	return best, cluster.TLS.Insecure, nil
}

func NewConn(cluster *arvados.Cluster) *rpc.Conn {
	url, insecure, err := FindRailsAPI(cluster)
	if err != nil {
		panic(err)
	}
	return rpc.NewConn(cluster.ClusterID, url, insecure, provideIncomingToken)
}

func provideIncomingToken(ctx context.Context) ([]string, error) {
	incoming, ok := ctx.Value(auth.ContextKeyCredentials).(*auth.Credentials)
	if !ok {
		return nil, errors.New("no token provided")
	}
	return incoming.Tokens, nil
}
