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
	var out []byte
	if !lp.did {
		out = append(out, lp.Prefix...)
	}
	lp.did = p[len(p)-1] != '\n'
	out = append(out, bytes.Replace(p[:len(p)-1], []byte("\n"), append([]byte("\n"), lp.Prefix...), -1)...)
	out = append(out, p[len(p)-1])
	_, err := lp.Writer.Write(out)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}
