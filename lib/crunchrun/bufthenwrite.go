// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"bytes"
	"io"
	"sync"
)

type bufThenWrite struct {
	buf bytes.Buffer
	w   io.Writer
	mtx sync.Mutex
}

func (btw *bufThenWrite) SetWriter(w io.Writer) error {
	btw.mtx.Lock()
	defer btw.mtx.Unlock()
	btw.w = w
	_, err := io.Copy(w, &btw.buf)
	return err
}

func (btw *bufThenWrite) Write(p []byte) (int, error) {
	btw.mtx.Lock()
	defer btw.mtx.Unlock()
	if btw.w == nil {
		btw.w = &btw.buf
	}
	return btw.w.Write(p)
}
