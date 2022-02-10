// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { WrappedFieldProps, Field, formValues, FormName, WrappedFieldInputProps, WrappedFieldMetaProps, change } from 'redux-form';
import { compose } from 'redux';
import { Autocomplete } from 'components/autocomplete/autocomplete';
import { Vocabulary, isStrictTag, getTagValues, getTagValueID, getTagValueLabel, PropFieldSuggestion, getPreferredTagValues } from 'models/vocabulary';
import { PROPERTY_KEY_FIELD_ID, PROPERTY_KEY_FIELD_NAME } from 'views-components/resource-properties-form/property-key-field';
import {
    handleSelect,
    handleBlur,
    VocabularyProp,
    ValidationProp,
    connectVocabulary,
    buildProps
} from 'views-components/resource-properties-form/property-field-common';
import { TAG_VALUE_VALIDATION } from 'validators/validators';
import { escapeRegExp } from 'common/regexp';
import { ChangeEvent } from 'react';

interface PropertyKeyProp {
    propertyKeyId: string;
    propertyKeyName: string;
}

interface PropertyValueInputProp {
    disabled: boolean;
}

type PropertyValueFieldProps = VocabularyProp & PropertyKeyProp & ValidationProp & PropertyValueInputProp;

export const PROPERTY_VALUE_FIELD_NAME = 'value';
export const PROPERTY_VALUE_FIELD_ID = 'valueID';

const connectVocabularyAndPropertyKey = compose(
    connectVocabulary,
    formValues({
        propertyKeyId: PROPERTY_KEY_FIELD_ID,
        propertyKeyName: PROPERTY_KEY_FIELD_NAME,
    }),
);

export const PropertyValueField = connectVocabularyAndPropertyKey(
    ({ skipValidation, ...props }: PropertyValueFieldProps) =>
        <span data-cy='property-field-value'>
        <Field
            name={PROPERTY_VALUE_FIELD_NAME}
            component={PropertyValueInput}
            validate={skipValidation ? undefined : getValidation(props)}
            {...{...props, disabled: !props.propertyKeyName}} />
        </span>
);

const PropertyValueInput = ({ vocabulary, propertyKeyId, propertyKeyName, ...props }: WrappedFieldProps & PropertyValueFieldProps) =>
    <FormName children={data => (
        <Autocomplete
            {...buildProps(props)}
            label='Value'
            disabled={props.disabled}
            suggestions={getSuggestions(props.input.value, propertyKeyId, vocabulary)}
            renderSuggestion={(s: PropFieldSuggestion) => (s.description || s.label)}
            onSelect={handleSelect(PROPERTY_VALUE_FIELD_ID, data.form, props.input, props.meta)}
            onBlur={() => {
                // Case-insensitive search for the value in the vocabulary
                const foundValueID =  getTagValueID(propertyKeyId, props.input.value, vocabulary);
                if (foundValueID !== '') {
                    props.input.value = getTagValueLabel(propertyKeyId, foundValueID, vocabulary);
                }
                handleBlur(PROPERTY_VALUE_FIELD_ID, data.form, props.meta, props.input, foundValueID)();
            }}
            onChange={(e: ChangeEvent<HTMLInputElement>) => {
                const newValue = e.currentTarget.value;
                const tagValueID = getTagValueID(propertyKeyId, newValue, vocabulary);
                handleChange(data.form, tagValueID, props.input, props.meta, newValue);
            }}
        />
    )} />;

const getValidation = (props: PropertyValueFieldProps) =>
    isStrictTag(props.propertyKeyId, props.vocabulary)
        ? [...TAG_VALUE_VALIDATION, matchTagValues(props)]
        : TAG_VALUE_VALIDATION;

const matchTagValues = ({ vocabulary, propertyKeyId }: PropertyValueFieldProps) =>
    (value: string) =>
        getTagValues(propertyKeyId, vocabulary).find(v => v.label === value)
            ? undefined
            : 'Incorrect value';

const getSuggestions = (value: string, tagName: string, vocabulary: Vocabulary) => {
    const re = new RegExp(escapeRegExp(value), "i");
    return getPreferredTagValues(tagName, vocabulary, value !== '').filter(
        v => re.test((v.description || v.label)) && v.label !== value);
};

const handleChange = (
    formName: string,
    tagValueID: string,
    { onChange }: WrappedFieldInputProps,
    { dispatch }: WrappedFieldMetaProps,
    value: string) => {
        onChange(value);
        dispatch(change(formName, PROPERTY_VALUE_FIELD_NAME, value));
        dispatch(change(formName, PROPERTY_VALUE_FIELD_ID, tagValueID));
    };