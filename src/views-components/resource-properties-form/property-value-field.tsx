// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { change, WrappedFieldProps, WrappedFieldMetaProps, WrappedFieldInputProps, Field, formValues } from 'redux-form';
import { compose } from 'redux';
import { Autocomplete } from '~/components/autocomplete/autocomplete';
import { Vocabulary } from '~/models/vocabulary';
import { PROPERTY_KEY_FIELD_NAME } from '~/views-components/resource-properties-form/property-key-field';
import { VocabularyProp, connectVocabulary, buildProps, PropFieldSuggestion } from '~/views-components/resource-properties-form/property-field-common';
import { TAG_VALUE_VALIDATION } from '~/validators/validators';
import { COLLECTION_TAG_FORM_NAME } from '~/store/collection-panel/collection-panel-action';

interface PropertyKeyProp {
    propertyKey: string;
}

export type PropertyValueFieldProps = VocabularyProp & PropertyKeyProp;

export const PROPERTY_VALUE_FIELD_NAME = 'value';
export const PROPERTY_VALUE_FIELD_ID = 'valueID';

export const PropertyValueField = compose(
    connectVocabulary,
    formValues({ propertyKey: PROPERTY_KEY_FIELD_NAME })
)(
    (props: PropertyValueFieldProps) =>
        <div>
            <Field
                name={PROPERTY_VALUE_FIELD_NAME}
                component={PropertyValueInput}
                validate={getValidation(props)}
                {...props} />
            <Field
                name={PROPERTY_VALUE_FIELD_ID}
                type='hidden'
                component='input' />
        </div>
);

export const PropertyValueInput = ({ vocabulary, propertyKey, ...props }: WrappedFieldProps & PropertyValueFieldProps) =>
    <Autocomplete
        label='Value'
        suggestions={getSuggestions(props.input.value, propertyKey, vocabulary)}
        onSelect={handleSelect(props.input, props.meta)}
        {...buildProps(props)}
    />;

const getValidation = (props: PropertyValueFieldProps) =>
    isStrictTag(props.propertyKey, props.vocabulary)
        ? [...TAG_VALUE_VALIDATION, matchTagValues(props)]
        : TAG_VALUE_VALIDATION;

const matchTagValues = ({ vocabulary, propertyKey }: PropertyValueFieldProps) =>
    (value: string) =>
        getTagValues(propertyKey, vocabulary).find(v => v.label === value)
            ? undefined
            : 'Incorrect value';

const getSuggestions = (value: string, tagKey: string, vocabulary: Vocabulary) =>
    getTagValues(tagKey, vocabulary).filter(v => v.label.toLowerCase().includes(value.toLowerCase()));

const isStrictTag = (tagKey: string, vocabulary: Vocabulary) => {
    const tag = vocabulary.tags[getTagID(tagKey, vocabulary)];
    return tag ? tag.strict : false;
};

const getTagID = (tagKeyLabel:string, vocabulary: Vocabulary) =>
    Object.keys(vocabulary.tags).find(
        k => vocabulary.tags[k].labels.find(
            l => l.label === tagKeyLabel) !== undefined) || tagKeyLabel;

const getTagValues = (tagKey: string, vocabulary: Vocabulary) => {
    const tag = vocabulary.tags[getTagID(tagKey, vocabulary)];
    const ret = tag && tag.values
        ? Object.keys(tag.values).map(
            tagValueID => tag.values![tagValueID].labels
                ? {"id": tagValueID, "label": tag.values![tagValueID].labels[0].label}
                : {"id": tagValueID, "label": tagValueID})
        : [];
    return ret;
};

const handleSelect = (
    { onChange }: WrappedFieldInputProps,
    { dispatch }: WrappedFieldMetaProps) => {
        return (item:PropFieldSuggestion) => {
            onChange(item.label);
            dispatch(change(COLLECTION_TAG_FORM_NAME, PROPERTY_VALUE_FIELD_ID, item.id));
    };
};
