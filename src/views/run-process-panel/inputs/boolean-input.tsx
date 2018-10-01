// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { BooleanCommandInputParameter, getInputLabel, isRequiredInput } from '~/models/workflow';
import { Field, WrappedFieldProps } from 'redux-form';
import { TextField } from '~/components/text-field/text-field';
import { FormGroup, FormLabel, FormHelperText, Switch } from '@material-ui/core';

export interface BooleanInputProps {
    input: BooleanCommandInputParameter;
}
export const BooleanInput = ({ input }: BooleanInputProps) =>
    <Field
        name={input.id}
        label={getInputLabel(input)}
        component={BooleanInputComponent}
        normalize={(value, prevValue) => !prevValue}
    />;

const BooleanInputComponent = (props: WrappedFieldProps & { label?: string }) =>
    <FormGroup>
        <FormLabel>{props.label}</FormLabel>
        <Switch
            color='primary'
            checked={props.input.value}
            onChange={() => props.input.onChange(props.input.value)} />
    </FormGroup>;