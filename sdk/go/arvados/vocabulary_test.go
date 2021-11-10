// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"encoding/json"

	check "gopkg.in/check.v1"
)

type VocabularySuite struct {
	testVoc *Vocabulary
}

var _ = check.Suite(&VocabularySuite{})

func (s *VocabularySuite) SetUpTest(c *check.C) {
	s.testVoc = &Vocabulary{
		reservedTagKeys: map[string]bool{
			"reservedKey": true,
		},
		StrictTags: false,
		Tags: map[string]VocabularyTag{
			"IDTAGANIMALS": {
				Strict: false,
				Labels: []VocabularyLabel{{Label: "Animal"}, {Label: "Creature"}},
				Values: map[string]VocabularyTagValue{
					"IDVALANIMAL1": {
						Labels: []VocabularyLabel{{Label: "Human"}, {Label: "Homo sapiens"}},
					},
					"IDVALANIMAL2": {
						Labels: []VocabularyLabel{{Label: "Elephant"}, {Label: "Loxodonta"}},
					},
				},
			},
			"IDTAGIMPORTANCE": {
				Strict: true,
				Labels: []VocabularyLabel{{Label: "Importance"}, {Label: "Priority"}},
				Values: map[string]VocabularyTagValue{
					"IDVAL3": {
						Labels: []VocabularyLabel{{Label: "Low"}, {Label: "Low priority"}},
					},
					"IDVAL2": {
						Labels: []VocabularyLabel{{Label: "Medium"}, {Label: "Medium priority"}},
					},
					"IDVAL1": {
						Labels: []VocabularyLabel{{Label: "High"}, {Label: "High priority"}},
					},
				},
			},
			"IDTAGCOMMENT": {
				Strict: false,
				Labels: []VocabularyLabel{{Label: "Comment"}},
			},
		},
	}
	err := s.testVoc.validate()
	c.Assert(err, check.IsNil)
}

func (s *VocabularySuite) TestCheck(c *check.C) {
	tests := []struct {
		name          string
		strictVoc     bool
		props         string
		expectSuccess bool
		errMatches    string
	}{
		// Check succeeds
		{
			"Known key, known value",
			false,
			`{"IDTAGANIMALS":"IDVALANIMAL1"}`,
			true,
			"",
		},
		{
			"Unknown non-alias key on non-strict vocabulary",
			false,
			`{"foo":"bar"}`,
			true,
			"",
		},
		{
			"Known non-strict key, unknown non-alias value",
			false,
			`{"IDTAGANIMALS":"IDVALANIMAL3"}`,
			true,
			"",
		},
		{
			"Undefined but reserved key on strict vocabulary",
			true,
			`{"reservedKey":"bar"}`,
			true,
			"",
		},
		{
			"Known key, list of known values",
			false,
			`{"IDTAGANIMALS":["IDVALANIMAL1","IDVALANIMAL2"]}`,
			true,
			"",
		},
		{
			"Known non-strict key, list of unknown non-alias values",
			false,
			`{"IDTAGCOMMENT":["hello world","lorem ipsum"]}`,
			true,
			"",
		},
		// Check fails
		{
			"Known first key & value; known 2nd key, unknown 2nd value",
			false,
			`{"IDTAGANIMALS":"IDVALANIMAL1", "IDTAGIMPORTANCE": "blah blah"}`,
			false,
			"tag value.*is not valid for key.*",
		},
		{
			"Unknown non-alias key on strict vocabulary",
			true,
			`{"foo":"bar"}`,
			false,
			"tag key.*is not defined in the vocabulary",
		},
		{
			"Known non-strict key, known value alias",
			false,
			`{"IDTAGANIMALS":"Loxodonta"}`,
			false,
			"tag value.*for key.* is an alias, must be provided as.*",
		},
		{
			"Known strict key, unknown non-alias value",
			false,
			`{"IDTAGIMPORTANCE":"Unimportant"}`,
			false,
			"tag value.*is not valid for key.*",
		},
		{
			"Known strict key, lowercase value regarded as alias",
			false,
			`{"IDTAGIMPORTANCE":"idval1"}`,
			false,
			"tag value.*for key.* is an alias, must be provided as.*",
		},
		{
			"Known strict key, known value alias",
			false,
			`{"IDTAGIMPORTANCE":"High"}`,
			false,
			"tag value.* for key.*is an alias, must be provided as.*",
		},
		{
			"Known strict key, list of known alias values",
			false,
			`{"IDTAGIMPORTANCE":["High", "Low"]}`,
			false,
			"tag value.*for key.*is an alias, must be provided as.*",
		},
		{
			"Known strict key, list of unknown non-alias values",
			false,
			`{"IDTAGIMPORTANCE":["foo","bar"]}`,
			false,
			"tag value.*is not valid for key.*",
		},
		{
			"Invalid value type",
			false,
			`{"IDTAGANIMALS":1}`,
			false,
			"value type for tag key.* was.*, but expected a string or list of strings",
		},
		{
			"Value list of invalid type",
			false,
			`{"IDTAGANIMALS":[1]}`,
			false,
			"value list element type for tag key.* was.*, but expected a string",
		},
	}
	for _, tt := range tests {
		c.Log(c.TestName()+" ", tt.name)
		s.testVoc.StrictTags = tt.strictVoc

		var data map[string]interface{}
		err := json.Unmarshal([]byte(tt.props), &data)
		c.Assert(err, check.IsNil)
		err = s.testVoc.Check(data)
		if tt.expectSuccess {
			c.Assert(err, check.IsNil)
		} else {
			c.Assert(err, check.NotNil)
			c.Assert(err.Error(), check.Matches, tt.errMatches)
		}
	}
}

