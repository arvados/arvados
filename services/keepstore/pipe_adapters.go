package main

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"sync"
)

// getWithPipe invokes getter and copies the resulting data into
// buf. If ctx is done before all data is copied, getWithPipe closes
// the pipe with an error, and returns early with an error.
func getWithPipe(ctx context.Context, loc string, buf []byte, getter func(context.Context, string, *io.PipeWriter)) (int, error) {
	piper, pipew := io.Pipe()
	go getter(ctx, loc, pipew)
	done := make(chan struct{})
	var size int
	var err error
	go func() {
		size, err = io.ReadFull(piper, buf)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			err = nil
		}
		close(done)
	}()
	select {
	case <-ctx.Done():
		piper.CloseWithError(ctx.Err())
		return 0, ctx.Err()
	case <-done:
		piper.Close()
		return size, err
	}
}

type errorReadCloser struct {
	*io.PipeReader
	err error
	mtx sync.Mutex
}

func (erc *errorReadCloser) Close() error {
	erc.mtx.Lock()
	defer erc.mtx.Unlock()
	erc.PipeReader.Close()
	return erc.err
}

func (erc *errorReadCloser) SetError(err error) {
	erc.mtx.Lock()
	defer erc.mtx.Unlock()
	erc.err = err
}

// putWithPipe invokes putter with a new pipe, and and copies data
// from buf into the pipe. If ctx is done before all data is copied,
// putWithPipe closes the pipe with an error, and returns early with
// an error.
func putWithPipe(ctx context.Context, loc string, buf []byte, putter func(context.Context, string, io.ReadCloser) error) error {
	piper, pipew := io.Pipe()
	copyErr := make(chan error)
	go func() {
		_, err := io.Copy(pipew, bytes.NewReader(buf))
		copyErr <- err
		close(copyErr)
	}()

	erc := errorReadCloser{
		PipeReader: piper,
		err:        nil,
	}
	putErr := make(chan error, 1)
	go func() {
		putErr <- putter(ctx, loc, &erc)
		close(putErr)
	}()

	var err error
	select {
	case err = <-copyErr:
	case err = <-putErr:
	case <-ctx.Done():
		err = ctx.Err()
	}

	// Ensure io.Copy goroutine isn't blocked writing to pipew
	// (otherwise, io.Copy is still using buf so it isn't safe to
	// return). This can cause pipew to receive corrupt data, so
	// we first ensure putter() will get an error when calling
	// erc.Close().
	erc.SetError(err)
	go pipew.CloseWithError(err)
	go io.Copy(ioutil.Discard, piper)
	<-copyErr

	// Note: io.Copy() is finished now, but putter() might still
	// be running. If we encounter an error before putter()
	// returns, we return right away without waiting for putter().

	if err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err = <-putErr:
		return err
	}
}
