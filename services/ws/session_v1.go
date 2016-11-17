package main

import (
	"database/sql"
	"errors"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

func NewSessionV1(ws wsConn, sendq chan<- interface{}, ac arvados.Client, db *sql.DB) (session, error) {
	return nil, errors.New("Not implemented")
}
