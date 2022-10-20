// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Duration is time.Duration but looks like "12s" in JSON, rather than
// a number of nanoseconds.
type Duration time.Duration

// UnmarshalJSON implements json.Unmarshaler.
func (d *Duration) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte(`"0"`)) || bytes.Equal(data, []byte(`0`)) {
		// Unitless 0 is not accepted by ParseDuration, but we
		// accept it as a reasonable spelling of 0
		// nanoseconds.
		*d = 0
		return nil
	}
	if data[0] == '"' {
		return d.Set(string(data[1 : len(data)-1]))
	}
	// Mimic error message returned by ParseDuration for a number
	// without units.
	return fmt.Errorf("missing unit in duration %q", data)
}

// MarshalJSON implements json.Marshaler.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// String returns a format similar to (time.Duration)String() but with
// "0m" and "0s" removed: e.g., "1h" instead of "1h0m0s".
func (d Duration) String() string {
	s := time.Duration(d).String()
	s = strings.Replace(s, "m0s", "m", 1)
	s = strings.Replace(s, "h0m", "h", 1)
	return s
}

// Duration returns a time.Duration.
func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

// Set implements the flag.Value interface and sets the duration value by using time.ParseDuration to parse the string.
func (d *Duration) Set(s string) error {
	dur, err := time.ParseDuration(s)
	*d = Duration(dur)
	return err
}
