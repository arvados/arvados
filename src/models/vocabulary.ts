// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { isObject, has, every } from 'lodash/fp';

export interface Vocabulary {
    strict: boolean;
    tags: Record<string, Tag>;
}

export interface Tag {
    strict?: boolean;
    values?: string[];
}

const VOCABULARY_VALIDATORS = [
    isObject,
    has('strict'),
    has('tags'),
];

export const isVocabulary = (value: any) =>
    every(validator => validator(value), VOCABULARY_VALIDATORS);