func (s *VocabularySuite) TestNewVocabulary(c *check.C) {
	tests := []struct {
		name       string
		data       string
		isValid    bool
		errMatches string
		expect     *Vocabulary
	}{
		{"Empty data", "", true, "", &Vocabulary{}},
		{"Invalid JSON", "foo", false, "invalid JSON format.*", nil},
		{"Valid, empty JSON", "{}", false, ".*doesn't match Vocabulary format.*", nil},
		{"Valid JSON, wrong data", `{"foo":"bar"}`, false, ".*doesn't match Vocabulary format.*", nil},
		{
			"Simple valid example",
			`{"tags":{
				"IDTAGANIMALS":{
					"strict": false,
					"labels": [{"label": "Animal"}, {"label": "Creature"}],
					"values": {
						"IDVALANIMAL1":{"labels":[{"label":"Human"}, {"label":"Homo sapiens"}]},
						"IDVALANIMAL2":{"labels":[{"label":"Elephant"}, {"label":"Loxodonta"}]}
					}
				}
			}}`,
			true, "",
			&Vocabulary{
				reservedTagKeys: map[string]bool{
					"type":                  true,
					"template_uuid":         true,
					"groups":                true,
					"username":              true,
					"image_timestamp":       true,
					"docker-image-repo-tag": true,
					"filters":               true,
					"container_request":     true,
				},
				StrictTags: false,
				Tags: map[string]VocabularyTag{
					"IDTAGANIMALS": {
						Strict: false,
						Labels: []VocabularyLabel{{Label: "Animal"}, {Label: "Creature"}},
						Values: map[string]VocabularyTagValue{
							"IDVALANIMAL1": {
								Labels: []VocabularyLabel{{Label: "Human"}, {Label: "Homo sapiens"}},
							},
							"IDVALANIMAL2": {
								Labels: []VocabularyLabel{{Label: "Elephant"}, {Label: "Loxodonta"}},
							},
						},
					},
				},
			},
		},
		{
			"Valid data, but uses reserved key",
			`{"tags":{
				"type":{
					"strict": false,
					"labels": [{"label": "Type"}]
				}
			}}`,
			false, "tag key.*is reserved", nil,
		},
	}

	for _, tt := range tests {
		c.Log(c.TestName()+" ", tt.name)
		voc, err := NewVocabulary([]byte(tt.data), []string{})
		if tt.isValid {
			c.Assert(err, check.IsNil)
		} else {
			c.Assert(err, check.NotNil)
			if tt.errMatches != "" {
				c.Assert(err, check.ErrorMatches, tt.errMatches)
			}
		}
		c.Assert(voc, check.DeepEquals, tt.expect)
	}
}

