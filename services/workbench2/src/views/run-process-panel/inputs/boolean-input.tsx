// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React from 'react';
import { memoize } from 'lodash/fp';
import { BooleanCommandInputParameter } from 'models/workflow';
import { Field } from 'redux-form';
import { Switch } from '@material-ui/core';
import { GenericInputProps, GenericInput } from './generic-input';

export interface BooleanInputProps {
    input: BooleanCommandInputParameter;
}
export const BooleanInput = ({ input }: BooleanInputProps) =>
    <Field
        name={input.id}
        commandInput={input}
        component={BooleanInputComponent}
        normalize={normalize}
    />;

const normalize = (_: any, prevValue: boolean) => !prevValue;

const BooleanInputComponent = (props: GenericInputProps) =>
    <GenericInput
        component={Input}
        {...props} />;

const Input = ({ input, commandInput }: GenericInputProps) =>
    <Switch
        color='primary'
        checked={input.value}
        onChange={handleChange(input.onChange, input.value)}
        disabled={commandInput.disabled} />;

const handleChange = memoize(
    (onChange: (value: string) => void, value: string) => () => onChange(value)
);
