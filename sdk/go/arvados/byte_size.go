// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ByteSize int64

var prefixValue = map[string]int64{
	"":   1,
	"K":  1000,
	"Ki": 1 << 10,
	"M":  1000000,
	"Mi": 1 << 20,
	"G":  1000000000,
	"Gi": 1 << 30,
	"T":  1000000000000,
	"Ti": 1 << 40,
	"P":  1000000000000000,
	"Pi": 1 << 50,
	"E":  1000000000000000000,
	"Ei": 1 << 60,
}

func (n *ByteSize) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || data[0] != '"' {
		var i int64
		err := json.Unmarshal(data, &i)
		if err != nil {
			return err
		}
		*n = ByteSize(i)
		return nil
	}
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	split := strings.LastIndexAny(s, "0123456789") + 1
	if split == 0 {
		return fmt.Errorf("invalid byte size %q", s)
	}
	var val int64
	err = json.Unmarshal([]byte(s[:split]), &val)
	if err != nil {
		return err
	}
	if split == len(s) {
		return nil
	}
	prefix := strings.Trim(s[split:], " ")
	if strings.HasSuffix(prefix, "B") {
		prefix = prefix[:len(prefix)-1]
	}
	pval, ok := prefixValue[prefix]
	if !ok {
		return fmt.Errorf("invalid unit %q", strings.Trim(s[split:], " "))
	}
	if pval > 1 && (val*pval)/pval != val {
		return fmt.Errorf("size %q overflows int64", s)
	}
	*n = ByteSize(val * pval)
	return nil
}
