// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { WrappedFieldProps, Field } from 'redux-form';
import { connect } from 'react-redux';
import { identity } from 'lodash';
import { RootState } from '~/store/store';
import { getVocabulary } from '~/store/vocabulary/vocabulary-selctors';
import { Autocomplete } from '~/components/autocomplete/autocomplete';
import { Vocabulary } from '~/models/vocabulary';
import { require } from '~/validators/require';

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

export const PropertyValueField = connect(mapStateToProps)(
    (props: PropertyValueFieldProps) =>
        <Field
            name='value'
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
