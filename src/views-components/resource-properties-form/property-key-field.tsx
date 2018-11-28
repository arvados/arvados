// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { WrappedFieldProps, Field } from 'redux-form';
import { memoize } from 'lodash';
import { Autocomplete } from '~/components/autocomplete/autocomplete';
import { Vocabulary } from '~/models/vocabulary';
import { connectVocabulary, VocabularyProp, buildProps } from '~/views-components/resource-properties-form/property-field-common';
import { TAG_KEY_VALIDATION } from '~/validators/validators';

export const PROPERTY_KEY_FIELD_NAME = 'key';

export const PropertyKeyField = connectVocabulary(
    ({ vocabulary }: VocabularyProp) =>
        <Field
            name={PROPERTY_KEY_FIELD_NAME}
            component={PropertyKeyInput}
            vocabulary={vocabulary}
            validate={getValidation(vocabulary)} />);

const PropertyKeyInput = ({ vocabulary, ...props }: WrappedFieldProps & VocabularyProp) =>
    <Autocomplete
        label='Key'
        suggestions={getSuggestions(props.input.value, vocabulary)}
        {...buildProps(props)}
    />;

const getValidation = memoize(
    (vocabulary: Vocabulary) =>
        vocabulary.strict
            ? [...TAG_KEY_VALIDATION, matchTags(vocabulary)]
            : TAG_KEY_VALIDATION);

const matchTags = (vocabulary: Vocabulary) =>
    (value: string) =>
        getTagsList(vocabulary).find(tag => tag.includes(value))
            ? undefined
            : 'Incorrect key';

const getSuggestions = (value: string, vocabulary: Vocabulary) =>
    getTagsList(vocabulary).filter(tag => tag.includes(value) && tag !== value);

const getTagsList = ({ tags }: Vocabulary) =>
    Object.keys(tags);
