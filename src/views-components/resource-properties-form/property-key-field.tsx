// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { change, WrappedFieldProps, WrappedFieldMetaProps, WrappedFieldInputProps, Field } from 'redux-form';
import { memoize } from 'lodash';
import { Autocomplete } from '~/components/autocomplete/autocomplete';
import { Vocabulary } from '~/models/vocabulary';
import { connectVocabulary, VocabularyProp, buildProps, PropFieldSuggestion } from '~/views-components/resource-properties-form/property-field-common';
import { TAG_KEY_VALIDATION } from '~/validators/validators';
import { COLLECTION_TAG_FORM_NAME } from '~/store/collection-panel/collection-panel-action';

export const PROPERTY_KEY_FIELD_NAME = 'key';
export const PROPERTY_KEY_FIELD_ID = 'keyID';

export const PropertyKeyField = connectVocabulary(
    ({ vocabulary }: VocabularyProp) =>
        <div>
            <Field
                name={PROPERTY_KEY_FIELD_NAME}
                component={PropertyKeyInput}
                vocabulary={vocabulary}
                validate={getValidation(vocabulary)} />
            <Field
                name={PROPERTY_KEY_FIELD_ID}
                type='hidden'
                component='input' />
        </div>
);

export const PropertyKeyInput = ({ vocabulary, ...props }: WrappedFieldProps & VocabularyProp) =>
    <Autocomplete
        label='Key'
        suggestions={getSuggestions(props.input.value, vocabulary)}
        onSelect={handleSelect(props.input, props.meta)}
        {...buildProps(props)}
        onBlur={handleBlur(props.meta, props.input, vocabulary)}
    />;

const getValidation = memoize(
    (vocabulary: Vocabulary) =>
        vocabulary.strict_tags
            ? [...TAG_KEY_VALIDATION, matchTags(vocabulary)]
            : TAG_KEY_VALIDATION);

const matchTags = (vocabulary: Vocabulary) =>
    (value: string) =>
        getTagsList(vocabulary).find(tag => tag.label === value)
            ? undefined
            : 'Incorrect key';

const getSuggestions = (value: string, vocabulary: Vocabulary) =>
    getTagsList(vocabulary).filter(tag => tag.label.toLowerCase().includes(value.toLowerCase()));

const getTagsList = ({ tags }: Vocabulary) => {
    const ret = tags && Object.keys(tags)
        ? Object.keys(tags).map(
            tagID => tags[tagID].labels
                ? {"id": tagID, "label": tags[tagID].labels[0].label}
                : {"id": tagID, "label": tagID})
        : [];
    return ret;
};

const getTagKeyID = (tagKeyLabel:string, vocabulary: Vocabulary) =>
    Object.keys(vocabulary.tags).find(
        k => vocabulary.tags[k].labels.find(
            l => l.label === tagKeyLabel) !== undefined) || '';

// Attempts to match a manually typed key label with a key ID, when the user
// doesn't select the key from the suggestions list.
const handleBlur = (
    { dispatch }: WrappedFieldMetaProps,
    { onBlur, value }: WrappedFieldInputProps,
    vocabulary: Vocabulary) =>
    () => {
        dispatch(change(COLLECTION_TAG_FORM_NAME, PROPERTY_KEY_FIELD_ID, getTagKeyID(value, vocabulary)));
        onBlur(value);
    };

// When selecting a property key, save its ID for later usage.
const handleSelect = (
    { onChange }: WrappedFieldInputProps,
    { dispatch }: WrappedFieldMetaProps) => {
        return (item:PropFieldSuggestion) => {
            onChange(item.label);
            dispatch(change(COLLECTION_TAG_FORM_NAME, PROPERTY_KEY_FIELD_ID, item.id));
    };
};
