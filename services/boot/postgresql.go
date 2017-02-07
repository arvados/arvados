package main

import (
	"context"
	"time"
)

var postgresql = &pgBooter{}

type pgBooter struct {}

func (pb *pgBooter) Boot(ctx context.Context) error {
	// TODO: return nil if this isn't the database host.
	if pb.check(ctx) == nil {
		return nil
	}
	if err := (&osPackage{
		Debian: "postgresql",
	}).Boot(ctx); err != nil {
		return err
	}
	if err := command("service", "postgresql", "start").Run(); err != nil {
		return err
	}
	return waitCheck(ctx, 30*time.Second, pb.check)
}

func (pb *pgBooter) check(ctx context.Context) error {
	return command("pg_isready").Run()
}
