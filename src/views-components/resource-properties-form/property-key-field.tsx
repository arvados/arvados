// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { WrappedFieldProps, Field } from 'redux-form';
import { connect } from 'react-redux';
import { identity, memoize } from 'lodash';
import { RootState } from '~/store/store';
import { getVocabulary } from '~/store/vocabulary/vocabulary-selctors';
import { Autocomplete } from '~/components/autocomplete/autocomplete';
import { Vocabulary } from '~/models/vocabulary';
import { require } from '~/validators/require';

interface VocabularyProp {
    vocabulary: Vocabulary;
}

const mapStateToProps = (state: RootState): VocabularyProp => ({
    vocabulary: getVocabulary(state.properties),
});

export const PROPERTY_KEY_FIELD_NAME = 'key';

export const PropertyKeyField = connect(mapStateToProps)(
    ({ vocabulary }: VocabularyProp) =>
        <Field
            name={PROPERTY_KEY_FIELD_NAME}
            component={PropertyKeyInput}
            vocabulary={vocabulary}
            validate={getValidation(vocabulary)} />);

const PropertyKeyInput = ({ input, meta, vocabulary }: WrappedFieldProps & VocabularyProp) =>
    <Autocomplete
        value={input.value}
        onChange={input.onChange}
        label='Key'
        suggestions={getSuggestions(input.value, vocabulary)}
        items={ITEMS_PLACEHOLDER}
        onSelect={input.onChange}
        renderSuggestion={identity}
        error={meta.invalid}
        helperText={meta.error}
    />;

const getValidation = memoize(
    (vocabulary: Vocabulary) =>
        vocabulary.strict
            ? [require, matchTags(vocabulary)]
            : [require]);

const matchTags = (vocabulary: Vocabulary) =>
    (value: string) =>
        getTagsList(vocabulary).find(tag => tag.includes(value))
            ? undefined
            : 'Incorrect key';

const getSuggestions = (value: string, vocabulary: Vocabulary) =>
    getTagsList(vocabulary).filter(tag => tag.includes(value) && tag !== value);

const getTagsList = ({ tags }: Vocabulary) =>
    Object.keys(tags);

const ITEMS_PLACEHOLDER: string[] = [];
