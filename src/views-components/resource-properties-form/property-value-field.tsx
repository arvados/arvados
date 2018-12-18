// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { WrappedFieldProps, Field, formValues } from 'redux-form';
import { compose } from 'redux';
import { Autocomplete } from '~/components/autocomplete/autocomplete';
import { Vocabulary } from '~/models/vocabulary';
import { PROPERTY_KEY_FIELD_NAME } from '~/views-components/resource-properties-form/property-key-field';
import { VocabularyProp, connectVocabulary, buildProps } from '~/views-components/resource-properties-form/property-field-common';
import { TAG_VALUE_VALIDATION } from '~/validators/validators';

interface PropertyKeyProp {
    propertyKey: string;
}

export type PropertyValueFieldProps = VocabularyProp & PropertyKeyProp;

export const PROPERTY_VALUE_FIELD_NAME = 'value';

export const PropertyValueField = compose(
    connectVocabulary,
    formValues({ propertyKey: PROPERTY_KEY_FIELD_NAME })
)(
    (props: PropertyValueFieldProps) =>
        <Field
            name={PROPERTY_VALUE_FIELD_NAME}
            component={PropertyValueInput}
            validate={getValidation(props)}
            {...props} />);

export const PropertyValueInput = ({ vocabulary, propertyKey, ...props }: WrappedFieldProps & PropertyValueFieldProps) =>
    <Autocomplete
        label='Value'
        suggestions={getSuggestions(props.input.value, propertyKey, vocabulary)}
        {...buildProps(props)}
    />;

const getValidation = (props: PropertyValueFieldProps) =>
    isStrictTag(props.propertyKey, props.vocabulary)
        ? [...TAG_VALUE_VALIDATION, matchTagValues(props)]
        : TAG_VALUE_VALIDATION;

const matchTagValues = ({ vocabulary, propertyKey }: PropertyValueFieldProps) =>
    (value: string) =>
        getTagValues(propertyKey, vocabulary).find(v => v.includes(value))
            ? undefined
            : 'Incorrect value';

const getSuggestions = (value: string, tagName: string, vocabulary: Vocabulary) =>
    getTagValues(tagName, vocabulary).filter(v => v.includes(value) && v !== value);

const isStrictTag = (tagName: string, vocabulary: Vocabulary) => {
    const tag = vocabulary.tags[tagName];
    return tag ? tag.strict : false;
};

const getTagValues = (tagName: string, vocabulary: Vocabulary) => {
    const tag = vocabulary.tags[tagName];
    return tag && tag.values ? tag.values : [];
};
