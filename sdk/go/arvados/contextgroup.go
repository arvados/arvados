package arvados

import (
	"context"
	"sync"
)

// A contextGroup is a context-aware variation on sync.WaitGroup. It
// provides a child context for the added funcs to use, so they can
// exit early if another added func returns an error. Its Wait()
// method returns the first error returned by any added func.
//
// Example:
//
//	err := errors.New("oops")
//	cg := newContextGroup()
//	defer cg.Cancel()
//	cg.Go(func() error {
//		someFuncWithContext(cg.Context())
//		return nil
//	})
//	cg.Go(func() error {
//		return err // this cancels cg.Context()
//	})
//	return cg.Wait() // returns err after both goroutines have ended
type contextGroup struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	err    error
	mtx    sync.Mutex
}

// newContextGroup returns a new contextGroup. The caller must
// eventually call the Cancel() method of the returned contextGroup.
func newContextGroup(ctx context.Context) *contextGroup {
	ctx, cancel := context.WithCancel(ctx)
	return &contextGroup{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Cancel cancels the context group.
func (cg *contextGroup) Cancel() {
	cg.cancel()
}

// Context returns a context.Context which will be canceled when all
// funcs have succeeded or one has failed.
func (cg *contextGroup) Context() context.Context {
	return cg.ctx
}

// Go calls f in a new goroutine. If f returns an error, the
// contextGroup is canceled.
//
// If f notices cg.Context() is done, it should abandon further work
// and return. In this case, f's return value will be ignored.
func (cg *contextGroup) Go(f func() error) {
	cg.mtx.Lock()
	defer cg.mtx.Unlock()
	if cg.err != nil {
		return
	}
	cg.wg.Add(1)
	go func() {
		defer cg.wg.Done()
		err := f()
		cg.mtx.Lock()
		defer cg.mtx.Unlock()
		if err != nil && cg.err == nil {
			cg.err = err
			cg.cancel()
		}
	}()
}

// Wait waits for all added funcs to return, and returns the first
// non-nil error.
//
// If the parent context is canceled before a func returns an error,
// Wait returns the parent context's Err().
//
// Wait returns nil if all funcs return nil before the parent context
// is canceled.
func (cg *contextGroup) Wait() error {
	cg.wg.Wait()
	cg.mtx.Lock()
	defer cg.mtx.Unlock()
	if cg.err != nil {
		return cg.err
	}
	return cg.ctx.Err()
}
