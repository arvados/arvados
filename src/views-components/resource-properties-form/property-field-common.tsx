// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from 'react-redux';
import { WrappedFieldMetaProps, WrappedFieldInputProps, WrappedFieldProps } from 'redux-form';
import { identity } from 'lodash';
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

export const hasError = ({ touched, invalid }: WrappedFieldMetaProps) =>
    touched && invalid;

export const getErrorMsg = (meta: WrappedFieldMetaProps) =>
    hasError(meta)
        ? meta.error
        : '';

export const handleBlur = ({ onBlur, value }: WrappedFieldInputProps) =>
    () =>
        onBlur(value);

export const buildProps = ({ input, meta }: WrappedFieldProps) => ({
    value: input.value,
    onChange: input.onChange,
    onBlur: handleBlur(input),
    items: ITEMS_PLACEHOLDER,
    onSelect: input.onChange,
    renderSuggestion: identity,
    error: hasError(meta),
    helperText: getErrorMsg(meta),
});
