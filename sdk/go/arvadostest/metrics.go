// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvadostest

import (
	"bytes"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

func GatherMetricsAsString(reg *prometheus.Registry) string {
	buf := bytes.NewBuffer(nil)
	enc := expfmt.NewEncoder(buf, expfmt.NewFormat(expfmt.TypeTextPlain))
	got, _ := reg.Gather()
	for _, mf := range got {
		enc.Encode(mf)
	}
	return buf.String()
}
