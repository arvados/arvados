// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package boot

import (
	"bytes"
	"io"
)

type logPrefixer struct {
	io.Writer
	Prefix []byte
	did    bool
}

func (lp *logPrefixer) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if !lp.did {
		lp.Writer.Write(lp.Prefix)
		lp.did = p[len(p)-1] != '\n'
	}
	out := append(bytes.Replace(p[:len(p)-1], []byte("\n"), append([]byte("\n"), lp.Prefix...), -1), p[len(p)-1])
	_, err := lp.Writer.Write(out)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}
