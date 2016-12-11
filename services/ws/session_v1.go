package main

import (
	"database/sql"
	"errors"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

func newSessionV1(ws wsConn, sendq chan<- interface{}, db *sql.DB, pc permChecker, ac *arvados.Client) (session, error) {
	return nil, errors.New("Not implemented")
}
