// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { BooleanCommandInputParameter } from '~/models/workflow';
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
        normalize={(value, prevValue) => !prevValue}
    />;

const BooleanInputComponent = (props: GenericInputProps) =>
    <GenericInput
        component={Input}
        {...props} />;

const Input = (props: GenericInputProps) =>
    <Switch
        color='primary'
        checked={props.input.value}
        onChange={() => props.input.onChange(props.input.value)} />;