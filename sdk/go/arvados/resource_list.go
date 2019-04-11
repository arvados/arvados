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
	var elements []interface{}
	err := json.Unmarshal(data, &elements)
	if err != nil {
		return err
	}
	if len(elements) != 3 {
		return fmt.Errorf("invalid filter %q: must have 3 elements", data)
	}
	attr, ok := elements[0].(string)
	if !ok {
		return fmt.Errorf("invalid filter attr %q", elements[0])
	}
	op, ok := elements[1].(string)
	if !ok {
		return fmt.Errorf("invalid filter operator %q", elements[1])
	}
	operand := elements[2]
	switch operand.(type) {
	case string, float64, []interface{}:
	default:
		return fmt.Errorf("invalid filter operand %q", elements[2])
	}
	*f = Filter{attr, op, operand}
	return nil
}
