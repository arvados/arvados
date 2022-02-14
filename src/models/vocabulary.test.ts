// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as Vocabulary from './vocabulary';

describe('Vocabulary', () => {
    let vocabulary: Vocabulary.Vocabulary;

    beforeEach(() => {
        vocabulary = {
            strict_tags: false,
            tags: {
                IDKEYCOMMENT: {
                    labels: []
                },
                IDKEYANIMALS: {
                    strict: false,
                    labels: [
                        {label: "Animal" },
                        {label: "Creature"},
                        {label: "Beast"},
                    ],
                    values: {
                        IDVALANIMALS1: {
                            labels: [
                                {label: "Human"},
                                {label: "Homo sapiens"}
                            ]
                        },
                        IDVALANIMALS2: {
                            labels: [
                                {label: "Dog"},
                                {label: "Canis lupus familiaris"}
                            ]
                        },
                    }
                },
                IDKEYSIZES: {
                    labels: [{label: "Sizes"}],
                    values: {
                        IDVALSIZES1: {
                            labels: [{label: "Small"}, {label: "S"}, {label: "Little"}]
                        },
                        IDVALSIZES2: {
                            labels: [{label: "Medium"}, {label: "M"}]
                        },
                        IDVALSIZES3: {
                            labels: [{label: "Large"}, {label: "L"}]
                        },
                        IDVALSIZES4: {
                            labels: []
                        }
                    }
                }
            }
        }
    });

    it('returns the list of tag keys', () => {
        const tagKeys = Vocabulary.getTags(vocabulary);
        // Alphabetically ordered by label
        expect(tagKeys).toEqual([
            {id: "IDKEYANIMALS", label: "Animal"},
            {id: "IDKEYANIMALS", label: "Beast"},
            {id: "IDKEYANIMALS", label: "Creature"},
            {id: "IDKEYCOMMENT", label: "IDKEYCOMMENT"},
            {id: "IDKEYSIZES", label: "Sizes"},
        ]);
    });

    it('returns the list of preferred tag keys', () => {
        const preferredTagKeys = Vocabulary.getPreferredTags(vocabulary);
        // Alphabetically ordered by label
        expect(preferredTagKeys).toEqual([
            {id: "IDKEYANIMALS", label: "Animal", synonyms: []},
            {id: "IDKEYCOMMENT", label: "IDKEYCOMMENT", synonyms: []},
            {id: "IDKEYSIZES", label: "Sizes", synonyms: []},
        ]);
    });

    it('returns the list of preferred tag keys with matching synonyms', () => {
        const preferredTagKeys = Vocabulary.getPreferredTags(vocabulary, 'creat');
        // Alphabetically ordered by label
        expect(preferredTagKeys).toEqual([
            {id: "IDKEYANIMALS", label: "Animal", synonyms: ["Creature"]},
            {id: "IDKEYCOMMENT", label: "IDKEYCOMMENT", synonyms: []},
            {id: "IDKEYSIZES", label: "Sizes", synonyms: []},
        ]);
    });

    it('returns the tag values for a given key', () => {
        const tagValues = Vocabulary.getTagValues('IDKEYSIZES', vocabulary);
        // Alphabetically ordered by label
        expect(tagValues).toEqual([
            {id: "IDVALSIZES4", label: "IDVALSIZES4"},
            {id: "IDVALSIZES3", label: "L"},
            {id: "IDVALSIZES3", label: "Large"},
            {id: "IDVALSIZES1", label: "Little"},
            {id: "IDVALSIZES2", label: "M"},
            {id: "IDVALSIZES2", label: "Medium"},
            {id: "IDVALSIZES1", label: "S"},
            {id: "IDVALSIZES1", label: "Small"},
        ])
    });

    it('returns the preferred tag values for a given key', () => {
        const preferredTagValues = Vocabulary.getPreferredTagValues('IDKEYSIZES', vocabulary);
        // Alphabetically ordered by label
        expect(preferredTagValues).toEqual([
            {id: "IDVALSIZES4", label: "IDVALSIZES4", synonyms: []},
            {id: "IDVALSIZES3", label: "Large", synonyms: []},
            {id: "IDVALSIZES2", label: "Medium", synonyms: []},
            {id: "IDVALSIZES1", label: "Small", synonyms: []},
        ])
    });

    it('returns the preferred tag values with matching synonyms for a given key', () => {
        const preferredTagValues = Vocabulary.getPreferredTagValues('IDKEYSIZES', vocabulary, 'litt');
        // Alphabetically ordered by label
        expect(preferredTagValues).toEqual([
            {id: "IDVALSIZES4", label: "IDVALSIZES4", synonyms: []},
            {id: "IDVALSIZES3", label: "Large", synonyms: []},
            {id: "IDVALSIZES2", label: "Medium", synonyms: []},
            {id: "IDVALSIZES1", label: "Small", synonyms: ["Little"]},
        ])
    });

    it('returns an empty list of values for an non-existent key', () => {
        const tagValues = Vocabulary.getTagValues('IDNONSENSE', vocabulary);
        expect(tagValues).toEqual([]);
    });

    it('returns a key id for a given key label', () => {
        const testCases = [
            // Two labels belonging to the same ID
            {keyLabel: 'Animal', expected: 'IDKEYANIMALS'},
            {keyLabel: 'Creature', expected: 'IDKEYANIMALS'},
            // Non-existent label returns empty string
            {keyLabel: 'ThisKeyLabelDoesntExist', expected: ''},
        ]
        testCases.forEach(tc => {
            const tagValueID = Vocabulary.getTagKeyID(tc.keyLabel, vocabulary);
            expect(tagValueID).toEqual(tc.expected);
        });
    });

    it('returns an key label for a given key id', () => {
        const testCases = [
            // ID with many labels return the first one
            {keyID: 'IDKEYANIMALS', expected: 'Animal'},
            // Key IDs without any labels or unknown keys should return the literal
            // key from the API's response (that is, the key 'id')
            {keyID: 'IDKEYCOMMENT', expected: 'IDKEYCOMMENT'},
            {keyID: 'FOO', expected: 'FOO'},
        ]
        testCases.forEach(tc => {
            const tagValueID = Vocabulary.getTagKeyLabel(tc.keyID, vocabulary);
            expect(tagValueID).toEqual(tc.expected);
        });
    });

    it('returns a value id for a given key id and value label', () => {
        const testCases = [
            // Key ID and value label known
            {keyID: 'IDKEYANIMALS', valueLabel: 'Human', expected: 'IDVALANIMALS1'},
            {keyID: 'IDKEYANIMALS', valueLabel: 'Homo sapiens', expected: 'IDVALANIMALS1'},
            // Key ID known, value label unknown
            {keyID: 'IDKEYANIMALS', valueLabel: 'Dinosaur', expected: ''},
            // Key ID unknown
            {keyID: 'IDNONSENSE', valueLabel: 'Does not matter', expected: ''},
        ]
        testCases.forEach(tc => {
            const tagValueID = Vocabulary.getTagValueID(tc.keyID, tc.valueLabel, vocabulary);
            expect(tagValueID).toEqual(tc.expected);
        });
    });

    it('returns a value label for a given key & value id pair', () => {
        const testCases = [
            // Known key & value ids with multiple value labels: returns the first label
            {keyId: 'IDKEYANIMALS', valueId: 'IDVALANIMALS1', expected: 'Human'},
            // Values without label or unknown values should return the literal value from
            // the API's response (that is, the value 'id')
            {keyId: 'IDKEYSIZES', valueId: 'IDVALSIZES4', expected: 'IDVALSIZES4'},
            {keyId: 'IDKEYCOMMENT', valueId: 'FOO', expected: 'FOO'},
            {keyId: 'IDKEYANIMALS', valueId: 'BAR', expected: 'BAR'},
            {keyId: 'IDKEYNONSENSE', valueId: 'FOOBAR', expected: 'FOOBAR'},
        ]
        testCases.forEach(tc => {
            const tagValueLabel = Vocabulary.getTagValueLabel(tc.keyId, tc.valueId, vocabulary);
            expect(tagValueLabel).toEqual(tc.expected);
        });
    });
});
