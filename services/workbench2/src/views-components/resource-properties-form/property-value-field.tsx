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
import { TAG_VALUE_VALIDATION, REQUIRED_LENGTH255_VALIDATION } from 'validators/validators';
import { escapeRegExp } from 'common/regexp';
import { ChangeEvent } from 'react';
import { memoize } from 'lodash';
import { useStateWithValidation } from 'common/useStateWithValidation';
import { Validator } from 'validators/validators';

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
            validate={skipValidation ? undefined : getValidation(props.propertyKeyId, props.vocabulary)}
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
            renderSuggestion={
                (s: PropFieldSuggestion) => s.synonyms && s.synonyms.length > 0
                    ? `${s.label} (${s.synonyms.join('; ')})`
                    : s.label
            }
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

type DialogPropertyValueInputProps = VocabularyProp & {
    showErrors?: boolean,
    skipValidation?: boolean,
    propertyKeyId: string,
    clearValueSignal?: {},
    onSelect: (value: string) => void,
    setValueErrors: (errors: string[]) => void,
};

export const DialogPropertyValueInput = ({ vocabulary, propertyKeyId, showErrors, skipValidation, clearValueSignal, onSelect, setValueErrors }: DialogPropertyValueInputProps) => {
    const validationArray = skipValidation ? [] : getValueValidation(propertyKeyId, vocabulary);
    const [value, setValue, valueErrs] = useStateWithValidation('', validationArray, 'Value');

    React.useEffect(() => {
            setValue('');
    }, [clearValueSignal]);

    React.useEffect(() => {
        setValueErrors(valueErrs);
    }, [valueErrs]);

    return <Autocomplete
        label='Value'
        items={[]}
        value={value}
        error={showErrors && valueErrs.length > 0}
        helperText={showErrors ? valueErrs.join(', ') : undefined}
        disabled={!propertyKeyId}
        suggestions={getSuggestions(value, propertyKeyId, vocabulary)}
        renderSuggestion={
            (s: PropFieldSuggestion) => s.synonyms && s.synonyms.length > 0
                ? `${s.label} (${s.synonyms.join('; ')})`
                : s.label
        }
        onSelect={(selectedSuggestion: PropFieldSuggestion) => {
            onSelect(selectedSuggestion.label);
            setValue(selectedSuggestion.label);
        }}
        onBlur={() => {
            // Case-insensitive search for the value in the vocabulary
            const foundValueID = getTagValueID(propertyKeyId, value, vocabulary);
            if (foundValueID !== '') {
                setValue(getTagValueLabel(propertyKeyId, foundValueID, vocabulary));
            }
        }}
        onChange={(e: ChangeEvent<HTMLInputElement>) => {
            const newValue = e.currentTarget.value;
            setValue(newValue);
            if (vocabulary.tags[propertyKeyId] && vocabulary.tags[propertyKeyId].strict === false) {
                onSelect(newValue);
            }
        }}
    />
};

/**
 * getValidation must be memoized to prevent infinite re-renders due to Field
 * checking it for changes
 */
const getValidation = memoize((propertyKeyId: string, vocabulary: Vocabulary) =>
    isStrictTag(propertyKeyId, vocabulary)
        ? [...TAG_VALUE_VALIDATION, matchTagValues(propertyKeyId, vocabulary)]
        : TAG_VALUE_VALIDATION);

const matchTagValues = (propertyKeyId: string, vocabulary: Vocabulary) =>
    (value: string) =>
        getTagValues(propertyKeyId, vocabulary).find(v => !value || v.label === value)
            ? undefined
            : 'Incorrect value';

const createStrictValueValidator = (propertyKeyId: string, vocabulary: Vocabulary): Validator => {
    const validValues = getTagValues(propertyKeyId, vocabulary).map(value => value.label);
    const validValueSet = new Set(validValues);

    return ((value: string) =>
        validValueSet.has(value) ? undefined : 'Incorrect value'
    ) as Validator;
};

const getValueValidation = (propertyKeyId: string, vocabulary: Vocabulary) => {
    if (isStrictTag(propertyKeyId, vocabulary)) {
        return [...REQUIRED_LENGTH255_VALIDATION, createStrictValueValidator(propertyKeyId, vocabulary)];
    }
    return REQUIRED_LENGTH255_VALIDATION;
};

const getSuggestions = (value: string, tagName: string, vocabulary: Vocabulary) => {
    const re = new RegExp(escapeRegExp(value), "i");
    return getPreferredTagValues(tagName, vocabulary, value).filter(
        val => (val.label !== value && re.test(val.label)) ||
            (val.synonyms && val.synonyms.some(s => re.test(s))));
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
