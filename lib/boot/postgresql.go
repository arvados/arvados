// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package boot

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/lib/pq"
)

func runPostgres(ctx context.Context, boot *Booter, ready chan<- bool) error {
	buf := bytes.NewBuffer(nil)
	err := boot.RunProgram(ctx, boot.tempdir, buf, nil, "pg_config", "--bindir")
	if err != nil {
		return err
	}
	datadir := filepath.Join(boot.tempdir, "pgdata")

	err = os.Mkdir(datadir, 0755)
	if err != nil {
		return err
	}
	bindir := strings.TrimSpace(buf.String())

	err = boot.RunProgram(ctx, boot.tempdir, nil, nil, filepath.Join(bindir, "initdb"), "-D", datadir)
	if err != nil {
		return err
	}

	err = boot.RunProgram(ctx, boot.tempdir, nil, nil, "cp", "server.crt", "server.key", datadir)
	if err != nil {
		return err
	}

	port := boot.cluster.PostgreSQL.Connection["port"]

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		for {
			if ctx.Err() != nil {
				return
			}
			if exec.CommandContext(ctx, "pg_isready", "--timeout=10", "--host="+boot.cluster.PostgreSQL.Connection["host"], "--port="+port).Run() == nil {
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
			boot.logger.WithError(err).Error("db open failed")
			cancel()
			return
		}
		defer db.Close()
		conn, err := db.Conn(ctx)
		if err != nil {
			boot.logger.WithError(err).Error("db conn failed")
			cancel()
			return
		}
		defer conn.Close()
		_, err = conn.ExecContext(ctx, `CREATE USER `+pq.QuoteIdentifier(boot.cluster.PostgreSQL.Connection["user"])+` WITH SUPERUSER ENCRYPTED PASSWORD `+pq.QuoteLiteral(boot.cluster.PostgreSQL.Connection["password"]))
		if err != nil {
			boot.logger.WithError(err).Error("createuser failed")
			cancel()
			return
		}
		_, err = conn.ExecContext(ctx, `CREATE DATABASE `+pq.QuoteIdentifier(boot.cluster.PostgreSQL.Connection["dbname"]))
		if err != nil {
			boot.logger.WithError(err).Error("createdb failed")
			cancel()
			return
		}
		close(ready)
		return
	}()

	return boot.RunProgram(ctx, boot.tempdir, nil, nil, filepath.Join(bindir, "postgres"),
		"-l",          // enable ssl
		"-D", datadir, // data dir
		"-k", datadir, // socket dir
		"-p", boot.cluster.PostgreSQL.Connection["port"],
	)
}
