// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { IntCommandInputParameter, getInputLabel, isRequiredInput } from '~/models/workflow';
import { Field } from 'redux-form';
import { isInteger } from '~/validators/is-integer';
import { GenericInputProps, GenericInput } from '~/views/run-process-panel/inputs/generic-input';
import { Input as MaterialInput } from '@material-ui/core';

export interface IntInputProps {
    input: IntCommandInputParameter;
}
export const IntInput = ({ input }: IntInputProps) =>
    <Field
        name={input.id}
        commandInput={input}
        component={IntInputComponent}
        parse={value => parseInt(value, 10)}
        format={value => isNaN(value) ? '' : JSON.stringify(value)}
        validate={[
            isRequiredInput(input)
                ? isInteger
                : () => undefined,
        ]} />;

const IntInputComponent = (props: GenericInputProps) =>
    <GenericInput
        component={Input}
        {...props} />;


const Input = (props: GenericInputProps) =>
    <MaterialInput type='number' {...props.input} />;

