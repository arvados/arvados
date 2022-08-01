// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchslurm

import (
	"strings"
)

func execScript(args []string, env map[string]string) string {
	s := "#!/bin/sh\n"
	for k, v := range env {
		s += k + `='`
		s += strings.Replace(v, `'`, `'\''`, -1)
		s += `' `
	}
	s += `exec`
	for _, w := range args {
		s += ` '`
		s += strings.Replace(w, `'`, `'\''`, -1)
		s += `'`
	}
	return s + "\n"
}
