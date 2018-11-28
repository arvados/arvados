// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from 'react-redux';
import { Vocabulary } from '~/models/vocabulary';
import { RootState } from '~/store/store';
import { getVocabulary } from '~/store/vocabulary/vocabulary-selctors';
import { WrappedFieldMetaProps, WrappedFieldInputProps } from 'redux-form';

export interface VocabularyProp {
    vocabulary: Vocabulary;
}

export const mapStateToProps = (state: RootState): VocabularyProp => ({
    vocabulary: getVocabulary(state.properties),
});

export const connectVocabulary = connect(mapStateToProps);

export const ITEMS_PLACEHOLDER: string[] = [];

export const hasError = ({ touched, invalid }: WrappedFieldMetaProps) =>
    touched && invalid;

export const getErrorMsg = (meta: WrappedFieldMetaProps) =>
    hasError(meta)
        ? meta.error
        : '';

export const handleBlur = ({ onBlur, value }: WrappedFieldInputProps) =>
    () =>
        onBlur(value);
