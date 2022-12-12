// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package boot

import (
	"context"
	"fmt"
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

// Run checks for and applies any pending Rails database migrations.
//
// If running a dev/test environment, and the database is empty, it
// initializes the database.
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

	// Check for pending migrations before running rake.
	//
	// In principle, we could use "rake db:migrate:status" or skip
	// this check entirely and let "rake db:migrate" be a no-op
	// most of the time.  However, in the most common case when
	// there are no new migrations, that would add ~2s to startup
	// time / downtime during service restart.

	todo, err := migrationList(appdir)
	if err != nil {
		return err
	}

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

func migrationList(dir string) (map[string]bool, error) {
	todo := map[string]bool{}

	// list versions in db/migrate/{version}_{name}.rb
	err := fs.WalkDir(os.DirFS(dir), "db/migrate", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		fnm := d.Name()
		if !strings.HasSuffix(fnm, ".rb") {
			return fmt.Errorf("unexpected file in db/migrate dir: %s", fnm)
		}
		for i, c := range fnm {
			if i > 0 && c == '_' {
				todo[fnm[:i]] = true
				break
			}
			if c < '0' || c > '9' {
				// non-numeric character before the
				// first '_' means this is not a
				// migration
				return fmt.Errorf("unexpected file in db/migrate dir: %s", fnm)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return todo, nil
}
