// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package selfsigned

import (
	"testing"
)

func TestCert(t *testing.T) {
	cert, err := CertGenerator{Bits: 1024, Hosts: []string{"localhost"}, IsCA: false}.Generate()
	if err != nil {
		t.Error(err)
	}
	if len(cert.Certificate) < 1 {
		t.Error("no certificate!")
	}
	cert, err = CertGenerator{Bits: 2048, Hosts: []string{"localhost"}, IsCA: true}.Generate()
	if err != nil {
		t.Error(err)
	}
	if len(cert.Certificate) < 1 {
		t.Error("no certificate!")
	}
}
