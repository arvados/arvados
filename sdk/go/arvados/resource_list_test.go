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
