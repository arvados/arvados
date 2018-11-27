// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { PropertiesState, getProperty } from '~/store/properties/properties';
import { Vocabulary } from '~/models/vocabulary';

export const VOCABULARY_PROPERTY_NAME = 'vocabulary';

export const DEFAULT_VOCABULARY: Vocabulary = {
    strict: false,
    tags: {},
};

export const getVocabulary = (state: PropertiesState) =>
    getProperty<Vocabulary>(VOCABULARY_PROPERTY_NAME)(state) || DEFAULT_VOCABULARY;
