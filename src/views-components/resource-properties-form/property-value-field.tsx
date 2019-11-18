// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { WrappedFieldProps, Field, formValues, FormName } from 'redux-form';
import { compose } from 'redux';
import { Autocomplete } from '~/components/autocomplete/autocomplete';
import { Vocabulary, isStrictTag, getTagValues, getTagValueID } from '~/models/vocabulary';
import { PROPERTY_KEY_FIELD_ID } from '~/views-components/resource-properties-form/property-key-field';
import { handleSelect, handleBlur, VocabularyProp, connectVocabulary, buildProps } from '~/views-components/resource-properties-form/property-field-common';
import { TAG_VALUE_VALIDATION } from '~/validators/validators';
import { escapeRegExp } from '~/common/regexp.ts';

interface PropertyKeyProp {
    propertyKey: string;
}

interface ValidationProp {
    skipValidation?: boolean;
}

type PropertyValueFieldProps = VocabularyProp & PropertyKeyProp;

export const PROPERTY_VALUE_FIELD_NAME = 'value';
export const PROPERTY_VALUE_FIELD_ID = 'valueID';

const connectVocabularyAndPropertyKey = compose<any>(
    connectVocabulary,
    formValues({ propertyKey: PROPERTY_KEY_FIELD_ID }),
);

export const PropertyValueField = connectVocabularyAndPropertyKey(
    ({skipValidation, ...props}: PropertyValueFieldProps & ValidationProp) =>
        <Field
            name={PROPERTY_VALUE_FIELD_NAME}
            component={PropertyValueInput}
            validate={skipValidation ? undefined : getValidation(props)}
            {...props} />
);

const PropertyValueInput = ({ vocabulary, propertyKey, ...props }: WrappedFieldProps & PropertyValueFieldProps) =>
    <FormName children={data => (
        <Autocomplete
            label='Value'
            suggestions={getSuggestions(props.input.value, propertyKey, vocabulary)}
            onSelect={handleSelect(PROPERTY_VALUE_FIELD_ID, data.form, props.input, props.meta)}
            onBlur={handleBlur(PROPERTY_VALUE_FIELD_ID, data.form, props.meta, props.input, getTagValueID(propertyKey, props.input.value, vocabulary))}
            {...buildProps(props)}
        />
    )}/>;

const getValidation = (props: PropertyValueFieldProps) =>
    isStrictTag(props.propertyKey, props.vocabulary)
        ? [...TAG_VALUE_VALIDATION, matchTagValues(props)]
        : TAG_VALUE_VALIDATION;

const matchTagValues = ({ vocabulary, propertyKey }: PropertyValueFieldProps) =>
    (value: string) =>
        getTagValues(propertyKey, vocabulary).find(v => v.label === value)
            ? undefined
            : 'Incorrect value';

const getSuggestions = (value: string, tagName: string, vocabulary: Vocabulary) => {
    const re = new RegExp(escapeRegExp(value), "i");
    return getTagValues(tagName, vocabulary).filter(v => re.test(v.label) && v.label !== value);
};

