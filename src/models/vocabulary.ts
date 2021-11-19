// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

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

export const getTagValueID = (tagKeyID:string, tagValueLabel:string, vocabulary: Vocabulary) =>
    (tagKeyID && vocabulary.tags[tagKeyID] && vocabulary.tags[tagKeyID].values)
    ? Object.keys(vocabulary.tags[tagKeyID].values!).find(
        k => vocabulary.tags[tagKeyID].values![k].labels.find(
            l => l.label.toLowerCase() === tagValueLabel.toLowerCase()) !== undefined) || ''
    : '';

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

export const getTagValues = (tagKeyID: string, vocabulary: Vocabulary) => {
    const tag = vocabulary.tags[tagKeyID];
    const ret = tag && tag.values
        ? Object.keys(tag.values).map(
            tagValueID => tag.values![tagValueID].labels && tag.values![tagValueID].labels.length > 0
                ? tag.values![tagValueID].labels.map(
                    lbl => Object.assign({}, {"id": tagValueID, "label": lbl.label}))
                : [{"id": tagValueID, "label": tagValueID}])
            .reduce((prev, curr) => [...prev, ...curr], [])
            .sort(compare)
        : [];
    return ret;
};

export const getTags = ({ tags }: Vocabulary) => {
    const ret = tags && Object.keys(tags)
        ? Object.keys(tags).map(
            tagID => tags[tagID].labels && tags[tagID].labels.length > 0
                ? tags[tagID].labels.map(
                    lbl => Object.assign({}, {"id": tagID, "label": lbl.label}))
                : [{"id": tagID, "label": tagID}])
            .reduce((prev, curr) => [...prev, ...curr], [])
            .sort(compare)
        : [];
    return ret;
};

export const getTagKeyID = (tagKeyLabel:string, vocabulary: Vocabulary) =>
    Object.keys(vocabulary.tags).find(
        k => vocabulary.tags[k].labels.find(
            l => l.label.toLowerCase() === tagKeyLabel.toLowerCase()) !== undefined
        ) || '';

export const getTagKeyLabel = (tagKeyID:string, vocabulary: Vocabulary) =>
    vocabulary.tags[tagKeyID] && vocabulary.tags[tagKeyID].labels.length > 0
    ? vocabulary.tags[tagKeyID].labels[0].label
    : tagKeyID;
