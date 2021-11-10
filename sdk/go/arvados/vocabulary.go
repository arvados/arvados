// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

type Vocabulary struct {
	reservedTagKeys map[string]bool          `json:"-"`
	StrictTags      bool                     `json:"strict_tags"`
	Tags            map[string]VocabularyTag `json:"tags"`
}

type VocabularyTag struct {
	Strict bool                          `json:"strict"`
	Labels []VocabularyLabel             `json:"labels"`
	Values map[string]VocabularyTagValue `json:"values"`
}

// Cannot have a constant map in Go, so we have to use a function
func (v *Vocabulary) systemTagKeys() map[string]bool {
	return map[string]bool{
		"type":                  true,
		"template_uuid":         true,
		"groups":                true,
		"username":              true,
		"image_timestamp":       true,
		"docker-image-repo-tag": true,
		"filters":               true,
		"container_request":     true,
	}
}

type VocabularyLabel struct {
	Label string `json:"label"`
}

type VocabularyTagValue struct {
	Labels []VocabularyLabel `json:"labels"`
}

// NewVocabulary creates a new Vocabulary from a JSON definition and a list
// of reserved tag keys that will get special treatment when strict mode is
// enabled.
func NewVocabulary(data []byte, managedTagKeys []string) (voc *Vocabulary, err error) {
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
	voc.reservedTagKeys = make(map[string]bool)
	for _, managedKey := range managedTagKeys {
		voc.reservedTagKeys[managedKey] = true
	}
	for systemKey := range voc.systemTagKeys() {
		voc.reservedTagKeys[systemKey] = true
	}
	err = voc.validate()
	if err != nil {
		return nil, err
	}
	return voc, nil
}

func (v *Vocabulary) validate() error {
	if v == nil {
		return nil
	}
	tagKeys := map[string]string{}
	// Checks for Vocabulary strictness
	if v.StrictTags && len(v.Tags) == 0 {
		return fmt.Errorf("vocabulary is strict but no tags are defined")
	}
	// Checks for collisions between tag keys, reserved tag keys
	// and tag key labels.
	for key := range v.Tags {
		if v.reservedTagKeys[key] {
			return fmt.Errorf("tag key %q is reserved", key)
		}
		lcKey := strings.ToLower(key)
		if tagKeys[lcKey] != "" {
			return fmt.Errorf("duplicate tag key %q", key)
		}
		tagKeys[lcKey] = key
		for _, lbl := range v.Tags[key].Labels {
			label := strings.ToLower(lbl.Label)
			if tagKeys[label] != "" {
				return fmt.Errorf("tag label %q for key %q already seen as a tag key or label", lbl.Label, key)
			}
			tagKeys[label] = lbl.Label
		}
		// Checks for value strictness
		if v.Tags[key].Strict && len(v.Tags[key].Values) == 0 {
			return fmt.Errorf("tag key %q is configured as strict but doesn't provide values", key)
		}
		// Checks for collisions between tag values and tag value labels.
		tagValues := map[string]string{}
		for val := range v.Tags[key].Values {
			lcVal := strings.ToLower(val)
			if tagValues[lcVal] != "" {
				return fmt.Errorf("duplicate tag value %q for tag %q", val, key)
			}
			tagValues[lcVal] = val
			for _, tagLbl := range v.Tags[key].Values[val].Labels {
				label := strings.ToLower(tagLbl.Label)
				if tagValues[label] != "" {
					return fmt.Errorf("tag value label %q for pair (%q:%q) already seen as a value key or label", tagLbl.Label, key, val)
				}
				tagValues[label] = tagLbl.Label
			}
		}
	}
	return nil
}

func (v *Vocabulary) getLabelsToKeys() (labels map[string]string) {
	if v == nil {
		return
	}
	labels = make(map[string]string)
	for key, val := range v.Tags {
		for _, lbl := range val.Labels {
			label := strings.ToLower(lbl.Label)
			labels[label] = key
		}
	}
	return labels
}

func (v *Vocabulary) getLabelsToValues(key string) (labels map[string]string) {
	if v == nil {
		return
	}
	labels = make(map[string]string)
	if _, ok := v.Tags[key]; ok {
		for val := range v.Tags[key].Values {
			labels[strings.ToLower(val)] = val
			for _, tagLbl := range v.Tags[key].Values[val].Labels {
				label := strings.ToLower(tagLbl.Label)
				labels[label] = val
			}
		}
	}
	return labels
}

func (v *Vocabulary) checkValue(key, val string) error {
	if _, ok := v.Tags[key].Values[val]; !ok {
		lcVal := strings.ToLower(val)
		correctValue, ok := v.getLabelsToValues(key)[lcVal]
		if ok {
			return fmt.Errorf("tag value %q for key %q is an alias, must be provided as %q", val, key, correctValue)
		} else if v.Tags[key].Strict {
			return fmt.Errorf("tag value %q is not valid for key %q", val, key)
		}
	}
	return nil
}

// Check validates the given data against the vocabulary.
func (v *Vocabulary) Check(data map[string]interface{}) error {
	if v == nil {
		return nil
	}
	for key, val := range data {
		// Checks for key validity
		if v.reservedTagKeys[key] {
			// Allow reserved keys to be used even if they are not defined in
			// the vocabulary no matter its strictness.
			continue
		}
		if _, ok := v.Tags[key]; !ok {
			lcKey := strings.ToLower(key)
			correctKey, ok := v.getLabelsToKeys()[lcKey]
			if ok {
				return fmt.Errorf("tag key %q is an alias, must be provided as %q", key, correctKey)
			} else if v.StrictTags {
				return fmt.Errorf("tag key %q is not defined in the vocabulary", key)
			}
			// If the key is not defined, we don't need to check the value
			continue
		}
		// Checks for value validity -- key is defined
		switch val := val.(type) {
		case string:
			err := v.checkValue(key, val)
			if err != nil {
				return err
			}
		case []interface{}:
			for _, singleVal := range val {
				switch singleVal := singleVal.(type) {
				case string:
					err := v.checkValue(key, singleVal)
					if err != nil {
						return err
					}
				default:
					return fmt.Errorf("value list element type for tag key %q was %T, but expected a string", key, singleVal)
				}
			}
		default:
			return fmt.Errorf("value type for tag key %q was %T, but expected a string or list of strings", key, val)
		}
	}
	return nil
}
