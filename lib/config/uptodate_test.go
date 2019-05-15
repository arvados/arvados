// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package config

import (
	"bytes"
	"io/ioutil"
	"testing"
)

func TestUpToDate(t *testing.T) {
	src := "config.default.yml"
	srcdata, err := ioutil.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(srcdata, DefaultYAML) {
		t.Fatalf("content of %s differs from DefaultYAML -- you need to run 'go generate' and commit", src)
	}
}
