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
	"os/user"
	"path/filepath"
	"strconv"
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

	if super.ClusterType == "production" {
		return nil
	}

	iamroot := false
	if u, err := user.Current(); err != nil {
		return fmt.Errorf("user.Current(): %w", err)
	} else if u.Uid == "0" {
		iamroot = true
	}

	buf := bytes.NewBuffer(nil)
	err = super.RunProgram(ctx, super.tempdir, runOptions{output: buf}, "pg_config", "--bindir")
	if err != nil {
		return err
	}
	bindir := strings.TrimSpace(buf.String())

	datadir := filepath.Join(super.tempdir, "pgdata")
	err = os.Mkdir(datadir, 0700)
	if err != nil {
		return err
	}
	prog, args := filepath.Join(bindir, "initdb"), []string{"-D", datadir, "-E", "utf8"}
	opts := runOptions{}
	if iamroot {
		postgresUser, err := user.Lookup("postgres")
		if err != nil {
			return fmt.Errorf("user.Lookup(\"postgres\"): %s", err)
		}
		postgresUID, err := strconv.Atoi(postgresUser.Uid)
		if err != nil {
			return fmt.Errorf("user.Lookup(\"postgres\"): non-numeric uid?: %q", postgresUser.Uid)
		}
		postgresGid, err := strconv.Atoi(postgresUser.Gid)
		if err != nil {
			return fmt.Errorf("user.Lookup(\"postgres\"): non-numeric gid?: %q", postgresUser.Gid)
		}
		err = os.Chown(super.tempdir, 0, postgresGid)
		if err != nil {
			return err
		}
		err = os.Chmod(super.tempdir, 0710)
		if err != nil {
			return err
		}
		err = os.Chown(datadir, postgresUID, 0)
		if err != nil {
			return err
		}
		opts.user = "postgres"
	}
	err = super.RunProgram(ctx, super.tempdir, opts, prog, args...)
	if err != nil {
		return err
	}

	err = super.RunProgram(ctx, super.tempdir, runOptions{}, "cp", "server.crt", "server.key", datadir)
	if err != nil {
		return err
	}
	if iamroot {
		err = super.RunProgram(ctx, super.tempdir, runOptions{}, "chown", "postgres", datadir+"/server.crt", datadir+"/server.key")
		if err != nil {
			return err
		}
	}

	port := super.cluster.PostgreSQL.Connection["port"]

	super.waitShutdown.Add(1)
	go func() {
		defer super.waitShutdown.Done()
		prog, args := filepath.Join(bindir, "postgres"), []string{
			"-l",          // enable ssl
			"-D", datadir, // data dir
			"-k", datadir, // socket dir
			"-h", super.cluster.PostgreSQL.Connection["host"],
			"-p", super.cluster.PostgreSQL.Connection["port"],
		}
		opts := runOptions{}
		if iamroot {
			opts.user = "postgres"
		}
		fail(super.RunProgram(ctx, super.tempdir, opts, prog, args...))
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
	pgconn := arvados.PostgreSQLConnection{
		"host":   datadir,
		"port":   port,
		"dbname": "postgres",
	}
	if iamroot {
		pgconn["user"] = "postgres"
	}
	db, err := sql.Open("postgres", pgconn.String())
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
	_, err = conn.ExecContext(ctx, `CREATE DATABASE `+pq.QuoteIdentifier(super.cluster.PostgreSQL.Connection["dbname"])+` WITH TEMPLATE template0 ENCODING 'utf8'`)
	if err != nil {
		return fmt.Errorf("createdb failed: %s", err)
	}
	return nil
}
