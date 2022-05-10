// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchslurm

import (
	"strings"
)

func execScript(args []string) string {
	s := "#!/bin/sh\nexec"
	for _, w := range args {
		s += ` '`
		s += strings.Replace(w, `'`, `'\''`, -1)
		s += `'`
	}
	return s + "\n"
}
