// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"encoding/json"
	"fmt"
)

// ResourceListParams expresses which results are requested in a
// list/index API.
type ResourceListParams struct {
	Select             []string `json:"select,omitempty"`
	Filters            []Filter `json:"filters,omitempty"`
	IncludeTrash       bool     `json:"include_trash,omitempty"`
	IncludeOldVersions bool     `json:"include_old_versions,omitempty"`
	Limit              *int     `json:"limit,omitempty"`
	Offset             int      `json:"offset,omitempty"`
	Order              string   `json:"order,omitempty"`
	Distinct           bool     `json:"distinct,omitempty"`
	Count              string   `json:"count,omitempty"`
}

// A Filter restricts the set of records returned by a list/index API.
type Filter struct {
	Attr     string
	Operator string
	Operand  interface{}
}

// MarshalJSON encodes a Filter to a JSON array.
func (f *Filter) MarshalJSON() ([]byte, error) {
	return json.Marshal([]interface{}{f.Attr, f.Operator, f.Operand})
}

// UnmarshalJSON decodes a JSON array to a Filter.
func (f *Filter) UnmarshalJSON(data []byte) error {
	var decoded interface{}
	err := json.Unmarshal(data, &decoded)
	if err != nil {
		return err
	}
	switch decoded := decoded.(type) {
	case string:
		// Accept "(foo < bar)" as a more obvious way to spell
		// ["(foo < bar)","=",true]
		*f = Filter{decoded, "=", true}
	case []interface{}:
		if len(decoded) != 3 {
			return fmt.Errorf("invalid filter %q: must have 3 decoded", data)
		}
		attr, ok := decoded[0].(string)
		if !ok {
			return fmt.Errorf("invalid filter attr %q", decoded[0])
		}
		op, ok := decoded[1].(string)
		if !ok {
			return fmt.Errorf("invalid filter operator %q", decoded[1])
		}
		operand := decoded[2]
		switch operand.(type) {
		case string, float64, []interface{}, nil, bool:
		default:
			return fmt.Errorf("invalid filter operand %q", decoded[2])
		}
		*f = Filter{attr, op, operand}
	default:
		return fmt.Errorf("invalid filter: json decoded as %T instead of array or string", decoded)
	}
	return nil
}
