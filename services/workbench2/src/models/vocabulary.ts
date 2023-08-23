// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { escapeRegExp } from 'common/regexp';
import { isObject, has, every } from 'lodash/fp';

export interface Vocabulary {
    strict_tags: boolean;
    tags: Record<string, Tag>;
}

export interface Label {
    lang?: string;
    label: string;
}

export interface TagValue {
    labels: Label[];
}

export interface Tag {
    strict?: boolean;
    labels: Label[];
    values?: Record<string, TagValue>;
}

export interface PropFieldSuggestion {
    id: string;
    label: string;
    synonyms?: string[];
}

const VOCABULARY_VALIDATORS = [
    isObject,
    has('strict_tags'),
    has('tags'),
];

export const isVocabulary = (value: any) =>
    every(validator => validator(value), VOCABULARY_VALIDATORS);

export const isStrictTag = (tagKeyID: string, vocabulary: Vocabulary) => {
    const tag = vocabulary.tags[tagKeyID];
    return tag ? tag.strict : false;
};

export const getTagValueID = (tagKeyID:string, tagValueLabel:string, vocabulary: Vocabulary) => {
    if (tagKeyID && vocabulary.tags[tagKeyID] && vocabulary.tags[tagKeyID].values) {
        const values = vocabulary.tags[tagKeyID].values!;
        return Object.keys(values).find(k =>
            (k.toLowerCase() === tagValueLabel.toLowerCase())
            || values[k].labels.find(
                l => l.label.toLowerCase() === tagValueLabel.toLowerCase()) !== undefined)
            || '';
    };
    return '';
};

export const getTagValueLabel = (tagKeyID:string, tagValueID:string, vocabulary: Vocabulary) =>
    vocabulary.tags[tagKeyID] &&
    vocabulary.tags[tagKeyID].values &&
    vocabulary.tags[tagKeyID].values![tagValueID] &&
    vocabulary.tags[tagKeyID].values![tagValueID].labels.length > 0
        ? vocabulary.tags[tagKeyID].values![tagValueID].labels[0].label
        : tagValueID;

const compare = (a: PropFieldSuggestion, b: PropFieldSuggestion) => {
    if (a.label < b.label) {return -1;}
    if (a.label > b.label) {return 1;}
    return 0;
};

export const getTagValues = (tagKeyID: string, vocabulary: Vocabulary): PropFieldSuggestion[] => {
    const tag = vocabulary.tags[tagKeyID];
    return tag && tag.values
        ? Object.keys(tag.values).map(
            tagValueID => tag.values![tagValueID].labels && tag.values![tagValueID].labels.length > 0
                ? tag.values![tagValueID].labels.map(
                    lbl => Object.assign({}, {"id": tagValueID, "label": lbl.label}))
                : [{"id": tagValueID, "label": tagValueID}])
            .reduce((prev, curr) => [...prev, ...curr], [])
            .sort(compare)
        : [];
};

export const getPreferredTagValues = (tagKeyID: string, vocabulary: Vocabulary, withMatch?: string): PropFieldSuggestion[] => {
    const tag = vocabulary.tags[tagKeyID];
    const regex = !!withMatch ? new RegExp(escapeRegExp(withMatch), 'i') : undefined;
    return tag && tag.values
        ? Object.keys(tag.values).map(
            tagValueID => tag.values![tagValueID].labels && tag.values![tagValueID].labels.length > 0
                ? {
                    "id": tagValueID,
                    "label": tag.values![tagValueID].labels[0].label,
                    "synonyms": !!withMatch && tag.values![tagValueID].labels.length > 1
                        ? tag.values![tagValueID].labels.slice(1)
                            .filter(l => !!regex ? regex.test(l.label) : true)
                            .map(l => l.label)
                        : []
                }
                : {"id": tagValueID, "label": tagValueID, "synonyms": []})
            .sort(compare)
        : [];
};

export const getTags = ({ tags }: Vocabulary): PropFieldSuggestion[] => {
    return tags && Object.keys(tags)
        ? Object.keys(tags).map(
            tagID => tags[tagID].labels && tags[tagID].labels.length > 0
                ? tags[tagID].labels.map(
                    lbl => Object.assign({}, {"id": tagID, "label": lbl.label}))
                : [{"id": tagID, "label": tagID}])
            .reduce((prev, curr) => [...prev, ...curr], [])
            .sort(compare)
        : [];
};

export const getPreferredTags = ({ tags }: Vocabulary, withMatch?: string): PropFieldSuggestion[] => {
    const regex = !!withMatch ? new RegExp(escapeRegExp(withMatch), 'i') : undefined;
    return tags && Object.keys(tags)
        ? Object.keys(tags).map(
            tagID => tags[tagID].labels && tags[tagID].labels.length > 0
                ? {
                    "id": tagID,
                    "label": tags[tagID].labels[0].label,
                    "synonyms": !!withMatch && tags[tagID].labels.length > 1
                        ? tags[tagID].labels.slice(1)
                                .filter(l => !!regex ? regex.test(l.label) : true)
                                .map(lbl => lbl.label)
                        : []
                }
                : {"id": tagID, "label": tagID, "synonyms": []})
            .sort(compare)
        : [];
};

export const getTagKeyID = (tagKeyLabel: string, vocabulary: Vocabulary) =>
    Object.keys(vocabulary.tags).find(k => (k.toLowerCase() === tagKeyLabel.toLowerCase())
        || vocabulary.tags[k].labels.find(
            l => l.label.toLowerCase() === tagKeyLabel.toLowerCase()) !== undefined)
        || '';

export const getTagKeyLabel = (tagKeyID:string, vocabulary: Vocabulary) =>
    vocabulary.tags[tagKeyID] && vocabulary.tags[tagKeyID].labels.length > 0
    ? vocabulary.tags[tagKeyID].labels[0].label
    : tagKeyID;
