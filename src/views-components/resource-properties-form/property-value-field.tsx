// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { WrappedFieldProps, Field, formValues } from 'redux-form';
import { connect } from 'react-redux';
import { identity } from 'lodash';
import { compose } from 'redux';
import { RootState } from '~/store/store';
import { getVocabulary } from '~/store/vocabulary/vocabulary-selctors';
import { Autocomplete } from '~/components/autocomplete/autocomplete';
import { Vocabulary } from '~/models/vocabulary';
import { require } from '~/validators/require';
import { PROPERTY_KEY_FIELD_NAME } from '~/views-components/resource-properties-form/property-key-field';

interface VocabularyProp {
    vocabulary: Vocabulary;
}

interface PropertyKeyProp {
    propertyKey: string;
}

type PropertyValueFieldProps = VocabularyProp & PropertyKeyProp;

const mapStateToProps = (state: RootState): VocabularyProp => ({
    vocabulary: getVocabulary(state.properties),
});

export const PROPERTY_VALUE_FIELD_NAME = 'value';

export const PropertyValueField = compose(
    connect(mapStateToProps),
    formValues({ propertyKey: PROPERTY_KEY_FIELD_NAME })
)(
    (props: PropertyValueFieldProps) =>
        <Field
            name={PROPERTY_VALUE_FIELD_NAME}
            component={PropertyValueInput}
            validate={getValidation(props)}
            {...props} />);

const PropertyValueInput = ({ input, meta, vocabulary, propertyKey }: WrappedFieldProps & PropertyValueFieldProps) =>
    <Autocomplete
        value={input.value}
        onChange={input.onChange}
        label='Value'
        suggestions={getSuggestions(input.value, propertyKey, vocabulary)}
        items={ITEMS_PLACEHOLDER}
        onSelect={input.onChange}
        renderSuggestion={identity}
        error={meta.invalid}
        helperText={meta.error}
    />;

const getValidation = (props: PropertyValueFieldProps) =>
    isStrictTag(props.propertyKey, props.vocabulary)
        ? [require, matchTagValues(props)]
        : [require];

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
    return tag ? tag.values : [];
};

const ITEMS_PLACEHOLDER: string[] = [];
