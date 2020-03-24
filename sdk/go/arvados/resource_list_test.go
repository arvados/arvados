// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestMarshalFiltersWithNanoseconds(t *testing.T) {
	t0 := time.Now()
	t0str := t0.Format(time.RFC3339Nano)
	buf, err := json.Marshal([]Filter{
		{Attr: "modified_at", Operator: "=", Operand: t0}})
	if err != nil {
		t.Fatal(err)
	}
	if expect := []byte(`[["modified_at","=","` + t0str + `"]]`); 0 != bytes.Compare(buf, expect) {
		t.Errorf("Encoded as %q, expected %q", buf, expect)
	}
}

func TestMarshalFiltersWithNil(t *testing.T) {
	buf, err := json.Marshal([]Filter{
		{Attr: "modified_at", Operator: "=", Operand: nil}})
	if err != nil {
		t.Fatal(err)
	}
	if expect := []byte(`[["modified_at","=",null]]`); 0 != bytes.Compare(buf, expect) {
		t.Errorf("Encoded as %q, expected %q", buf, expect)
	}
}

func TestUnmarshalFiltersWithNil(t *testing.T) {
	buf := []byte(`["modified_at","=",null]`)
	f := &Filter{}
	err := f.UnmarshalJSON(buf)
	if err != nil {
		t.Fatal(err)
	}
	expect := Filter{Attr: "modified_at", Operator: "=", Operand: nil}
	if f.Attr != expect.Attr || f.Operator != expect.Operator || f.Operand != expect.Operand {
		t.Errorf("Decoded as %q, expected %q", f, expect)
	}
}

func TestMarshalFiltersWithBoolean(t *testing.T) {
	buf, err := json.Marshal([]Filter{
		{Attr: "is_active", Operator: "=", Operand: true}})
	if err != nil {
		t.Fatal(err)
	}
	if expect := []byte(`[["is_active","=",true]]`); 0 != bytes.Compare(buf, expect) {
		t.Errorf("Encoded as %q, expected %q", buf, expect)
	}
}

func TestUnmarshalFiltersWithBoolean(t *testing.T) {
	buf := []byte(`["is_active","=",true]`)
	f := &Filter{}
	err := f.UnmarshalJSON(buf)
	if err != nil {
		t.Fatal(err)
	}
	expect := Filter{Attr: "is_active", Operator: "=", Operand: true}
	if f.Attr != expect.Attr || f.Operator != expect.Operator || f.Operand != expect.Operand {
		t.Errorf("Decoded as %q, expected %q", f, expect)
	}
}
