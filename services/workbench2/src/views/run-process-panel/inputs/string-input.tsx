// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { memoize } from 'lodash/fp';
import { isRequiredInput, StringCommandInputParameter } from 'models/workflow';
import { Field } from 'redux-form';
import { require } from 'validators/require';
import { GenericInputProps, GenericInput } from 'views/run-process-panel/inputs/generic-input';
import { Input as MaterialInput } from '@material-ui/core';

export interface StringInputProps {
    input: StringCommandInputParameter;
}
export const StringInput = ({ input }: StringInputProps) =>
    <Field
        name={input.id}
        commandInput={input}
        component={StringInputComponent}
        validate={getValidation(input)} />;

const getValidation = memoize(
    (input: StringCommandInputParameter) => ([
        isRequiredInput(input)
            ? require
            : () => undefined,
    ]));

const StringInputComponent = (props: GenericInputProps) =>
    <GenericInput
        component={Input}
        {...props} />;

const Input = (props: GenericInputProps) =>
    <MaterialInput
        fullWidth
        error={props.meta.touched && !!props.meta.error}
        disabled={props.commandInput.disabled}
        {...props.input} />;