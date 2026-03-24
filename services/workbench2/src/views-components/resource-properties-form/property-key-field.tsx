// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { WrappedFieldProps, Field, FormName, reset, change, WrappedFieldInputProps, WrappedFieldMetaProps } from 'redux-form';
import { Autocomplete } from 'components/autocomplete/autocomplete';
import {
    Vocabulary,
    getTags,
    getTagKeyID,
    getTagKeyLabel,
    getPreferredTags,
    PropFieldSuggestion
} from 'models/vocabulary';
import {
    handleSelect,
    handleBlur,
    connectVocabulary,
    VocabularyProp,
    ValidationProp,
    buildProps
} from 'views-components/resource-properties-form/property-field-common';
import { TAG_KEY_VALIDATION, REQUIRED_LENGTH255_VALIDATION, Validator } from 'validators/validators';
import { escapeRegExp } from 'common/regexp';
import { ChangeEvent } from 'react';
import { useStateWithValidation } from 'common/useStateWithValidation';

export const PROPERTY_KEY_FIELD_NAME = 'key';
export const PROPERTY_KEY_FIELD_ID = 'keyID';

export const PropertyKeyField = connectVocabulary(
    ({ vocabulary, skipValidation, clearPropertyKeyOnSelect }: VocabularyProp & ValidationProp) =>
        <span data-cy='property-field-key'>
        <Field
            clearPropertyKeyOnSelect
            name={PROPERTY_KEY_FIELD_NAME}
            component={PropertyKeyInput}
            vocabulary={vocabulary}
            validate={skipValidation ? undefined : getValidation(vocabulary)} />
        </span>
);

const PropertyKeyInput = ({ vocabulary, ...props }: WrappedFieldProps & VocabularyProp & { clearPropertyKeyOnSelect?: boolean }) =>
    <FormName children={data => (
        <Autocomplete
            {...buildProps(props)}
            label='Key'
            suggestions={getSuggestions(props.input.value, vocabulary)}
            renderSuggestion={
                (s: PropFieldSuggestion) => s.synonyms && s.synonyms.length > 0
                    ? `${s.label} (${s.synonyms.join('; ')})`
                    : s.label
            }
            onFocus={() => {
                if (props.clearPropertyKeyOnSelect && props.input.value) {
                    props.meta.dispatch(reset(props.meta.form));
                }
            }}
            onSelect={handleSelect(PROPERTY_KEY_FIELD_ID, data.form, props.input, props.meta)}
            onBlur={() => {
                // Case-insensitive search for the key in the vocabulary
                const foundKeyID = getTagKeyID(props.input.value, vocabulary);
                if (foundKeyID !== '') {
                    props.input.value = getTagKeyLabel(foundKeyID, vocabulary);
                }
                handleBlur(PROPERTY_KEY_FIELD_ID, data.form, props.meta, props.input, foundKeyID)();
            }}
            onChange={(e: ChangeEvent<HTMLInputElement>) => {
                const newValue = e.currentTarget.value;
                handleChange(data.form, props.input, props.meta, newValue);
            }}
        />
    )} />;

const handleChange = (
    formName: string,
    { onChange }: WrappedFieldInputProps,
    { dispatch }: WrappedFieldMetaProps,
    value: string) => {
        // Properties' values are dependant on the keys, if any value is
        // pre-existant, a change on the property key should mean that the
        // previous value is invalid, so we better reset the whole form before
        // setting the new tag key.
        dispatch(reset(formName));

        onChange(value);
        dispatch(change(formName, PROPERTY_KEY_FIELD_NAME, value));
    };

type DialogPropertyKeyInputProps = VocabularyProp & {
    showErrors?: boolean
    skipValidation?: boolean,
    clearPropertyKeyOnSelect?: boolean,
    setCurrentValue?: (value: string | undefined) => void,
    onSelect: (value: string) => void,
    setKeyErrors: (errors: string[]) => void,
};

export const DialogPropertyKeyInput = ({ vocabulary, showErrors, skipValidation, clearPropertyKeyOnSelect, onSelect, setKeyErrors, setCurrentValue }: DialogPropertyKeyInputProps) => {
    const validationArray = skipValidation ? [] : getKeyValidation(vocabulary);
    const [key, setKey, keyErrs] = useStateWithValidation('', validationArray, 'Key');

    // report errors to parent component
    React.useEffect(() => {
        setKeyErrors(keyErrs);
    }, [keyErrs]);

    const handleSetKey = (newKey: string) => {
        if (setCurrentValue) {
            setCurrentValue(undefined);
        }
        setKey(newKey);
        onSelect(newKey);
    }

    return <Autocomplete
        label='Key'
        items={[]}
        value={key}
        error={showErrors && keyErrs.length > 0}
        helperText={showErrors ? keyErrs.join(', ') : undefined}
        suggestions={getSuggestions(key, vocabulary)}
        renderSuggestion={
            (s: PropFieldSuggestion) => s.synonyms && s.synonyms.length > 0
                ? `${s.label} (${s.synonyms.join('; ')})`
                : s.label
        }
        onFocus={() => {
            setKey('');
            onSelect('');
            if (clearPropertyKeyOnSelect && key && setCurrentValue) {
                setCurrentValue(undefined);
            }
        }}
        onSelect={(selectedSuggestion: PropFieldSuggestion) => {
            handleSetKey(selectedSuggestion.label);
        }}
        onBlur={() => {
            // Case-insensitive search for the key in the vocabulary
            const foundKeyID = getTagKeyID(key, vocabulary);
            if (foundKeyID !== '') {
                const foundKeyLabel = getTagKeyLabel(foundKeyID, vocabulary);
                handleSetKey(foundKeyLabel);
            }
        }}
        onChange={(e: ChangeEvent<HTMLInputElement>) => {
            const newValue = e.currentTarget.value;
            if (vocabulary.strict_tags === false) {
                handleSetKey(newValue);
            } else {
                setKey(newValue);
            }
        }}
    />
};

const getValidation =
    (vocabulary: Vocabulary) =>
        vocabulary.strict_tags
            ? [...TAG_KEY_VALIDATION, matchTags(vocabulary)]
            : TAG_KEY_VALIDATION

const createStrictTagValidator = (vocabulary: Vocabulary): Validator => {
    const validTags = getTags(vocabulary).map(tag => tag.label);
    const validTagSet = new Set(validTags);

    return ((value: string) =>
        validTagSet.has(value) ? undefined : 'Incorrect key'
    ) as Validator;
};

const getKeyValidation = (vocabulary: Vocabulary) => {
    if (vocabulary.strict_tags) {
        return [...REQUIRED_LENGTH255_VALIDATION, createStrictTagValidator(vocabulary)];
    }
    return REQUIRED_LENGTH255_VALIDATION;
}

const matchTags = (vocabulary: Vocabulary) =>
    (value: string) =>
        getTags(vocabulary).find(tag => tag.label === value)
            ? undefined
            : 'Incorrect key';

const getSuggestions = (value: string, vocabulary: Vocabulary): PropFieldSuggestion[] => {
    const re = new RegExp(escapeRegExp(value), "i");
    return getPreferredTags(vocabulary, value).filter(
        tag => (tag.label !== value && re.test(tag.label)) ||
            (tag.synonyms && tag.synonyms.some(s => re.test(s))));
};
