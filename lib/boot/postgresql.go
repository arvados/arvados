// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package boot

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/lib/pq"
)

// Run a postgresql server in a private data directory. Set up a db
// user, database, and TCP listener that match the supervisor's
// configured database connection info.
type runPostgreSQL struct{}

func (runPostgreSQL) String() string {
	return "postgresql"
}

func (runPostgreSQL) Run(ctx context.Context, fail func(error), super *Supervisor) error {
	err := super.wait(ctx, createCertificates{})
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(nil)
	err = super.RunProgram(ctx, super.tempdir, buf, nil, "pg_config", "--bindir")
	if err != nil {
		return err
	}
	bindir := strings.TrimSpace(buf.String())

	datadir := filepath.Join(super.tempdir, "pgdata")
	err = os.Mkdir(datadir, 0755)
	if err != nil {
		return err
	}
	err = super.RunProgram(ctx, super.tempdir, nil, nil, filepath.Join(bindir, "initdb"), "-D", datadir)
	if err != nil {
		return err
	}

	err = super.RunProgram(ctx, super.tempdir, nil, nil, "cp", "server.crt", "server.key", datadir)
	if err != nil {
		return err
	}

	port := super.cluster.PostgreSQL.Connection["port"]

	super.waitShutdown.Add(1)
	go func() {
		defer super.waitShutdown.Done()
		fail(super.RunProgram(ctx, super.tempdir, nil, nil, filepath.Join(bindir, "postgres"),
			"-l",          // enable ssl
			"-D", datadir, // data dir
			"-k", datadir, // socket dir
			"-p", super.cluster.PostgreSQL.Connection["port"],
		))
	}()

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if exec.CommandContext(ctx, "pg_isready", "--timeout=10", "--host="+super.cluster.PostgreSQL.Connection["host"], "--port="+port).Run() == nil {
			break
		}
		time.Sleep(time.Second / 2)
	}
	db, err := sql.Open("postgres", arvados.PostgreSQLConnection{
		"host":   datadir,
		"port":   port,
		"dbname": "postgres",
	}.String())
	if err != nil {
		return fmt.Errorf("db open failed: %s", err)
	}
	defer db.Close()
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("db conn failed: %s", err)
	}
	defer conn.Close()
	_, err = conn.ExecContext(ctx, `CREATE USER `+pq.QuoteIdentifier(super.cluster.PostgreSQL.Connection["user"])+` WITH SUPERUSER ENCRYPTED PASSWORD `+pq.QuoteLiteral(super.cluster.PostgreSQL.Connection["password"]))
	if err != nil {
		return fmt.Errorf("createuser failed: %s", err)
	}
	_, err = conn.ExecContext(ctx, `CREATE DATABASE `+pq.QuoteIdentifier(super.cluster.PostgreSQL.Connection["dbname"]))
	if err != nil {
		return fmt.Errorf("createdb failed: %s", err)
	}
	return nil
}
