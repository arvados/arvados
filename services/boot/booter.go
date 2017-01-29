package main

import (
	"context"
	"fmt"
	"sync"
)

// A Booter ensures some piece of the system ("target") is correctly
// installed, configured, running, or working.
type Booter interface {
	// Inspect, repair, and report the current state of the target.
	Boot(context.Context) error
}

var cfgKey = &struct{}{}

func cfg(ctx context.Context) *Config {
	return ctx.Value(cfgKey).(*Config)
}

func withCfg(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, cfgKey, cfg)
}

type Series []Booter

func (sb Series) Boot(ctx context.Context) error {
	for _, b := range sb {
		err := b.Boot(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

type Concurrent []Booter

func (cb Concurrent) Boot(ctx context.Context) error {
	errs := make([]error, len(cb))
	var wg sync.WaitGroup
	wg.Add(len(cb))
	for i, b := range cb {
		i, b := i, b
		go func() {
			defer wg.Done()
			errs[i] = b.Boot(ctx)
		}()
	}
	wg.Wait()
	return NewMultipleError(errs)
}

type MultipleError struct {
	error
	errors []error
}

func NewMultipleError(errs []error) error {
	var errors []error
	for _, err := range errs {
		switch err := err.(type) {
		case *MultipleError:
			errors = append(errors, err.errors...)
		case nil:
		default:
			errors = append(errors, err)
		}
	}
	if len(errors) == 0 {
		return nil
	}
	if len(errors) == 1 {
		return errors[0]
	}
	return &MultipleError{
		error:  fmt.Errorf("%d errors %q", len(errors), errors),
		errors: errors,
	}
}
