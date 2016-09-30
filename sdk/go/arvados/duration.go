package arvados

import (
	"encoding/json"
	"fmt"
	"time"
)

// Duration is time.Duration but looks like "12s" in JSON, rather than
// a number of nanoseconds.
type Duration time.Duration

// UnmarshalJSON implements json.Unmarshaler
func (d *Duration) UnmarshalJSON(data []byte) error {
	if data[0] == '"' {
		return d.Set(string(data[1 : len(data)-1]))
	}
	return fmt.Errorf("duration must be given as a string like \"600s\" or \"1h30m\"")
}

// MarshalJSON implements json.Marshaler
func (d *Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// String implements fmt.Stringer
func (d Duration) String() string {
	return time.Duration(d).String()
}

// Duration returns a time.Duration
func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

// Value implements flag.Value
func (d *Duration) Set(s string) error {
	dur, err := time.ParseDuration(s)
	*d = Duration(dur)
	return err
}
