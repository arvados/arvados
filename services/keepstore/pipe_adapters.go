// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
)

// getWithPipe invokes getter and copies the resulting data into
// buf. If ctx is done before all data is copied, getWithPipe closes
// the pipe with an error, and returns early with an error.
func getWithPipe(ctx context.Context, loc string, buf []byte, br BlockReader) (int, error) {
	piper, pipew := io.Pipe()
	go func() {
		pipew.CloseWithError(br.ReadBlock(ctx, loc, pipew))
	}()
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

// putWithPipe invokes putter with a new pipe, and copies data
// from buf into the pipe. If ctx is done before all data is copied,
// putWithPipe closes the pipe with an error, and returns early with
// an error.
func putWithPipe(ctx context.Context, loc string, buf []byte, bw BlockWriter) error {
	piper, pipew := io.Pipe()
	copyErr := make(chan error)
	go func() {
		_, err := io.Copy(pipew, bytes.NewReader(buf))
		copyErr <- err
		close(copyErr)
	}()

	putErr := make(chan error, 1)
	go func() {
		putErr <- bw.WriteBlock(ctx, loc, piper)
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
	// return). This can cause pipew to receive corrupt data if
	// err came from copyErr or ctx.Done() before the copy
	// finished. That's OK, though: in that case err != nil, and
	// CloseWithErr(err) ensures putter() will get an error from
	// piper.Read() before seeing EOF.
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
