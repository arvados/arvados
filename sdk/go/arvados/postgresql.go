// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import "strings"

func (c PostgreSQLConnection) String() string {
	s := ""
	for k, v := range c {
		if v == "" {
			continue
		}
		s += strings.ToLower(k)
		s += "='"
		s += strings.Replace(
			strings.Replace(v, `\`, `\\`, -1),
			`'`, `\'`, -1)
		s += "' "
	}
	return s
}
