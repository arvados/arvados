// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
)

type Vocabulary struct {
	StrictTags bool                     `json:"strict_tags"`
	Tags       map[string]VocabularyTag `json:"tags"`
}

type VocabularyTag struct {
	Strict bool                          `json:"strict"`
	Labels []VocabularyLabel             `json:"labels"`
	Values map[string]VocabularyTagValue `json:"values"`
}

type VocabularyLabel struct {
	Label string `json:"label"`
}

type VocabularyTagValue struct {
	Labels []VocabularyLabel `json:"labels"`
}

func NewVocabulary(data []byte) (voc *Vocabulary, err error) {
	if r := bytes.Compare(data, []byte("")); r == 0 {
		return &Vocabulary{}, nil
	}
	err = json.Unmarshal(data, &voc)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON format error: %q", err)
	}
	if reflect.DeepEqual(voc, &Vocabulary{}) {
		return nil, fmt.Errorf("JSON data provided doesn't match Vocabulary format: %q", data)
	}
	err = voc.Validate()
	if err != nil {
		return nil, err
	}
	return voc, nil
}

func (v *Vocabulary) Validate() error {
	tagKeys := map[string]bool{}
	// Checks for Vocabulary strictness
	if v.StrictTags && len(v.Tags) == 0 {
		return fmt.Errorf("vocabulary is strict but no tags are defined")
	}
	// Checks for duplicate tag keys
	for key := range v.Tags {
		if tagKeys[key] {
			return fmt.Errorf("duplicate tag key %q", key)
		}
		tagKeys[key] = true
		for _, lbl := range v.Tags[key].Labels {
			if tagKeys[lbl.Label] {
				return fmt.Errorf("tag label %q for key %q already seen as a tag key or label", lbl.Label, key)
			}
			tagKeys[lbl.Label] = true
		}
		// Checks for value strictness
		if v.Tags[key].Strict && len(v.Tags[key].Values) == 0 {
			return fmt.Errorf("tag key %q is configured as strict but doesn't provide values", key)
		}
		// Checks for value duplication within a key
		tagValues := map[string]bool{}
		for val := range v.Tags[key].Values {
			if tagValues[val] {
				return fmt.Errorf("duplicate tag value %q for tag %q", val, key)
			}
			tagValues[val] = true
			for _, tagLbl := range v.Tags[key].Values[val].Labels {
				if tagValues[tagLbl.Label] {
					return fmt.Errorf("tag value label %q for value %q[%q] already seen as a value key or label", tagLbl.Label, key, val)
				}
				tagValues[tagLbl.Label] = true
			}
		}
	}
	return nil
}
