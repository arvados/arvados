// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"context"
	"io"
	"net/http"

	"git.arvados.org/arvados.git/sdk/go/ctxlog"
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

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()
		_, err := io.CopyN(conn, sshconn.Bufrw, int64(sshconn.Bufrw.Reader.Buffered()))
		if err == nil {
			_, err = io.Copy(conn, sshconn.Conn)
		}
		if err != nil {
			ctxlog.FromContext(req.Context()).WithError(err).Error("error copying downstream")
		}
	}()
	go func() {
		defer cancel()
		_, err := io.CopyN(sshconn.Conn, bufrw, int64(bufrw.Reader.Buffered()))
		if err == nil {
			_, err = io.Copy(sshconn.Conn, conn)
		}
		if err != nil {
			ctxlog.FromContext(req.Context()).WithError(err).Error("error copying upstream")
		}
	}()
	<-ctx.Done()
}
