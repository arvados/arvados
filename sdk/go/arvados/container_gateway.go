// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"context"
	"io"
	"net/http"
	"sync"

	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"github.com/sirupsen/logrus"
)

func (cresp ConnectionResponse) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer cresp.Conn.Close()
	conn, bufrw, err := http.NewResponseController(w).Hijack()
	if err != nil {
		http.Error(w, "connection upgrade failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()
	conn.Write([]byte("HTTP/1.1 101 Switching Protocols\r\n"))
	w.Header().Set("Connection", "upgrade")
	for k, v := range cresp.Header {
		w.Header()[k] = v
	}
	w.Header().Write(conn)
	conn.Write([]byte("\r\n"))
	httpserver.ExemptFromDeadline(req)

	var bytesIn, bytesOut int64
	ctx, cancel := context.WithCancel(req.Context())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()
		n, err := io.CopyN(conn, cresp.Bufrw, int64(cresp.Bufrw.Reader.Buffered()))
		bytesOut += n
		if err == nil {
			n, err = io.Copy(conn, cresp.Conn)
			bytesOut += n
		}
		if err != nil {
			ctxlog.FromContext(ctx).WithError(err).Error("error copying downstream")
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()
		n, err := io.CopyN(cresp.Conn, bufrw, int64(bufrw.Reader.Buffered()))
		bytesIn += n
		if err == nil {
			n, err = io.Copy(cresp.Conn, conn)
			bytesIn += n
		}
		if err != nil {
			ctxlog.FromContext(ctx).WithError(err).Error("error copying upstream")
		}
	}()
	<-ctx.Done()
	go func() {
		// Wait for both io.Copy goroutines to finish and increment
		// their byte counters.
		wg.Wait()
		if cresp.Logger != nil {
			cresp.Logger.WithFields(logrus.Fields{
				"bytesIn":  bytesIn,
				"bytesOut": bytesOut,
			}).Info("closed connection")
		}
	}()
}
