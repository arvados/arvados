// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { memoize } from 'lodash/fp';
import { FloatCommandInputParameter, isRequiredInput } from 'models/workflow';
import { Field } from 'redux-form';
import { isNumber } from 'validators/is-number';
import { GenericInputProps, GenericInput } from './generic-input';
import { FloatInput as FloatInputComponent } from 'components/float-input/float-input';
export interface FloatInputProps {
    input: FloatCommandInputParameter;
}
export const FloatInput = ({ input }: FloatInputProps) =>
    <Field
        name={input.id}
        commandInput={input}
        component={Input}
        parse={parseFloat}
        format={format}
        validate={getValidation(input)} />;

const format = (value: any) => isNaN(value) ? '' : JSON.stringify(value);

const getValidation = memoize(
    (input: FloatCommandInputParameter) => ([
        isRequiredInput(input)
            ? isNumber
            : () => undefined,])
);

const Input = (props: GenericInputProps) =>
    <GenericInput
        component={InputComponent}
        {...props} />;

const InputComponent = ({ input, meta, commandInput }: GenericInputProps) =>
    <FloatInputComponent
        fullWidth
        error={meta.touched && !!meta.error}
        disabled={commandInput.disabled}
        {...input} />;

