// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package boot

import (
	"context"
)

// Populate a blank database with arvados tables and seed rows.
type seedDatabase struct{}

func (seedDatabase) String() string {
	return "seedDatabase"
}

func (seedDatabase) Run(ctx context.Context, fail func(error), super *Supervisor) error {
	err := super.wait(ctx, runPostgreSQL{}, installPassenger{src: "services/api"})
	if err != nil {
		return err
	}
	if super.ClusterType == "production" {
		return nil
	}
	err = super.RunProgram(ctx, "services/api", runOptions{env: railsEnv}, "bundle", "exec", "rake", "db:setup")
	if err != nil {
		return err
	}
	err = super.RunProgram(ctx, "services/api", runOptions{env: railsEnv}, "bundle", "exec", "./script/get_anonymous_user_token.rb")
	if err != nil {
		return err
	}
	return nil
}
