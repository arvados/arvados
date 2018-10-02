// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import * as React from 'react';
import { EnumCommandInputParameter, CommandInputEnumSchema } from '~/models/workflow';
import { Field } from 'redux-form';
import { Select, MenuItem } from '@material-ui/core';
import { GenericInputProps, GenericInput } from './generic-input';

export interface EnumInputProps {
    input: EnumCommandInputParameter;
}
export const EnumInput = ({ input }: EnumInputProps) =>
    <Field
        name={input.id}
        commandInput={input}
        component={EnumInputComponent}
    />;

const EnumInputComponent = (props: GenericInputProps) =>
    <GenericInput
        component={Input}
        {...props} />;

const Input = (props: GenericInputProps) => {
    const type = props.commandInput.type as CommandInputEnumSchema;
    return <Select
        value={props.input.value}
        onChange={props.input.onChange}>
        {type.symbols.map(symbol =>
            <MenuItem key={symbol} value={symbol.split('/').pop()}>
                {symbol.split('/').pop()}
            </MenuItem>)}
    </Select>;
};