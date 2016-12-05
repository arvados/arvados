package main

import (
	"database/sql"
	"errors"
)

func newSessionV1(ws wsConn, sendq chan<- interface{}, db *sql.DB, pc permChecker) (session, error) {
	return nil, errors.New("Not implemented")
}
