// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { connect } from 'react-redux';
import { change, WrappedFieldMetaProps, WrappedFieldInputProps, WrappedFieldProps } from 'redux-form';
import { Vocabulary, PropFieldSuggestion } from 'models/vocabulary';
import { RootState } from 'store/store';
import { getVocabulary } from 'store/vocabulary/vocabulary-selectors';

export interface VocabularyProp {
    vocabulary: Vocabulary;
}

export interface ValidationProp {
    skipValidation?: boolean;
}

export const mapStateToProps = (state: RootState, ownProps: ValidationProp): VocabularyProp & ValidationProp => ({
    skipValidation: ownProps.skipValidation,
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

export const buildProps = ({ input, meta }: WrappedFieldProps) => {
    return {
        value: input.value,
        items: ITEMS_PLACEHOLDER,
        renderSuggestion: (item: PropFieldSuggestion) => item.label,
        error: hasError(meta),
        helperText: getErrorMsg(meta),
    };
};

// Attempts to match a manually typed value label with a value ID, when the user
// doesn't select the value from the suggestions list.
export const handleBlur = (
    fieldName: string,
    formName: string,
    { dispatch }: WrappedFieldMetaProps,
    { onBlur, value }: WrappedFieldInputProps,
    fieldValue: string) =>
    () => {
        dispatch(change(formName, fieldName, fieldValue));
        onBlur(value);
    };

// When selecting a property value, save its ID for later usage.
export const handleSelect = (
    fieldName: string,
    formName: string,
    { onChange }: WrappedFieldInputProps,
    { dispatch }: WrappedFieldMetaProps) =>
    (item: PropFieldSuggestion) => {
        if (item) {
            onChange(item.label);
            dispatch(change(formName, fieldName, item.id));
        }
    };
