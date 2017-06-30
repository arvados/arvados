// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package stats

import (
	"fmt"
	"strconv"
	"time"
)

// Duration is a duration that is displayed as a number of seconds in
// fixed-point notation.
type Duration time.Duration

// MarshalJSON implements json.Marshaler.
func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(d.String()), nil
}

// String implements fmt.Stringer.
func (d Duration) String() string {
	return fmt.Sprintf("%.6f", time.Duration(d).Seconds())
}

// UnmarshalJSON implements json.Unmarshaler
func (d *Duration) UnmarshalJSON(data []byte) error {
	return d.Set(string(data))
}

// Value implements flag.Value
func (d *Duration) Set(s string) error {
	sec, err := strconv.ParseFloat(s, 64)
	if err == nil {
		*d = Duration(sec * float64(time.Second))
	}
	return err
}
