// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"database/sql"
	"errors"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

// newSessionV1 returns a v1 session -- see
// https://dev.arvados.org/projects/arvados/wiki/Websocket_server
func newSessionV1(ws wsConn, sendq chan<- interface{}, db *sql.DB, pc permChecker, ac *arvados.Client) (session, error) {
	return nil, errors.New("Not implemented")
}
