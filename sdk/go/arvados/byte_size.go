// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type ByteSize int64

// ByteSizeOrPercent indicates either a number of bytes or a
// percentage from 1 to 100.
type ByteSizeOrPercent ByteSize

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
	split := strings.LastIndexAny(s, "0123456789.+-eE") + 1
	if split == 0 {
		return fmt.Errorf("invalid byte size %q", s)
	}
	if s[split-1] == 'E' {
		// We accepted an E as if it started the exponent part
		// of a json number, but if the next char isn't +, -,
		// or digit, then the E must have meant Exa. Instead
		// of "4.5E"+"iB" we want "4.5"+"EiB".
		split--
	}
	var val json.Number
	dec := json.NewDecoder(strings.NewReader(s[:split]))
	dec.UseNumber()
	err = dec.Decode(&val)
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
	if intval, err := val.Int64(); err == nil {
		if pval > 1 && (intval*pval)/pval != intval {
			return fmt.Errorf("size %q overflows int64", s)
		}
		*n = ByteSize(intval * pval)
		return nil
	} else if floatval, err := val.Float64(); err == nil {
		if floatval*float64(pval) > math.MaxInt64 {
			return fmt.Errorf("size %q overflows int64", s)
		}
		*n = ByteSize(int64(floatval * float64(pval)))
		return nil
	} else {
		return fmt.Errorf("bug: json.Number for %q is not int64 or float64: %s", s, err)
	}
}

func (n ByteSizeOrPercent) MarshalJSON() ([]byte, error) {
	if n < 0 && n >= -100 {
		return []byte(fmt.Sprintf("\"%d%%\"", -n)), nil
	} else {
		return json.Marshal(int64(n))
	}
}

func (n *ByteSizeOrPercent) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || data[0] != '"' {
		return (*ByteSize)(n).UnmarshalJSON(data)
	}
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	if s := strings.TrimSpace(s); len(s) > 0 && s[len(s)-1] == '%' {
		pct, err := strconv.ParseInt(strings.TrimSpace(s[:len(s)-1]), 10, 64)
		if err != nil {
			return err
		}
		if pct < 0 || pct > 100 {
			return fmt.Errorf("invalid value %q (percentage must be between 0 and 100)", s)
		}
		*n = ByteSizeOrPercent(-pct)
		return nil
	}
	return (*ByteSize)(n).UnmarshalJSON(data)
}

// ByteSize returns the absolute byte size specified by n, or 0 if n
// specifies a percent.
func (n ByteSizeOrPercent) ByteSize() ByteSize {
	if n >= -100 && n < 0 {
		return 0
	} else {
		return ByteSize(n)
	}
}

// ByteSize returns the percentage specified by n, or 0 if n specifies
// an absolute byte size.
func (n ByteSizeOrPercent) Percent() int64 {
	if n >= -100 && n < 0 {
		return int64(-n)
	} else {
		return 0
	}
}
