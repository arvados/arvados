// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package stats

import (
	"testing"
	"time"
)

func TestString(t *testing.T) {
	d := Duration(123123123123 * time.Nanosecond)
	if s, expect := d.String(), "123.123123"; s != expect {
		t.Errorf("got %s, expect %s", s, expect)
	}
}

func TestSet(t *testing.T) {
	var d Duration
	if err := d.Set("123.456"); err != nil {
		t.Fatal(err)
	}
	if got, expect := time.Duration(d).Nanoseconds(), int64(123456000000); got != expect {
		t.Errorf("got %d, expect %d", got, expect)
	}
}
