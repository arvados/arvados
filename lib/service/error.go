// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"context"
	"net/http"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/sirupsen/logrus"
)

// ErrorHandler returns a Handler that reports itself as unhealthy and
// responds 500 to all requests.  ErrorHandler itself logs the given
// error once, and the handler logs it again for each incoming
// request.
func ErrorHandler(ctx context.Context, _ *arvados.Cluster, err error) Handler {
	logger := ctxlog.FromContext(ctx)
	logger.WithError(err).Error("unhealthy service")
	return errorHandler{err, logger}
}

type errorHandler struct {
	err    error
	logger logrus.FieldLogger
}

func (eh errorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	eh.logger.WithError(eh.err).Error("unhealthy service")
	http.Error(w, "", http.StatusInternalServerError)
}

func (eh errorHandler) CheckHealth() error {
	return eh.err
}