func (s *VocabularySuite) TestValidationErrors(c *check.C) {
	tests := []struct {
		name       string
		voc        *Vocabulary
		errMatches string
	}{
		{
			"Strict vocabulary, no keys",
			&Vocabulary{
				StrictTags: true,
			},
			"vocabulary is strict but no tags are defined",
		},
		{
			"Collision between tag key and tag key label",
			&Vocabulary{
				StrictTags: false,
				Tags: map[string]VocabularyTag{
					"IDTAGANIMALS": {
						Strict: false,
						Labels: []VocabularyLabel{{Label: "Animal"}, {Label: "Creature"}},
					},
					"IDTAGCOMMENT": {
						Strict: false,
						Labels: []VocabularyLabel{{Label: "Comment"}, {Label: "IDTAGANIMALS"}},
					},
				},
			},
			"", // Depending on how the map is sorted, this could be one of two errors
		},
		{
			"Collision between tag key and tag key label (case-insensitive)",
			&Vocabulary{
				StrictTags: false,
				Tags: map[string]VocabularyTag{
					"IDTAGANIMALS": {
						Strict: false,
						Labels: []VocabularyLabel{{Label: "Animal"}, {Label: "Creature"}},
					},
					"IDTAGCOMMENT": {
						Strict: false,
						Labels: []VocabularyLabel{{Label: "Comment"}, {Label: "IdTagAnimals"}},
					},
				},
			},
			"", // Depending on how the map is sorted, this could be one of two errors
		},
		{
			"Collision between tag key labels",
			&Vocabulary{
				StrictTags: false,
				Tags: map[string]VocabularyTag{
					"IDTAGANIMALS": {
						Strict: false,
						Labels: []VocabularyLabel{{Label: "Animal"}, {Label: "Creature"}},
					},
					"IDTAGCOMMENT": {
						Strict: false,
						Labels: []VocabularyLabel{{Label: "Comment"}, {Label: "Animal"}},
					},
				},
			},
			"tag label.*for key.*already seen.*",
		},
		{
			"Collision between tag value and tag value label",
			&Vocabulary{
				StrictTags: false,
				Tags: map[string]VocabularyTag{
					"IDTAGANIMALS": {
						Strict: false,
						Labels: []VocabularyLabel{{Label: "Animal"}, {Label: "Creature"}},
						Values: map[string]VocabularyTagValue{
							"IDVALANIMAL1": {
								Labels: []VocabularyLabel{{Label: "Human"}, {Label: "Mammal"}},
							},
							"IDVALANIMAL2": {
								Labels: []VocabularyLabel{{Label: "Elephant"}, {Label: "IDVALANIMAL1"}},
							},
						},
					},
				},
			},
			"", // Depending on how the map is sorted, this could be one of two errors
		},
		{
			"Collision between tag value and tag value label (case-insensitive)",
			&Vocabulary{
				StrictTags: false,
				Tags: map[string]VocabularyTag{
					"IDTAGANIMALS": {
						Strict: false,
						Labels: []VocabularyLabel{{Label: "Animal"}, {Label: "Creature"}},
						Values: map[string]VocabularyTagValue{
							"IDVALANIMAL1": {
								Labels: []VocabularyLabel{{Label: "Human"}, {Label: "Mammal"}},
							},
							"IDVALANIMAL2": {
								Labels: []VocabularyLabel{{Label: "Elephant"}, {Label: "IDValAnimal1"}},
							},
						},
					},
				},
			},
			"", // Depending on how the map is sorted, this could be one of two errors
		},
		{
			"Collision between tag value labels",
			&Vocabulary{
				StrictTags: false,
				Tags: map[string]VocabularyTag{
					"IDTAGANIMALS": {
						Strict: false,
						Labels: []VocabularyLabel{{Label: "Animal"}, {Label: "Creature"}},
						Values: map[string]VocabularyTagValue{
							"IDVALANIMAL1": {
								Labels: []VocabularyLabel{{Label: "Human"}, {Label: "Mammal"}},
							},
							"IDVALANIMAL2": {
								Labels: []VocabularyLabel{{Label: "Elephant"}, {Label: "Mammal"}},
							},
						},
					},
				},
			},
			"tag value label.*for pair.*already seen.*",
		},
		{
			"Strict tag key, with no values",
			&Vocabulary{
				StrictTags: false,
				Tags: map[string]VocabularyTag{
					"IDTAGANIMALS": {
						Strict: true,
						Labels: []VocabularyLabel{{Label: "Animal"}, {Label: "Creature"}},
					},
				},
			},
			"tag key.*is configured as strict but doesn't provide values",
		},
	}
	for _, tt := range tests {
		c.Log(c.TestName()+" ", tt.name)
		err := tt.voc.validate()
		c.Assert(err, check.NotNil)
		if tt.errMatches != "" {
			c.Assert(err, check.ErrorMatches, tt.errMatches)
		}
	}
}
