// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { WrappedFieldProps, Field } from 'redux-form';
import { identity, memoize } from 'lodash';
import { Autocomplete } from '~/components/autocomplete/autocomplete';
import { Vocabulary } from '~/models/vocabulary';
import { require } from '~/validators/require';
import { ITEMS_PLACEHOLDER, connectVocabulary, VocabularyProp, hasError, getErrorMsg, handleBlur } from '~/views-components/resource-properties-form/property-field-common';

export const PROPERTY_KEY_FIELD_NAME = 'key';

export const PropertyKeyField = connectVocabulary(
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
        onBlur={handleBlur(input)}
        label='Key'
        suggestions={getSuggestions(input.value, vocabulary)}
        items={ITEMS_PLACEHOLDER}
        onSelect={input.onChange}
        renderSuggestion={identity}
        error={hasError(meta)}
        helperText={getErrorMsg(meta)}
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
