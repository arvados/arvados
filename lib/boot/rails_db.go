// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package boot

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"git.arvados.org/arvados.git/lib/controller/dblock"
	"git.arvados.org/arvados.git/lib/ctrlctx"
)

type railsDatabase struct{}

func (runner railsDatabase) String() string {
	return "railsDatabase"
}

func (runner railsDatabase) Run(ctx context.Context, fail func(error), super *Supervisor) error {
	err := super.wait(ctx, runPostgreSQL{}, installPassenger{src: "services/api"})
	if err != nil {
		return err
	}

	// determine path to installed rails app or source tree
	var appdir string
	if super.ClusterType == "production" {
		appdir = "/var/lib/arvados/railsapi"
	} else {
		appdir = filepath.Join(super.SourcePath, "services/api")
	}

	// list versions in db/migrate/{version}_{name}.rb
	todo := map[string]bool{}
	fs.WalkDir(os.DirFS(appdir), "db/migrate", func(path string, d fs.DirEntry, err error) error {
		if cut := strings.Index(d.Name(), "_"); cut > 0 && strings.HasSuffix(d.Name(), ".rb") {
			todo[d.Name()[:cut]] = true
		}
		return nil
	})

	// read schema_migrations table (list of migrations already
	// applied) and remove those entries from todo
	dbconnector := ctrlctx.DBConnector{PostgreSQL: super.cluster.PostgreSQL}
	defer dbconnector.Close()
	db, err := dbconnector.GetDB(ctx)
	if err != nil {
		return err
	}
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		if super.ClusterType == "production" {
			return err
		}
		super.logger.WithError(err).Info("schema_migrations query failed, trying db:setup")
		return super.RunProgram(ctx, "services/api", runOptions{env: railsEnv}, "bundle", "exec", "rake", "db:setup")
	}
	for rows.Next() {
		var v string
		err = rows.Scan(&v)
		if err != nil {
			return err
		}
		delete(todo, v)
	}
	err = rows.Close()
	if err != nil {
		return err
	}

	// if nothing remains in todo, all available migrations are
	// done, so return without running any [relatively slow]
	// ruby/rake commands
	if len(todo) == 0 {
		return nil
	}

	super.logger.Infof("%d migrations pending", len(todo))
	if !dblock.RailsMigrations.Lock(ctx, dbconnector.GetDB) {
		return context.Canceled
	}
	defer dblock.RailsMigrations.Unlock()
	return super.RunProgram(ctx, appdir, runOptions{env: railsEnv}, "bundle", "exec", "rake", "db:migrate")
}
