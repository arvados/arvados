// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package boot

import (
	"context"
)

type seedDatabase struct{}

func (seedDatabase) String() string {
	return "seedDatabase"
}

func (seedDatabase) Run(ctx context.Context, fail func(error), boot *Booter) error {
	err := boot.wait(ctx, runPostgreSQL{})
	if err != nil {
		return err
	}
	err = boot.RunProgram(ctx, "services/api", nil, nil, "bundle", "exec", "rake", "db:setup")
	if err != nil {
		return err
	}
	return nil
}
