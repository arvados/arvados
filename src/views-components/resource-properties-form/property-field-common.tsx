// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from 'react-redux';
import { Vocabulary } from '~/models/vocabulary';
import { RootState } from '~/store/store';
import { getVocabulary } from '~/store/vocabulary/vocabulary-selctors';

export interface VocabularyProp {
    vocabulary: Vocabulary;
}

export const mapStateToProps = (state: RootState): VocabularyProp => ({
    vocabulary: getVocabulary(state.properties),
});

export const connectVocabulary = connect(mapStateToProps);

export const ITEMS_PLACEHOLDER: string[] = [];
