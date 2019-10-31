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

const VOCABULARY_VALIDATORS = [
    isObject,
    has('strict_tags'),
    has('tags'),
];

export const isVocabulary = (value: any) =>
    every(validator => validator(value), VOCABULARY_VALIDATORS);