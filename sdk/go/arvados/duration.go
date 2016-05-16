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
		dur, err := time.ParseDuration(string(data[1 : len(data)-1]))
		*d = Duration(dur)
		return err
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
