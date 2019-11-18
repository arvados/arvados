// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { WrappedFieldProps, Field, FormName } from 'redux-form';
import { memoize } from 'lodash';
import { Autocomplete } from '~/components/autocomplete/autocomplete';
import { Vocabulary, getTags, getTagKeyID } from '~/models/vocabulary';
import { handleSelect, handleBlur, connectVocabulary, VocabularyProp, buildProps } from '~/views-components/resource-properties-form/property-field-common';
import { TAG_KEY_VALIDATION } from '~/validators/validators';
import { escapeRegExp } from '~/common/regexp.ts';

export const PROPERTY_KEY_FIELD_NAME = 'key';
export const PROPERTY_KEY_FIELD_ID = 'keyID';

interface PropertyKeyFieldProps {
    skipValidation?: boolean;
}

export const PropertyKeyField = connectVocabulary(
    ({ vocabulary, skipValidation }: VocabularyProp & PropertyKeyFieldProps) =>
        <Field
            name={PROPERTY_KEY_FIELD_NAME}
            component={PropertyKeyInput}
            vocabulary={vocabulary}
            validate={skipValidation ? undefined : getValidation(vocabulary)} />
);

const PropertyKeyInput = ({ vocabulary, ...props }: WrappedFieldProps & VocabularyProp) =>
    <FormName children={data => (
        <Autocomplete
            label='Key'
            suggestions={getSuggestions(props.input.value, vocabulary)}
            onSelect={handleSelect(PROPERTY_KEY_FIELD_ID, data.form, props.input, props.meta)}
            onBlur={handleBlur(PROPERTY_KEY_FIELD_ID, data.form, props.meta, props.input, getTagKeyID(props.input.value, vocabulary))}
            {...buildProps(props)}
        />
    )}/>;

const getValidation = memoize(
    (vocabulary: Vocabulary) =>
        vocabulary.strict_tags
            ? [...TAG_KEY_VALIDATION, matchTags(vocabulary)]
            : TAG_KEY_VALIDATION);

const matchTags = (vocabulary: Vocabulary) =>
    (value: string) =>
        getTags(vocabulary).find(tag => tag.label === value)
            ? undefined
            : 'Incorrect key';

const getSuggestions = (value: string, vocabulary: Vocabulary) => {
    const re = new RegExp(escapeRegExp(value), "i");
    return getTags(vocabulary).filter(tag => re.test(tag.label) && tag.label !== value);
};
