// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { memoize } from 'lodash/fp';
import { IntCommandInputParameter, isRequiredInput } from 'models/workflow';
import { Field } from 'redux-form';
import { isInteger } from 'validators/is-integer';
import { GenericInputProps, GenericInput } from 'views/run-process-panel/inputs/generic-input';
import { IntInput as IntInputComponent } from 'components/int-input/int-input';

export interface IntInputProps {
    input: IntCommandInputParameter;
}
export const IntInput = ({ input }: IntInputProps) =>
    <Field
        name={input.id}
        commandInput={input}
        component={InputComponent}
        parse={parse}
        format={format}
        validate={getValidation(input)} />;

export const parse = (value: any) => value === '' ? '' : parseInt(value, 10);

export const format = (value: any) => isNaN(value) ? '' : JSON.stringify(value);

const getValidation = memoize(
    (input: IntCommandInputParameter) => ([
        isRequiredInput(input)
            ? isInteger
            : () => undefined,
    ]));

const InputComponent = (props: GenericInputProps) =>
    <GenericInput
        component={Input}
        {...props} />;


const Input = (props: GenericInputProps) =>
    <IntInputComponent
        fullWidth
        type='number'
        error={props.meta.touched && !!props.meta.error}
        disabled={props.commandInput.disabled}
        {...props.input} />;

