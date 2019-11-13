// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { change, WrappedFieldProps, WrappedFieldMetaProps, WrappedFieldInputProps,
    Field, FormName } from 'redux-form';
import { memoize } from 'lodash';
import { Autocomplete } from '~/components/autocomplete/autocomplete';
import { Vocabulary, getTags, getTagKeyID, PropFieldSuggestion } from '~/models/vocabulary';
import { connectVocabulary, VocabularyProp, buildProps } from '~/views-components/resource-properties-form/property-field-common';
import { TAG_KEY_VALIDATION } from '~/validators/validators';
import { escapeRegExp } from '~/common/regexp.ts';

export const PROPERTY_KEY_FIELD_NAME = 'key';
export const PROPERTY_KEY_FIELD_ID = 'keyID';

export const PropertyKeyField = connectVocabulary(
    ({ vocabulary }: VocabularyProp) =>
        <Field
            name={PROPERTY_KEY_FIELD_NAME}
            component={PropertyKeyInput}
            vocabulary={vocabulary}
            validate={getValidation(vocabulary)} />
);

export const PropertyKeyInput = ({ vocabulary, ...props }: WrappedFieldProps & VocabularyProp) =>
    <FormName children={data => (
        <Autocomplete
            label='Key'
            suggestions={getSuggestions(props.input.value, vocabulary)}
            onSelect={handleSelect(data.form, props.input, props.meta)}
            {...buildProps(props)}
            onBlur={handleBlur(data.form, props.meta, props.input, vocabulary)}
        />
    )}/>;

const getValidation = memoize(
    (vocabulary: Vocabulary) =>
        vocabulary.strict_tags
            ? [...TAG_KEY_VALIDATION, matchTags(vocabulary)]
            : TAG_KEY_VALIDATION);

const matchTags = (vocabulary: Vocabulary) =>
    (value: string) =>
        getTags(vocabulary).find(tag => tag.label === value)
            ? undefined
            : 'Incorrect key';

const getSuggestions = (value: string, vocabulary: Vocabulary) => {
    const re = new RegExp(escapeRegExp(value), "i");
    return getTags(vocabulary).filter(tag => re.test(tag.label) && tag.label !== value);
};

// Attempts to match a manually typed key label with a key ID, when the user
// doesn't select the key from the suggestions list.
const handleBlur = (
    formName: string,
    { dispatch }: WrappedFieldMetaProps,
    { onBlur, value }: WrappedFieldInputProps,
    vocabulary: Vocabulary) =>
    () => {
        dispatch(change(formName, PROPERTY_KEY_FIELD_ID, getTagKeyID(value, vocabulary)));
        onBlur(value);
    };

// When selecting a property key, save its ID for later usage.
const handleSelect = (
    formName: string,
    { onChange }: WrappedFieldInputProps,
    { dispatch }: WrappedFieldMetaProps) => {
        return (item:PropFieldSuggestion) => {
            onChange(item.label);
            dispatch(change(formName, PROPERTY_KEY_FIELD_ID, item.id));
    };
};
