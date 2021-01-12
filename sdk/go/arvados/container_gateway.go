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
	"github.com/sirupsen/logrus"
)

func (sshconn ContainerSSHConnection) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "ResponseWriter does not support connection upgrade", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Connection", "upgrade")
	w.Header().Set("Upgrade", "ssh")
	w.WriteHeader(http.StatusSwitchingProtocols)
	conn, bufrw, err := hj.Hijack()
	if err != nil {
		ctxlog.FromContext(req.Context()).WithError(err).Error("error hijacking ResponseWriter")
		return
	}
	defer conn.Close()

	var bytesIn, bytesOut int64
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()
		n, err := io.CopyN(conn, sshconn.Bufrw, int64(sshconn.Bufrw.Reader.Buffered()))
		bytesOut += n
		if err == nil {
			n, err = io.Copy(conn, sshconn.Conn)
			bytesOut += n
		}
		if err != nil {
			ctxlog.FromContext(req.Context()).WithError(err).Error("error copying downstream")
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()
		n, err := io.CopyN(sshconn.Conn, bufrw, int64(bufrw.Reader.Buffered()))
		bytesIn += n
		if err == nil {
			n, err = io.Copy(sshconn.Conn, conn)
			bytesIn += n
		}
		if err != nil {
			ctxlog.FromContext(req.Context()).WithError(err).Error("error copying upstream")
		}
	}()
	<-ctx.Done()
	if sshconn.Logger != nil {
		go func() {
			wg.Wait()
			sshconn.Logger.WithFields(logrus.Fields{
				"bytesIn":  bytesIn,
				"bytesOut": bytesOut,
			}).Info("closed connection")
		}()
	}
}
