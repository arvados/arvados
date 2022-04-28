// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepbalance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

// KeepService represents a keepstore server that is being rebalanced.
type KeepService struct {
	arvados.KeepService
	mounts []*KeepMount
	*ChangeSet
}

// String implements fmt.Stringer.
func (srv *KeepService) String() string {
	return fmt.Sprintf("%s (%s:%d, %s)", srv.UUID, srv.ServiceHost, srv.ServicePort, srv.ServiceType)
}

var ksSchemes = map[bool]string{false: "http", true: "https"}

// URLBase returns scheme://host:port for this server.
func (srv *KeepService) URLBase() string {
	return fmt.Sprintf("%s://%s:%d", ksSchemes[srv.ServiceSSLFlag], srv.ServiceHost, srv.ServicePort)
}

// CommitPulls sends the current list of pull requests to the storage
// server (even if the list is empty).
func (srv *KeepService) CommitPulls(ctx context.Context, c *arvados.Client) error {
	return srv.put(ctx, c, "pull", srv.ChangeSet.Pulls)
}

// CommitTrash sends the current list of trash requests to the storage
// server (even if the list is empty).
func (srv *KeepService) CommitTrash(ctx context.Context, c *arvados.Client) error {
	return srv.put(ctx, c, "trash", srv.ChangeSet.Trashes)
}

// Perform a PUT request at path, with data (as JSON) in the request
// body.
func (srv *KeepService) put(ctx context.Context, c *arvados.Client, path string, data interface{}) error {
	// We'll start a goroutine to do the JSON encoding, so we can
	// stream it to the http client through a Pipe, rather than
	// keeping the entire encoded version in memory.
	jsonR, jsonW := io.Pipe()

	// errC communicates any encoding errors back to our main
	// goroutine.
	errC := make(chan error, 1)

	go func() {
		enc := json.NewEncoder(jsonW)
		errC <- enc.Encode(data)
		jsonW.Close()
	}()

	url := srv.URLBase() + "/" + path
	req, err := http.NewRequestWithContext(ctx, "PUT", url, ioutil.NopCloser(jsonR))
	if err != nil {
		return fmt.Errorf("building request for %s: %v", url, err)
	}
	err = c.DoAndDecode(nil, req)

	// If there was an error encoding the request body, report
	// that instead of the response: obviously we won't get a
	// useful response if our request wasn't properly encoded.
	if encErr := <-errC; encErr != nil {
		return fmt.Errorf("encoding data for %s: %v", url, encErr)
	}

	return err
}

func (srv *KeepService) discoverMounts(c *arvados.Client) error {
	mounts, err := srv.Mounts(c)
	if err != nil {
		return fmt.Errorf("%s: error retrieving mounts: %v", srv, err)
	}
	srv.mounts = nil
	for _, m := range mounts {
		srv.mounts = append(srv.mounts, &KeepMount{
			KeepMount:   m,
			KeepService: srv,
		})
	}
	return nil
}

type KeepMount struct {
	arvados.KeepMount
	KeepService *KeepService
}

// String implements fmt.Stringer.
func (mnt *KeepMount) String() string {
	return fmt.Sprintf("%s (%s) on %s", mnt.UUID, mnt.DeviceID, mnt.KeepService)
}
