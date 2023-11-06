// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { WrappedFieldProps, Field, FormName, reset, change, WrappedFieldInputProps, WrappedFieldMetaProps } from 'redux-form';
import { memoize } from 'lodash';
import { Autocomplete } from 'components/autocomplete/autocomplete';
import {
    Vocabulary,
    getTags,
    getTagKeyID,
    getTagKeyLabel,
    getPreferredTags,
    PropFieldSuggestion
} from 'models/vocabulary';
import {
    handleSelect,
    handleBlur,
    connectVocabulary,
    VocabularyProp,
    ValidationProp,
    buildProps
} from 'views-components/resource-properties-form/property-field-common';
import { TAG_KEY_VALIDATION } from 'validators/validators';
import { escapeRegExp } from 'common/regexp';
import { ChangeEvent } from 'react';

export const PROPERTY_KEY_FIELD_NAME = 'key';
export const PROPERTY_KEY_FIELD_ID = 'keyID';

export const PropertyKeyField = connectVocabulary(
    ({ vocabulary, skipValidation, clearPropertyKeyOnSelect }: VocabularyProp & ValidationProp) =>
        <span data-cy='property-field-key'>
        <Field
            clearPropertyKeyOnSelect
            name={PROPERTY_KEY_FIELD_NAME}
            component={PropertyKeyInput}
            vocabulary={vocabulary}
            validate={skipValidation ? undefined : getValidation(vocabulary)} />
        </span>
);

const PropertyKeyInput = ({ vocabulary, ...props }: WrappedFieldProps & VocabularyProp & { clearPropertyKeyOnSelect?: boolean }) =>
    <FormName children={data => (
        <Autocomplete
            {...buildProps(props)}
            label='Key'
            suggestions={getSuggestions(props.input.value, vocabulary)}
            renderSuggestion={
                (s: PropFieldSuggestion) => s.synonyms && s.synonyms.length > 0
                    ? `${s.label} (${s.synonyms.join('; ')})`
                    : s.label
            }
            onFocus={() => {
                if (props.clearPropertyKeyOnSelect && props.input.value) {
                    props.meta.dispatch(reset(props.meta.form));
                }
            }}
            onSelect={handleSelect(PROPERTY_KEY_FIELD_ID, data.form, props.input, props.meta)}
            onBlur={() => {
                // Case-insensitive search for the key in the vocabulary
                const foundKeyID = getTagKeyID(props.input.value, vocabulary);
                if (foundKeyID !== '') {
                    props.input.value = getTagKeyLabel(foundKeyID, vocabulary);
                }
                handleBlur(PROPERTY_KEY_FIELD_ID, data.form, props.meta, props.input, foundKeyID)();
            }}
            onChange={(e: ChangeEvent<HTMLInputElement>) => {
                const newValue = e.currentTarget.value;
                handleChange(data.form, props.input, props.meta, newValue);
            }}
        />
    )} />;

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

const getSuggestions = (value: string, vocabulary: Vocabulary): PropFieldSuggestion[] => {
    const re = new RegExp(escapeRegExp(value), "i");
    return getPreferredTags(vocabulary, value).filter(
        tag => (tag.label !== value && re.test(tag.label)) ||
            (tag.synonyms && tag.synonyms.some(s => re.test(s))));
};

const handleChange = (
    formName: string,
    { onChange }: WrappedFieldInputProps,
    { dispatch }: WrappedFieldMetaProps,
    value: string) => {
        // Properties' values are dependant on the keys, if any value is
        // pre-existant, a change on the property key should mean that the
        // previous value is invalid, so we better reset the whole form before
        // setting the new tag key.
        dispatch(reset(formName));

        onChange(value);
        dispatch(change(formName, PROPERTY_KEY_FIELD_NAME, value));
    };